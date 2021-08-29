package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/metachris/eth-reorg-monitor/analysis"
	"github.com/metachris/eth-reorg-monitor/reorgutils"
	"github.com/metachris/eth-reorg-monitor/testutils"
)

func main() {
	log.SetOutput(os.Stdout)

	ethUriPtr := flag.String("eth", os.Getenv("ETH_NODES"), "Geth node URI")
	flag.Parse()

	ethUris := reorgutils.EthUrisFromString(*ethUriPtr)
	if len(ethUris) == 0 {
		log.Fatal("Missing eth node uri")
	}

	testutils.EthNodeUri = ethUris[0]
	reorgutils.Perror(testutils.ConnectClient(testutils.EthNodeUri))

	// CheckReorg(testutils.Test_13033424_13033425_d2_b5)
	// CheckReorg(testutils.TestD1and2)
	// CheckReorg(testutils.Test3xD1)
	CheckReorg(testutils.TestD3)

	// Test(testutils.Test_12996760_12996760_d1_b2)
	// Test(testutils.Test_12996750_12996750_d1_b3)
	// Test(testutils.Test_12991732_12991733_d2_b4)
	// Test(testutils.Test_12969887_12969889_d3_b6)
	// Test(testutils.Test_13017535_13017536_d2_b5)
	// Test(testutils.Test_13018369_13018370_d2_b4)
}

func CheckReorg(testCase testutils.TestCase) {
	testutils.ResetMon(testCase.Name)

	// Add the blocks
	for _, ethBlock := range testutils.BlocksForStrings(testCase.BlockInfo) {
		block := analysis.NewBlock(ethBlock, analysis.OriginSubscription, testutils.EthNodeUri)
		testutils.Monitor.AddBlock(block)
	}

	fmt.Println("")
	analysis, err := testutils.Monitor.AnalyzeTree(100, 0)
	if err != nil {
		fmt.Println(err)
		return
	}

	analysis.Tree.Print()
	fmt.Println("")
	analysis.Print()

	// testutils.ReorgCheckAndPrint()
}

func TestAndVerify(testCase testutils.TestCase) {
	// Create a new monitor
	testutils.ResetMon(testCase.Name)

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
