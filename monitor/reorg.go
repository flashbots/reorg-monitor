package monitor

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type Reorg struct {
	StartBlockHeight uint64
	EndBlockHeight   uint64
	Depth            uint64
	NumChains        uint64 // needs better detection of double-reorgs
	IsCompleted      bool
	SeenLive         bool

	BlocksInvolved map[common.Hash]*types.Block
	Segments       []*ChainSegment
}

func NewReorg() *Reorg {
	return &Reorg{
		BlocksInvolved: make(map[common.Hash]*types.Block),
		Segments:       make([]*ChainSegment, 0),
	}
}

func (r *Reorg) Id() string {
	return fmt.Sprintf("%d_%d_d%d_b%d", r.StartBlockHeight, r.EndBlockHeight, r.Depth, len(r.BlocksInvolved))
}

func (r *Reorg) String() string {
	return fmt.Sprintf("Reorg %s: live=%v, blocks %d - %d, depth: %d, numBlocks: %d, numChains: %d", r.Id(), r.SeenLive, r.StartBlockHeight, r.EndBlockHeight, r.Depth, len(r.BlocksInvolved), len(r.Segments))
}

func (r *Reorg) AddBlock(block *types.Block) {
	if _, found := r.BlocksInvolved[block.Hash()]; found {
		// alredy known
		return
	}

	r.BlocksInvolved[block.Hash()] = block

	// Add to segment
	addedToExistingSegment := false
	for _, segment := range r.Segments {
		// Segment found if last block is this one's parent
		if block.ParentHash() == segment.LastBlock.Hash() {
			segment.AddBlock(block)
			addedToExistingSegment = true
			break
		}
	}

	if !addedToExistingSegment {
		// need to create new segment
		// todo: what if parent already in the middle of another segment?
		newSegment := NewChainSegment(block)
		r.Segments = append(r.Segments, newSegment)
		// fmt.Println("Added new segment for", block.Number(), block.Hash())
	}
}

func (r *Reorg) Finalize(firstBlockWithoutSiblings *types.Block) {
	r.IsCompleted = true
	r.EndBlockHeight = firstBlockWithoutSiblings.NumberU64() - 1

	// Find which is the main chain
	for _, segment := range r.Segments {
		if segment.LastBlock.Hash() == firstBlockWithoutSiblings.ParentHash() {
			segment.IsMainChain = true
			break
		}
	}
}
