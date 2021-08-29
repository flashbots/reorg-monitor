package main

import (
	"flag"
	"log"
	"os"

	"github.com/metachris/eth-reorg-monitor/analysis"
	"github.com/metachris/eth-reorg-monitor/database"
	"github.com/metachris/eth-reorg-monitor/reorgutils"
	"github.com/metachris/eth-reorg-monitor/testutils"
)

var db *database.DatabaseService

func main() {
	dbConfig := database.PostgresConfig{
		User:       "user1",
		Password:   "password",
		Host:       "localhost",
		Name:       "reorg",
		DisableTLS: true,
	}
	db = database.NewDatabaseService(dbConfig)

	log.SetOutput(os.Stdout)

	ethUriPtr := flag.String("eth", os.Getenv("GETH_ETH1LOCAL"), "Geth node URI")
	flag.Parse()

	if *ethUriPtr == "" {
		log.Fatal("Missing eth node uri")
	}

	testutils.EthNodeUri = *ethUriPtr
	reorgutils.Perror(testutils.ConnectClient(*ethUriPtr))

	// Test(testutils.Test_12996760_12996760_d1_b2)
	// Test(testutils.Test_12996750_12996750_d1_b3)
	// Test(testutils.Test_12991732_12991733_d2_b4)
	// Test(testutils.Test_12969887_12969889_d3_b6)
	// Test(testutils.Test_13017535_13017536_d2_b5)
	// Test(testutils.Test_13018369_13018370_d2_b4)

	// test := testutils.Test_13018369_13018370_d2_b4
	// test := testutils.Test_12996750_12996750_d1_b3_twouncles
	test := testutils.Test_Tmp
	testutils.ResetMon("")

	// Add the blocks
	for _, ethBlock := range testutils.BlocksForStrings(test.BlockInfo) {
		block := analysis.NewBlock(ethBlock, analysis.OriginSubscription, *ethUriPtr)
		testutils.Monitor.AddBlock(block)
	}

	// reorgs := testutils.ReorgCheckAndPrint()
	// testutils.Pcheck("NumReorgs", len(reorgs), 1)

	// reorg := reorgs[0]
	// fmt.Println(reorg)

	// entry := database.NewReorgEntry(reorg)
	// db.AddReorgEntry(entry)

	// // blocks
	// for _, block := range reorg.BlocksInvolved {
	// 	blockEntry := database.NewBlockEntry(block, reorg)
	// 	db.AddBlockEntry(blockEntry)
	// }
}
