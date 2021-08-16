package main

/*
Interesting reorgs for tests:

2021/08/16 03:55:05 Reorg 13033424_13033425_d2_b5 (/server/geth.ipc): live=false  blocks 13033424 - 13033425, depth: 2, numBlocks: 5, numChains: 4
- ChainSegment [orph], blocks: 2 - 0x42087d1b5230fd48172c19d301691aa0866d62739b00804e21ad9668a30fa461, 0x448b1495889c0a1759e83d08b5550b14c7992b6dd84b0eda84414bbd62675337
- ChainSegment [orph], blocks: 1 - 0x5c4aaa79df3c48f53282340c1893a4547d4d42c4ca3af5376dba34646832925d
- ChainSegment [orph], blocks: 1 - 0xd608ccc185b058eae7c12547712244c38ddd98e515e8ba1df7e0af29b468bd3c
- ChainSegment [main], blocks: 1 - 0x5996b0838dbd0d23458664633d1f7beef77be74a69abafdd588825b01ab1f15a
stateDiagram-v2
    0x29af3e566af450ec7443b1857944a7c266aed19707f348e24657bc4abc32ec9f --> 0x42087d1b5230fd48172c19d301691aa0866d62739b00804e21ad9668a30fa461
    0x42087d1b5230fd48172c19d301691aa0866d62739b00804e21ad9668a30fa461 --> 0x448b1495889c0a1759e83d08b5550b14c7992b6dd84b0eda84414bbd62675337
    0x29af3e566af450ec7443b1857944a7c266aed19707f348e24657bc4abc32ec9f --> 0x5c4aaa79df3c48f53282340c1893a4547d4d42c4ca3af5376dba34646832925d
    0x42087d1b5230fd48172c19d301691aa0866d62739b00804e21ad9668a30fa461 --> 0xd608ccc185b058eae7c12547712244c38ddd98e515e8ba1df7e0af29b468bd3c
    0x42087d1b5230fd48172c19d301691aa0866d62739b00804e21ad9668a30fa461 --> 0x5996b0838dbd0d23458664633d1f7beef77be74a69abafdd588825b01ab1f15a
    0x5996b0838dbd0d23458664633d1f7beef77be74a69abafdd588825b01ab1f15a --> 0xb63bb5b0ee3bffb39c6afec9e7569150c9a9b170aba84f5a34c960ffb27046e4
*/

var Test_1Uncle = TestCase{
	Name:           "uncles: 1",
	BlockInfo:      []string{"12996760", "12996763"},
	ExpectedResult: ReorgTestResult{StartBlock: 12996760, EndBlock: 12996760, Depth: 1, NumBlocks: 2, NumChains: 2},
}

var Test_2Uncles = TestCase{
	Name:           "uncles: 2",
	BlockInfo:      []string{"12996749", "12996751"},
	ExpectedResult: ReorgTestResult{StartBlock: 12996750, EndBlock: 12996750, Depth: 1, NumBlocks: 3, NumChains: 3},
}

var Test_ReorgD2 = TestCase{
	Name: "depth: 2",
	BlockInfo: []string{
		"12991730",
		"0xc5d7c2d6da0a4dba574ca6b7697b5850477d646fdb067b20d908060b0d5651c7", // 12991733
		"0x61d0546aba46a166c185c584673e5afe911673e22ca75a754f165454b161e72a", // 12991734
		"12991736",
	},
	ExpectedResult: ReorgTestResult{StartBlock: 12991732, EndBlock: 12991733, Depth: 2, NumBlocks: 4, NumChains: 2},
}

var Test_DoubleReorgD3 = TestCase{
	Name: "double, d3",
	BlockInfo: []string{
		"12969885",
		"0xae396e35c045b8603de015e182ce1349c579c68bb00396bfb8a7b5946a4fa87c", // 12969889
		"0xdca194ddb314c1c4e3de10ccfcb88bf9183a78118a393e1b3860e5eb10dd7c6c", // 12969889
		"12969891",
	},
	ExpectedResult: ReorgTestResult{StartBlock: 12969887, EndBlock: 12969889, Depth: 3, NumBlocks: 6, NumChains: 3},
}

var Test_ReorgD2B5 = TestCase{ // 3 blocks at 13017535, 2 blocks at 13017536
	Name: "reorg b3+2",
	BlockInfo: []string{
		"0xd633f8b768ae1e6975eb0fbd8f5d7ef7b06151a9106a23c17b0ee1b4f74a9bed", // 13017533
		"0xab672fe4e5ca25f44d8cf5c8be556a155d976ddc27a21e069172b3dda7335dad", // 13017534
		"0xd24bb816d9416fe504dea1d2480e560f31d59a50035cd967142cbb118782a015", // 13017535
		"0xfa5314344ed60908988e30524fbcdf4b1fef23a050339368c53c96c5461c956b", // 13017535
		"0x990e488c4eebcb83d17c739311b639c41c200c3906093b6d80ed10d2a75c503b", // 13017536
		"0xdc9a6e449e959ca888da7365d529ac9e05d98d9f7e88adc0e5016da13cef10b7", // 13017535
		"0x3e0e26323edfe6728a4ded45716c138b9e85df50342eea4059f2354ac2937d08", // 13017536
		"0xebf21cef1a406e30bb7b4d482591ca82e444f779efcb76ce67d09c2f548b4c82", // 13017537
		"0xb22ff4c5759adb7e14da8644d2dfdef98bb0e43f3d548cdcdbb0c7fa78675413", // 13017538
	},
	ExpectedResult: ReorgTestResult{StartBlock: 13017535, EndBlock: 13017536, Depth: 2, NumBlocks: 5, NumChains: 4},
}

var Test_ReorgD2B4 = TestCase{
	Name: "n1",
	BlockInfo: []string{
		"0xae416859b2ae32ac70dee15d3b164d81f27c5990312b72419bd0d15c856911bc", // 13018368
		"0x9282169b84cde985685d6157438ef5a4ff7fa83a895ff31a4893c9400e87b0c9", // 13018369
		"0xd9fb42a0296ebb85924366ada87d5acf8eae2069c408111c52c585b48bbb0ec0", // 13018369
		"0xf06bf47c3332361f93cc24c45954949755fa4474631bdfeaa176b48929a56663", // 13018370
		"0x94e290ab3ddaea782b826ea66094c429db1aeb632f80fe7fd003faad9e0a2001", // 13018371
		"0x15d97a6e60b229e143e20bd1a810c3568f13b59a7c9a1f1098928e17389c355d", // 13018370
		"0x6a5706073f58b14949fbb47c9107c480fbb34f9b96c72b34385ae3d06de489b6", // 13018372
		"0x6b501f2591c5f16398497cf71ea7fcc845029847a39f90ebc01f76408a3c665f", // 13018373
		"0xcfba4b54e919631d3b678ab82c9f22bc0cfc4e26341825088a447830d46a7ba1", // 13018374
	},
	ExpectedResult: ReorgTestResult{StartBlock: 13018369, EndBlock: 13018370, Depth: 2, NumBlocks: 4, NumChains: 2},
}

var Test_13033424_13033425_d2_b5 = TestCase{
	Name: "depth 2, blocks 5",
	BlockInfo: []string{
		"0x29af3e566af450ec7443b1857944a7c266aed19707f348e24657bc4abc32ec9f",
		"0x42087d1b5230fd48172c19d301691aa0866d62739b00804e21ad9668a30fa461",
		"0x5996b0838dbd0d23458664633d1f7beef77be74a69abafdd588825b01ab1f15a",
		"0x448b1495889c0a1759e83d08b5550b14c7992b6dd84b0eda84414bbd62675337",
		"0x5c4aaa79df3c48f53282340c1893a4547d4d42c4ca3af5376dba34646832925d",
		"0xd608ccc185b058eae7c12547712244c38ddd98e515e8ba1df7e0af29b468bd3c",
		"0xb63bb5b0ee3bffb39c6afec9e7569150c9a9b170aba84f5a34c960ffb27046e4",
	},
	ExpectedResult: ReorgTestResult{StartBlock: 13033424, EndBlock: 13033425, Depth: 2, NumBlocks: 5, NumChains: 4},
}
