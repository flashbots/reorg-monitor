package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/metachris/eth-reorg-monitor/monitor"
)

var reorgs map[string]*monitor.Reorg = make(map[string]*monitor.Reorg)

func main() {
	log.SetOutput(os.Stdout)

	ethUriPtr := flag.String("eth", os.Getenv("ETH_NODE"), "Geth node URI")
	debugPtr := flag.Bool("debug", false, "print debug information")
	flag.Parse()

	if *ethUriPtr == "" {
		log.Fatal("Missing eth node uri")
	}

	// Handle reorgs from many monitors
	reorgChan := make(chan *monitor.Reorg)
	go func() {
		for reorg := range reorgChan {
			handleReorg(reorg)
		}
	}()

	// Start a monitor
	mon := monitor.NewReorgMonitor(*ethUriPtr, "eth1", *debugPtr)
	mon.SubscribeAndStart(reorgChan)
}

func handleReorg(reorg *monitor.Reorg) {
	_, found := reorgs[reorg.Id()]
	if found {
		return
	}

	// new reorg
	reorgs[reorg.Id()] = reorg
	fmt.Println("xx new reorg:", reorg)

	// Todo:
	// - Get coinbase diff
	// - Save to database
}
