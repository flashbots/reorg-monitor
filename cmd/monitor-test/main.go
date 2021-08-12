package main

import (
	"flag"
	"log"
	"os"

	"github.com/metachris/eth-reorg-monitor/reorgutils"
)

func main() {
	log.SetOutput(os.Stdout)

	ethUriPtr := flag.String("eth", os.Getenv("ETH_NODE"), "Geth node URI")
	flag.Parse()

	if *ethUriPtr == "" {
		log.Fatal("Missing eth node uri")
	}

	reorgutils.Perror(ConnectClient(*ethUriPtr))

	// TestBlockWith1Uncle()
	// TestBlockWith2Uncles()
	// TestReorg1()
	TestDoubleReorg()
}

func TestDoubleReorg() {
	ResetMon("double reorg")

	// 2021/08/06 08:43:16 Reorg with depth=3 in block 12969889
	// Last common block:
	// - 12969886 0xc8c4ed6507d118168f1901236aa43d073006cd8c612dc78e9c6b333df29e7639
	// C1
	// - 12969887 0x7b5c2c2a5b31d7436b3e541f44a868c2cafa837e629b267dbfab749fd6c9e3d4 (now uncle)
	// - 12969888 0x647411d024c0b007556f80e48cf4ec02c601c37584e00b9e6c8ec08b8f2b0252
	// - 12969889 0xae396e35c045b8603de015e182ce1349c579c68bb00396bfb8a7b5946a4fa87c
	// C2
	// - 12969887 0x7f31fa36b85af1b5ec33a68c4fb82e395eb8123dba66fa43322656f043e80576
	// - 12969888 0x647411d024c0b007556f80e48cf4ec02c601c37584e00b9e6c8ec08b8f2b0252 (now uncle)
	// - 12969888 0x698d17e71f65006661f45af5f5fbea50fd1bcbc7236fe5e37255d727675148b8
	// - 12969889 0xdca194ddb314c1c4e3de10ccfcb88bf9183a78118a393e1b3860e5eb10dd7c6c
	AddBlockAndPrintNewline(
		GetBlockByNumber(12969885),
		GetBlockByNumber(12969886),
		GetBlockByHashStr("0xae396e35c045b8603de015e182ce1349c579c68bb00396bfb8a7b5946a4fa87c"), // 12969889
		GetBlockByHashStr("0xdca194ddb314c1c4e3de10ccfcb88bf9183a78118a393e1b3860e5eb10dd7c6c"), // 12969889
		GetBlockByNumber(12969890),
		GetBlockByNumber(12969891),
	)

	ReorgCheckAndPrint()
}

func TestReorg1() {
	// 2021/08/09 17:26:14 Reorg with depth=2 in block 12991734
	// Last common block:
	// - 12991731 0x6c607c8769eecf20fc6dccdcbcbbb60598623ecf1d1cbe9216d616761bed1656 /  25 tx, miner 0xbCC817f057950b0df41206C5D7125E6225Cae18e, earnings: 2.4328 ETH
	// Old chain (replaced blocks):
	// - 12991732 0xcb895219946bbb37aa08c7ef55779ac5c801bb43abbc025e88539562d7a18e93 / 306 tx, miner 0x5A0b54D5dc17e0AadC383d2db43B0a0D3E029c4c, earnings: 2.2183 ETH (now uncle)
	// - 12991733 0x155a5b3724dc8dc0c0d407fce58d74bce777ac48cb8958b695f335888cfa00e5 / 180 tx, miner 0xe206e3DCa498258f1B7EEc1c640B5AEE7BB88Fd0, earnings: 2.4299 ETH
	// - 12991733 0xc5d7c2d6da0a4dba574ca6b7697b5850477d646fdb067b20d908060b0d5651c7 / 158 tx, miner 0x5A0b54D5dc17e0AadC383d2db43B0a0D3E029c4c, earnings: 2.4184 ETH
	// New chain after reorg:
	// - 12991732 0xcd66a2f7f1f56f4ca0715c2ceed680180c5d49a939c3a3a3663bc9fec7b58e05 / 290 tx, miner 0xEA674fdDe714fd979de3EdF0F56AA9716B898ec8, earnings: 2.2369 ETH
	// - 12991733 0x16094a7feb1d83a8153ffbd68a631dcb914f5fbc08e213db9cd6e6475f94e2fa / 144 tx, miner 0xEA674fdDe714fd979de3EdF0F56AA9716B898ec8, earnings: 2.4809 ETH
	// - 12991734 0x61d0546aba46a166c185c584673e5afe911673e22ca75a754f165454b161e72a / 440 tx, miner 0xEA674fdDe714fd979de3EdF0F56AA9716B898ec8, earnings: 2.4328 ETH

	ResetMon("reorg1")

	AddBlockAndPrintNewline(
		GetBlockByNumber(12991730),
		GetBlockByNumber(12991731),
		GetBlockByHashStr("0xc5d7c2d6da0a4dba574ca6b7697b5850477d646fdb067b20d908060b0d5651c7"), // 12991733
		GetBlockByHashStr("0x61d0546aba46a166c185c584673e5afe911673e22ca75a754f165454b161e72a"), // 12991734
		GetBlockByNumber(12991735),
		GetBlockByNumber(12991736),
	)

	ReorgCheckAndPrint()
}

func TestBlockWith2Uncles() {
	// 12996751 0xe6b8556d6b8be89721c8fee126e7e66bb615057164b20562b69717bec6e24841 uncles: 2
	ResetMon("uncles:2")

	AddBlockAndPrintNewline(
		GetBlockByNumber(12996748),
		GetBlockByNumber(12996749),
		GetBlockByNumber(12996750),
		GetBlockByNumber(12996751), // 2 uncles
		GetBlockByNumber(12996752),
		GetBlockByNumber(12996753),
	)

	reorgs := ReorgCheckAndPrint()
	Pcheck("NumReorgs", len(reorgs), 1)

	reorg1, found := reorgs[12996750]
	Pcheck("FoundReorg", found, true)
	Pcheck("StartBlockHeight", reorg1.StartBlockHeight, uint64(12996750))
	Pcheck("EndBlockHeight", reorg1.EndBlockHeight, uint64(12996750))
	Pcheck("Depth", reorg1.Depth, uint64(1))
	Pcheck("NumChains", reorg1.NumChains, uint64(3)) // 2 uncles -> parent has 2 siblings
}

func TestBlockWith1Uncle() {
	// 12996762 0xa98aa57369c7c1ba75a88b86d5ddd685cf77d5e25f0521e3ee77cff81b67ede3 uncles: 1

	ResetMon("uncles:1")

	AddBlockAndPrintNewline(
		GetBlockByHashStr("0x64f5389fbb9c97ca2d164a034d6168449a27eb7c91d4e51b1a2cfbee3ad810a2"),
		GetBlockByHashStr("0xc7079a2686e92351812928407eca33d1d6f4884bdc37d018df978ec95aedb9ca"),
		GetBlockByHashStr("0xa98aa57369c7c1ba75a88b86d5ddd685cf77d5e25f0521e3ee77cff81b67ede3"), // block with 1 uncle
		GetBlockByHashStr("0xeb818c21ad41d1db9c64ee9dbdecd4ada0a2b86c753fa16c5802a6bf79f8c6ff"),
	)

	ReorgCheckAndPrint()
}
