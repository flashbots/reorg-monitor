package monitor

import (
	"context"
	"fmt"
	"log"

	"github.com/pkg/errors"

	"github.com/ethereum/go-ethereum/common"
)

type ReorgMonitor struct {
	maxBlocksInCache int

	gethNodeUris []string
	connections  map[string]*GethConnection
	verbose      bool

	NewBlockChan chan *Block
	NewReorgChan chan<- *Reorg

	BlockByHash    map[common.Hash]*Block
	BlocksByHeight map[uint64]map[common.Hash]*Block

	EarliestBlockNumber uint64
	LatestBlockNumber   uint64

	Reorgs map[string]*Reorg
}

func NewReorgMonitor(gethNodeUris []string, reorgChan chan<- *Reorg, verbose bool) *ReorgMonitor {
	return &ReorgMonitor{
		verbose:          verbose,
		maxBlocksInCache: 1000,

		gethNodeUris: gethNodeUris,
		connections:  make(map[string]*GethConnection),

		NewBlockChan: make(chan *Block, 100),
		NewReorgChan: reorgChan,

		BlockByHash:    make(map[common.Hash]*Block),
		BlocksByHeight: make(map[uint64]map[common.Hash]*Block),
		Reorgs:         make(map[string]*Reorg),
	}
}

func (mon *ReorgMonitor) String() string {
	return fmt.Sprintf("ReorgMonitor: %d - %d, %d blocks", mon.EarliestBlockNumber, mon.LatestBlockNumber, len(mon.BlockByHash))
}

func (mon *ReorgMonitor) ConnectClients() error {
	for _, nodeUri := range mon.gethNodeUris {
		gethConn, err := NewGethConnection(nodeUri, mon.NewBlockChan)
		if err != nil {
			return err
		}

		mon.connections[nodeUri] = gethConn
	}

	return nil
}

// SubscribeAndListen subscribes to new blocks from all geth connections, and waits for new blocks to process.
// After adding a new block, a reorg check takes place. If a new completed reorg is detected, it is sent to the channel.
func (mon *ReorgMonitor) SubscribeAndListen() {
	// Subscribe to new blocks from all clients
	for _, conn := range mon.connections {
		go conn.Subscribe()
	}

	// Wait for new blocks and process them (blocking)
	lastBlockHeight := uint64(0)
	for block := range mon.NewBlockChan {
		mon.AddBlock(block)

		// Do nothing if block is at previous height
		if block.Number == lastBlockHeight {
			continue
		}

		// Check for reorgs once we are at new block height
		lastBlockHeight = block.Number
		completedReorgs := mon.CheckForReorgs(0, 2)
		for _, reorg := range completedReorgs {
			// Send new reorgs to channel
			if _, isKnownReorg := mon.Reorgs[reorg.Id()]; !isKnownReorg {
				mon.Reorgs[reorg.Id()] = reorg
				mon.NewReorgChan <- reorg
			}
		}
	}
}

// AddBlock adds a block to history if it hasn't been seen before, and download unknown referenced blocks (parent, uncles).
func (mon *ReorgMonitor) AddBlock(block *Block) bool {
	// If known, then only overwrite if known was by uncle
	knownBlock, isKnown := mon.BlockByHash[block.Hash]
	if isKnown && knownBlock.Origin != OriginUncle {
		return false
	}

	// Only accept blocks that are after the earliest known (some nodes might be further back)
	if block.Number < mon.EarliestBlockNumber {
		return false
	}

	// Print
	blockInfo := fmt.Sprintf("[%25s] Add%s \t %-12s \t %s", block.NodeUri, block.String(), block.Origin, mon)
	log.Println(blockInfo)

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
		err := mon.CheckBlockForReferences(block)
		if err != nil {
			log.Println(err)
		}
	}

	mon.TrimCache()
	return true
}

func (mon *ReorgMonitor) TrimCache() {
	for currentHeight := mon.EarliestBlockNumber; currentHeight < mon.LatestBlockNumber; currentHeight++ {
		blocks, heightExists := mon.BlocksByHeight[currentHeight]
		if !heightExists {
			continue
		}

		// Set new lowest block number
		mon.EarliestBlockNumber = currentHeight

		// Stop if trimmed enough
		if len(mon.BlockByHash) <= mon.maxBlocksInCache {
			return
		}

		// Trim
		delete(mon.BlocksByHeight, currentHeight)
		for hash := range blocks {
			delete(mon.BlockByHash, hash)
		}
	}
}

func (mon *ReorgMonitor) CheckBlockForReferences(block *Block) error {
	// Check parent
	_, found := mon.BlockByHash[block.ParentHash]
	if !found {
		// fmt.Printf("- parent of %d %s not found (%s), downloading...\n", block.Number, block.Hash, block.ParentHash)
		_, _, err := mon.EnsureBlock(block.ParentHash, OriginGetParent, block.NodeUri)
		if err != nil {
			return errors.Wrap(err, "get-parent error")
		}
	}

	// Check uncles
	for _, uncleHeader := range block.Block.Uncles() {
		// fmt.Printf("- block %d %s has uncle: %s\n", block.Number, block.Hash, uncleHeader.Hash())
		_, _, err := mon.EnsureBlock(uncleHeader.Hash(), OriginUncle, block.NodeUri)
		if err != nil {
			return errors.Wrap(err, "get-uncle error")
		}
	}

	// ro.DebugPrintln(fmt.Sprintf("- added block %d %s", block.NumberU64(), block.Hash()))
	return nil
}

func (mon *ReorgMonitor) EnsureBlock(blockHash common.Hash, origin BlockOrigin, nodeUri string) (block *Block, alreadyExisted bool, err error) {
	// Check and potentially download block
	var found bool
	block, found = mon.BlockByHash[blockHash]
	if found {
		return block, true, nil
	}

	fmt.Printf("- block %s (%s) not found, downloading from %s...\n", blockHash, origin, nodeUri)
	conn := mon.connections[nodeUri]
	ethBlock, err := conn.Client.BlockByHash(context.Background(), blockHash)
	if err != nil {
		fmt.Println("- err block not found:", blockHash, err) // todo: try other clients
		msg := fmt.Sprintf("EnsureBlock error for hash %s", blockHash)
		return nil, false, errors.Wrap(err, msg)
	}

	block = NewBlock(ethBlock, origin, nodeUri)
	// mon.AddBlock(block)
	mon.NewBlockChan <- block
	return block, false, nil
}

func (mon *ReorgMonitor) CheckForReorgs(maxBlocks uint64, distanceToLastBlockHeight uint64) []*Reorg {
	reorgs := make([]*Reorg, 0)

	// Set end height of search
	endBlockNumber := mon.LatestBlockNumber - distanceToLastBlockHeight

	// Set start height of search
	startBlockNumber := mon.EarliestBlockNumber
	if maxBlocks > 0 && endBlockNumber-maxBlocks > mon.EarliestBlockNumber {
		startBlockNumber = endBlockNumber - maxBlocks
	}

	// Pointer to the current open reorg
	var currentReorg *Reorg

	// Iterate over all blocks in the search range to find reorgs
	for height := startBlockNumber; height <= endBlockNumber; height++ {
		// mon.DebugPrintln("CheckForReorgs at height", height)
		numBlocksAtHeight := len(mon.BlocksByHeight[height])
		if numBlocksAtHeight == 0 {
			fmt.Printf("CheckForReorgs: no block at height %d found\n", height)
		} else if numBlocksAtHeight == 1 {
			if currentReorg == nil {
				continue
			}

			// end of reorg when only 1 block is left
			for _, currentBlock := range mon.BlocksByHeight[height] {
				currentReorg.Finalize(currentBlock)
				reorgs = append(reorgs, currentReorg)
				currentReorg = nil
			}
		} else { // > 1 block at this height, reorg in progress
			if currentReorg == nil {
				currentReorg = NewReorg()
			}

			// Add all blocks at this height. TODO: a new reorg could be started here
			for _, block := range mon.BlocksByHeight[height] {
				currentReorg.AddBlock(block)
			}
		}
	}

	return reorgs
}
