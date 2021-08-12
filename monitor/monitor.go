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
	_, knownBlock := ro.NodeByHash[block.Hash()]
	if !knownBlock {
		// Add for access by hash
		ro.NodeByHash[block.Hash()] = block

		// Add to array of blocks by height
		if _, found := ro.NodesByHeight[block.NumberU64()]; !found {
			ro.NodesByHeight[block.NumberU64()] = make([]*types.Block, 1)
		}
		ro.NodesByHeight[block.NumberU64()] = append(ro.NodesByHeight[block.NumberU64()], block)
	}

	if ro.EarliestBlockNumber == 0 {
		ro.EarliestBlockNumber = block.NumberU64()
	}

	if ro.LatestBlockNumber < block.NumberU64() {
		ro.LatestBlockNumber = block.NumberU64()
	}

	ro.DebugPrintln(fmt.Sprintf("Added block %d %s. %s", block.NumberU64(), block.Hash(), ro.String()))

	// Check and potentially download parent (back until start of monitoring)
	if block.NumberU64() > ro.EarliestBlockNumber {
		_, found := ro.NodeByHash[block.ParentHash()]
		if !found {
			ro.DebugPrintln(fmt.Sprintf("- Parent hash not found (%s)", block.ParentHash()))
			block, err := ro.client.BlockByHash(context.Background(), block.ParentHash())
			if err != nil {
				return errors.Wrap(err, "get unknown parent error")
			}
			ro.AddBlock(block)
		}
	}

	return nil
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
