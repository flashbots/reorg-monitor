// Incoming Ethereum Block from a client, with some information where it came from.
package analysis

import (
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type BlockOrigin string

const (
	OriginSubscription BlockOrigin = "Subscription"
	OriginGetParent    BlockOrigin = "GetParent"
	OriginUncle        BlockOrigin = "Uncle"
)

// Block is an geth Block and information about where it came from
type Block struct {
	Block   *types.Block
	Origin  BlockOrigin
	NodeUri string

	// some helpers
	Number     uint64
	Hash       common.Hash
	ParentHash common.Hash
}

func NewBlock(block *types.Block, origin BlockOrigin, nodeUri string) *Block {
	return &Block{
		Block:   block,
		Origin:  origin,
		NodeUri: nodeUri,

		Number:     block.NumberU64(),
		Hash:       block.Hash(),
		ParentHash: block.ParentHash(),
	}
}

func (block *Block) String() string {
	t := time.Unix(int64(block.Block.Time()), 0).UTC()
	return fmt.Sprintf("Block %d %s / %s / tx: %4d, uncles: %d", block.Number, block.Hash, t, len(block.Block.Transactions()), len(block.Block.Uncles()))
}
