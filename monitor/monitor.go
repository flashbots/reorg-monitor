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

// type Node struct {
// 	Block    *types.Block
// 	Parent   *Node
// 	Siblings []*Node
// }

// func NewNode
// type ChainSegment struct {
// 	Nodes map[common.Hash]*Node
// }

type ReorgMonitor struct { // monitor one node
	client   *ethclient.Client
	debug    bool
	nickname string

	NodeByHash    map[common.Hash]*types.Block
	NodesByHeight map[uint64][]*types.Block

	EarliestBlockNumber uint64
	LatestBlockNumber   uint64
}

func NewReorgMonitor(client *ethclient.Client, nickname string, debug bool) *ReorgMonitor {
	return &ReorgMonitor{
		client:        client,
		debug:         debug,
		nickname:      nickname,
		NodeByHash:    make(map[common.Hash]*types.Block),
		NodesByHeight: make(map[uint64][]*types.Block),
	}
}

func (ro *ReorgMonitor) DebugPrintln(msg string) {
	if ro.debug {
		fmt.Println(msg)
	}
}

// AddBlock adds a block to history if it hasn't been seen before. Also will query and download
// the chain of parents if not found
func (ro *ReorgMonitor) AddBlock(block *types.Block) error {
	ro.DebugPrintln(fmt.Sprintf("AddBlock %d %s - %s", block.NumberU64(), block.Hash(), ro.String()))
	_, knownBlock := ro.NodeByHash[block.Hash()]
	if !knownBlock {
		// Add for access by hash
		ro.NodeByHash[block.Hash()] = block

		// Add to array of blocks by height
		if _, found := ro.NodesByHeight[block.NumberU64()]; !found {
			ro.NodesByHeight[block.NumberU64()] = make([]*types.Block, 0)
		}
		ro.NodesByHeight[block.NumberU64()] = append(ro.NodesByHeight[block.NumberU64()], block)
	}

	if ro.EarliestBlockNumber == 0 {
		ro.EarliestBlockNumber = block.NumberU64()
	}

	if ro.LatestBlockNumber < block.NumberU64() {
		ro.LatestBlockNumber = block.NumberU64()
	}

	if block.NumberU64() > ro.EarliestBlockNumber { // check backhistory only if we are past the earliest block
		for _, uncleHeader := range block.Uncles() {
			fmt.Printf("- block %d has uncle: %s\n", block.NumberU64(), uncleHeader.Hash())
			_, err := ro.EnsureBlock(uncleHeader.Hash())
			if err != nil {
				return errors.Wrap(err, "get uncle error")
			}
		}

		// Check and potentially download parent (back until start of monitoring)
		_, found := ro.NodeByHash[block.ParentHash()]
		if !found {
			fmt.Printf("- Parent hash of %d %s not found. Downloading %s\n", block.NumberU64(), block.Hash(), block.ParentHash())
			_, err := ro.EnsureBlock(block.ParentHash())
			if err != nil {
				return errors.Wrap(err, "get unknown parent error")
			}
		}
	}

	ro.DebugPrintln(fmt.Sprintf("- added block %d %s", block.NumberU64(), block.Hash()))
	return nil
}

func (ro *ReorgMonitor) EnsureBlock(blockHash common.Hash) (*types.Block, error) {
	// Check and potentially download block
	block, found := ro.NodeByHash[blockHash]
	if found {
		return block, nil
	}

	fmt.Printf("- Block with hash %s not found, downloading...\n", blockHash)
	block, err := ro.client.BlockByHash(context.Background(), blockHash)
	if err != nil {
		return nil, errors.Wrap(err, "get unknown parent error")
	}

	ro.AddBlock(block)
	return block, nil
}

func (ro *ReorgMonitor) String() string {
	return fmt.Sprintf("ReorgMonitor[%s]: %d .. %d - %d blocks", ro.nickname, ro.EarliestBlockNumber, ro.LatestBlockNumber, len(ro.NodeByHash))
}

func (ro *ReorgMonitor) SubscribeAndStart() string {
	// Subscribe to new blocks
	headers := make(chan *types.Header)
	sub, err := ro.client.SubscribeNewHead(context.Background(), headers)
	reorgutils.Perror(err)

	for {
		select {
		case err := <-sub.Err():
			log.Fatal(err)
		case header := <-headers:
			// Fetch full block information
			block, err := ro.client.BlockByHash(context.Background(), header.Hash())
			if err != nil {
				log.Println("error", err)
				continue
			}

			reorgutils.PrintBlock(block)

			err = ro.AddBlock(block)
			if err != nil {
				log.Println("error", err)
				continue
			}
		}
	}
}

type Reorg struct {
	StartBlockHeight uint64
	EndBlockHeight   uint64
	Depth            uint64
}

func (ro *ReorgMonitor) CheckForReorgs() (reorgs map[uint64]*Reorg) {
	ro.DebugPrintln("CheckForReorgs")
	reorgs = make(map[uint64]*Reorg)

	var currentReorgStartHeight uint64
	var currentReorgEndHeight uint64
	// var currentReorgHeight uint64

	for height := ro.EarliestBlockNumber; height <= ro.LatestBlockNumber; height++ {
		numBlocksAtHeight := len(ro.NodesByHeight[height])
		if numBlocksAtHeight > 1 {
			fmt.Printf("- sibling blocks at %d: %v\n", height, ro.NodesByHeight[height])

			// detect reorg start
			if currentReorgStartHeight == 0 {
				reorgs[height] = &Reorg{StartBlockHeight: height}
				currentReorgStartHeight = height
			}

			reorgs[height].Depth += 1
			currentReorgEndHeight = height

		} else if numBlocksAtHeight == 0 {
			fmt.Printf("- no block at height %d found\n", height)
		} else {
			// 1 block, end of reorg
			if currentReorgStartHeight != 0 {
				reorgs[currentReorgStartHeight].EndBlockHeight = currentReorgEndHeight
				currentReorgStartHeight = 0
			}
		}
	}

	return reorgs
}
