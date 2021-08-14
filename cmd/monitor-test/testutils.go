package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"reflect"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/metachris/eth-reorg-monitor/monitor"
	"github.com/metachris/eth-reorg-monitor/reorgutils"
)

var Client *ethclient.Client
var EthNodeUri string
var Monitor *monitor.ReorgMonitor

func ConnectClient(uri string) error {
	// Connect to geth node
	var err error
	fmt.Printf("Connecting to %s...", uri)
	Client, err = ethclient.Dial(uri)
	if err != nil {
		return err
	}
	fmt.Printf(" ok\n")
	return nil
}

func ResetMon(nick string) {
	Monitor = monitor.NewReorgMonitor(EthNodeUri, false, true)
	// fmt.Println(Monitor.String())
}

func BlocksForStrings(blockStrings []string) (ret []*types.Block) {
	ret = make([]*types.Block, len(blockStrings))
	for i, blockStr := range blockStrings {
		if len(blockStr) < 10 {
			blockNum, err := strconv.Atoi(blockStr)
			reorgutils.Perror(err)
			ret[i] = GetBlockByNumber(int64(blockNum))
		} else {
			ret[i] = GetBlockByHashStr(blockStr)
		}
	}
	return ret
}

func AddBlockAndPrintNewline(blocks ...*types.Block) {
	for _, block := range blocks {
		Monitor.AddBlock(block, "test")
		// fmt.Println("")
	}
}

func ReorgCheckAndPrint() (ret []*monitor.Reorg) {
	ret = make([]*monitor.Reorg, 0)

	fmt.Println("\n---\n ")
	reorgs, _, _ := Monitor.CheckForReorgs(100, 0)
	// fmt.Println("\n---\n ")

	fmt.Println("Summary for", Monitor.String())
	fmt.Println("")
	for _, reorg := range reorgs {
		fmt.Println(reorg)
		reorg.PrintSegments()
		// for i, segment := range reorg.Segments {
		// 	fmt.Printf("- segment %d: %s - %s\n", i, segment, strings.Join(segment.BlockHashes(), ", "))
		// }
		ret = append(ret, reorg)
	}
	fmt.Println("")
	return ret
}

type ReorgTestResult struct {
	StartBlock uint64
	EndBlock   uint64
	Depth      uint64
	NumBlocks  int
	NumChains  int
}

func GetBlockByHashStr(hashStr string) *types.Block {
	hash := common.HexToHash(hashStr)
	block, err := Client.BlockByHash(context.Background(), hash)
	reorgutils.Perror(err)
	return block
}

func GetBlockByNumber(number int64) *types.Block {
	block, err := Client.BlockByNumber(context.Background(), big.NewInt(number))
	reorgutils.Perror(err)
	return block
}

func Assert(condition bool, errorMsg string) {
	if !condition {
		log.Fatal(errorMsg)
	}
}

func Check(f string, got, want interface{}) error {
	if !reflect.DeepEqual(got, want) {
		return fmt.Errorf("%s mismatch: got %v, want %v", f, got, want)
	}
	return nil
}

func Pcheck(f string, got, want interface{}) {
	err := Check(f, got, want)
	if err != nil {
		log.Fatal(err)
	}
}
