package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/metachris/eth-reorg-monitor/reorgutils"
)

type TestCase struct {
	Name           string
	BlockInfo      []string
	ExpectedResult ReorgTestResult
}

func main() {
	log.SetOutput(os.Stdout)

	ethUriPtr := flag.String("eth", os.Getenv("ETH_NODE"), "Geth node URI")
	flag.Parse()

	if *ethUriPtr == "" {
		log.Fatal("Missing eth node uri")
	}

	EthNodeUri = *ethUriPtr
	reorgutils.Perror(ConnectClient(*ethUriPtr))

	// Test(Test_12996760_12996760_d1_b2)
	// Test(Test_12996750_12996750_d1_b3)
	Test(Test_12991732_12991733_d2_b4)
	// Test(Test_12969887_12969889_d3_b6)
	// Test(Test_13017535_13017536_d2_b5)
	// Test(Test_13018369_13018370_d2_b4)
	// Test(Test_13033424_13033425_d2_b5)
}

func Test(testCase TestCase) {
	// Create a new monitor
	ResetMon(testCase.Name)

	// Add the blocks
	for _, block := range BlocksForStrings(testCase.BlockInfo) {
		Monitor.AddBlock(block, "test")
	}

	reorgs := ReorgCheckAndPrint()
	Pcheck("NumReorgs", len(reorgs), 1)

	reorg := reorgs[0]
	Pcheck("StartBlock", reorg.StartBlockHeight, testCase.ExpectedResult.StartBlock)
	Pcheck("EndBlock", reorg.EndBlockHeight, testCase.ExpectedResult.EndBlock)
	Pcheck("Depth", reorg.Depth, testCase.ExpectedResult.Depth)
	Pcheck("NumBlocks", len(reorg.BlocksInvolved), testCase.ExpectedResult.NumBlocks)
	Pcheck("NumReplacedBlocks", reorg.NumReplacedBlocks, testCase.ExpectedResult.NumReplacedBlocks)

	if testCase.ExpectedResult.MustBeLive {
		Pcheck("MustBeLive", reorg.SeenLive, true)
	}

	fmt.Println(reorg.MermaidSyntax())
}
