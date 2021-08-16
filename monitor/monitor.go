package monitor

import (
	"context"
	"fmt"
	"log"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/metachris/eth-reorg-monitor/reorgutils"
	"github.com/pkg/errors"
)

type ReorgMonitor struct { // monitor one node
	client  *ethclient.Client
	debug   bool
	verbose bool

	nodeUri string

	BlockByHash    map[common.Hash]*types.Block
	BlocksByHeight map[uint64][]*types.Block

	EarliestBlockNumber uint64
	LatestBlockNumber   uint64

	Reorgs            map[string]*Reorg
	BlockViaUncleInfo map[common.Hash]bool // true if that block was not seen live but rather seen through uncle information of child block. Used to know if we saw a reorg live
}

func NewReorgMonitor(ethNodeUri string, debug bool, verbose bool) *ReorgMonitor {
	// Connect to geth node
	fmt.Printf("[%s] Connecting to node...", ethNodeUri)
	client, err := ethclient.Dial(ethNodeUri)
	reorgutils.Perror(err)
	fmt.Printf(" ok\n")

	return &ReorgMonitor{
		client:  client,
		debug:   debug,
		verbose: verbose,
		nodeUri: ethNodeUri,

		BlockByHash:       make(map[common.Hash]*types.Block),
		BlocksByHeight:    make(map[uint64][]*types.Block),
		Reorgs:            make(map[string]*Reorg),
		BlockViaUncleInfo: make(map[common.Hash]bool),
	}
}

func (mon *ReorgMonitor) String() string {
	return fmt.Sprintf("ReorgMonitor[%s]: %d - %d, %d blocks", mon.nodeUri, mon.EarliestBlockNumber, mon.LatestBlockNumber, len(mon.BlockByHash))
}

func (mon *ReorgMonitor) DebugPrintln(a ...interface{}) {
	if mon.debug {
		fmt.Println(a...)
	}
}

// AddBlock adds a block to history if it hasn't been seen before. Also will query and download
// the chain of parents if not found
func (mon *ReorgMonitor) AddBlock(block *types.Block, info string) error {
	_, knownBlock := mon.BlockByHash[block.Hash()]
	if knownBlock {
		return nil
	}

	blockInfo := fmt.Sprintf("[%s] Add%s \t %10s \t %s", mon.nodeUri, reorgutils.SprintBlock(block), info, mon)
	fmt.Println(blockInfo)

	// Add for access by hash
	mon.BlockByHash[block.Hash()] = block

	// Create array of blocks at this height, if necessary
	if _, found := mon.BlocksByHeight[block.NumberU64()]; !found {
		mon.BlocksByHeight[block.NumberU64()] = make([]*types.Block, 0)
	}

	// Add to array of blocks at this height
	mon.BlocksByHeight[block.NumberU64()] = append(mon.BlocksByHeight[block.NumberU64()], block)

	// Set earliest block
	if mon.EarliestBlockNumber == 0 {
		mon.EarliestBlockNumber = block.NumberU64()
	}

	// Set latest block
	if mon.LatestBlockNumber < block.NumberU64() {
		mon.LatestBlockNumber = block.NumberU64()
	}

	if block.NumberU64() > mon.EarliestBlockNumber { // check backhistory only if we are past the earliest block
		// Check and potentially download parent (back until start of monitoring)
		_, found := mon.BlockByHash[block.ParentHash()]
		if !found {
			mon.DebugPrintln(fmt.Sprintf("- parent of %d %s not found (%s), downloading...", block.NumberU64(), block.Hash(), block.ParentHash()))
			_, _, err := mon.EnsureBlock(block.ParentHash(), "get-parent")
			if err != nil {
				return errors.Wrap(err, "get unknown parent error")
			}
		}

		// Check uncles
		for _, uncleHeader := range block.Uncles() {
			mon.DebugPrintln(fmt.Sprintf("- block %d %s has uncle: %s", block.NumberU64(), block.Hash(), uncleHeader.Hash()))
			_, existed, err := mon.EnsureBlock(uncleHeader.Hash(), "get-uncle")
			if err != nil {
				return errors.Wrap(err, "get uncle error")
			}
			if !existed {
				mon.BlockViaUncleInfo[uncleHeader.Hash()] = true
			}
		}
	}

	// ro.DebugPrintln(fmt.Sprintf("- added block %d %s", block.NumberU64(), block.Hash()))
	return nil
}

func (mon *ReorgMonitor) EnsureBlock(blockHash common.Hash, info string) (block *types.Block, alreadyExisted bool, err error) {
	// Check and potentially download block
	var found bool
	block, found = mon.BlockByHash[blockHash]
	if found {
		return block, true, nil
	}

	mon.DebugPrintln(fmt.Sprintf("- block with hash %s not found, downloading...", blockHash))
	block, err = mon.client.BlockByHash(context.Background(), blockHash)
	if err != nil {
		return nil, false, errors.Wrap(err, "EnsureBlock error")
	}

	mon.AddBlock(block, info)
	return block, false, nil
}

func (mon *ReorgMonitor) SubscribeAndStart(reorgChan chan<- *Reorg) string {
	// Subscribe to new blocks
	headers := make(chan *types.Header)
	sub, err := mon.client.SubscribeNewHead(context.Background(), headers)
	reorgutils.Perror(err)

	for {
		select {
		case err := <-sub.Err():
			log.Fatal(err)
		case header := <-headers:
			// Fetch full block information
			block, err := mon.client.BlockByHash(context.Background(), header.Hash())
			if err != nil {
				log.Println("error", err)
				continue
			}

			// Add the block
			err = mon.AddBlock(block, "subscription")
			if err != nil {
				log.Println("error", err)
				continue
			}

			// Check for reorgs
			completedReorgs := mon.CheckForCompletedReorgs(100, 2)
			for _, reorg := range completedReorgs {
				reorgId := reorg.Id()
				_, found := mon.Reorgs[reorgId]
				if !found {
					// log.Println("New completed reorg found:", reorg.String())
					mon.Reorgs[reorgId] = reorg
					reorgChan <- reorg
				}
			}
		}
	}
}

func (mon *ReorgMonitor) CheckForReorgs(maxBlocks uint64, distanceToLastBlockHeight uint64) (reorgs map[uint64]*Reorg, isOngoingReorg bool, lastReorgEndHeight uint64) {
	// ro.DebugPrintln("c")
	reorgs = make(map[uint64]*Reorg)

	var currentReorgStartHeight uint64
	var currentReorgEndHeight uint64

	startBlockNumber := mon.EarliestBlockNumber
	if maxBlocks > 0 && mon.LatestBlockNumber-maxBlocks > mon.EarliestBlockNumber {
		startBlockNumber = mon.LatestBlockNumber - maxBlocks
	}

	endBlockNumber := mon.LatestBlockNumber - distanceToLastBlockHeight
	for height := startBlockNumber; height <= endBlockNumber; height++ {
		mon.DebugPrintln("CheckForReorgs at height", height)
		numBlocksAtHeight := uint64(len(mon.BlocksByHeight[height]))
		if numBlocksAtHeight > 1 {
			// fmt.Printf("- sibling blocks at %d: %v\n", height, ro.NodesByHeight[height])

			// detect reorg start
			if currentReorgStartHeight == 0 {
				currentReorgStartHeight = height

				reorgs[height] = NewReorg(mon.nodeUri)
				reorgs[height].StartBlockHeight = height
				// reorgs[height].NumChains = numBlocksAtHeight

				// Was seen live if none of the first siblings was detected through uncle unformation of child
				reorgs[height].SeenLive = true
				for _, block := range mon.BlocksByHeight[height] {
					if mon.BlockViaUncleInfo[block.Hash()] {
						reorgs[height].SeenLive = false
					}
				}
			}

			// Add all blocks at this height to it's own segment
			for _, block := range mon.BlocksByHeight[height] {
				reorgs[currentReorgStartHeight].AddBlock(block)
			}

			// Increase depth
			reorgs[currentReorgStartHeight].Depth += 1
			currentReorgEndHeight = height

		} else if numBlocksAtHeight == 0 {
			fmt.Printf("CheckForReorgs: no block at height %d found\n", height)
		} else {
			// 1 block, end of reorg
			if currentReorgStartHeight != 0 {
				reorgs[currentReorgStartHeight].Finalize(mon.BlocksByHeight[height][0])
				currentReorgStartHeight = 0
			}
		}
	}

	isOngoingReorg = currentReorgStartHeight != 0
	return reorgs, isOngoingReorg, currentReorgEndHeight
}

func (mon *ReorgMonitor) CheckForCompletedReorgs(maxBlocks uint64, distanceToLastBlockHeight uint64) (reorgs []*Reorg) {
	reorgs = make([]*Reorg, 0)
	_reorgs, _, _ := mon.CheckForReorgs(maxBlocks, distanceToLastBlockHeight)
	for _, _reorg := range _reorgs {
		if _reorg.IsCompleted {
			reorgs = append(reorgs, _reorg)
		}
	}
	return reorgs
}
