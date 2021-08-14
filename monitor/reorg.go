package monitor

import (
	"fmt"
	"strings"

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
	NodeUri          string

	BlocksInvolved map[common.Hash]*types.Block
	ChainSegments  []*ChainSegment
}

func NewReorg(nodeUri string) *Reorg {
	return &Reorg{
		NodeUri:        nodeUri,
		BlocksInvolved: make(map[common.Hash]*types.Block),
		ChainSegments:  make([]*ChainSegment, 0),
	}
}

func (r *Reorg) Id() string {
	return fmt.Sprintf("%d_%d_d%d_b%d", r.StartBlockHeight, r.EndBlockHeight, r.Depth, len(r.BlocksInvolved))
}

func (r *Reorg) String() string {
	return fmt.Sprintf("Reorg %s (%s): live=%-6v blocks %d - %d, depth: %d, numBlocks: %d, numChains: %d", r.Id(), r.NodeUri, r.SeenLive, r.StartBlockHeight, r.EndBlockHeight, r.Depth, len(r.BlocksInvolved), len(r.ChainSegments))
}

func (r *Reorg) PrintSegments() {
	for _, segment := range r.ChainSegments {
		fmt.Printf("- %s - %s\n", segment, strings.Join(segment.BlockHashes(), ", "))
	}
}

func (r *Reorg) AddBlock(block *types.Block) {
	if _, found := r.BlocksInvolved[block.Hash()]; found {
		// alredy known
		return
	}

	r.BlocksInvolved[block.Hash()] = block

	// Add to segment
	addedToExistingSegment := false
	for _, segment := range r.ChainSegments {
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
		r.ChainSegments = append(r.ChainSegments, newSegment)
		// fmt.Println("Added new segment for", block.Number(), block.Hash())
	}
}

func (r *Reorg) Finalize(firstBlockWithoutSiblings *types.Block) {
	r.IsCompleted = true
	r.EndBlockHeight = firstBlockWithoutSiblings.NumberU64() - 1

	// Find which is the main chain
	for _, segment := range r.ChainSegments {
		if segment.LastBlock.Hash() == firstBlockWithoutSiblings.ParentHash() {
			segment.IsMainChain = true
			break
		}
	}
}
