package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/metachris/eth-reorg-monitor/monitor"
	"github.com/metachris/eth-reorg-monitor/reorgutils"
)

func main() {
	log.SetOutput(os.Stdout)

	ethUriPtr := flag.String("eth", os.Getenv("ETH_NODE"), "Geth node URI")
	flag.Parse()

	if *ethUriPtr == "" {
		log.Fatal("Missing eth node uri")
	}

	// Connect to geth node
	fmt.Printf("Connecting to %s...", *ethUriPtr)
	client, err := ethclient.Dial(*ethUriPtr)
	reorgutils.Perror(err)
	fmt.Printf(" ok\n")

	// Monitor
	mon := monitor.NewReorgMonitor(client, "eth1", true)
	fmt.Println(mon.String())
	mon.SubscribeAndStart()
}
