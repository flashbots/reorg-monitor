package main

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

type EarningsService struct {
	client                   *ethclient.Client
	MinerEarningsByBlockHash map[common.Hash]*big.Int
}

func NewEarningsService(client *ethclient.Client) *EarningsService {
	return &EarningsService{
		client:                   client,
		MinerEarningsByBlockHash: make(map[common.Hash]*big.Int),
	}
}

func (es *EarningsService) GetBlockCoinbaseEarningsWithoutCache(block *types.Block) (*big.Int, error) {
	balanceAfterBlock, err := es.client.BalanceAt(context.Background(), block.Coinbase(), block.Number())
	if err != nil {
		return nil, err
	}

	balanceBeforeBlock, err := es.client.BalanceAt(context.Background(), block.Coinbase(), new(big.Int).Sub(block.Number(), common.Big1))
	if err != nil {
		return nil, err
	}

	earnings := new(big.Int).Sub(balanceAfterBlock, balanceBeforeBlock)

	// Iterate over all transactions - add sent value back into earnings, remove received value
	for _, tx := range block.Transactions() {
		from, err := types.Sender(types.LatestSignerForChainID(tx.ChainId()), tx)
		if err != nil {
			fmt.Println("getsender error", err, tx.Hash())
			continue
		}

		if from == block.Coinbase() {
			earnings = new(big.Int).Add(earnings, tx.Value())
		}

		to := tx.To()
		if to != nil && *to == block.Coinbase() {
			earnings = new(big.Int).Sub(earnings, tx.Value())
		}
	}

	return earnings, nil
}

func (es *EarningsService) GetBlockCoinbaseEarnings(block *types.Block) (*big.Int, error) {
	var err error

	earnings, found := es.MinerEarningsByBlockHash[block.Hash()]
	if found {
		return earnings, nil
	}

	earnings, err = es.GetBlockCoinbaseEarningsWithoutCache(block)
	if err != nil {
		return nil, err
	}

	es.MinerEarningsByBlockHash[block.Hash()] = earnings
	return earnings, nil
}
