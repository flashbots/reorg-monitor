# Ethereum Reorg Monitor

Detect Ethereum reorgs with the depth.

Work in progress, early prototype and experiments.

---

### Usage

You can set the geth node with `-eth <geth_node_url>` or use an `ETH_NODE` environment variable.

```shell
# Run in normal mode
go run .

# Run silently (only print reorgs)
go run . -silent
```

