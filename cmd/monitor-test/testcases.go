package main

type TestCase struct {
	Name           string
	BlockInfo      []string
	ExpectedResult ReorgTestResult
}

var Test_2Uncles = TestCase{
	Name:           "uncles: 2",
	BlockInfo:      []string{"12996749", "12996751"},
	ExpectedResult: ReorgTestResult{StartBlock: 12996750, EndBlock: 12996750, Depth: 1, NumBlocks: 3, NumChains: 3},
}

var Test_1Uncle = TestCase{
	Name:           "uncles: 1",
	BlockInfo:      []string{"12996760", "12996763"},
	ExpectedResult: ReorgTestResult{StartBlock: 12996760, EndBlock: 12996760, Depth: 1, NumBlocks: 2, NumChains: 2},
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
