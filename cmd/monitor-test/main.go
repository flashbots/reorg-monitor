package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/metachris/eth-reorg-monitor/reorgutils"
)

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
	Test(Test_DoubleReorgD3)
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
	Pcheck("NumChains", len(reorg.Segments), testCase.ExpectedResult.NumChains)

	hasMainChain := false
	for _, segment := range reorg.Segments {
		if segment.IsMainChain {
			hasMainChain = true
		}
	}
	Pcheck("HasMainChain", hasMainChain, true)

	fmt.Println("All check passed")
}

func TestBlockWith1Uncle() {
	// 12996762 0xa98aa57369c7c1ba75a88b86d5ddd685cf77d5e25f0521e3ee77cff81b67ede3 uncles: 1

	ResetMon("uncles:1")

	AddBlockAndPrintNewline(
		GetBlockByHashStr("0x64f5389fbb9c97ca2d164a034d6168449a27eb7c91d4e51b1a2cfbee3ad810a2"),
		GetBlockByHashStr("0xc7079a2686e92351812928407eca33d1d6f4884bdc37d018df978ec95aedb9ca"),
		GetBlockByHashStr("0xa98aa57369c7c1ba75a88b86d5ddd685cf77d5e25f0521e3ee77cff81b67ede3"), // block with 1 uncle
		GetBlockByHashStr("0xeb818c21ad41d1db9c64ee9dbdecd4ada0a2b86c753fa16c5802a6bf79f8c6ff"),
	)

	ReorgCheckAndPrint()
}
