package monitor

import (
	"context"
	"fmt"
	"log"

	"github.com/pkg/errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

type BlockOrigin string

const (
	OriginSubscription BlockOrigin = "Subscription"
	OriginGetParent    BlockOrigin = "GetParent"
	OriginUncle        BlockOrigin = "Uncle"
)

// Block is an geth Block and information about where it came from
type Block struct {
	Block   *types.Block
	Origin  BlockOrigin
	NodeUri string

	// some helpers
	Number     uint64
	Hash       common.Hash
	ParentHash common.Hash
}

func NewBlock(block *types.Block, origin BlockOrigin, nodeUri string) *Block {
	return &Block{
		Block:   block,
		Origin:  origin,
		NodeUri: nodeUri,

		Number:     block.NumberU64(),
		Hash:       block.Hash(),
		ParentHash: block.ParentHash(),
	}
}

func (block *Block) String() string {
	return fmt.Sprintf("Block %d %s \t tx: %3d, uncles: %d", block.Number, block.Hash, len(block.Block.Transactions()), len(block.Block.Uncles()))
}

type ReorgMonitor2 struct { // monitor one node
	clients map[string]*ethclient.Client
	verbose bool

	NewBlockChannel chan (*Block)
	BlockByHash     map[common.Hash]*Block
	BlocksByHeight  map[uint64]map[common.Hash]*Block

	EarliestBlockNumber uint64
	LatestBlockNumber   uint64

	Reorgs map[string]*Reorg
}

func NewReorgMonitor2(verbose bool) *ReorgMonitor2 {
	return &ReorgMonitor2{
		verbose: verbose,
		clients: make(map[string]*ethclient.Client),

		NewBlockChannel: make(chan *Block),
		BlockByHash:     make(map[common.Hash]*Block),
		BlocksByHeight:  make(map[uint64]map[common.Hash]*Block),
		Reorgs:          make(map[string]*Reorg),
	}
}

func (mon *ReorgMonitor2) String() string {
	return fmt.Sprintf("ReorgMonitor2: %d - %d, %d blocks", mon.EarliestBlockNumber, mon.LatestBlockNumber, len(mon.BlockByHash))
}

func (mon *ReorgMonitor2) ConnectGethInstance(nodeUri string) error {
	fmt.Printf("[%s] Connecting to geth node...", nodeUri)
	client, err := ethclient.Dial(nodeUri)
	if err != nil {
		return err
	}

	fmt.Printf(" ok\n")
	mon.clients[nodeUri] = client
	return nil
}

func (mon *ReorgMonitor2) SubscribeAndStart(reorgChan chan<- *Reorg) error {
	// Subscribe to new blocks from all clients
	for clientNodeUri, client := range mon.clients {
		headers := make(chan *types.Header)
		sub, err := client.SubscribeNewHead(context.Background(), headers)
		if err != nil {
			return err
		}

		go func(_client *ethclient.Client, nodeUri string) {
			for {
				select {
				case err := <-sub.Err():
					fmt.Printf("Subscription error on node %s: %v\n", nodeUri, err)
				case header := <-headers:
					// Fetch full block information from same client
					ethBlock, err := _client.BlockByHash(context.Background(), header.Hash())
					if err != nil {
						log.Println("newHeader -> getBlockByHash error", err)
						continue
					}

					// Add the block
					newBlock := NewBlock(ethBlock, OriginSubscription, nodeUri)
					mon.NewBlockChannel <- newBlock
				}
			}
		}(client, clientNodeUri)
	}

	for block := range mon.NewBlockChannel {
		mon.AddBlock(block)
	}

	return nil
}

// AddBlock adds a block to history if it hasn't been seen before. Also will query and download
// the chain of parents if not found
func (mon *ReorgMonitor2) AddBlock(block *Block) bool {
	knownBlock, isKnown := mon.BlockByHash[block.Hash]

	blockInfo := fmt.Sprintf("[%25s] Add%s \t %-12s \t %s", block.NodeUri, block.String(), block.Origin, mon)
	fmt.Println(blockInfo)

	// Do nothing if block is known and not via uncle
	if isKnown && knownBlock.Origin != OriginUncle {
		return false
	}

	// Add for access by hash
	mon.BlockByHash[block.Hash] = block

	// Create array of blocks at this height, if necessary
	if _, found := mon.BlocksByHeight[block.Number]; !found {
		mon.BlocksByHeight[block.Number] = make(map[common.Hash]*Block)
	}

	// Add to map of blocks at this height
	mon.BlocksByHeight[block.Number][block.Hash] = block

	// Set earliest block
	if mon.EarliestBlockNumber == 0 || block.Number < mon.EarliestBlockNumber {
		mon.EarliestBlockNumber = block.Number
	}

	// Set latest block
	if block.Number > mon.LatestBlockNumber {
		mon.LatestBlockNumber = block.Number
	}

	// Check if further blocks can be downloaded from this one
	if block.Number > mon.EarliestBlockNumber { // check backhistory only if we are past the earliest block
		mon.CheckBlockForReferences(block)
	}

	return true
}

func (mon *ReorgMonitor2) CheckBlockForReferences(block *Block) error {
	// Check parent
	_, found := mon.BlockByHash[block.ParentHash]
	if !found {
		fmt.Printf("- parent of %d %s not found (%s), downloading...\n", block.Number, block.Hash, block.ParentHash)
		_, _, err := mon.EnsureBlock(block.ParentHash, block.Origin, block.NodeUri)
		if err != nil {
			return errors.Wrap(err, "get-parent error")
		}
	}

	// Check uncles
	for _, uncleHeader := range block.Block.Uncles() {
		fmt.Printf("- block %d %s has uncle: %s\n", block.Number, block.Hash, uncleHeader.Hash())
		_, _, err := mon.EnsureBlock(uncleHeader.Hash(), block.Origin, block.NodeUri)
		if err != nil {
			return errors.Wrap(err, "get-uncle error")
		}
	}

	// ro.DebugPrintln(fmt.Sprintf("- added block %d %s", block.NumberU64(), block.Hash()))
	return nil
}

func (mon *ReorgMonitor2) EnsureBlock(blockHash common.Hash, origin BlockOrigin, nodeUri string) (block *Block, alreadyExisted bool, err error) {
	// Check and potentially download block
	var found bool
	block, found = mon.BlockByHash[blockHash]
	if found {
		return block, true, nil
	}

	fmt.Printf("- block with hash %s not found, downloading from %s...", blockHash, nodeUri)
	client := mon.clients[nodeUri]
	ethBlock, err := client.BlockByHash(context.Background(), blockHash)
	if err != nil {
		return nil, false, errors.Wrap(err, "EnsureBlock error")
	}

	block = NewBlock(ethBlock, origin, nodeUri)
	mon.AddBlock(block)
	return block, false, nil
}
