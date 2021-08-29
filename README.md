# Ethereum Reorg Monitor

Detect Ethereum reorgs with the depth, and print replaced and new blocks.

* Can save reorg summaries and block info in a Postgres database
* Can query a mev-geth instance for block value
* Can monitor multiple geth nodes at once (the more the better)

It's work in progress is not yet bug-free.

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
        Geth node URI
  -mevgeth string
        mev-geth node URI
  -sim
        simulate blocks in mev-geth
```

The monitor needs a subscription to one or multiple geth/mev-geth nodes, either a local IPC connection or a `ws://` URI.
You can set the geth node with `-eth <geth_node_url>` or use an `ETH_NODES` environment variable.

Notes: 

* You can find more infos about the children of uncles via AlchemyApi: https://composer.alchemyapi.io/

---

## TODO

* Reorg detection: currently counts 2x depth-1 reorg as one depth-2 reorg
* GethConnection: reconnect, retry with backoff

Less important:

* add webserver / API
  * get status of nodes
  * add or remove nodes
* cmd to simulate old blocks in the database (eg. which had an error before)
* pool of mev-geth instances for simulating blocks

Errors, trying to reproduce:

* `err in Finalize` (can't find parent in reorg.InvolvedBlocks)

---

## Helpers

```bash
# Show AddBlock from logs
grep -v "AddBlock" output/monitor-new/run13.txt 

# Get depth: 2 and higher reorgs
grep "Reorg " output/monitor-new/run13.txt | grep -v "depth: 1"
```

Mermaid:

* https://mermaid-js.github.io/mermaid/#/stateDiagram
* https://mermaid-js.github.io/mermaid-live-editor


See also:

* https://etherscan.io/blocks_forked
