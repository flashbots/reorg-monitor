package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/metachris/eth-reorg-monitor/monitor"
	"github.com/metachris/eth-reorg-monitor/reorgutils"
)

var client *ethclient.Client

func main() {
	var err error
	log.SetOutput(os.Stdout)

	ethUriPtr := flag.String("eth", os.Getenv("ETH_NODE"), "Geth node URI")
	flag.Parse()

	if *ethUriPtr == "" {
		log.Fatal("Missing eth node uri")
	}

	// Connect to geth node
	fmt.Printf("Connecting to %s...", *ethUriPtr)
	client, err = ethclient.Dial(*ethUriPtr)
	reorgutils.Perror(err)
	fmt.Printf(" ok\n")

	TestBlockWithUncles()
}

func GetBlockByHash(hashStr string) *types.Block {
	hash := common.HexToHash(hashStr)
	block, err := client.BlockByHash(context.Background(), hash)
	reorgutils.Perror(err)
	return block
}

func GetBlockByNumber(number int64) *types.Block {
	block, err := client.BlockByNumber(context.Background(), big.NewInt(number))
	reorgutils.Perror(err)
	return block
}

func TestBlockWithUncles() {
	// 12996762 0xa98aa57369c7c1ba75a88b86d5ddd685cf77d5e25f0521e3ee77cff81b67ede3 uncles: 1
	// 12996751 0xe6b8556d6b8be89721c8fee126e7e66bb615057164b20562b69717bec6e24841 uncles: 2

	// Monitor
	mon := monitor.NewReorgMonitor(client, "eth1", true)
	fmt.Println(mon.String())
	// mon.SubscribeAndStart()

	// 1 Uncle
	// block_12996760 := GetBlock("0x64f5389fbb9c97ca2d164a034d6168449a27eb7c91d4e51b1a2cfbee3ad810a2")
	// block_12996761 := GetBlock("0xc7079a2686e92351812928407eca33d1d6f4884bdc37d018df978ec95aedb9ca")
	// block_12996762 := GetBlock("0xa98aa57369c7c1ba75a88b86d5ddd685cf77d5e25f0521e3ee77cff81b67ede3") // block with 1 uncle
	// block_12996763 := GetBlock("0xeb818c21ad41d1db9c64ee9dbdecd4ada0a2b86c753fa16c5802a6bf79f8c6ff")

	addBlockAndNewline := func(blocks ...*types.Block) {
		for _, block := range blocks {
			mon.AddBlock(block)
			fmt.Println("")
		}
	}

	addBlockAndNewline(
		GetBlockByNumber(12996748),
		GetBlockByNumber(12996749),
		GetBlockByNumber(12996750),
		GetBlockByNumber(12996751), // 2 uncles
		GetBlockByNumber(12996752),
		GetBlockByNumber(12996753),
	)

	fmt.Println("\n---\n ")
	reorgs := mon.CheckForReorgs()
	fmt.Println("\n---\n ")

	for i, reorg := range reorgs {
		fmt.Printf("reorg #%d: started at height %d, depth: %d, end: %d\n", i+1, reorg.StartBlockHeight, reorg.Depth, reorg.EndBlockHeight)
	}
}

// func TestReorg1() {
// 	// Monitor
// 	mon := monitor.NewReorgMonitor(client, "eth1", true)
// 	fmt.Println(mon.String())
// 	// mon.SubscribeAndStart()

// 	BlockM2_12969885 := GetBlockByHash("0x81f00ccfe1be19abf8772eb909398ee2d3581ab34a872cf501e3114aaf817da0")
// 	BlockM1_12969886 := GetBlockByHash("0xc8c4ed6507d118168f1901236aa43d073006cd8c612dc78e9c6b333df29e7639")
// 	_, _ = BlockM2_12969885, BlockM1_12969886

// 	BlockC1_12969887 := GetBlockByHash("0x7b5c2c2a5b31d7436b3e541f44a868c2cafa837e629b267dbfab749fd6c9e3d4")
// 	BlockC1_12969888 := GetBlockByHash("0x647411d024c0b007556f80e48cf4ec02c601c37584e00b9e6c8ec08b8f2b0252")
// 	BlockC1_12969889 := GetBlockByHash("0xae396e35c045b8603de015e182ce1349c579c68bb00396bfb8a7b5946a4fa87c")
// 	_, _, _ = BlockC1_12969887, BlockC1_12969888, BlockC1_12969889

// 	BlockC2_12969887 := GetBlockByHash("0x7f31fa36b85af1b5ec33a68c4fb82e395eb8123dba66fa43322656f043e80576")
// 	BlockC2_12969888 := GetBlockByHash("0x698d17e71f65006661f45af5f5fbea50fd1bcbc7236fe5e37255d727675148b8")
// 	BlockC2_12969889 := GetBlockByHash("0xdca194ddb314c1c4e3de10ccfcb88bf9183a78118a393e1b3860e5eb10dd7c6c")
// 	BlockC2_12969890 := GetBlockByHash("0x235387014a35db9a67ccf89793cb6f0da53888b7813b4adeba3ec1c992c96d8c")
// 	_, _, _, _ = BlockC2_12969887, BlockC2_12969888, BlockC2_12969889, BlockC2_12969890

// 	mon.AddBlock(BlockM2_12969885)
// 	mon.AddBlock(BlockC1_12969887)
// }
