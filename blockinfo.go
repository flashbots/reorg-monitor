package main

import (
	"math/big"
	"strconv"

	"github.com/ethereum/go-ethereum/core/types"
)

// BlockInfo is used to cache
type BlockInfo struct {
	Block              *types.Block
	CoinbaseDifference *big.Int
	IsReorged          bool
	IsUncle            bool
	IsChild            bool
	ReorgDepth         int // 1 if uncle, 2 if first child, etc
}

func NewBlockInfoFromBlock(block *types.Block, earnings *big.Int) *BlockInfo {
	return &BlockInfo{
		Block:              block,
		CoinbaseDifference: earnings,
		IsReorged:          false,
		IsUncle:            false,
		IsChild:            false,
		ReorgDepth:         0,
	}
}

func (b *BlockInfo) ToCsvRecord() []string {
	return []string{
		b.Block.Number().String(),
		b.Block.Hash().String(),
		b.Block.ParentHash().String(),
		strconv.FormatUint(b.Block.Time(), 10),
		b.Block.Coinbase().String(),

		b.Block.Difficulty().String(),
		strconv.Itoa(len(b.Block.Uncles())),
		strconv.Itoa(len(b.Block.Transactions())),

		strconv.FormatBool(b.IsReorged),
		strconv.FormatBool(b.IsUncle),
		strconv.FormatBool(b.IsChild),
		strconv.Itoa(b.ReorgDepth),

		BalanceToEthStr(b.CoinbaseDifference),
	}
}

var BlockInfoCsvRecordHeader []string = []string{
	"block_number",
	"block_hash",
	"parent_hash",
	"block_timestamp",
	"coinbase_address",

	"difficulty",
	"num_uncles",
	"num_tx",

	"isReorged",
	"isUncle",
	"isChild",
	"reorg_depth",

	"coinbase_diff",
}
