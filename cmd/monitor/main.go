package main

import (
	"flag"
	"log"
	"os"

	"github.com/metachris/eth-reorg-monitor/monitor"
)

var Reorgs map[string]*monitor.Reorg = make(map[string]*monitor.Reorg)
var ColorGreen = "\033[1;32m%s\033[0m"

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
	_, found := Reorgs[reorg.Id()]
	if found {
		return
	}

	// new reorg
	Reorgs[reorg.Id()] = reorg

	log.Printf(ColorGreen, reorg)

	// Todo:
	// - Get coinbase diff
	// - Save to database
}