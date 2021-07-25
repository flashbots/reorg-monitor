# Ethereum Reorg Monitor

Detect Ethereum reorgs with the depth.

Work in progress, early prototype and experiments.

---

### Installation

You can either clone this repository or build and install it like this:

```bash
$ go install github.com/metachris/eth-reorg-monitor@latest
$ eth-reorg-monitor -help
Usage of eth-reorg-monitor:
  -eth string
    	Geth node URI
  -mindepth int
    	minimum reorg depth to notify (default 1)
  -silent
    	only print alerts, no info about every block
```

### Usage

You can set the geth node with `-eth <geth_node_url>` or use an `ETH_NODE` environment variable.

```bash
# Run in verbose mode
eth-reorg-monitor

# Run silently (only print reorgs)
eth-reorg-monitor -silent

# Set a minimum reorg depth for notifications (default: 1)
eth-reorg-monitor -mindepth 2

# You might want to pipe the output into a file like this:
eth-reorg-monitor -silent -mindepth 2 | tee log.txt
```
