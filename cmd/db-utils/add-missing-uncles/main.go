// In case of service downtime, add at least the known uncles to the database
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/big"
	"os"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/flashbots/reorg-monitor/analysis"
	"github.com/flashbots/reorg-monitor/database"
	"github.com/flashbots/reorg-monitor/reorgutils"
	flashbotsrpc "github.com/metachris/flashbots-rpc"
)

var db *database.DatabaseService
var rpc *flashbotsrpc.FlashbotsRPC
var client *ethclient.Client
var mevGethClient *ethclient.Client
var callBundlePrivKey, _ = crypto.GenerateKey()
var ethUriPtr *string

func main() {
	log.SetOutput(os.Stdout)

	ethUriPtr = flag.String("eth", "", "mev-geth node URI")
	mevGethUriPtr := flag.String("mevgeth", "", "mev-geth node URI")
	startBlock := flag.Int64("startblock", 0, "start blockheight")
	endBlock := flag.Int64("endblock", 0, "end blockheight")
	flag.Parse()

	if *ethUriPtr == "" {
		log.Fatal("Missing -eth argument")
	}

	if *mevGethUriPtr == "" {
		log.Fatal("Missing -mevgeth argument")
	}

	if *startBlock == 0 {
		log.Fatal("Missing -startblock argument")
	}

	if *endBlock == 0 {
		log.Fatal("Missing -endblock argument")
	}

	var err error
	fmt.Printf("Connecting to %s...", *ethUriPtr)
	client, err = ethclient.Dial(*ethUriPtr)
	reorgutils.Perror(err)
	fmt.Printf(" ok\n")

	fmt.Printf("Connecting to %s...", *mevGethUriPtr)
	mevGethClient, err = ethclient.Dial(*mevGethUriPtr)
	reorgutils.Perror(err)
	fmt.Printf(" ok\n")

	rpc = flashbotsrpc.NewFlashbotsRPC(*mevGethUriPtr)

	db, err = database.NewDatabaseService(os.Getenv("POSTGRES_DSN"))
	reorgutils.Perror(err)
	fmt.Println("Connected to database")

	blockChan := make(chan *types.Block, 100)

	// Start block processor
	var analyzeLock sync.Mutex
	go func() {
		analyzeLock.Lock()
		defer analyzeLock.Unlock() // we unlock when done

		for block := range blockChan {
			fmt.Printf("Block %d %s \t miner: %s \t tx=%-4d \t gas=%d \t %d\n", block.NumberU64(), block.Hash(), block.Coinbase(), len(block.Transactions()), block.GasUsed(), len(block.Uncles()))

			// Download the uncles for processing later
			for _, uncleHeader := range block.Uncles() {
				fmt.Printf("Downloading uncle %s...\n", uncleHeader.Hash())
				uncleBlock, err := mevGethClient.BlockByHash(context.Background(), uncleHeader.Hash())
				if err != nil {
					fmt.Println("- error:", err)
					continue
				}

				addUncle(uncleBlock, block)
			}
		}
	}()

	// Start getting blocks
	reorgutils.GetBlocks(blockChan, client, *startBlock, *endBlock, 15)

	// Wait until all blocks have been processed
	close(blockChan)
	analyzeLock.Lock()

}

func addUncle(uncleBlock *types.Block, mainchainBlock *types.Block) {
	fmt.Println("addUncle", uncleBlock.Hash())

	// Add to database now
	_, err := db.BlockEntry(uncleBlock.Hash())
	if err == nil { // already exists
		fmt.Println("- block already exists in db, skipping update")
		return
	}

	// get next block
	nextHeight := new(big.Int).Add(uncleBlock.Number(), common.Big1)
	mainChainBlockChild1, err := client.BlockByNumber(context.Background(), nextHeight)
	reorgutils.Perror(err)

	observed := time.Now().UTC().UnixNano()
	reorg := analysis.Reorg{
		IsFinished:           true,
		SeenLive:             false,
		StartBlockHeight:     uncleBlock.NumberU64(),
		EndBlockHeight:       uncleBlock.NumberU64(),
		Chains:               make(map[common.Hash][]*analysis.Block),
		Depth:                1,
		BlocksInvolved:       make(map[common.Hash]*analysis.Block),
		MainChainHash:        mainchainBlock.Hash(),
		MainChainBlocks:      make(map[common.Hash]*analysis.Block),
		NumReplacedBlocks:    1,
		EthNodesInvolved:     make(map[string]bool),
		FirstBlockAfterReorg: analysis.NewBlock(mainChainBlockChild1, analysis.OriginUncle, *ethUriPtr, observed),
	}

	_uncleBlock := analysis.NewBlock(uncleBlock, analysis.OriginUncle, *ethUriPtr, observed)
	_mainChainBlock := analysis.NewBlock(mainchainBlock, analysis.OriginSubscription, *ethUriPtr, observed)

	// Update reorg details
	reorg.Chains[_mainChainBlock.Hash] = []*analysis.Block{_mainChainBlock}
	reorg.Chains[_uncleBlock.Hash] = []*analysis.Block{_uncleBlock}
	reorg.BlocksInvolved[_mainChainBlock.Hash] = _mainChainBlock
	reorg.BlocksInvolved[_uncleBlock.Hash] = _uncleBlock
	reorg.MainChainBlocks[_mainChainBlock.Hash] = _mainChainBlock
	reorg.EthNodesInvolved[*ethUriPtr] = true

	// Create block entries
	fmt.Println("- simulating uncle block...")
	uncleSimRes, err := rpc.FlashbotsSimulateBlock(callBundlePrivKey, uncleBlock, 0)
	if err != nil {
		fmt.Println("-", err)
		return
	}
	uncleBlockEntry := database.NewBlockEntry(_uncleBlock, &reorg)
	uncleBlockEntry.UpdateWitCallBundleResponse(uncleSimRes)

	fmt.Println("- simulating mainchain block...")
	mainChainSimRes, err := rpc.FlashbotsSimulateBlock(callBundlePrivKey, uncleBlock, 0)
	if err != nil {
		fmt.Println("-", err)
		return
	}
	mainChainBlockEntry := database.NewBlockEntry(_mainChainBlock, &reorg)
	mainChainBlockEntry.UpdateWitCallBundleResponse(mainChainSimRes)

	// Time to use for created_at is time of next block
	t := time.Unix(int64(mainChainBlockChild1.Header().Time), 0).UTC()
	tf := t.Format("2006-01-02 15:04:05")

	// Save
	reorgEntry := database.NewReorgEntry(&reorg)

	err = db.AddReorgEntry(reorgEntry)
	if err != nil {
		fmt.Println("-", err)
		return
	}

	_, err = db.DB.Exec("Update reorg_summary SET Created_At=$1 WHERE Key=$2", tf, reorgEntry.Key)
	if err != nil {
		fmt.Println("-", err)
		return
	}

	err = db.AddBlockEntry(uncleBlockEntry)
	if err != nil {
		fmt.Println("-", err)
		return
	}

	err = db.AddBlockEntry(mainChainBlockEntry)
	if err != nil {
		fmt.Println("-", err)
		return
	}

	_, err = db.DB.Exec("Update reorg_block SET Created_At=$1 WHERE Reorg_Key=$2", tf, reorgEntry.Key)
	if err != nil {
		fmt.Println("-", err)
		return
	}
}
