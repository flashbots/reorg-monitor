package analysis

import (
	"fmt"
)

type TreeAnalysis struct {
	Tree *BlockTree

	StartBlockHeight uint64 // first block number with siblings
	EndBlockHeight   uint64

	NumBlocks          int
	NumBlocksMainChain int
	NumReorgs          int
	IsSplitOngoing     bool

	Reorgs map[string]*Reorg
}

func NewTreeAnalysis(t *BlockTree) (*TreeAnalysis, error) {
	if t.FirstNode == nil {
		return nil, fmt.Errorf("error in BlockTree.Analyze: tree has no first block")
	}

	lastNode := t.LatestNodes[0] // there's always at least one

	analysis := TreeAnalysis{
		Tree:             t,
		StartBlockHeight: t.FirstNode.Block.Number,
		EndBlockHeight:   lastNode.Block.Number,
		Reorgs:           make(map[string]*Reorg),
	}

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
	fmt.Printf("TreeAnalysis %d - %d \t nodes: %d, mainchain: %d, many children: %d\n", a.StartBlockHeight, a.EndBlockHeight, a.NumBlocks, a.NumBlocksMainChain, a.NumReorgs)
	if a.IsSplitOngoing {
		fmt.Println("- split ongoing")
	}

	for _, reorg := range a.Reorgs {
		fmt.Println("")
		fmt.Println(reorg.String())
		fmt.Println("- common parent:", reorg.CommonParent.Hash)
		fmt.Println("- first block after:", reorg.FirstBlockAfterReorg.Hash)

		for _, chain := range reorg.Chains {
			fmt.Printf("- chain l=%d: ", len(chain))
			for _, block := range chain {
				fmt.Printf("%s ", block.Hash)
			}
			fmt.Print("\n")
		}
	}
}
