package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/flashbots/reorg-monitor/analysis"
	"github.com/flashbots/reorg-monitor/database"
	"github.com/flashbots/reorg-monitor/monitor"
	flashbotsrpc "github.com/metachris/flashbots-rpc"
)

var saveToDb = false
var simulateBlocksWithMevGeth = false

var Reorgs map[string]*analysis.Reorg = make(map[string]*analysis.Reorg)
var db *database.DatabaseService
var rpc *flashbotsrpc.FlashbotsRPC
var callBundlePrivKey, _ = crypto.GenerateKey()

var ColorGreen = "\033[1;32m%s\033[0m"

var (
	version = "dev" // is set during build process

	// default values
	defaultEthNodes    = os.Getenv("ETH_NODES")
	defaultDebug       = os.Getenv("DEBUG") == "1"
	defaultListenAddr  = os.Getenv("LISTEN_ADDR")
	defaultSimBlocks   = os.Getenv("SIM_BLOCKS") == "1"
	defaultSimURI      = os.Getenv("MEVGETH_NODE")
	defaultPostgresDSN = os.Getenv("POSTGRES_DSN")

	// cli flags
	ethUriPtr      = flag.String("eth", defaultEthNodes, "One or more geth node URIs for subscription (comma separated)")
	debugPtr       = flag.Bool("debug", defaultDebug, "print RPC call debug information")
	httpAddrPtr    = flag.String("http", defaultListenAddr, "http service address")
	mevGethSimPtr  = flag.Bool("sim", defaultSimBlocks, "simulate blocks in mev-geth")
	mevGethUriPtr  = flag.String("mevgeth", defaultSimURI, "mev-geth node URI")
	postgresDSNPtr = flag.String("postgres", defaultPostgresDSN, "postgres DSN")
)

func main() {
	log.SetOutput(os.Stdout)
	flag.Parse()

	log.Printf("reorg-monitor %s", version)

	ethUris := []string{}
	if *ethUriPtr != "" {
		ethUris = strings.Split(*ethUriPtr, ",")
	} else {
		// Try parsing ETH_NODE_* env vars
		for _, entry := range os.Environ() {
			if strings.HasPrefix(entry, "ETH_NODE_") {
				ethUris = append(ethUris, strings.Split(entry, "=")[1])
			}
		}
	}
	if len(ethUris) == 0 {
		log.Fatal("Missing eth node uri")
	} else {
		log.Println("eth nodes:", ethUris)
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

	if *postgresDSNPtr != "" {
		saveToDb = true
		db = database.NewDatabaseService(*postgresDSNPtr)
		fmt.Println("Connected to database at", "uri", strings.Split(*postgresDSNPtr, "@")[1])
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

	if *httpAddrPtr != "" {
		fmt.Printf("Starting webserver on %s\n", *httpAddrPtr)
		ws := monitor.NewMonitorWebserver(mon, *httpAddrPtr)
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
		err := db.AddReorgEntry(entry)
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
