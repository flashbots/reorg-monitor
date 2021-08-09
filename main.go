package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

var blockByHash map[common.Hash]*types.Block = make(map[common.Hash]*types.Block) // for looking up known parents, to detect reorgs
var blockHashesByHeight map[uint64][]common.Hash = make(map[uint64][]common.Hash)
var earningsService *EarningsService

var silent bool
var minReorgDepthToNotify uint64

var blockInfoCache map[common.Hash]*BlockInfo = make(map[common.Hash]*BlockInfo) // wait 5 blocks before writing, because instantly we don't know if a block is going to be reorged

var csvWriter *csv.Writer
var latestHeightWrittenToCsv uint64

func main() {
	log.SetOutput(os.Stdout)

	ethUriPtr := flag.String("eth", os.Getenv("ETH_NODE"), "Geth node URI")
	silentPtr := flag.Bool("silent", false, "only print alerts, no info about every block")
	minReorgDepthPtr := flag.Uint64("mindepth", 1, "minimum reorg depth to notify")
	csvFilename := flag.String("csv", "", "CSV file for saving blocks")
	flag.Parse()

	if *ethUriPtr == "" {
		log.Fatal("Missing eth node uri")
	}

	// Setup the CSV writer
	if *csvFilename != "" {
		if FileExists(*csvFilename) {
			log.Fatal("File already exists:", *csvFilename)
		}

		file, err := os.Create(*csvFilename)
		Perror(err)
		defer file.Close()
		csvWriter = csv.NewWriter(file)
		defer csvWriter.Flush()

		// write CSV header
		csvAddLine(BlockInfoCsvRecordHeader)
	}

	silent = *silentPtr
	minReorgDepthToNotify = *minReorgDepthPtr

	// Connect to geth node
	fmt.Printf("Connecting to %s...", *ethUriPtr)
	client, err := ethclient.Dial(*ethUriPtr)
	Perror(err)
	fmt.Printf(" ok\n")

	// Setup the service to query block earnings
	earningsService = NewEarningsService(client)

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
			earnings, err := earningsService.GetBlockCoinbaseEarnings(block)
			if err != nil {
				log.Println("getEarnings error:", err)
				earnings = big.NewInt(-1)
			}

			// Print block
			if !silent {
				t := time.Unix(int64(header.Time), 0).UTC()
				log.Printf("Block %s %s \t %s \t tx: %3d, uncles: %d, earnings: %s \n", block.Number(), block.Hash(), t, len(block.Transactions()), len(block.Uncles()), BalanceToEthStr(earnings))
			}

			// Check block
			reorgFound := checkBlock(block, client)

			// Add block to history
			addBlockToHistory(block)

			// Write to CSV
			csvWriteBlockInfo(header.Number.Uint64(), reorgFound)
		}
	}
}

func addBlockToHistory(block *types.Block) {
	blockByHash[block.Hash()] = block
	blockHashesByHeight[block.NumberU64()] = append(blockHashesByHeight[block.NumberU64()], block.Hash())
}

func checkBlock(block *types.Block, client *ethclient.Client) (reorgFound bool) {
	earnings, _ := earningsService.GetBlockCoinbaseEarnings(block)
	blockInfoCache[block.Hash()] = NewBlockInfoFromBlock(block, earnings)

	// Check if we know parent. If not then it's a reorg (this node got first the future uncle and possibly children).
	_, found := blockByHash[block.ParentHash()]
	if found || len(blockByHash) == 0 { // parent was found, no reorg
		return false
	}

	// We have a reorg. Find the new full chain of new blocks
	newBlocks := findNewChain(block.Header(), client)
	if len(newBlocks) == 0 {
		log.Println("Possible reorg, but didn't couldn't determine new blocks (probably didn't run long enough to find common ancestor)")
		return true
	}

	// Find reorg depth and replaced blocks
	reorgDepth, replacedBlocks := findReplacedBlocks(newBlocks)

	// Add new blocks to history and blockInfoCache
	for _, newBlock := range newBlocks {
		addBlockToHistory(newBlock)

		earnings, _ := earningsService.GetBlockCoinbaseEarnings(newBlock)
		blockInfoCache[newBlock.Hash()] = NewBlockInfoFromBlock(newBlock, earnings)
	}

	// Add replaced (reorged) blocks to blockInfoCache
	for blockIndex, block := range replacedBlocks {
		earnings, _ := earningsService.GetBlockCoinbaseEarnings(block)
		blockInfoCache[block.Hash()] = &BlockInfo{
			Block:              block,
			CoinbaseDifference: earnings,
			IsReorged:          true,
			IsUncle:            blockIndex == 0,
			IsChild:            blockIndex > 0,
			ReorgDepth:         blockIndex + 1,
		}
	}

	// Alert
	if reorgDepth >= minReorgDepthToNotify {
		reorgAlert(block, reorgDepth, replacedBlocks, newBlocks)
	}

	return true
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

func findReplacedBlocks(newBlocks []*types.Block) (depth uint64, replacedBlocks []*types.Block) {
	replacedBlocks = make([]*types.Block, 0)

	earliestNewBlock := *newBlocks[len(newBlocks)-1]
	lastCommonBlockHash := earliestNewBlock.ParentHash() // parent of last (earliest) new block
	lastCommonBlock, found := blockByHash[lastCommonBlockHash]
	if !found {
		log.Printf("findReplacedBlocks error - couldn't find last common block %s\n", lastCommonBlockHash)
		return 0, replacedBlocks
	}

	lastCommonBlockNumber := lastCommonBlock.Header().Number.Uint64()
	blockNumber := lastCommonBlockNumber // iterate forward starting at last common block
	for {
		blockNumber += 1
		blockHashesAtHeight, found := blockHashesByHeight[blockNumber]
		if !found {
			break
		}

		for _, hash := range blockHashesAtHeight {
			block := blockByHash[hash]
			replacedBlocks = append(replacedBlocks, block)
		}
	}

	reorgDepth := blockNumber - lastCommonBlockNumber - 1
	return reorgDepth, replacedBlocks
}

func reorgAlert(latestBlock *types.Block, depth uint64, replacedChainSement []*types.Block, newChainSegment []*types.Block) {
	msg := fmt.Sprintf("Reorg with depth=%d in block %s", depth, latestBlock.Header().Number)
	log.Println(msg)

	earliestNewBlock := *newChainSegment[len(newChainSegment)-1]
	lastCommonBlockHash := earliestNewBlock.ParentHash() // parent of last (earliest) new block
	lastCommonBlock := blockByHash[lastCommonBlockHash]
	lastCommonBlockNumber := lastCommonBlock.Header().Number.Uint64()

	fmt.Println("Last common block:")
	earnings, _ := earningsService.GetBlockCoinbaseEarnings(latestBlock)
	fmt.Printf("- %d %3s / %3d tx, miner %s, earnings: %s ETH\n", lastCommonBlockNumber, lastCommonBlockHash, len(lastCommonBlock.Transactions()), lastCommonBlock.Coinbase(), BalanceToEthStr(earnings))

	// fmt.Println("Old chain (replaced blocks):")
	// blockNumber := lastCommonBlockNumber
	// for {
	// 	blockNumber += 1
	// 	hashes, found := blockHashesByHeight[blockNumber]
	// 	if !found {
	// 		break
	// 	}

	// 	for _, hash := range hashes { // block can have more than 1 uncles
	// 		blockInfo := ""
	// 		replacedBlock, found := blockByHash[hash]
	// 		if found {
	// 			blockInfo += fmt.Sprintf("/ %3d tx, miner %s, ", len(replacedBlock.Transactions()), replacedBlock.Coinbase())
	// 		}

	// 		earnings, _ := earningsService.GetBlockCoinbaseEarnings(replacedBlock)
	// 		blockInfo += fmt.Sprintf("earnings: %s ETH", BalanceToEthStr(earnings))
	// 		if blockNumber == lastCommonBlockNumber+1 {
	// 			blockInfo += " (now uncle)"
	// 		}
	// 		fmt.Printf("- %d %s %s\n", blockNumber, hash.String(), blockInfo)
	// 	}
	// }

	fmt.Println("Old chain (replaced blocks):")
	for _, replacedBlock := range replacedChainSement {
		earnings, _ := earningsService.GetBlockCoinbaseEarnings(replacedBlock)
		blockInfo := fmt.Sprintf("- %d %s / %3d tx, miner %s, earnings: %s ETH", replacedBlock.Number(), replacedBlock.Hash(), len(replacedBlock.Transactions()), replacedBlock.Coinbase(), BalanceToEthStr(earnings))
		if replacedBlock.NumberU64() == lastCommonBlockNumber+1 {
			blockInfo += " (now uncle)"
		}
		fmt.Println(blockInfo)
	}

	fmt.Println("New chain after reorg:")
	for i := len(newChainSegment) - 1; i >= 0; i-- {
		earnings, _ := earningsService.GetBlockCoinbaseEarnings(newChainSegment[i])
		fmt.Printf("- %s %s / %3d tx, miner %s, earnings: %s ETH\n", newChainSegment[i].Number(), newChainSegment[i].Hash(), len(newChainSegment[i].Transactions()), newChainSegment[i].Coinbase(), BalanceToEthStr(earnings))
	}

	earnings, _ = earningsService.GetBlockCoinbaseEarnings(latestBlock)
	fmt.Printf("- %d %s / %3d tx, miner %s, earnings: %s ETH\n", latestBlock.Number(), latestBlock.Hash(), len(latestBlock.Transactions()), latestBlock.Coinbase(), BalanceToEthStr(earnings))
	fmt.Println("")
}

func csvAddLine(record []string) {
	if csvWriter != nil {
		err := csvWriter.Write(record)
		csvWriter.Flush()
		if err != nil {
			log.Println("error writing to csv:", err)
		}
	}
}

func csvWriteBlockInfo(latestHeight uint64, debug bool) {
	if latestHeightWrittenToCsv == 0 {
		latestHeightWrittenToCsv = latestHeight - 1
		return
	}

	if latestHeightWrittenToCsv < latestHeight-5 {
		if debug {
			fmt.Printf("csvWriteBlockInfo debug: latestHeightWrittenToCsv: %d, latestHeight: %d\n", latestHeightWrittenToCsv, latestHeight)
		}
		for height := latestHeightWrittenToCsv + 1; height <= latestHeight-5; height++ {
			// Get all hashes for this height
			hashes, found := blockHashesByHeight[height]
			if debug {
				fmt.Printf("- height: %d, found: %v, hashes: %v \n", height, found, hashes)
			}
			if !found {
				continue
			}

			// For all blocks at this height, save to CSV
			for _, hash := range hashes {
				blockInfo, found := blockInfoCache[hash]
				if debug {
					fmt.Printf("-- hash: %s, blockInfo found: %v \n", hash, found)
				}
				if !found {
					continue
				}

				csvAddLine(blockInfo.ToCsvRecord())
			}
			latestHeightWrittenToCsv = height
		}
	}
}
