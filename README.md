# Ethereum Reorg Monitor

Detect Ethereum reorgs with the depth, and print replaced and new blocks.

* Can save reorg summaries and block info in a Postgres database
* Can query a mev-geth instance for block value

---

## Getting started

* Clone this repository
* Create a `.env` file based on `.env.example`
* Start the monitor


```bash
$ go run cmd/monitor/main.go -h
  -db
        save reorgs to database
  -debug
        print debug information
  -eth string
        Geth node URI
  -mevgeth string
        mev-geth node URI
  -sim
        simulate blocks in mev-geth
```

The code needs a subscription to a geth node, either a local IPC connection or a `ws://` URI.
You can set the geth node with `-eth <geth_node_url>` or use an `ETH_NODE` environment variable.

Notes: 

* You can find more infos about the children of uncles via AlchemyApi: https://composer.alchemyapi.io/

---

## TODO

* Limit memory growth by pruning old blocks.
* Add `seenLive` to block db entries.

---

## Helpers

```bash
# Show only reorgs with ChainSegments
grep -v "AddBlock" output/monitor-new/run13.txt 

# Get depth: 2 and higher reorgs
grep "Reorg 1" output/monitor-new/run13.txt | grep -v "depth: 1"
```

Mermaid:

* https://mermaid-js.github.io/mermaid/#/stateDiagram
* https://mermaid-js.github.io/mermaid-live-editor


See also:

* https://etherscan.io/blocks_forked
