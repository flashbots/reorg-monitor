package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/flashbots/reorg-monitor/analysis"
	"github.com/flashbots/reorg-monitor/database"
	"github.com/flashbots/reorg-monitor/monitor"
	"github.com/flashbots/reorg-monitor/reorgutils"
	flashbotsrpc "github.com/metachris/flashbots-rpc"
)

var saveToDb = false
var simulateBlocksWithMevGeth = false

var Reorgs map[string]*analysis.Reorg = make(map[string]*analysis.Reorg)
var db *database.DatabaseService
var rpc *flashbotsrpc.FlashbotsRPC
var callBundlePrivKey, _ = crypto.GenerateKey()

var ColorGreen = "\033[1;32m%s\033[0m"

func main() {
	log.SetOutput(os.Stdout)

	ethUriPtr := flag.String("eth", os.Getenv("ETH_NODES"), "One or more geth node URIs for subscription (comma separated)")
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
		dbCfg := database.GetDbConfig()
		db = database.NewDatabaseService(dbCfg)
		fmt.Println("Connected to database at", dbCfg.Host)
	}

	// Start healthcheck pings
	go healthPing()

	// Channel to receive reorgs from monitor
	reorgChan := make(chan *analysis.Reorg)

	// Setup and start the monitor
	mon := monitor.NewReorgMonitor(ethUris, reorgChan, true)
	numConnectedClients := mon.ConnectClients()
	if numConnectedClients == 0 {
		log.Fatal("could not connect to any clients")
	}

	if *webserverPortPtr > 0 {
		fmt.Printf("Starting webserver on port %d\n", *webserverPortPtr)
		ws := monitor.NewMonitorWebserver(mon, *webserverPortPtr)
		go ws.ListenAndServe()
	}

	// In the background, subscribe to new blocks and listen for updates
	go mon.SubscribeAndListen()

	// Wait for reorgs
	for reorg := range reorgChan {
		handleReorg(reorg)
	}
}

func healthPing() {
	url := os.Getenv("URL_HC_PING")
	if url == "" {
		return
	}
	for {
		http.Get(url)
		time.Sleep(1 * time.Minute)
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
	fmt.Println("- common parent:    ", reorg.CommonParent.Hash)
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
		_, err := db.AddReorgEntry(entry)
		if err != nil {
			log.Printf("error at db.AddReorgEntry: %+v\n", err)
		}

		for _, block := range reorg.BlocksInvolved {
			blockEntry := database.NewBlockEntry(block, reorg)

			// If block has no transactions, then it has 0 miner value (no need to simulate)
			if simulateBlocksWithMevGeth && len(block.Block.Transactions()) > 0 {
				res, err := rpc.FlashbotsSimulateBlock(callBundlePrivKey, block.Block, 0)
				if err != nil {
					log.Println("error: sim failed of block", block.Hash, "-", err)
				} else {
					fmt.Printf("- sim of block %s: CoinbaseDiff=%20s, GasFees=%20s, EthSentToCoinbase=%20s\n", block.Hash, res.CoinbaseDiff, res.GasFees, res.EthSentToCoinbase)
					blockEntry.UpdateWitCallBundleResponse(res)
				}
			}

			_, err := db.AddBlockEntry(blockEntry)
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
