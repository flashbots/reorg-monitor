// TreeAnalysis takes in a BlockTree and collects information about reorgs
package analysis

import (
	"fmt"
)

type TreeAnalysis struct {
	Tree *BlockTree

	StartBlockHeight uint64 // first block number with siblings
	EndBlockHeight   uint64
	IsSplitOngoing   bool

	NumBlocks          int
	NumBlocksMainChain int

	Reorgs map[string]*Reorg
}

func NewTreeAnalysis(t *BlockTree) (*TreeAnalysis, error) {
	analysis := TreeAnalysis{
		Tree:   t,
		Reorgs: make(map[string]*Reorg),
	}

	if t.FirstNode == nil { // empty analysis for empty tree
		return &analysis, nil
	}

	analysis.StartBlockHeight = t.FirstNode.Block.Number
	analysis.EndBlockHeight = t.LatestNodes[0].Block.Number

	if len(t.LatestNodes) > 1 {
		analysis.IsSplitOngoing = true
	}

	analysis.NumBlocks = len(t.NodeByHash)
	analysis.NumBlocksMainChain = len(t.MainChainNodeByHash)

	// Find reorgs
	for _, node := range t.NodeByHash {
		if len(node.Children) > 1 {
			reorg, err := NewReorg(node)
			if err != nil {
				return nil, err
			}
			analysis.Reorgs[reorg.Id()] = reorg
		}
	}

	return &analysis, nil
}

func (a *TreeAnalysis) Print() {
	fmt.Printf("TreeAnalysis %d - %d, nodes: %d, mainchain: %d, reorgs: %d\n", a.StartBlockHeight, a.EndBlockHeight, a.NumBlocks, a.NumBlocksMainChain, len(a.Reorgs))
	if a.IsSplitOngoing {
		fmt.Println("- split ongoing")
	}

	for _, reorg := range a.Reorgs {
		fmt.Println("")
		fmt.Println(reorg.String())
		fmt.Printf("- common parent: %d %s, first block after: %d %s\n", reorg.CommonParent.Number, reorg.CommonParent.Hash, reorg.FirstBlockAfterReorg.Number, reorg.FirstBlockAfterReorg.Hash)

		for chainKey, chain := range reorg.Chains {
			if chainKey == reorg.MainChainHash {
				fmt.Printf("- mainchain l=%d: ", len(chain))
			} else {
				fmt.Printf("- sidechain l=%d: ", len(chain))
			}
			for _, block := range chain {
				fmt.Printf("%s ", block.Hash)
			}
			fmt.Print("\n")
		}
	}
}
