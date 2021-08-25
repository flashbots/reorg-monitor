// Wraps a geth connection and tries to reconnect on error
package monitor

import (
	"context"
	"fmt"
	"log"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

type GethConnection struct {
	NodeUri string
	Client  *ethclient.Client
}

func NewGethConnection(nodeUri string) (*GethConnection, error) {
	conn := GethConnection{
		NodeUri: nodeUri,
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
		return fmt.Errorf("err at syncProgress: %v", err)
	}

	if syncProgress != nil {
		return fmt.Errorf("err: sync in progress")
	}

	fmt.Printf("ok\n")
	return nil
}

func (conn *GethConnection) Subscribe(newBlockChan chan<- *Block) {
	headers := make(chan *types.Header)
	sub, err := conn.Client.SubscribeNewHead(context.Background(), headers)
	if err != nil {
		fmt.Printf("error: SubscribeNewHead failed on node %s: %v\n", conn.NodeUri, err)
	}

	for {
		select {
		case err := <-sub.Err():
			fmt.Printf("error: Subscription failed on node %s: %v\n", conn.NodeUri, err)
			continue
			// // try to reconnect
			// sub, err = _client.SubscribeNewHead(context.Background(), headers)
			// if err != nil {
			// 	fmt.Printf("- Resubscription failed: %v. Trying to reconnect...\n", err)
			// 	_client, err = ethclient.Dial(nodeUri)
			// 	if err != nil {
			// 		fmt.Printf("- Reconnect failed: %v\n", err)
			// 		return // die
			// 	} else {
			// 		sub, err = _client.SubscribeNewHead(context.Background(), headers)
			// 		if err != nil {
			// 			fmt.Printf("- Resubscribe after reconnect failed: %v\n", err)
			// 			return // die
			// 		}
			// 	}
			// }
		case header := <-headers:
			// Fetch full block information from same client
			ethBlock, err := conn.Client.BlockByHash(context.Background(), header.Hash())
			if err != nil {
				log.Println("newHeader -> getBlockByHash error", err)
				continue
			}

			// Add the block
			newBlock := NewBlock(ethBlock, OriginSubscription, conn.NodeUri)
			newBlockChan <- newBlock
		}
	}
}
