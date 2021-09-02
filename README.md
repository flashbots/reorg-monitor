# Ethereum Reorg Monitor

Watch and document Ethereum reorgs, including miner values of blocks.

* Monitor multiple geth nodes (a WebSocket or IPC connection is nevessary for subscribing to new blocks)
* Capture miner value (gas fees and smart contract payments) by simulating blocks with [mev-geth](https://github.com/flashbots/mev-geth/)
* Collect data in a Postgres database (summary and individual block info)
* Webserver that shows status information and recent reorgs

This project is currently work in progress and there may be bugs, although it works pretty stable now. 
Please open issues if you have ideas, questions or want to contribute :)

---

## Getting started

* Clone this repository
* For database testing, you can use `docker-compose up` to start a local Postgres database and adminer
* See [`.env.example`](https://github.com/metachris/eth-reorg-monitor/blob/master/.env.example) for environment variables you can use (eg create `.env.local` and use them with `source .env.local`).
* Start the monitor:


```bash
# Normal run, print only
$ go run cmd/monitor/main.go -eth ws://geth_node:8546

# Simulate blocks in a reorg 
$ go run cmd/monitor/main.go -sim

# Save to database
$ go run cmd/monitor/main.go -sim -db

# Get status from webserver
$ curl localhost:9094
```

You can also install the reorg monitor with `go install`:

```bash
$ go install github.com/metachris/eth-reorg-monitor/cmd/reorg-monitor@latest
$ reorg-monitor -h
```

---

## Codebase Overview & Architecture

See also: [Story of an Ethereum Reorg](https://docs.google.com/presentation/d/1ZHJp2HFOFeZxQAyPETRvcXW0oSOkZHAUhm7G-MoYyoQ/edit?usp=sharing)

Code layout:

* [`cmd/reorg-monitor`](https://github.com/metachris/eth-reorg-monitor/blob/master/cmd/reorg-monitor/main.go) is the main command-line entrypoint
* [`cmd/reorg-monitor-test`](https://github.com/metachris/eth-reorg-monitor/blob/master/cmd/reorg-monitor-test/main.go) is used for local testing and development
* [`monitor` module](https://github.com/metachris/eth-reorg-monitor/tree/master/monitor) - block collection: subscription to geth nodes, building a history of as many blocks as possible
* [`analysis` module](https://github.com/metachris/eth-reorg-monitor/tree/master/analysis) - detect reorgs by building a tree data structure of all known blocks (blocks with >1 child start a reorg)

---

## TODO

Less important:

* pool of mev-geth instances for simulating blocks
* move simulation into monitor

---

## Notes & References

* [etherscan.io/chart/uncles](https://etherscan.io/chart/uncles)
* [etherscan.io/blocks_forked](https://etherscan.io/blocks_forked)
* [Story of an Ethereum Reorg](https://docs.google.com/presentation/d/1ZHJp2HFOFeZxQAyPETRvcXW0oSOkZHAUhm7G-MoYyoQ/edit?usp=sharing)
* [go-ethereum `WriteBlock` function](https://github.com/ethereum/go-ethereum/blob/525116dbff916825463931361f75e75e955c12e2/core/blockchain.go#L860), which calls the `reorg` method if a block is seen whos parent is not the current block
* [Ethereum Whitepaper: Modified GHOST Implementation](https://ethereum.org/en/whitepaper/#modified-ghost-implementation)
* [Ethereum Yellow Paper](https://ethereum.github.io/yellowpaper/paper.pdf)
* [Ghost whitepaper](https://eprint.iacr.org/2013/881.pdf)
* For Ethereum 2.0: [Combining Ghost and Casper](https://arxiv.org/abs/2003.03052)

See also:

* [An Empirical Analysis of Chain Reorganizations and Double-Spend Attacks on Proof-of-Work Cryptocurrencies](https://static1.squarespace.com/static/59aae5e9a803bb10bedeb03e/t/5f08d13a1cd5592cb330a0d0/1594413374526/LovejoyJamesP-meng-eecs-2020.pdf) (pdf)

Tools:

* https://composer.alchemyapi.io - to find out more about non-mainchain blocks ([`eth_getBlockByHash`](https://composer.alchemyapi.io/?composer_state=%7B%22chain%22%3A0%2C%22network%22%3A0%2C%22methodName%22%3A%22eth_getBlockByHash%22%2C%22paramValues%22%3A%5B%22YOUR_BLOCK_HASH_HERE%22%2Ctrue%5D%7D))
* https://mermaid-js.github.io/mermaid-live-editor

