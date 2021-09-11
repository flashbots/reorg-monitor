package monitor

import (
	"context"
	"fmt"
	"log"

	"github.com/metachris/eth-reorg-monitor/analysis"
	"github.com/pkg/errors"

	"github.com/ethereum/go-ethereum/common"
)

type ReorgMonitor struct {
	maxBlocksInCache int

	gethNodeUris []string
	connections  map[string]*GethConnection
	verbose      bool

	NewBlockChan chan *analysis.Block
	NewReorgChan chan<- *analysis.Reorg

	BlockByHash    map[common.Hash]*analysis.Block
	BlocksByHeight map[uint64]map[common.Hash]*analysis.Block

	EarliestBlockNumber uint64
	LatestBlockNumber   uint64

	Reorgs map[string]*analysis.Reorg
}

func NewReorgMonitor(gethNodeUris []string, reorgChan chan<- *analysis.Reorg, verbose bool) *ReorgMonitor {
	return &ReorgMonitor{
		verbose:          verbose,
		maxBlocksInCache: 1000,

		gethNodeUris: gethNodeUris,
		connections:  make(map[string]*GethConnection),

		NewBlockChan: make(chan *analysis.Block, 100),
		NewReorgChan: reorgChan,

		BlockByHash:    make(map[common.Hash]*analysis.Block),
		BlocksByHeight: make(map[uint64]map[common.Hash]*analysis.Block),
		Reorgs:         make(map[string]*analysis.Reorg),
	}
}

func (mon *ReorgMonitor) String() string {
	return fmt.Sprintf("ReorgMonitor: %d - %d, %d blocks", mon.EarliestBlockNumber, mon.LatestBlockNumber, len(mon.BlockByHash))
}

func (mon *ReorgMonitor) ConnectClients() (connectedClients int) {
	for _, nodeUri := range mon.gethNodeUris {
		gethConn, err := NewGethConnection(nodeUri, mon.NewBlockChan)
		if err != nil { // in case of an error, just print it but still continue to add it
			fmt.Println(err)
		} else {
			connectedClients += 1
		}
		mon.connections[nodeUri] = gethConn
	}

	return connectedClients
}

// SubscribeAndListen is the main monitor loop: subscribes to new blocks from all geth connections, and waits for new blocks to process.
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

		if len(mon.BlocksByHeight) < 3 {
			continue
		}

		// Analyze blocks once a new height has been reached
		lastBlockHeight = block.Number
		analysis, err := mon.AnalyzeTree(0, 2)
		if err != nil {
			log.Println("error in SubscribeAndListen->AnalyzeTree", err)
			continue
		}

		for _, reorg := range analysis.Reorgs {
			if !reorg.IsFinished { // don't care about unfinished reorgs
				continue
			}

			// Send new finished reorgs to channel
			if _, isKnownReorg := mon.Reorgs[reorg.Id()]; !isKnownReorg {
				mon.Reorgs[reorg.Id()] = reorg
				mon.NewReorgChan <- reorg
			}
		}
	}
}

// AddBlock adds a block to history if it hasn't been seen before, and download unknown referenced blocks (parent, uncles).
func (mon *ReorgMonitor) AddBlock(block *analysis.Block) bool {
	// If known, then only overwrite if known was by uncle
	knownBlock, isKnown := mon.BlockByHash[block.Hash]
	if isKnown && knownBlock.Origin != analysis.OriginUncle {
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
		mon.BlocksByHeight[block.Number] = make(map[common.Hash]*analysis.Block)
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

func (mon *ReorgMonitor) CheckBlockForReferences(block *analysis.Block) error {
	// Check parent
	_, found := mon.BlockByHash[block.ParentHash]
	if !found {
		// fmt.Printf("- parent of %d %s not found (%s), downloading...\n", block.Number, block.Hash, block.ParentHash)
		_, _, err := mon.EnsureBlock(block.ParentHash, analysis.OriginGetParent, block.NodeUri)
		if err != nil {
			return errors.Wrap(err, "get-parent error")
		}
	}

	// Check uncles
	for _, uncleHeader := range block.Block.Uncles() {
		// fmt.Printf("- block %d %s has uncle: %s\n", block.Number, block.Hash, uncleHeader.Hash())
		_, _, err := mon.EnsureBlock(uncleHeader.Hash(), analysis.OriginUncle, block.NodeUri)
		if err != nil {
			return errors.Wrap(err, "get-uncle error")
		}
	}

	// ro.DebugPrintln(fmt.Sprintf("- added block %d %s", block.NumberU64(), block.Hash()))
	return nil
}

func (mon *ReorgMonitor) EnsureBlock(blockHash common.Hash, origin analysis.BlockOrigin, nodeUri string) (block *analysis.Block, alreadyExisted bool, err error) {
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

	block = analysis.NewBlock(ethBlock, origin, nodeUri)

	// Add a new block without sending to channel, because that makes reorg.AddBlock() asynchronous, but we want reorg.AddBlock() to wait until all references are added.
	mon.AddBlock(block)
	return block, false, nil
}

func (mon *ReorgMonitor) AnalyzeTree(maxBlocks uint64, distanceToLastBlockHeight uint64) (*analysis.TreeAnalysis, error) {
	// Set end height of search
	endBlockNumber := mon.LatestBlockNumber - distanceToLastBlockHeight

	// Set start height of search
	startBlockNumber := mon.EarliestBlockNumber
	if maxBlocks > 0 && endBlockNumber-maxBlocks > mon.EarliestBlockNumber {
		startBlockNumber = endBlockNumber - maxBlocks
	}

	// Build tree datastructure
	tree := analysis.NewBlockTree()
	for height := startBlockNumber; height <= endBlockNumber; height++ {
		numBlocksAtHeight := len(mon.BlocksByHeight[height])
		if numBlocksAtHeight == 0 {
			err := fmt.Errorf("error in monitor.AnalyzeTree: no blocks at height %d", height)
			return nil, err
		}

		// Start tree only when 1 block at this height. If more blocks then skip.
		if tree.FirstNode == nil && numBlocksAtHeight > 1 {
			continue
		}

		// Add all blocks at this height to the tree
		for _, currentBlock := range mon.BlocksByHeight[height] {
			err := tree.AddBlock(currentBlock)
			if err != nil {
				return nil, errors.Wrap(err, "monitor.AnalyzeTree->tree.AddBlock error")
			}
		}
	}

	// Get analysis of tree
	analysis, err := analysis.NewTreeAnalysis(tree)
	if err != nil {
		return nil, errors.Wrap(err, "monitor.AnalyzeTree->NewTreeAnalysis error")
	}

	return analysis, nil
}
