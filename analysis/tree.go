// Tree of blocks, used to traverse up (from children to parents) and down (from parents to children).
// Reorgs start on each node with more than one child.
package analysis

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

type BlockTree struct {
	FirstNode           *TreeNode
	LatestNodes         []*TreeNode // Nodes at latest blockheight (can be more than 1)
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
	// First block is a special case
	if t.FirstNode == nil {
		node := NewTreeNode(block, nil)
		t.FirstNode = node
		t.LatestNodes = []*TreeNode{node}
		t.NodeByHash[block.Hash] = node
		return nil
	}

	// All other blocks are inserted as child of it's parent parent
	parent, parentFound := t.NodeByHash[block.ParentHash]
	if !parentFound {
		err := fmt.Errorf("error in BlockTree.AddBlock(): parent not found. block: %d %s, parent: %s", block.Number, block.Hash, block.ParentHash)
		return err
	}

	node := NewTreeNode(block, parent)
	t.NodeByHash[block.Hash] = node
	parent.AddChild(node)

	// Remember nodes at latest block height
	if len(t.LatestNodes) == 0 {
		t.LatestNodes = []*TreeNode{node}
	} else {
		if block.Number == t.LatestNodes[0].Block.Number { // add to list of latest nodes!
			t.LatestNodes = append(t.LatestNodes, node)
		} else if block.Number > t.LatestNodes[0].Block.Number { // replace
			t.LatestNodes = []*TreeNode{node}
		}
	}

	// Mark main-chain nodes as such. Step 1: reset all nodes to non-main-chain
	t.MainChainNodeByHash = make(map[common.Hash]*TreeNode)
	for _, n := range t.NodeByHash {
		n.IsMainChain = false
	}

	// Step 2: Traverse backwards and mark main chain. If there's more than 1 nodes at latest height, then we don't yet know which chain will be the main-chain
	if len(t.LatestNodes) == 1 {
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
	indent := "-"
	fmt.Printf("%s %s\n", indent, node.String())
	for _, childNode := range node.Children {
		PrintNodeAndChildren(childNode, depth+1)
	}
}

func (t *BlockTree) Print() {
	fmt.Printf("BlockTree: nodes=%d\n", len(t.NodeByHash))

	if t.FirstNode == nil {
		return
	}

	// Print tree by traversing from parent to all children
	PrintNodeAndChildren(t.FirstNode, 1)

	// Print latest nodes
	fmt.Printf("Latest nodes:\n")
	for _, n := range t.LatestNodes {
		fmt.Println("-", n.String())
	}
}
