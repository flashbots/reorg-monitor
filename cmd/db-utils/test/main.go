package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/flashbots/reorg-monitor/analysis"
	"github.com/flashbots/reorg-monitor/database"
	"github.com/flashbots/reorg-monitor/reorgutils"
	"github.com/flashbots/reorg-monitor/testutils"
)

var db *database.DatabaseService

func main() {
	var (
		postgresDsn = "postgres://user1:password@localhost:5432/reorg?sslmode=disable"
		err         error
	)

	db, err = database.NewDatabaseService(postgresDsn)
	reorgutils.Perror(err)

	log.SetOutput(os.Stdout)

	ethUriPtr := flag.String("eth", os.Getenv("GETH_ETH1LOCAL"), "Geth node URI")
	flag.Parse()

	if *ethUriPtr == "" {
		log.Fatal("Missing eth node uri")
	}

	testutils.EthNodeUri = *ethUriPtr
	_, err = testutils.ConnectClient(*ethUriPtr)
	reorgutils.Perror(err)

	// Test(testutils.Test_12996760_12996760_d1_b2)
	// Test(testutils.Test_12996750_12996750_d1_b3)
	// Test(testutils.Test_12991732_12991733_d2_b4)
	// Test(testutils.Test_12969887_12969889_d3_b6)
	// Test(testutils.Test_13017535_13017536_d2_b5)
	// Test(testutils.Test_13018369_13018370_d2_b4)

	// test := testutils.Test_13018369_13018370_d2_b4
	// test := testutils.Test_12996750_12996750_d1_b3_twouncles
	// test := testutils.Test3xD1
	test := testutils.TestCase{
		BlockInfo: []string{
			"13400397",
			"0xe11507e3ab485efa72d4221e5a2e58d35809425f0bb486d702dc38efe894c830",
			"0x5824ce21441e5d30e8685a90cb70bb24d09ca6533fb3063f35bd5b9a05e480b8",
			"13400402",
		},
	}
	testutils.ResetMon()

	// Add the blocks
	for _, ethBlock := range testutils.BlocksForStrings(test.BlockInfo) {
		observed := time.Now().UTC().UnixNano()
		block := analysis.NewBlock(ethBlock, analysis.OriginSubscription, *ethUriPtr, observed)
		testutils.Monitor.AddBlock(block)
	}

	analysis, err := testutils.Monitor.AnalyzeTree(100, 0)
	if err != nil {
		fmt.Println(err)
		return
	}

	// analysis.Tree.Print()
	// fmt.Println("")
	analysis.Print()
	fmt.Println("")

	for _, reorg := range analysis.Reorgs {
		fmt.Println(reorg)

		entry := database.NewReorgEntry(reorg)
		db.AddReorgEntry(entry)

		// blocks
		for _, block := range reorg.BlocksInvolved {
			blockEntry := database.NewBlockEntry(block, reorg)
			db.AddBlockEntry(blockEntry)
		}
	}
}
