package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/metachris/eth-reorg-monitor/analysis"
	"github.com/metachris/eth-reorg-monitor/database"
	"github.com/metachris/eth-reorg-monitor/monitor"
	"github.com/metachris/eth-reorg-monitor/reorgutils"
	flashbotsrpc "github.com/metachris/flashbots-rpc"
)

var saveToDb = false
var simulateBlocksWithMevGeth = false

var Reorgs map[string]*analysis.Reorg = make(map[string]*analysis.Reorg)
var db *database.DatabaseService
var rpc *flashbotsrpc.FlashbotsRPC
var callBundlePrivKey, _ = crypto.GenerateKey()

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

	ethUriPtr := flag.String("eth", os.Getenv("ETH_NODES"), "Geth node URIs (comma separated)")
	debugPtr := flag.Bool("debug", false, "print debug information")
	saveToDbPtr := flag.Bool("db", false, "save reorgs to database")

	webserverPortPtr := flag.Int("webserver", 8094, "port for the webserver (0 to disable)")

	mevGethSimPtr := flag.Bool("sim", false, "simulate blocks in mev-geth")
	mevGethUriPtr := flag.String("mevgeth", os.Getenv("MEVGETH_NODE"), "mev-geth node URI")
	flag.Parse()

	ethUris := reorgutils.EthUrisFromString(*ethUriPtr)
	if len(ethUris) == 0 {
		log.Fatal("Missing eth node uri")
	}

	if *mevGethSimPtr {
		if *mevGethUriPtr == "" {
			log.Fatal("Missing mevgeth node uri")
		}

		simulateBlocksWithMevGeth = true
		rpc = flashbotsrpc.NewFlashbotsRPC(*mevGethUriPtr)
		rpc.Debug = *debugPtr
		fmt.Printf("Using mev-geth node at %s for simulations\n", *mevGethUriPtr)
	}

	if *saveToDbPtr {
		saveToDb = *saveToDbPtr
		dbCfg := getDbConfig()
		db = database.NewDatabaseService(dbCfg)
		fmt.Println("Connected to database at", dbCfg.Host)
	}

	// Channel to receive reorgs from monitor
	reorgChan := make(chan *analysis.Reorg)

	// Setup and start the monitor
	mon := monitor.NewReorgMonitor(ethUris, reorgChan, true)
	err := mon.ConnectClients()
	reorgutils.Perror(err)
	go mon.SubscribeAndListen()

	if *webserverPortPtr > 0 {
		fmt.Printf("Starting webserver on port %d\n", *webserverPortPtr)
		ws := monitor.NewMonitorWebserver(mon, *webserverPortPtr)
		go ws.ListenAndServe()
	}

	// Wait for reorgs
	for reorg := range reorgChan {
		handleReorg(reorg)
	}
}

func handleReorg(reorg *analysis.Reorg) {
	_, found := Reorgs[reorg.Id()]
	if found {
		return
	}

	// new reorg: remember and print
	Reorgs[reorg.Id()] = reorg

	log.Println(reorg.String())
	fmt.Println("- common parent:", reorg.CommonParent.Hash)
	fmt.Println("- first block after:", reorg.FirstBlockAfterReorg.Hash)
	for chainKey, chain := range reorg.Chains {
		if chainKey == reorg.MainChainHash {
			fmt.Printf("- mainchain l=%d: ", len(chain))
		} else {
			fmt.Printf("- sidechain l=%d: ", len(chain))
		}

		for _, block := range chain {
			fmt.Printf("%s ", block.Hash)
		}
		fmt.Print("\n")
	}

	if saveToDb {
		entry := database.NewReorgEntry(reorg)
		err := db.AddReorgEntry(entry)
		if err != nil {
			log.Printf("error at db.AddReorgEntry: %+v\n", err)
		}

		for _, block := range reorg.BlocksInvolved {
			blockEntry := database.NewBlockEntry(block, reorg)

			// If block has no transactions, then it has 0 miner value (no need to simulate)
			if len(block.Block.Transactions()) == 0 {
				blockEntry.MevGeth_CoinbaseDiffWei = "0"
				blockEntry.MevGeth_GasFeesWei = "0"
				blockEntry.MevGeth_EthSentToCoinbaseWei = "0"
				blockEntry.MevGeth_CoinbaseDiffEth = "0.000000"
				blockEntry.MevGeth_EthSentToCoinbase = "0.000000"

			} else if simulateBlocksWithMevGeth { // simulate the block now
				res, err := rpc.FlashbotsSimulateBlock(callBundlePrivKey, block.Block, 0)
				if err != nil {
					log.Println("error: sim failed of block", block.Hash, "-", err)
				} else {
					fmt.Printf("- sim of block %s: CoinbaseDiff=%20s, GasFees=%20s, EthSentToCoinbase=%20s\n", block.Hash, res.CoinbaseDiff, res.GasFees, res.EthSentToCoinbase)
					blockEntry.UpdateWitCallBundleResponse(res)
				}
			}

			err := db.AddBlockEntry(blockEntry)
			if err != nil {
				log.Println("error at db.AddBlockEntry:", err)
			}
		}
	}

	if reorg.NumReplacedBlocks > 1 {
		fmt.Println(reorg.MermaidSyntax())
	}
	fmt.Println("")
}
