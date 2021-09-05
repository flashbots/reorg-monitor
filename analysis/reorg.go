// Summary of a specific reorg
package analysis

import (
	"fmt"
	"reflect"
	"sort"

	"github.com/ethereum/go-ethereum/common"
)

type Reorg struct {
	IsFinished bool
	SeenLive   bool

	StartBlockHeight uint64 // first block in a reorg (block number after common parent)
	EndBlockHeight   uint64 // last block in a reorg

	Chains map[common.Hash][]*Block

	Depth int

	BlocksInvolved    map[common.Hash]*Block
	MainChainHash     common.Hash
	MainChainBlocks   map[common.Hash]*Block
	NumReplacedBlocks int
	EthNodesInvolved  map[string]bool

	CommonParent         *Block
	FirstBlockAfterReorg *Block
}

func NewReorg(parentNode *TreeNode) (*Reorg, error) {
	if len(parentNode.Children) < 2 {
		return nil, fmt.Errorf("cannot create reorg because parent node with < 2 children")
	}

	reorg := Reorg{
		CommonParent:     parentNode.Block,
		StartBlockHeight: parentNode.Block.Number + 1,

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

	// Truncate the longest chain to the second, which is when the reorg actually stopped
	for key, chain := range reorg.Chains {
		if len(chain) > reorg.Depth {
			reorg.FirstBlockAfterReorg = chain[reorg.Depth] // first block that will be truncated
			reorg.Chains[key] = chain[:reorg.Depth]
			reorg.MainChainHash = key
		}
	}

	// If two chains with same height, then the reorg isn't yet finalized
	if chainLengths[0] == chainLengths[1] {
		reorg.IsFinished = false
	} else {
		reorg.IsFinished = true
	}

	// Build final list of involved blocks, and get end blockheight
	for chainHash, chain := range reorg.Chains {
		for _, block := range chain {
			reorg.BlocksInvolved[block.Hash] = block
			reorg.EthNodesInvolved[block.NodeUri] = true

			if block.Origin != OriginSubscription && block.Origin != OriginGetParent {
				reorg.SeenLive = false
			}

			if chainHash == reorg.MainChainHash {
				reorg.MainChainBlocks[block.Hash] = block
				reorg.EndBlockHeight = block.Number
			}
		}
	}

	reorg.NumReplacedBlocks = len(reorg.BlocksInvolved) - reorg.Depth

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
	nodeUris := reflect.ValueOf(r.EthNodesInvolved).MapKeys()
	return fmt.Sprintf("Reorg %s: live=%-5v chains=%d, depth=%d, replaced=%d, nodes: %v", r.Id(), r.SeenLive, len(r.Chains), r.Depth, r.NumReplacedBlocks, nodeUris)
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
