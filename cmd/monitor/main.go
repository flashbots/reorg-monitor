package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/metachris/eth-reorg-monitor/database"
	"github.com/metachris/eth-reorg-monitor/monitor"
)

var Reorgs map[string]*monitor.Reorg = make(map[string]*monitor.Reorg)
var saveToDb = false
var db *database.DatabaseService

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
	flag.Parse()

	if *ethUriPtr == "" {
		log.Fatal("Missing eth node uri")
	}

	if *saveToDbPtr {
		saveToDb = *saveToDbPtr
		db = database.NewDatabaseService(getDbConfig())
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
		db.AddReorgEntry(entry)
	}

	// Todo:
	// - Get coinbase diff
	// - Save to database
}
