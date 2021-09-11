// Test various aspects of the reorg monitor based on custom block inputs (block height and hash)
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/metachris/eth-reorg-monitor/analysis"
	"github.com/metachris/eth-reorg-monitor/monitor"
	"github.com/metachris/eth-reorg-monitor/reorgutils"
	"github.com/metachris/eth-reorg-monitor/testutils"
)

var mon *monitor.ReorgMonitor

func main() {
	log.SetOutput(os.Stdout)

	ethUriPtr := flag.String("eth", os.Getenv("ETH_NODES"), "Geth node URI")
	flag.Parse()

	ethUris := reorgutils.EthUrisFromString(*ethUriPtr)
	if len(ethUris) == 0 {
		log.Fatal("Missing eth node uri")
	}

	// Connect a geth client to fetch the custom blocks
	_, err := testutils.ConnectClient(ethUris[0])
	reorgutils.Perror(err)

	// Create a new monitor instance (and connect its geth clients to fetch further blocks if required)
	mon = monitor.NewReorgMonitor(ethUris, make(chan *analysis.Reorg), true)
	numConnectedClients := mon.ConnectClients()
	if numConnectedClients == 0 {
		log.Fatal("could not connect to any clients")
	}

	CheckReorg([]string{
		"0xd690a9bb26d27d665768f67b8839ea6571a1e10c95ad69e38dac16d9686f5d9f",
		"0xc24323315f3735017026f754b3f6ee1bca3cdbb8e3f6b2476dd09aec4e40cb38",
		"0x1eba1d62712b43067cb8f913e682190eabf2dd020a4e954fdd4460a6099f09a7",
	})
}

// blockIds can include block numbers and hashes
func CheckReorg(blockIds []string) {
	// Add the blocks
	for _, ethBlock := range testutils.BlocksForStrings(blockIds) {
		block := analysis.NewBlock(ethBlock, analysis.OriginSubscription, testutils.EthNodeUri)
		mon.AddBlock(block)
	}

	// Analyze
	analysis, err := mon.AnalyzeTree(0, 0)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Print tree and result
	fmt.Println("")
	analysis.Tree.Print()
	fmt.Println("")
	analysis.Print()
}

func TestAndVerify(testCase testutils.TestCase) {
	// Create a new monitor
	testutils.ResetMon()

	// Add the blocks
	for _, ethBlock := range testutils.BlocksForStrings(testCase.BlockInfo) {
		block := analysis.NewBlock(ethBlock, analysis.OriginSubscription, testutils.EthNodeUri)
		testutils.Monitor.AddBlock(block)
	}

	// reorgs := testutils.ReorgCheckAndPrint()
	// testutils.Pcheck("NumReorgs", len(reorgs), 1)

	// reorg := reorgs[0]
	// testutils.Pcheck("StartBlock", reorg.StartBlockHeight, testCase.ExpectedResult.StartBlock)
	// testutils.Pcheck("EndBlock", reorg.EndBlockHeight, testCase.ExpectedResult.EndBlock)
	// testutils.Pcheck("Depth", reorg.Depth, testCase.ExpectedResult.Depth)
	// testutils.Pcheck("NumBlocks", len(reorg.BlocksInvolved), testCase.ExpectedResult.NumBlocks)
	// testutils.Pcheck("NumReplacedBlocks", reorg.NumReplacedBlocks, testCase.ExpectedResult.NumReplacedBlocks)

	// if testCase.ExpectedResult.MustBeLive {
	// 	testutils.Pcheck("MustBeLive", reorg.SeenLive, true)
	// }

	// fmt.Println(reorg.MermaidSyntax())
}
