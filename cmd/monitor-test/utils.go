package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"reflect"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/metachris/eth-reorg-monitor/monitor"
	"github.com/metachris/eth-reorg-monitor/reorgutils"
)

var Client *ethclient.Client
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
	Monitor = monitor.NewReorgMonitor(Client, nick, true)
	fmt.Println(Monitor.String())
}

func AddBlockAndPrintNewline(blocks ...*types.Block) {
	for _, block := range blocks {
		Monitor.AddBlock(block)
		fmt.Println("")
	}
}

func ReorgCheckAndPrint() map[uint64]*monitor.Reorg {
	fmt.Println("\n---\n ")
	reorgs, _ := Monitor.CheckForReorgs()
	fmt.Println("\n---\n ")

	fmt.Println(Monitor.String())
	for k, reorg := range reorgs {
		fmt.Printf("reorg at block %d with depth %d: blocks %d - %d\n", k, reorg.Depth, reorg.StartBlockHeight, reorg.EndBlockHeight)
		fmt.Println(len(reorg.BlocksInvolved), "blocks involved:")
		for _, block := range reorg.BlocksInvolved {
			fmt.Printf("- %d %s\n", block.NumberU64(), block.Hash())
		}
	}
	fmt.Println("")
	return reorgs
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
