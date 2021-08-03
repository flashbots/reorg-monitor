# Ethereum Reorg Monitor

Detect Ethereum reorgs with the depth, and print replaced and new blocks.

The code is working, but it's still an early prototype / work in progress.

---

## Installation

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

## Usage

The code needs a subscription to a geth node, either a local IPC connection or a `ws://` URI.
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

Example output:

```log
2021/08/02 18:05:24 Reorg with depth=2 in block 12946890 0xe20d8759446ef48f9a3f0bccbd85f61899c10a33561bccce4a406af3e9957d57
Last common block:
- 12946887 0xa882637384fb3817a909f7b31a2cde61ce31bbae909272a10e0745d517f42fec
Old chain (replaced blocks):
- 12946888 0x3824bd5b26ce01b21649abde7a4917c623a685c479050bebd4253ea6f595d2a6 (now uncle)
- 12946889 0xf172992557e00bdb32baaae1962e34267f003a2d4183604c24c538bbb8a42699
New chain after reorg:
- 12946888 0xa3363dedf7cc491fd2c81215d29e64491b5747ce4bfc6659b357761b38c88c60
- 12946889 0xcf7412e3503c8d02a5436dd16c9bf437da7538c6343a5462fb7b1050d68be67f
- 12946890 0xe20d8759446ef48f9a3f0bccbd85f61899c10a33561bccce4a406af3e9957d57
```

---

## TODO

* Limit memory growth by pruning old blocks.
* For each new header, get the full block (with tx receipts?) to inspect tx in case of reorg
