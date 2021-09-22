package testutils

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
	"github.com/flashbots/reorg-monitor/analysis"
	"github.com/flashbots/reorg-monitor/monitor"
	"github.com/flashbots/reorg-monitor/reorgutils"
)

var Client *ethclient.Client
var EthNodeUri string
var Monitor *monitor.ReorgMonitor

func ConnectClient(uri string) (client *ethclient.Client, err error) {
	EthNodeUri = uri

	// Connect to geth node
	fmt.Printf("Connecting to %s...", uri)
	Client, err = ethclient.Dial(uri)
	if err != nil {
		return nil, err
	}
	fmt.Printf(" ok\n")
	return Client, nil
}

func ResetMon() {
	reorgChan := make(chan *analysis.Reorg)
	Monitor = monitor.NewReorgMonitor([]string{EthNodeUri}, reorgChan, true)
	numConnectedClients := Monitor.ConnectClients()
	if numConnectedClients == 0 {
		log.Fatal("could not connect to any clients")
	}
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

func ReorgCheckAndPrint() {
	fmt.Println("\n---\n ")
	analysis, err := Monitor.AnalyzeTree(100, 0)
	if err != nil {
		fmt.Println(err)
		return
	}

	analysis.Tree.Print()
	fmt.Println("")
	analysis.Print()
}

func GetBlockByHashStr(hashStr string) *types.Block {
	hash := common.HexToHash(hashStr)
	block, err := Client.BlockByHash(context.Background(), hash)
	if err != nil {
		log.Fatalf("GetBlockByHashStr couldn't find block %s: %v\n", hash, err)
	}
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
