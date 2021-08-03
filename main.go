package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

var blockHeaderByHash map[common.Hash]*types.Header = make(map[common.Hash]*types.Header) // for looking up known parents, to detect reorgs
var blockHashesByHeight map[uint64][]common.Hash = make(map[uint64][]common.Hash)

var silent bool
var minReorgDepthToNotify int64

func main() {
	log.SetOutput(os.Stdout)

	ethUriPtr := flag.String("eth", os.Getenv("ETH_NODE"), "Geth node URI")
	silentPtr := flag.Bool("silent", false, "only print alerts, no info about every block")
	minReorgDepthPtr := flag.Int64("mindepth", 1, "minimum reorg depth to notify")
	flag.Parse()

	if *ethUriPtr == "" {
		log.Fatal("Missing eth node uri")
	}

	silent = *silentPtr
	minReorgDepthToNotify = *minReorgDepthPtr

	// Connect to geth node
	fmt.Printf("Connecting to %s...", *ethUriPtr)
	client, err := ethclient.Dial(*ethUriPtr)
	Perror(err)
	fmt.Printf(" ok\n")

	// Subscribe to new blocks
	headers := make(chan *types.Header)
	sub, err := client.SubscribeNewHead(context.Background(), headers)
	Perror(err)

	for {
		select {
		case err := <-sub.Err():
			log.Fatal(err)
		case header := <-headers:
			// Print block
			if !silent {
				t := time.Unix(int64(header.Time), 0).UTC()
				log.Printf("Block %s \t %s \t %s\n", header.Number, t, header.Hash())
			}

			// Check block
			checkBlockHeader(header, client)

			// Add block to history
			blockHeaderByHash[header.Hash()] = header
			blockHashesByHeight[header.Number.Uint64()] = append(blockHashesByHeight[header.Number.Uint64()], header.Hash())
		}
	}
}

func Perror(err error) {
	if err != nil {
		panic(err)
	}
}

func checkBlockHeader(header *types.Header, client *ethclient.Client) {
	if len(blockHeaderByHash) == 0 { // nothing to do if we have no history yet
		return
	}

	// // Check if a sibling exists (then next block will have an uncle)
	// blockHashes, found := blockHashesByHeight[header.Number.Uint64()]
	// if found {
	// 	fmt.Printf("- block %s / %s has %d already known siblings: %s\n", header.Number, header.Hash(), len(blockHashes), blockHashes)
	// }

	// Check if we know parent. If not then it's a reorg (probably block based on uncle)
	_, found := blockHeaderByHash[header.ParentHash]
	if !found {
		reorgDepth, newHeaders := findReorgDepth(header, client)
		if reorgDepth >= minReorgDepthToNotify {
			reorgAlert(header, reorgDepth, newHeaders)
		}
	}
}

// findReorgDepth
func findReorgDepth(header *types.Header, client *ethclient.Client) (depth int64, newBlockHeaders []*types.Header) {
	newBlockHeaders = make([]*types.Header, 0)

	limit := 100
	if len(blockHeaderByHash) < limit {
		limit = len(blockHeaderByHash)
	}

	parentHash := header.ParentHash
	for {
		// Finish search when finding a known parent
		_, found := blockHeaderByHash[parentHash]
		if found {
			return depth, newBlockHeaders
		}

		// Avoid an endless loop
		if depth == int64(limit) {
			log.Printf("findReorgDepth error on block %d %s: search limit reached without finding parent\n", header.Number.Uint64(), header.Hash())
			return -1, newBlockHeaders
		}

		// Step back one more block to check if the parent is known
		depth += 1
		checkBlockHeader, err := client.HeaderByHash(context.Background(), parentHash)
		Perror(err)
		newBlockHeaders = append(newBlockHeaders, checkBlockHeader)
		parentHash = checkBlockHeader.ParentHash
		// fmt.Printf("findReorgDepth check depth %d: block %d %s. It's parent: %s\n", depth, checkBlockHeader.Number.Uint64(), checkBlockHeader.Hash(), parentHash)
	}
}

func reorgAlert(newHeader *types.Header, depth int64, newBlockHeaders []*types.Header) {
	log.Printf("Reorg with depth=%d in block %s %s: parent block not found with hash %s\n", depth, newHeader.Number, newHeader.Hash(), newHeader.ParentHash)

	earliestNewBlock := *newBlockHeaders[len(newBlockHeaders)-1]
	lastCommonBlockHash := earliestNewBlock.ParentHash // parent of last (earliest) new block
	lastCommonBlockHeader := blockHeaderByHash[lastCommonBlockHash]
	lastCommonBlockNumber := lastCommonBlockHeader.Number.Uint64()

	fmt.Println("Last common block:")
	fmt.Printf("- %d %s\n", lastCommonBlockNumber, lastCommonBlockHash)

	fmt.Println("Old chain (replaced blocks):")
	blockNumber := lastCommonBlockNumber
	for {
		blockNumber += 1
		hashes, found := blockHashesByHeight[blockNumber]
		if !found {
			break
		}

		// hashes could be more than 1 if it has siblings. pretty print
		hashesStr := hashes[0].String()
		if len(hashes) > 1 {
			hashesStr = fmt.Sprintf("%s", hashes)
		}

		// add uncle information (first replaced block in old path became the uncle)
		uncleStr := "(now uncle)"
		if blockNumber > lastCommonBlockNumber+1 {
			uncleStr = ""
		}
		fmt.Printf("- %d %s %s\n", blockNumber, hashesStr, uncleStr)

		// Here is a good place to save data from transactions that are children of the uncle and not included in the node DB
	}

	fmt.Println("New chain after reorg:")
	for i := len(newBlockHeaders) - 1; i >= 0; i-- {
		fmt.Printf("- %d %s\n", newBlockHeaders[i].Number.Uint64(), newBlockHeaders[i].Hash())
	}
	fmt.Printf("- %d %s\n", newHeader.Number, newHeader.Hash())

	// Note: add custom notification logic here
}
