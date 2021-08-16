package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/metachris/eth-reorg-monitor/database"
	"github.com/metachris/eth-reorg-monitor/monitor"
	flashbotsrpc "github.com/metachris/flashbots-rpc"
)

var saveToDb = false
var simulateBlocksWithMevGeth = false

var Reorgs map[string]*monitor.Reorg = make(map[string]*monitor.Reorg)
var db *database.DatabaseService
var rpc *flashbotsrpc.FlashbotsRPC

var ColorGreen = "\033[1;32m%s\033[0m"

func getDbConfig() database.PostgresConfig {
	return database.PostgresConfig{
		User:       os.Getenv("DB_USER"),
		Password:   os.Getenv("DB_PASS"),
		Host:       os.Getenv("DB_HOST"),
		Name:       os.Getenv("DB_NAME"),
		DisableTLS: len(os.Getenv("DB_DISABLE_TLS")) > 0,
	}
}

func main() {
	log.SetOutput(os.Stdout)

	ethUriPtr := flag.String("eth", os.Getenv("ETH_NODE"), "Geth node URI")
	debugPtr := flag.Bool("debug", false, "print debug information")
	saveToDbPtr := flag.Bool("db", false, "save reorgs to database")

	mevGethSimPtr := flag.Bool("sim", false, "simulate blocks in mev-geth")
	mevGethUriPtr := flag.String("mevgeth", os.Getenv("MEVGETH_NODE"), "mev-geth node URI")
	flag.Parse()

	if *ethUriPtr == "" {
		log.Fatal("Missing eth node uri")
	}

	if *mevGethSimPtr {
		if *mevGethUriPtr == "" {
			log.Fatal("Missing mevgeth node uri")
		}

		simulateBlocksWithMevGeth = true
		rpc = flashbotsrpc.NewFlashbotsRPC(*mevGethUriPtr)
		rpc.Debug = *debugPtr
	}

	if *saveToDbPtr {
		saveToDb = *saveToDbPtr
		dbCfg := getDbConfig()
		db = database.NewDatabaseService(dbCfg)
		fmt.Println("Connected to database at", dbCfg.Host)
	}

	// Handle reorgs from many monitors
	reorgChan := make(chan *monitor.Reorg)
	go func() {
		for reorg := range reorgChan {
			handleReorg(reorg)
		}
	}()

	// Start a monitor
	mon := monitor.NewReorgMonitor(*ethUriPtr, *debugPtr, true)
	mon.SubscribeAndStart(reorgChan)
}

func handleReorg(reorg *monitor.Reorg) {
	_, found := Reorgs[reorg.Id()]
	if found {
		return
	}

	// new reorg
	Reorgs[reorg.Id()] = reorg

	log.Println(reorg)
	fmt.Println("- mainchain:", strings.Join(reorg.GetMainChainHashes(), ", "))
	fmt.Println("- discarded:", strings.Join(reorg.GetReplacedBlockHashes(), ", "))

	if reorg.Depth > 1 {
		fmt.Println(reorg.MermaidSyntax())
	}

	if saveToDb {
		entry := database.NewReorgEntry(reorg)
		err := db.AddReorgEntry(entry)
		if err != nil {
			log.Println("err at db.AddReorgEntry:", err)
		}
	}

	if simulateBlocksWithMevGeth {
		privateKey, _ := crypto.GenerateKey()

		for _, block := range reorg.BlocksInvolved {
			res, err := rpc.FlashbotsSimulateBlock(privateKey, block, 0)
			if err != nil {
				log.Println("error: sim failed of block", block.Hash(), "-", err)
			} else {
				fmt.Printf("- sim of block %s: CoinbaseDiff=%s, GasFees=%s, EthSentToCoinbase=%s\n", block.Hash(), res.CoinbaseDiff, res.GasFees, res.EthSentToCoinbase)
			}

			if saveToDb {
				response := &res
				if err != nil {
					response = nil
				}
				blockEntry := database.NewBlockEntry(block, reorg, response)
				db.AddBlockEntry(blockEntry)
			}
		}
	}
}
