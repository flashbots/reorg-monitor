package monitor

import (
	"fmt"

	"github.com/ethereum/go-ethereum/core/types"
)

type ChainSegment struct {
	FirstBlock *types.Block
	LastBlock  *types.Block
	Blocks     []*types.Block

	IsMainChain bool
}

func NewChainSegment(firstBlock *types.Block) *ChainSegment {
	segment := &ChainSegment{
		FirstBlock: firstBlock,
		LastBlock:  firstBlock,
		Blocks:     make([]*types.Block, 1),
	}
	segment.Blocks[0] = firstBlock
	return segment
}

func (s *ChainSegment) String() string {
	segmentType := "main"
	if !s.IsMainChain {
		segmentType = "orph"
	}
	return fmt.Sprintf("ChainSegment [%s], blocks: %d", segmentType, len(s.Blocks))
}

func (s *ChainSegment) BlockHashes() []string {
	ret := make([]string, len(s.Blocks))
	for i, b := range s.Blocks {
		ret[i] = b.Hash().String()
	}
	return ret
}

func (s *ChainSegment) BlockHashesShort() []string {
	ret := make([]string, len(s.Blocks))
	for i, b := range s.Blocks {
		h := b.Hash().String()
		ret[i] = h[0:8] + "..." + h[len(h)-6:]
	}
	return ret
}

// Add a new block to the segment. Must be child of current last block.
func (segment *ChainSegment) AddBlock(block *types.Block) error {
	if block.ParentHash() != segment.LastBlock.Hash() {
		return fmt.Errorf("ChainSegment.AddBlock: block has different parent (%s) than last block in this segment (%s)", block.ParentHash(), segment.LastBlock.Hash())
	}
	segment.Blocks = append(segment.Blocks, block)
	segment.LastBlock = block
	return nil
}
