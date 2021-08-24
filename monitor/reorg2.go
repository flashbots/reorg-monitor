package monitor

import (
	"fmt"
	"reflect"

	"github.com/ethereum/go-ethereum/common"
)

type Reorg2 struct {
	IsCompleted bool
	SeenLive    bool

	StartBlockHeight  uint64 // first block number with siblings
	EndBlockHeight    uint64
	Depth             int
	NumReplacedBlocks int

	BlocksInvolved  map[common.Hash]*Block
	NodesInvolved   map[string]bool
	MainChain       []*Block // chain of remaining blocks
	MainChainHashes map[common.Hash]bool

	LastBlockHashBeforeReorg common.Hash
	FirstBlockAfterReorg     *Block
}

func NewReorg2() *Reorg2 {
	return &Reorg2{
		BlocksInvolved:  make(map[common.Hash]*Block),
		NodesInvolved:   make(map[string]bool),
		MainChain:       make([]*Block, 0),
		MainChainHashes: make(map[common.Hash]bool),
	}
}

func (r *Reorg2) Id() string {
	id := fmt.Sprintf("%d_%d_d%d_b%d", r.StartBlockHeight, r.EndBlockHeight, r.Depth, len(r.BlocksInvolved))
	if r.SeenLive {
		id += "_l"
	}
	return id
}

func (r *Reorg2) String() string {
	nodeUris := reflect.ValueOf(r.NodesInvolved).MapKeys()
	return fmt.Sprintf("Reorg2 %s: live=%-6v blocks %d - %d, depth: %d, lenMainChain: %d, numBlocks: %d, replacedBlocks: %d, nodesInvolved: %v", r.Id(), r.SeenLive, r.StartBlockHeight, r.EndBlockHeight, r.Depth, len(r.MainChain), len(r.BlocksInvolved), r.NumReplacedBlocks, nodeUris)
}

func (r *Reorg2) GetMainChainHashes() []string {
	blockHashes := []string{}
	for _, block := range r.MainChain {
		blockHashes = append(blockHashes, block.Hash.String())
	}
	return blockHashes
}

func (r *Reorg2) GetReplacedBlockHashes() []string {
	hashes := []string{}
	for hash := range r.BlocksInvolved {
		_, isMainChainBlock := r.MainChainHashes[hash]
		if !isMainChainBlock {
			hashes = append(hashes, hash.String())
		}

	}
	return hashes
}

func (r *Reorg2) AddBlock(block *Block) {
	if _, found := r.BlocksInvolved[block.Hash]; found {
		// alredy known
		return
	}

	if len(r.BlocksInvolved) == 0 { // first block
		r.LastBlockHashBeforeReorg = block.ParentHash
	}

	r.BlocksInvolved[block.Hash] = block

	if block.Number > r.EndBlockHeight {
		r.EndBlockHeight = block.Number
	}
}

func (r *Reorg2) Finalize(firstBlockWithoutSiblings *Block) {
	r.IsCompleted = true
	r.FirstBlockAfterReorg = firstBlockWithoutSiblings

	// Find canonical mainchain
	r.MainChain = make([]*Block, 0)
	r.MainChainHashes = make(map[common.Hash]bool)

	curBlockHash := firstBlockWithoutSiblings.ParentHash // start from last block in reorg and work backwards until start
	for {
		if curBlockHash == r.LastBlockHashBeforeReorg { // done, as we are now before the reorg
			break
		}

		block, found := r.BlocksInvolved[curBlockHash]
		if !found {
			fmt.Printf("err in Finalize: couldn't find block")
			break
		}

		r.MainChain = append(r.MainChain, block)
		r.MainChainHashes[block.Hash] = true
		curBlockHash = block.ParentHash
	}

	// Reverse the array, so earliest block comes first
	for i, j := 0, len(r.MainChain)-1; i < j; i, j = i+1, j-1 {
		r.MainChain[i], r.MainChain[j] = r.MainChain[j], r.MainChain[i]
	}

	// Find all involved nodes
	for _, block := range r.BlocksInvolved {
		r.NodesInvolved[block.NodeUri] = true
	}

	r.NumReplacedBlocks = len(r.BlocksInvolved) - len(r.MainChain)
	r.Depth = int(r.EndBlockHeight) - int(r.StartBlockHeight) + 1
}

func (r *Reorg2) MermaidSyntax() string {
	ret := "stateDiagram-v2\n"

	for _, block := range r.BlocksInvolved {
		ret += fmt.Sprintf("    %s --> %s\n", block.ParentHash, block.Hash)
	}

	// Add first block after reorg
	ret += fmt.Sprintf("    %s --> %s", r.FirstBlockAfterReorg.ParentHash, r.FirstBlockAfterReorg.Hash)
	return ret
}
