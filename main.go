package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

var blockByHash map[common.Hash]*types.Block = make(map[common.Hash]*types.Block) // for looking up known parents, to detect reorgs
var blockHashesByHeight map[uint64][]common.Hash = make(map[uint64][]common.Hash)
var minerEarningsByBlockHash map[common.Hash]*big.Int = make(map[common.Hash]*big.Int)
var silent bool
var minReorgDepthToNotify uint64

func Perror(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	log.SetOutput(os.Stdout)

	ethUriPtr := flag.String("eth", os.Getenv("ETH_NODE"), "Geth node URI")
	silentPtr := flag.Bool("silent", false, "only print alerts, no info about every block")
	minReorgDepthPtr := flag.Uint64("mindepth", 1, "minimum reorg depth to notify")
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
			// Fetch full block information
			block, err := client.BlockByHash(context.Background(), header.Hash())
			if err != nil {
				log.Println("error", err)
				continue
			}

			// Get miner earnings
			earnings, err := GetBlockCoinbaseEarnings(client, block)
			if err != nil {
				log.Println("error", err)
				continue
			}
			minerEarningsByBlockHash[header.Hash()] = earnings

			// Print block
			// if !silent || len(block.Uncles()) > 0 {
			if !silent {
				t := time.Unix(int64(header.Time), 0).UTC()
				log.Printf("Block %s %s \t %s \t tx: %3d, uncles: %d, earnings: %s \n", block.Number(), block.Hash(), t, len(block.Transactions()), len(block.Uncles()), BalanceToEth(earnings).Text('f', 4))
			}

			// Check block
			checkBlock(block, client)

			// Add block to history
			blockByHash[header.Hash()] = block
			blockHashesByHeight[header.Number.Uint64()] = append(blockHashesByHeight[header.Number.Uint64()], header.Hash())
		}
	}
}

func checkBlock(block *types.Block, client *ethclient.Client) {
	if len(blockByHash) == 0 { // nothing to do if we have no history yet
		return
	}

	// // Check if a sibling exists (then next block will have an uncle)
	// blockHashes, found := blockHashesByHeight[header.Number.Uint64()]
	// if found {
	// 	fmt.Printf("- block %s / %s has %d already known siblings: %s\n", header.Number, header.Hash(), len(blockHashes), blockHashes)
	// }

	// Check if we know parent. If not then it's a reorg (this node got first the future uncle and possibly children).
	_, found := blockByHash[block.ParentHash()]
	if !found {
		newBlocks := findNewChain(block.Header(), client)
		if len(newBlocks) == 0 {
			log.Println("Possible reorg, but didn't couldn't determine new blocks (probably didn't run long enough to find common ancestor)")
			return
		}

		reorgDepth := findReplacedBlockDepth(newBlocks)
		if reorgDepth >= minReorgDepthToNotify {
			reorgAlert(block, reorgDepth, newBlocks)
		}
	}
}

// findReorgDepth: count chain of replaced blocks
func findNewChain(header *types.Header, client *ethclient.Client) (newBlocks []*types.Block) {
	newBlocks = make([]*types.Block, 0)

	limit := 100
	if len(blockByHash) < limit {
		limit = len(blockByHash)
	}

	parentHash := header.ParentHash
	for {
		// Finish search when finding a known parent
		_, found := blockByHash[parentHash]
		if found {
			return newBlocks
		}

		// Avoid an endless loop
		if len(newBlocks) == limit {
			log.Printf("findReorgDepth error on block %d %s: search limit reached without finding parent\n", header.Number.Uint64(), header.Hash())
			return newBlocks
		}

		// Step back one more block to check if the parent is known
		checkBlock, err := client.BlockByHash(context.Background(), parentHash)
		Perror(err)
		newBlocks = append(newBlocks, checkBlock)
		parentHash = checkBlock.ParentHash()
		// fmt.Printf("findReorgDepth check depth %d: block %d %s. It's parent: %s\n", depth, checkBlockHeader.Number.Uint64(), checkBlockHeader.Hash(), parentHash)
	}
}

func findReplacedBlockDepth(newBlocks []*types.Block) uint64 {
	earliestNewBlock := *newBlocks[len(newBlocks)-1]
	lastCommonBlockHash := earliestNewBlock.ParentHash() // parent of last (earliest) new block
	lastCommonBlock := blockByHash[lastCommonBlockHash]
	lastCommonBlockNumber := lastCommonBlock.Header().Number.Uint64()

	blockNumber := lastCommonBlockNumber // iterate forward starting at last common block
	for {
		blockNumber += 1
		_, found := blockHashesByHeight[blockNumber]
		if !found {
			break
		}
	}
	return blockNumber - lastCommonBlockNumber - 1
}

func reorgAlert(latestBlock *types.Block, depth uint64, newChainSegment []*types.Block) {
	msg := fmt.Sprintf("Reorg with depth=%d in block %s", depth, latestBlock.Header().Number)
	// if depth > 1 {
	// 	msg = fmt.Sprintf("\033[1;33m%s\033[0m", msg)
	// }
	log.Println(msg)
	// log.Printf("Reorg with depth=%d in block %s %s: parent block not found with hash %s\n", depth, newBlock.Header().Number, newBlock.Header().Hash(), newBlock.Header().ParentHash)

	earliestNewBlock := *newChainSegment[len(newChainSegment)-1]
	lastCommonBlockHash := earliestNewBlock.ParentHash() // parent of last (earliest) new block
	lastCommonBlock := blockByHash[lastCommonBlockHash]
	lastCommonBlockNumber := lastCommonBlock.Header().Number.Uint64()

	fmt.Println("Last common block:")
	fmt.Printf("- %d %3s / %d tx, miner %s\n", lastCommonBlockNumber, lastCommonBlockHash, len(lastCommonBlock.Transactions()), lastCommonBlock.Coinbase())

	fmt.Println("Old chain (replaced blocks):")
	blockNumber := lastCommonBlockNumber
	for {
		blockNumber += 1
		hashes, found := blockHashesByHeight[blockNumber]
		if !found {
			break
		}

		// hashes could be more than 1 if it has siblings. pretty print
		for _, hash := range hashes {
			blockInfo := ""
			replacedBlock, found := blockByHash[hash]
			if found {
				blockInfo += fmt.Sprintf("/ %3d tx, miner %s, ", len(replacedBlock.Transactions()), replacedBlock.Coinbase())
			}
			minerEarnings := minerEarningsByBlockHash[hash]
			blockInfo += fmt.Sprintf("earnings: %s ETH", BalanceToEth(minerEarnings).Text('f', 2))
			if blockNumber == lastCommonBlockNumber+1 {
				blockInfo += " (now uncle)"
			}
			fmt.Printf("- %d %s %s\n", blockNumber, hash.String(), blockInfo)
		}

		// Here is a good place to save data from transactions that are children of the uncle and not included in the node DB
	}

	fmt.Println("New chain after reorg:")
	for i := len(newChainSegment) - 1; i >= 0; i-- {
		fmt.Printf("- %s %s / %3d tx, miner %s\n", newChainSegment[i].Number(), newChainSegment[i].Hash(), len(newChainSegment[i].Transactions()), newChainSegment[i].Coinbase())
	}
	fmt.Printf("- %d %s / %3d tx, miner %s\n", latestBlock.Number(), latestBlock.Hash(), len(latestBlock.Transactions()), latestBlock.Coinbase())
	fmt.Println("")
	// Note: add custom notification logic here
}

func BalanceToEth(balance *big.Int) *big.Float {
	fbalance := new(big.Float)
	fbalance.SetInt(balance)
	// fbalance.SetString(balance)
	ethValue := new(big.Float).Quo(fbalance, big.NewFloat(math.Pow10(18)))
	return ethValue
}

func GetBlockCoinbaseEarnings(client *ethclient.Client, block *types.Block) (*big.Int, error) {
	balanceAfterBlock, err := client.BalanceAt(context.Background(), block.Coinbase(), block.Number())
	if err != nil {
		return nil, err
	}

	balanceBeforeBlock, err := client.BalanceAt(context.Background(), block.Coinbase(), new(big.Int).Sub(block.Number(), common.Big1))
	if err != nil {
		return nil, err
	}

	earnings := new(big.Int).Sub(balanceAfterBlock, balanceBeforeBlock)

	// Iterate over all transactions - add sent value back into earnings, remove received value
	for _, tx := range block.Transactions() {
		from, err := types.Sender(types.NewLondonSigner(tx.ChainId()), tx)
		if err != nil {
			fmt.Println("getsender error", err, tx.Hash())
			continue
		}

		if from == block.Coinbase() {
			earnings = new(big.Int).Add(earnings, tx.Value())
		}

		to := tx.To()
		if to != nil && *to == block.Coinbase() {
			earnings = new(big.Int).Sub(earnings, tx.Value())
		}
	}

	return earnings, nil
}
