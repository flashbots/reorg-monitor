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

var blockHeightByHash map[common.Hash]uint64 = make(map[common.Hash]uint64) // for looking up known parents, to detect reorgs
var blockHashesByHeight map[uint64][]common.Hash = make(map[uint64][]common.Hash)
var silent bool

func main() {
	log.SetOutput(os.Stdout)

	ethUriPtr := flag.String("eth", os.Getenv("ETH_NODE"), "Geth node URI")
	silentPtr := flag.Bool("silent", false, "don't print info about every block")
	flag.Parse()

	if *ethUriPtr == "" {
		log.Fatal("Missing eth node uri")
	}

	silent = *silentPtr

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
				fmt.Printf("%s \t %s \t %s\n", header.Number, t, header.Hash())
			}

			// Check block
			checkBlockHeader(header, client)

			// Add block to history
			blockHashesByHeight[header.Number.Uint64()] = append(blockHashesByHeight[header.Number.Uint64()], header.Hash())
			blockHeightByHash[header.Hash()] = header.Number.Uint64()
		}
	}
}

func Perror(err error) {
	if err != nil {
		panic(err)
	}
}

func checkBlockHeader(header *types.Header, client *ethclient.Client) {
	if len(blockHeightByHash) == 0 { // nothing to do if we have no history yet
		return
	}

	// Check if a sibling exists (then next block will have an uncle)
	blockHashes, found := blockHashesByHeight[header.Number.Uint64()]
	if found {
		fmt.Printf("- block %s / %s has %d already known siblings: %s\n", header.Number, header.Hash(), len(blockHashes), blockHashes)
	}

	// Check if we know parent. If not then it's a reorg (probably block based on uncle)
	_, found = blockHeightByHash[header.ParentHash]
	if !found {
		reorgDepth := findReorgDepth(header, client)
		fmt.Printf("- reorg with depth=%d in block %s / %s: parent block not found with hash %s\n", reorgDepth, header.Number, header.Hash(), header.ParentHash)
	}
}

func findReorgDepth(header *types.Header, client *ethclient.Client) (depth int64) {
	parentHash := header.ParentHash

	limit := 100
	if len(blockHashesByHeight) < limit {
		limit = len(blockHashesByHeight)
	}

	for {
		// Is a parent already known?
		_, found := blockHeightByHash[parentHash]
		if found {
			return depth
		}

		// No locally known parent, step back one block and check if it's parents is known
		depth += 1
		if depth == int64(limit) {
			log.Println("findReorgDepth limit reached")
			return -1
		}

		cheeckBlock, err := client.HeaderByHash(context.Background(), parentHash)
		Perror(err)
		parentHash = cheeckBlock.ParentHash
	}
}
