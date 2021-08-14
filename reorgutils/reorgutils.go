package reorgutils

import (
	"fmt"
	"math"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
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