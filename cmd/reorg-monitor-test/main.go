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
	var err error
	log.SetOutput(os.Stdout)

	ethUriPtr := flag.String("eth", os.Getenv("ETH_NODES"), "Geth node URI")
	flag.Parse()

	ethUris := reorgutils.EthUrisFromString(*ethUriPtr)
	if len(ethUris) == 0 {
		log.Fatal("Missing eth node uri")
	}

	// Connect a geth client to fetch the custom blocks
	_, err = testutils.ConnectClient(ethUris[0])
	reorgutils.Perror(err)

	// Create a new monitor instance
	mon = monitor.NewReorgMonitor(ethUris, make(chan *analysis.Reorg), true)
	err = mon.ConnectClients()
	reorgutils.Perror(err)

	// https://mermaid-js.github.io/mermaid-live-editor/view/#eyJjb2RlIjoic3RhdGVEaWFncmFtLXYyXG4gICAgMHhkMzE0NTMyNjgyZWM1Mjc3MzMxYjg1N2E2OGJmMjIwM2I1MjBhNGZkMWM4NWJlNzViMDMyOTY1OGYxOTY2YjAyIC0tPiAweDdkZTk2OWNkNGI4M2MwYTdmN2E0MjMxYmY2MDY0N2MyNDBiYzE2YTAxZmRmZGNhYjE1ZWU4Zjg3MjA1MDFjOTRcbiAgICAweGQzMTQ1MzI2ODJlYzUyNzczMzFiODU3YTY4YmYyMjAzYjUyMGE0ZmQxYzg1YmU3NWIwMzI5NjU4ZjE5NjZiMDIgLS0-IDB4MGFhMmMzYWI4MmQ4ODI3ZWUzYTIyNTIxZjQyNDE5NDFmYzdhZTAwZjVkZjk5NGNlNmY4OGZhZGQyNDRkZjQwYlxuICAgIDB4N2RlOTY5Y2Q0YjgzYzBhN2Y3YTQyMzFiZjYwNjQ3YzI0MGJjMTZhMDFmZGZkY2FiMTVlZThmODcyMDUwMWM5NCAtLT4gMHhhOTdjY2MwM2U4NmUzODVkYzRiOWJjMDNhZWU5OTc2YWQ5ODliYWZmMTExN2JhYmZhNDdhNWFlZjg3ZTg2ODAzXG4gICAgMHgwYWEyYzNhYjgyZDg4MjdlZTNhMjI1MjFmNDI0MTk0MWZjN2FlMDBmNWRmOTk0Y2U2Zjg4ZmFkZDI0NGRmNDBiIC0tPiAweGY5MTAxMzA0NDQ1YWViYzFkMmNlNmFlYzczNWM0YWU1OTY5Y2Q1YjQ1YmUxNzYzNzZiNjgwMmE2MTAwYjc1YzNcbiAgICAweGU2NWEwMTYzZWNkODI5YWVkODkwZWZkMDg4NWIwMGM3Yzg4NjEwZGVmYzFkYjZiNTU4NTRlYjc3M2RkOTg1N2UgLS0-IDB4ZDMxNDUzMjY4MmVjNTI3NzMzMWI4NTdhNjhiZjIyMDNiNTIwYTRmZDFjODViZTc1YjAzMjk2NThmMTk2NmIwMlxuICAgIDB4ZTY1YTAxNjNlY2Q4MjlhZWQ4OTBlZmQwODg1YjAwYzdjODg2MTBkZWZjMWRiNmI1NTg1NGViNzczZGQ5ODU3ZSAtLT4gMHhhODVmZjhlYzc2YWRjYWE4MGE0ZjhjNzkzMzUyYWYwNmY0M2ZjNWZlZTY3ZjE5NjFkOTkyODgyMjg0ODA3YzUxXG4gICAgMHhmOTEwMTMwNDQ0NWFlYmMxZDJjZTZhZWM3MzVjNGFlNTk2OWNkNWI0NWJlMTc2Mzc2YjY4MDJhNjEwMGI3NWMzIC0tPiAweGYzMGYwMjRkZjc5NzE0N2ZmNjFiMDZhNzRmMGM1NjllNzg0MGExYjRkOTc1ODMxNmUzOGJiYmQ5MDhiMzdlYjgiLCJtZXJtYWlkIjoie1xuICBcInRoZW1lXCI6IFwiZGVmYXVsdFwiXG59IiwidXBkYXRlRWRpdG9yIjp0cnVlLCJhdXRvU3luYyI6dHJ1ZSwidXBkYXRlRGlhZ3JhbSI6dHJ1ZX0
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
	fmt.Println("")
	analysis, err := mon.AnalyzeTree(0, 0)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Print tree and result
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
