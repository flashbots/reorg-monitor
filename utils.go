package main

import (
	"math"
	"math/big"
	"os"
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
