package analysis

import "fmt"

type TreeNode struct {
	Block    *Block
	Parent   *TreeNode
	Children []*TreeNode

	IsFirst     bool
	IsMainChain bool
}

func NewTreeNode(block *Block, parent *TreeNode) *TreeNode {
	return &TreeNode{
		Block:    block,
		Parent:   parent,
		Children: []*TreeNode{},
		IsFirst:  parent == nil,
	}
}

func (tn *TreeNode) String() string {
	return fmt.Sprintf("TreeNode %d %s main=%5v \t first=%5v, %d children", tn.Block.Number, tn.Block.Hash, tn.IsMainChain, tn.IsFirst, len(tn.Children))
}

func (tn *TreeNode) AddChild(node *TreeNode) {
	tn.Children = append(tn.Children, node)
}
