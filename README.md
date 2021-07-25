# Ethereum Reorg Monitor

Detect Ethereum reorgs with the depth.

Work in progress, early prototype and experiments.

---

### Usage

You can set the geth node with `-eth <geth_node_url>` or use an `ETH_NODE` environment variable.

```bash
# Run in verbose mode
go run .

# Run silently (only print reorgs)
go run . -silent

# Set a minimum reorg depth for notifications (default: 1)
go run . -mindepth 2

# You might want to pipe the output into a file like this:
go run . -silent -mindepth 2 | tee log.txt

# Show help
$ go run . -help
Usage of eth-reorg-monitor:
  -eth string
        Geth node URI
  -mindepth int
        minimum reorg depth to notify (default 1)
  -silent
        only print alerts, no info about every block
```
