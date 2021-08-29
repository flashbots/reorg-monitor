package analysis

import (
	"fmt"
	"sort"

	"github.com/ethereum/go-ethereum/common"
)

// type Chain struct {
// 	Blocks []*Block
// }

type Reorg struct {
	IsFinished bool
	SeenLive   bool

	StartBlockHeight uint64 // first block number with siblings
	EndBlockHeight   uint64

	Chains map[common.Hash][]*Block

	Depth             int
	Width             int
	NumReplacedBlocks int

	BlocksInvolved   map[common.Hash]*Block
	MainChainHash    common.Hash
	MainChainBlocks  map[common.Hash]*Block
	EthNodesInvolved map[string]bool

	CommonParent         *Block
	FirstBlockAfterReorg *Block
}

func NewReorg(parentNode *TreeNode) (*Reorg, error) {
	if len(parentNode.Children) < 2 {
		return nil, fmt.Errorf("cannot create reorg because parent node with < 2 children")
	}

	reorg := Reorg{
		CommonParent: parentNode.Block,

		Chains:           make(map[common.Hash][]*Block),
		BlocksInvolved:   make(map[common.Hash]*Block),
		EthNodesInvolved: make(map[string]bool),
		MainChainBlocks:  make(map[common.Hash]*Block),

		SeenLive: true, // will be set to false if any of the added blocks was received via uncle-info
	}

	// Build the individual chains until the enc, by iterating over children recursively
	for _, chainRootNode := range parentNode.Children {
		chain := make([]*Block, 0)

		var addChildToChainRecursive func(node *TreeNode)
		addChildToChainRecursive = func(node *TreeNode) {
			chain = append(chain, node.Block)

			for _, childNode := range node.Children {
				addChildToChainRecursive(childNode)
			}
		}
		addChildToChainRecursive(chainRootNode)
		reorg.Chains[chainRootNode.Block.Hash] = chain
	}

	// Find depth of chains
	chainLengths := []int{}
	for _, chain := range reorg.Chains {
		chainLengths = append(chainLengths, len(chain))
	}
	sort.Sort(sort.Reverse(sort.IntSlice(chainLengths)))

	// Depth is number of blocks in second chain
	reorg.Depth = chainLengths[1]
	reorg.Width = len(reorg.Chains)

	// If two chains with same height, then the reorg isn't yet finalized
	if chainLengths[0] == chainLengths[1] {
		reorg.IsFinished = false
	} else {
		reorg.IsFinished = true
	}

	// Truncate the longest chain to the second, which is when the reorg actually stopped
	for key, chain := range reorg.Chains {
		if len(chain) > chainLengths[1] {
			reorg.FirstBlockAfterReorg = chain[chainLengths[1]] // first block that will be truncated
			reorg.Chains[key] = chain[:chainLengths[1]]
			reorg.MainChainHash = key
		}
	}

	// Find final blockheight
	mainChain := reorg.Chains[reorg.MainChainHash]
	lastBlockInMainChain := mainChain[len(mainChain)-1]
	reorg.StartBlockHeight = mainChain[0].Number
	reorg.EndBlockHeight = lastBlockInMainChain.Number

	// Build final list of involved blocks, and get end blockheight
	for chainHash, chain := range reorg.Chains {
		for _, block := range chain {
			reorg.BlocksInvolved[block.Hash] = block
			reorg.EthNodesInvolved[block.NodeUri] = true

			if block.Origin == OriginUncle {
				reorg.SeenLive = false
			}

			if chainHash == reorg.MainChainHash {
				reorg.MainChainBlocks[block.Hash] = block
			}
		}
	}

	return &reorg, nil
}

func (r *Reorg) Id() string {
	id := fmt.Sprintf("%d_%d_d%d_b%d", r.StartBlockHeight, r.EndBlockHeight, r.Depth, len(r.BlocksInvolved))
	if r.SeenLive {
		id += "_l"
	}
	return id
}

func (r *Reorg) String() string {
	// nodeUris := reflect.ValueOf(r.EthNodesInvolved).MapKeys()
	return fmt.Sprintf("Reorg %s: chains: %d", r.Id(), len(r.Chains))
	// return fmt.Sprintf("Reorg %s: live=%-6v blocks %d - %d, depth: %d, lenMainChain: %d, numBlocks: %d, replacedBlocks: %d, nodesInvolved: %v", r.Id(), r.SeenLive, r.StartBlockHeight, r.EndBlockHeight, r.Depth, len(r.MainChain), len(r.BlocksInvolved), r.NumReplacedBlocks, nodeUris)
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
