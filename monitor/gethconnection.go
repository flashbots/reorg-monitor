// Wraps a geth connection and tries to reconnect on error
package monitor

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/metachris/eth-reorg-monitor/analysis"
)

type GethConnection struct {
	NodeUri      string
	Client       *ethclient.Client
	NewBlockChan chan<- *analysis.Block

	IsConnected         bool
	IsSubscribed        bool
	LastRetryTimeoutSec int64 // Wait time before retry. Starts at 5 seconds and doubles after each unsuccessful retry (max: 3 min).

	NumResubscribes int64
	NumReconnects   int64
	NumBlocks       uint64
}

func NewGethConnection(nodeUri string, newBlockChan chan<- *analysis.Block) (*GethConnection, error) {
	conn := GethConnection{
		NodeUri:             nodeUri,
		NewBlockChan:        newBlockChan,
		LastRetryTimeoutSec: 5,
	}

	err := conn.Connect()
	return &conn, err
}

func (conn *GethConnection) Connect() (err error) {
	fmt.Printf("[%25s] Connecting to geth node... ", conn.NodeUri)
	conn.Client, err = ethclient.Dial(conn.NodeUri)
	if err != nil {
		return err
	}

	syncProgress, err := conn.Client.SyncProgress(context.Background())
	if err != nil {
		return fmt.Errorf("error at SyncProgress: %v", err)
	}

	if syncProgress != nil {
		return fmt.Errorf("error: sync in progress")
	}

	fmt.Printf("ok\n")
	conn.IsConnected = true
	return nil
}

func (conn *GethConnection) Subscribe() error {
	headers := make(chan *types.Header)
	sub, err := conn.Client.SubscribeNewHead(context.Background(), headers)
	if err != nil {
		fmt.Printf("[conn %s] SubscribeNewHead error: %v\n", conn.NodeUri, err)
		conn.IsSubscribed = false
		return err
	}

	conn.IsSubscribed = true
	conn.LastRetryTimeoutSec = 5

	for {
		select {
		case err := <-sub.Err():
			fmt.Printf("[conn %s] Subscription error: %v\n", conn.NodeUri, err)
			conn.IsSubscribed = false
			conn.ResubscribeAfterTimeout()
			return err
		case header := <-headers:
			conn.NumBlocks += 1

			// Fetch full block information from same client
			ethBlock, err := conn.Client.BlockByHash(context.Background(), header.Hash())
			if err != nil {
				log.Printf("[conn %s] BlockByHash error: %v\n", conn.NodeUri, err)
				continue
			}

			// Add the block
			newBlock := analysis.NewBlock(ethBlock, analysis.OriginSubscription, conn.NodeUri)
			conn.NewBlockChan <- newBlock
		}
	}
}

func (conn *GethConnection) ResubscribeAfterTimeout() {
	conn.NumResubscribes += 1
	log.Printf("[conn %s] resubscribing in %d seconds...\n", conn.NodeUri, conn.LastRetryTimeoutSec)
	time.Sleep(time.Duration(conn.LastRetryTimeoutSec) * time.Second)
	log.Printf("[conn %s] resubscribing...\n", conn.NodeUri)

	// Now double time until next retry (max 3 min)
	conn.LastRetryTimeoutSec *= 2
	if conn.LastRetryTimeoutSec > 60*3 {
		conn.LastRetryTimeoutSec = 60 * 3
	}

	// step 1: get sync status
	syncProgress, err := conn.Client.SyncProgress(context.Background())
	if err != nil {
		log.Printf("[conn %s] err at ResubscribeAfterTimeout syncProgressCheck: %v\n", conn.NodeUri, err)

		// Reconnect
		conn.NumReconnects += 1
		conn.Client, err = ethclient.Dial(conn.NodeUri)
		if err != nil {
			log.Printf("[conn %s] err at ResubscribeAfterTimeout syncProgressCheck->reconnect: %v\n", conn.NodeUri, err)
			conn.IsConnected = false
			conn.ResubscribeAfterTimeout()
			return
		}

		conn.IsConnected = true
		syncProgress, err = conn.Client.SyncProgress(context.Background())
		if err != nil {
			log.Printf("[conn %s] err at ResubscribeAfterTimeout syncProgressCheck: %v\n", conn.NodeUri, err)
			conn.ResubscribeAfterTimeout()
			return
		}
	}

	if syncProgress != nil {
		log.Printf("[conn %s] err at ResubscribeAfterTimeout syncProgressCheck: sync in progress\n", conn.NodeUri)
		conn.ResubscribeAfterTimeout()
		return
	}

	// step 2: subscribe
	err = conn.Subscribe()
	if err != nil {
		log.Printf("[conn %s] err at ResubscribeAfterTimeout Subscribe: %v\n", conn.NodeUri, err)
		conn.ResubscribeAfterTimeout()
		return
	}
}
