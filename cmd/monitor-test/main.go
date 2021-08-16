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

	// Test(Test_1Uncle)
	// Test(Test_2Uncles)
	// Test(Test_ReorgD2)
	// Test(Test_DoubleReorgD3)
	// Test(Test_ReorgD2B5)
	Test(Test_13033424_13033425_d2_b5)
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
	Pcheck("NumChains", len(reorg.ChainSegments), testCase.ExpectedResult.NumChains)

	hasMainChain := false
	for _, segment := range reorg.ChainSegments {
		if segment.IsMainChain {
			hasMainChain = true
		}
	}
	Pcheck("HasMainChain", hasMainChain, true)

	if testCase.ExpectedResult.MustBeLive {
		Pcheck("MustBeLive", reorg.SeenLive, true)
	}

	fmt.Println("All check passed")

	fmt.Println(reorg.MermaidSyntax())
}
