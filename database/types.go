package database

import (
	"database/sql"
	"math/big"

	"github.com/flashbots/reorg-monitor/analysis"
	"github.com/flashbots/reorg-monitor/reorgutils"
	_ "github.com/lib/pq"
	"github.com/metachris/flashbotsrpc"
)

var Schema = `
CREATE TABLE IF NOT EXISTS reorg_summary (
    Id integer GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
	Created_At timestamp NOT NULL default current_timestamp,

    Key      VARCHAR (40) NOT NULL UNIQUE,
    SeenLive boolean NOT NULL,

    StartBlockNumber   integer NOT NULL,
    EndBlockNumber     integer NOT NULL,
    Depth              integer NOT NULL,
    NumChains          integer NOT NULL,
    NumBlocksInvolved  integer NOT NULL,
    NumBlocksReplaced  integer NOT NULL,
    MermaidSyntax text NOT NULL
);

CREATE TABLE IF NOT EXISTS reorg_block (
    Id integer GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
	Created_At timestamp NOT NULL default current_timestamp,

    Reorg_Key VARCHAR (40) REFERENCES reorg_summary (Key) NOT NULL,

    Origin        VARCHAR (20) NOT NULL,
    NodeUri       text NOT NULL,

    BlockNumber     integer NOT NULL,
    BlockHash       text NOT NULL,
    ParentHash      text NOT NULL,
    BlockTimestamp  integer NOT NULL,
    CoinbaseAddress text NOT NULL,

    Difficulty bigint  NOT NULL,
    NumUncles  integer NOT NULL,
    NumTx      integer NOT NULL,

    IsPartOfReorg boolean NOT NULL,
    IsMainChain   boolean NOT NULL,
    IsFirst       boolean NOT NULL,

    MevGeth_CoinbaseDiffWei      NUMERIC(48, 0),
    MevGeth_GasFeesWei           NUMERIC(48, 0),
    MevGeth_EthSentToCoinbaseWei NUMERIC(48, 0),

    MevGeth_CoinbaseDiffEth      VARCHAR(10),
    MevGeth_EthSentToCoinbase    VARCHAR(10)
);
`

type ReorgEntry struct {
	Id         int
	Created_At sql.NullTime

	Key      string
	SeenLive bool

	StartBlockNumber  uint64
	EndBlockNumber    uint64
	Depth             int
	NumChains         int
	NumBlocksInvolved int
	NumBlocksReplaced int
	MermaidSyntax     string
}

func NewReorgEntry(reorg *analysis.Reorg) ReorgEntry {
	return ReorgEntry{
		Key:               reorg.Id(),
		SeenLive:          reorg.SeenLive,
		StartBlockNumber:  reorg.StartBlockHeight,
		EndBlockNumber:    reorg.EndBlockHeight,
		Depth:             reorg.Depth,
		NumChains:         len(reorg.Chains),
		NumBlocksInvolved: len(reorg.BlocksInvolved),
		NumBlocksReplaced: reorg.NumReplacedBlocks,
		MermaidSyntax:     reorg.MermaidSyntax(),
	}
}

type BlockEntry struct {
	Id         int
	Created_At sql.NullTime

	Reorg_Key string
	Origin    string
	NodeUri   string

	BlockNumber     uint64
	BlockHash       string
	ParentHash      string
	BlockTimestamp  uint64
	CoinbaseAddress string

	Difficulty uint64
	NumUncles  int
	NumTx      int

	IsPartOfReorg bool
	IsMainChain   bool
	IsFirst       bool

	MevGeth_CoinbaseDiffWei      string
	MevGeth_GasFeesWei           string
	MevGeth_EthSentToCoinbaseWei string

	MevGeth_CoinbaseDiffEth   string
	MevGeth_EthSentToCoinbase string
}

func NewBlockEntry(block *analysis.Block, reorg *analysis.Reorg) BlockEntry {
	_, isPartOfReorg := reorg.BlocksInvolved[block.Hash]
	_, isMainChain := reorg.MainChainBlocks[block.Hash]

	blockEntry := BlockEntry{
		Reorg_Key: reorg.Id(),
		Origin:    string(block.Origin),
		NodeUri:   block.NodeUri,

		BlockNumber:     block.Number,
		BlockHash:       block.Hash.String(),
		ParentHash:      block.ParentHash.String(),
		BlockTimestamp:  block.Block.Time(),
		CoinbaseAddress: block.Block.Coinbase().String(),

		Difficulty: block.Block.Difficulty().Uint64(),
		NumUncles:  len(block.Block.Uncles()),
		NumTx:      len(block.Block.Transactions()),

		IsPartOfReorg: isPartOfReorg,
		IsMainChain:   isMainChain,
		IsFirst:       block.Number == reorg.StartBlockHeight,

		MevGeth_CoinbaseDiffWei:      "0",
		MevGeth_GasFeesWei:           "0",
		MevGeth_EthSentToCoinbaseWei: "0",

		MevGeth_CoinbaseDiffEth:   "0.000000",
		MevGeth_EthSentToCoinbase: "0.000000",
	}

	return blockEntry
}

func (e *BlockEntry) UpdateWitCallBundleResponse(callBundleResponse flashbotsrpc.FlashbotsCallBundleResponse) {
	coinbaseDiffWei := new(big.Int)
	coinbaseDiffWei.SetString(callBundleResponse.CoinbaseDiff, 10)
	coinbaseDiffEth := reorgutils.WeiToEth(coinbaseDiffWei)

	ethSentToCoinbaseWei := new(big.Int)
	ethSentToCoinbaseWei.SetString(callBundleResponse.EthSentToCoinbase, 10)
	ethSentToCoinbase := reorgutils.WeiToEth(ethSentToCoinbaseWei)

	e.MevGeth_CoinbaseDiffWei = callBundleResponse.CoinbaseDiff
	e.MevGeth_GasFeesWei = callBundleResponse.GasFees
	e.MevGeth_EthSentToCoinbaseWei = callBundleResponse.EthSentToCoinbase

	e.MevGeth_CoinbaseDiffEth = coinbaseDiffEth.Text('f', 6)
	e.MevGeth_EthSentToCoinbase = ethSentToCoinbase.Text('f', 6)
}
