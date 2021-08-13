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

// type ChainSegment struct {
// 	FirstBlock *types.Block
// 	LastBlock  *types.Block
// 	Blocks     map[common.Hash]*types.Block
// 	// Parent         *types.Block
// 	// Children       []*types.Block
// }

// func NewChainSegment(firstBlock *types.Block) *ChainSegment {
// 	segment := &ChainSegment{
// 		FirstBlock: firstBlock,
// 		LastBlock:  firstBlock,
// 		Blocks:     make(map[common.Hash]*types.Block),
// 	}
// 	segment.Blocks[firstBlock.Hash()] = firstBlock
// 	return segment
// }

// func (segment *ChainSegment) AddBlock(block *types.Block) error {
// 	if block.ParentHash() != segment.LastBlock.Hash() {
// 		return fmt.Errorf("ChainSegment.AddBlock: block has different parent (%s) than last block in this segment (%s)", block.ParentHash(), segment.LastBlock.Hash())
// 	}
// 	segment.Blocks[block.Hash()] = block
// 	segment.LastBlock = block
// 	return nil
// }

type Reorg struct {
	StartBlockHeight uint64
	EndBlockHeight   uint64
	Depth            uint64
	NumChains        uint64 // needs better detection of double-reorgs
	IsCompleted      bool
	SeenLive         bool

	BlocksInvolved map[common.Hash]*types.Block
	// Segments []*ChainSegment
}

func NewReorg() *Reorg {
	return &Reorg{
		BlocksInvolved: make(map[common.Hash]*types.Block),
	}
}

func (r *Reorg) Id() string {
	return fmt.Sprintf("%d_%d_d%d_b%d", r.StartBlockHeight, r.EndBlockHeight, r.Depth, len(r.BlocksInvolved))
}

func (r *Reorg) String() string {
	return fmt.Sprintf("Reorg %s: live: %v, blocks %d - %d, depth: %d, numBlocks: %d", r.Id(), r.SeenLive, r.StartBlockHeight, r.EndBlockHeight, r.Depth, len(r.BlocksInvolved))
}

type ReorgMonitor struct { // monitor one node
	client   *ethclient.Client
	debug    bool
	nickname string

	BlockByHash    map[common.Hash]*types.Block
	BlocksByHeight map[uint64][]*types.Block

	EarliestBlockNumber uint64
	LatestBlockNumber   uint64

	Reorgs            map[string]*Reorg
	BlockViaUncleInfo map[common.Hash]bool // true if that block was not seen live but rather seen through uncle information of child block. Used to know if we saw a reorg live
}

func NewReorgMonitor(ethUri string, nickname string, debug bool) *ReorgMonitor {
	// Connect to geth node
	fmt.Printf("[%s] Connecting to %s...", nickname, ethUri)
	client, err := ethclient.Dial(ethUri)
	reorgutils.Perror(err)
	fmt.Printf(" ok\n")

	return &ReorgMonitor{
		client:            client,
		debug:             debug,
		nickname:          nickname,
		BlockByHash:       make(map[common.Hash]*types.Block),
		BlocksByHeight:    make(map[uint64][]*types.Block),
		Reorgs:            make(map[string]*Reorg),
		BlockViaUncleInfo: make(map[common.Hash]bool),
	}
}

func (mon *ReorgMonitor) String() string {
	return fmt.Sprintf("ReorgMonitor[%s]: %d..%d, %d blocks", mon.nickname, mon.EarliestBlockNumber, mon.LatestBlockNumber, len(mon.BlockByHash))
}

func (mon *ReorgMonitor) DebugPrintln(a ...interface{}) {
	if mon.debug {
		fmt.Println(a...)
	}
}

// AddBlock adds a block to history if it hasn't been seen before. Also will query and download
// the chain of parents if not found
func (mon *ReorgMonitor) AddBlock(block *types.Block) error {
	mon.DebugPrintln(fmt.Sprintf("AddBlock %d %s - %s", block.NumberU64(), block.Hash(), mon.String()))
	_, knownBlock := mon.BlockByHash[block.Hash()]
	if !knownBlock {
		// Add for access by hash
		mon.BlockByHash[block.Hash()] = block

		// Add to array of blocks by height
		if _, found := mon.BlocksByHeight[block.NumberU64()]; !found {
			mon.BlocksByHeight[block.NumberU64()] = make([]*types.Block, 0)
		}
		mon.BlocksByHeight[block.NumberU64()] = append(mon.BlocksByHeight[block.NumberU64()], block)
	}

	if mon.EarliestBlockNumber == 0 {
		mon.EarliestBlockNumber = block.NumberU64()
	}

	if mon.LatestBlockNumber < block.NumberU64() {
		mon.LatestBlockNumber = block.NumberU64()
	}

	if block.NumberU64() > mon.EarliestBlockNumber { // check backhistory only if we are past the earliest block
		// Check and potentially download parent (back until start of monitoring)
		_, found := mon.BlockByHash[block.ParentHash()]
		if !found {
			fmt.Printf("- Parent of %d %s not found (%s), downloading...\n", block.NumberU64(), block.Hash(), block.ParentHash())
			_, err := mon.EnsureBlock(block.ParentHash())
			if err != nil {
				return errors.Wrap(err, "get unknown parent error")
			}
		}

		// Check uncles
		for _, uncleHeader := range block.Uncles() {
			fmt.Printf("- block %d %s has uncle: %s\n", block.NumberU64(), block.Hash(), uncleHeader.Hash())
			_, err := mon.EnsureBlock(uncleHeader.Hash())
			if err != nil {
				return errors.Wrap(err, "get uncle error")
			}
			mon.BlockViaUncleInfo[uncleHeader.Hash()] = true
		}
	}

	// ro.DebugPrintln(fmt.Sprintf("- added block %d %s", block.NumberU64(), block.Hash()))
	return nil
}

func (mon *ReorgMonitor) EnsureBlock(blockHash common.Hash) (*types.Block, error) {
	// Check and potentially download block
	block, found := mon.BlockByHash[blockHash]
	if found {
		return block, nil
	}

	fmt.Printf("- Block with hash %s not found, downloading...\n", blockHash)
	block, err := mon.client.BlockByHash(context.Background(), blockHash)
	if err != nil {
		return nil, errors.Wrap(err, "get unknown parent error")
	}

	mon.AddBlock(block)
	return block, nil
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

			blockInfo := reorgutils.SprintBlock(block) + " \t " + mon.String()
			log.Println(blockInfo)

			// Add the block
			err = mon.AddBlock(block)
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
					log.Println("New completed reorg found:", reorg.String())
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

				reorgs[height] = NewReorg()
				reorgs[height].StartBlockHeight = height
				reorgs[height].NumChains = numBlocksAtHeight

				// Was seen live if none of the siblings was detected through uncle unformation of child
				reorgs[height].SeenLive = true
				for _, block := range mon.BlocksByHeight[height] {
					if mon.BlockViaUncleInfo[block.Hash()] {
						reorgs[height].SeenLive = false
					}
				}
			}

			// Add all blocks at this height to it's own segment
			for _, block := range mon.BlocksByHeight[height] {
				reorgs[currentReorgStartHeight].BlocksInvolved[block.Hash()] = block
			}

			// Increase depth
			reorgs[currentReorgStartHeight].Depth += 1
			currentReorgEndHeight = height

		} else if numBlocksAtHeight == 0 {
			fmt.Printf("CheckForReorgs: no block at height %d found\n", height)
		} else {
			// 1 block, end of reorg
			if currentReorgStartHeight != 0 {
				reorgs[currentReorgStartHeight].IsCompleted = true
				reorgs[currentReorgStartHeight].EndBlockHeight = currentReorgEndHeight
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
