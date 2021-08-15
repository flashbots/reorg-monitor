# Ethereum Reorg Monitor

Detect Ethereum reorgs with the depth, and print replaced and new blocks.

---

## Installation

You can either clone this repository or build and install it like this:


```bash
TODO
$ go install github.com/metachris/eth-reorg-monitor/cmd/monitor@latest
$ eth-reorg-monitor -help
Usage of eth-reorg-monitor:
  -eth string
    	Geth node URI
  -mindepth int
    	minimum reorg depth to notify (default 1)
  -silent
    	only print alerts, no info about every block
```

## Usage

The code needs a subscription to a geth node, either a local IPC connection or a `ws://` URI.
You can set the geth node with `-eth <geth_node_url>` or use an `ETH_NODE` environment variable.

Note: You can find more infos about the children of uncles via AlchemyApi: https://composer.alchemyapi.io/

---

## TODO

* Printing replaced blocks: add miner
* Limit memory growth by pruning old blocks.
* For each new header, get the full block (with tx receipts?) to inspect tx in case of reorg

---

## Helpers

```bash
# Show only reorgs with ChainSegments
grep -v "AddBlock" output/monitor-new/run12.txt 

# Get depth: 2 and higher reorgs
grep "Reorg 1" output/monitor-new/run12.txt | grep -v "depth: 1"
```
