// Tree structure of the blocks
package analysis

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

// Block is an geth Block and information about where it came from
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
	// return fmt.Sprintf("TreeNode %d %s first=%v, %d children, %d siblings, %d uncles", tn.Block.Number, tn.Block.Hash, tn.IsFirst, len(tn.Children), len(tn.Siblings), len(tn.Uncles))
}

func (tn *TreeNode) AddChild(node *TreeNode) {
	tn.Children = append(tn.Children, node)
}

type BlockTree struct {
	FirstNode           *TreeNode
	LatestNodes         []*TreeNode // can be more than 1 at latest height
	NodeByHash          map[common.Hash]*TreeNode
	MainChainNodeByHash map[common.Hash]*TreeNode
}

func NewBlockTree() *BlockTree {
	return &BlockTree{
		LatestNodes:         []*TreeNode{},
		NodeByHash:          make(map[common.Hash]*TreeNode),
		MainChainNodeByHash: make(map[common.Hash]*TreeNode),
	}
}

func (t *BlockTree) AddBlock(block *Block) error {
	// fmt.Println("BlockTree.AddBlock", block.Number, block.Hash)

	// First block is a special case
	if t.FirstNode == nil {
		node := NewTreeNode(block, nil)
		t.FirstNode = node
		t.LatestNodes = []*TreeNode{node}
		t.NodeByHash[block.Hash] = node
		return nil
	}

	// All other blocks are inserted after parent
	parent, parentFound := t.NodeByHash[block.ParentHash]
	if !parentFound {
		err := fmt.Errorf("error in BlockTree.AddBlock(): parent not found. block: %d %s, parent: %s", block.Number, block.Hash, block.ParentHash)
		// fmt.Println(err)
		return err
	}

	node := NewTreeNode(block, parent)
	t.NodeByHash[block.Hash] = node
	parent.AddChild(node)

	// Check if this node is latest, then remember
	if len(t.LatestNodes) == 0 {
		t.LatestNodes = []*TreeNode{node}
	} else {
		if block.Number == t.LatestNodes[0].Block.Number { // add to list of latest nodes!
			t.LatestNodes = append(t.LatestNodes, node)
		} else if block.Number > t.LatestNodes[0].Block.Number { // replace
			t.LatestNodes = []*TreeNode{node}
		}
	}

	// Mark main-chain nodes as such. Step 1: reset
	t.MainChainNodeByHash = make(map[common.Hash]*TreeNode)
	for _, n := range t.NodeByHash {
		n.IsMainChain = false
	}

	if len(t.LatestNodes) == 1 {
		// Traverse backwards and mark main chain
		var TraverseMainChainFromLatestToEarliest func(node *TreeNode)
		TraverseMainChainFromLatestToEarliest = func(node *TreeNode) {
			if node == nil {
				return
			}
			node.IsMainChain = true
			t.MainChainNodeByHash[node.Block.Hash] = node
			TraverseMainChainFromLatestToEarliest(node.Parent)
		}
		TraverseMainChainFromLatestToEarliest(t.LatestNodes[0])
	}

	return nil
}

func PrintNodeAndChildren(node *TreeNode, depth int) {
	// indent := strings.Repeat("-", depth)
	indent := "-"
	fmt.Printf("%s %s\n", indent, node.String())
	for _, childNode := range node.Children {
		PrintNodeAndChildren(childNode, depth+1)
	}
}

func (t *BlockTree) Print() {
	fmt.Printf("BlockTree - %d nodes\n", len(t.NodeByHash))

	// Traverse to print (from parent to all children)
	if t.FirstNode == nil {
		return
	}

	PrintNodeAndChildren(t.FirstNode, 1)

	fmt.Printf("Latest nodes:\n")
	for _, n := range t.LatestNodes {
		fmt.Println("-", n.String())
	}
}
