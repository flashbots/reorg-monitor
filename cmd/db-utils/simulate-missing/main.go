// Fetch all blocks that haven't been simulated from the database -> simulate it -> update the db entry
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/metachris/eth-reorg-monitor/database"
	"github.com/metachris/eth-reorg-monitor/reorgutils"
	flashbotsrpc "github.com/metachris/flashbots-rpc"
)

var db *database.DatabaseService
var client *ethclient.Client
var rpc *flashbotsrpc.FlashbotsRPC
var callBundlePrivKey, _ = crypto.GenerateKey()

func main() {
	log.SetOutput(os.Stdout)

	ethUriPtr := flag.String("eth", "", "Geth node URI")
	flag.Parse()

	if *ethUriPtr == "" {
		log.Fatal("Missing eth node uri")
	}

	var err error
	fmt.Printf("Connecting to %s...", *ethUriPtr)
	client, err = ethclient.Dial(*ethUriPtr)
	reorgutils.Perror(err)
	fmt.Printf(" ok\n")

	rpc = flashbotsrpc.NewFlashbotsRPC(*ethUriPtr)

	dbCfg := database.GetDbConfig()
	db = database.NewDatabaseService(dbCfg)
	fmt.Println("Connected to database at", dbCfg.Host)

	blockEntries := []database.BlockEntry{}
	// db.DB.Select(&blockEntries, "SELECT * FROM reorg_block WHERE MevGeth_CoinbaseDiffWei=0 AND NumTx>0")
	db.DB.Select(&blockEntries, "SELECT * FROM reorg_block WHERE MevGeth_CoinbaseDiffWei=-1")
	if len(blockEntries) == 0 {
		return
	}
	fmt.Println(len(blockEntries))

	for _, blockEntry := range blockEntries {
		fmt.Println("Checking block", blockEntry.BlockNumber, blockEntry.BlockHash, blockEntry.NumTx)
		ethBlock, err := client.BlockByHash(context.Background(), common.HexToHash(blockEntry.BlockHash))
		if err != nil {
			fmt.Println("-", err)
			continue
		}

		if len(ethBlock.Transactions()) == 0 {
			blockEntry.MevGeth_CoinbaseDiffWei = "0"
			blockEntry.MevGeth_GasFeesWei = "0"
			blockEntry.MevGeth_EthSentToCoinbaseWei = "0"
			blockEntry.MevGeth_EthSentToCoinbase = "0"
			blockEntry.MevGeth_CoinbaseDiffEth = "0"
		} else {
			res, err := rpc.FlashbotsSimulateBlock(callBundlePrivKey, ethBlock, 0)
			if err != nil {
				fmt.Println("-", err)
				continue
			}
			blockEntry.UpdateWitCallBundleResponse(res)
		}

		_, err = db.DB.Exec("Update reorg_block SET MevGeth_CoinbaseDiffWei=$1, MevGeth_GasFeesWei=$2, MevGeth_EthSentToCoinbaseWei=$3, MevGeth_EthSentToCoinbase=$4, MevGeth_CoinbaseDiffEth=$5 WHERE id=$6",
			blockEntry.MevGeth_CoinbaseDiffWei, blockEntry.MevGeth_GasFeesWei, blockEntry.MevGeth_EthSentToCoinbaseWei, blockEntry.MevGeth_EthSentToCoinbase, blockEntry.MevGeth_CoinbaseDiffEth, blockEntry.Id)
		reorgutils.Perror(err)
		fmt.Println("updated block", blockEntry.Id)
		// return
	}
}
