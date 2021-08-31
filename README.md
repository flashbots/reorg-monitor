# Ethereum Reorg Monitor

Detect Ethereum reorgs:

* Monitor multiple geth nodes
* Simulate blocks at a mev-geth instance for to get miner value
* Save reorg summaries and block info in a Postgres database

This project is currently work in progress and there may be bugs. Please open issues if you find any, or want to contribute :)

---

## Getting started

* Clone this repository
* See `.env.example` for example environment variables
* Start the monitor:


```bash
$ go run cmd/monitor/main.go -h
  -db
        save reorgs to database
  -debug
        print debug information
  -eth string
        Geth node URIs (comma separated)
  -sim
        simulate blocks in mev-geth
  -mevgeth string
        mev-geth node URI for use with -sim
  -webserver int
        port for the webserver (0 to disable) (default 8094)
```

The monitor needs a subscription to one or multiple geth/mev-geth nodes, either a local IPC connection or a `ws://` URI.
You can set the geth node with `-eth <geth_node_urls>` or use an `ETH_NODES` environment variable (comma separated).

Notes: 

* You can find more infos about the children of uncles via AlchemyApi: https://composer.alchemyapi.io/

---

## TODO

Less important:

* cmd to simulate old blocks in the database (eg. which had an error before)
* pool of mev-geth instances for simulating blocks
* move simulation into monitor

---

## Helpers

```bash
# Show AddBlock from logs
grep -v "AddBlock" output.txt 

# Get reorgs with depth >1
grep "Reorg 1" output.txt | grep -v "depth=1"

# Get reorgs with >1 block replaced
grep "Reorg 1" output.txt | grep -v "replaced=1"

# Get reorgs with more than 2 chains
grep "Reorg 1" output.txt | grep -v "chains=2"
```

Mermaid:

* https://mermaid-js.github.io/mermaid/#/stateDiagram
* https://mermaid-js.github.io/mermaid-live-editor


See also:

* https://etherscan.io/blocks_forked
