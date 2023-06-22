package reorgutils

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/big"
	"os"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

func Perror(err error) {
	if err != nil {
		panic(err)
	}
}

func FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func BalanceToEth(balance *big.Int) *big.Float {
	fbalance := new(big.Float)
	fbalance.SetInt(balance)
	// fbalance.SetString(balance)
	ethValue := new(big.Float).Quo(fbalance, big.NewFloat(math.Pow10(18)))
	return ethValue
}

func BalanceToEthStr(balance *big.Int) string {
	if balance == nil {
		return "nil"
	}
	return BalanceToEth(balance).Text('f', 4)
}

func SprintBlock(block *types.Block) string {
	t := time.Unix(int64(block.Time()), 0).UTC()
	return fmt.Sprintf("Block %s %s \t %s \t tx: %3d, uncles: %d", block.Number(), block.Hash(), t, len(block.Transactions()), len(block.Uncles()))
}

func WeiToEth(wei *big.Int) (ethValue *big.Float) {
	// wei / 10^18
	fbalance := new(big.Float)
	fbalance.SetString(wei.String())
	ethValue = new(big.Float).Quo(fbalance, big.NewFloat(1e18))
	return
}

var ColorGreen = "\033[1;32m%s\033[0m"

func ColorPrintf(color, format string, a ...interface{}) {
	str := fmt.Sprintf(format, a...)
	fmt.Printf(string(color), str)
}

func GetBlocks(blockChan chan<- *types.Block, client *ethclient.Client, startBlock, endBlock int64, concurrency int) {
	var blockWorkerWg sync.WaitGroup
	blockHeightChan := make(chan int64, 100) // blockHeight to fetch with receipts

	// Start eth client thread pool
	for w := 1; w <= concurrency; w++ {
		blockWorkerWg.Add(1)

		// Worker gets a block height from blockHeightChan, downloads it, and puts it in the blockChan
		go func() {
			defer blockWorkerWg.Done()
			for blockHeight := range blockHeightChan {
				// fmt.Println(blockHeight)
				block, err := client.BlockByNumber(context.Background(), big.NewInt(blockHeight))
				if err != nil {
					log.Println("Error getting block:", blockHeight, err)
					continue
				}
				blockChan <- block
			}
		}()
	}

	// Push blocks into channel, for workers to pick up
	for currentBlockNumber := startBlock; currentBlockNumber <= endBlock; currentBlockNumber++ {
		blockHeightChan <- currentBlockNumber
	}

	// Close worker channel and wait for workers to finish
	close(blockHeightChan)
	blockWorkerWg.Wait()
}
