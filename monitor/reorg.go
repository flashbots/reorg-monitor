package monitor

import (
	"fmt"
	"reflect"

	"github.com/ethereum/go-ethereum/common"
)

type Reorg struct {
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

func NewReorg() *Reorg {
	return &Reorg{
		BlocksInvolved:  make(map[common.Hash]*Block),
		NodesInvolved:   make(map[string]bool),
		MainChain:       make([]*Block, 0),
		MainChainHashes: make(map[common.Hash]bool),
		SeenLive:        true, // will be set to false if any of the added blocks was received via uncle-info
	}
}

func (r *Reorg) Id() string {
	id := fmt.Sprintf("%d_%d_d%d_b%d", r.StartBlockHeight, r.EndBlockHeight, r.Depth, len(r.BlocksInvolved))
	if r.SeenLive {
		id += "_l"
	}
	return id
}

func (r *Reorg) String() string {
	nodeUris := reflect.ValueOf(r.NodesInvolved).MapKeys()
	return fmt.Sprintf("Reorg %s: live=%-6v blocks %d - %d, depth: %d, lenMainChain: %d, numBlocks: %d, replacedBlocks: %d, nodesInvolved: %v", r.Id(), r.SeenLive, r.StartBlockHeight, r.EndBlockHeight, r.Depth, len(r.MainChain), len(r.BlocksInvolved), r.NumReplacedBlocks, nodeUris)
}

func (r *Reorg) GetMainChainHashes() []string {
	blockHashes := []string{}
	for _, block := range r.MainChain {
		blockHashes = append(blockHashes, block.Hash.String())
	}
	return blockHashes
}

func (r *Reorg) GetReplacedBlockHashes() []string {
	hashes := []string{}
	for hash := range r.BlocksInvolved {
		_, isMainChainBlock := r.MainChainHashes[hash]
		if !isMainChainBlock {
			hashes = append(hashes, hash.String())
		}

	}
	return hashes
}

func (r *Reorg) AddBlock(block *Block) {
	if _, found := r.BlocksInvolved[block.Hash]; found { // alredy known
		return
	}

	// Remember this block
	r.BlocksInvolved[block.Hash] = block

	if block.Origin == OriginUncle { // one block as uncle means the reorg hasn't been seen live
		r.SeenLive = false
	}

	if r.StartBlockHeight == 0 || block.Number < r.StartBlockHeight {
		r.StartBlockHeight = block.Number
		r.LastBlockHashBeforeReorg = block.ParentHash // common parent must be block before earliest block in this reorg
	}

	if block.Number > r.EndBlockHeight {
		r.EndBlockHeight = block.Number
	}
}

func (r *Reorg) Finalize(firstBlockWithoutSiblings *Block) {
	r.IsCompleted = true
	r.FirstBlockAfterReorg = firstBlockWithoutSiblings

	// Find all involved nodes
	for _, block := range r.BlocksInvolved {
		r.NodesInvolved[block.NodeUri] = true
	}

	r.Depth = int(r.EndBlockHeight) - int(r.StartBlockHeight) + 1

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
			fmt.Printf("error in reorg.Finalize: couldn't find block %s, %s\n", curBlockHash, r.String())
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

	r.NumReplacedBlocks = len(r.BlocksInvolved) - len(r.MainChain)
}

func (r *Reorg) MermaidSyntax() string {
	ret := "stateDiagram-v2\n"

	for _, block := range r.BlocksInvolved {
		ret += fmt.Sprintf("    %s --> %s\n", block.ParentHash, block.Hash)
	}

	// Add first block after reorg
	ret += fmt.Sprintf("    %s --> %s", r.FirstBlockAfterReorg.ParentHash, r.FirstBlockAfterReorg.Hash)
	return ret
}
