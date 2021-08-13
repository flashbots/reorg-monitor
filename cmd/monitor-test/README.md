### More testcases:

```
2021/08/13 11:14:03 Block 13015940 0x5fab153450d7a2fe62886e0b9a3692903011328a10e7dda4cf4511f5782dbaee    2021-08-13 09:13:47 +0000 UTC   tx: 535, uncles: 0
2021/08/13 11:14:29 Block 13015941 0x45365dd8b21a99256fd7c2249f894665880e6a21fbd2e05a865973ca74a06b46    2021-08-13 09:14:03 +0000 UTC   tx: 207, uncles: 1
- block 13015941 has uncle: 0xed1d32f5ceacda53a128a1acc6aa1c631b78cacd0237955afbbf527a7a90e474
- Block with hash 0xed1d32f5ceacda53a128a1acc6aa1c631b78cacd0237955afbbf527a7a90e474 not found, downloading...
2021/08/13 11:14:29 New completed reorg found: Reorg 13015940_13015940_d1_b2: live: false, blocks 13015940 - 13015940, depth: 1, numBlocks: 2
2021/08/13 11:14:29 ReorgMonitor[eth1]: 13015877 .. 13015941 - 71 blocks
2021/08/13 11:14:46 Block 13015942 0x8c1c8d55b7ada8381b3d351b520f6c54b37bd7e3ad0b4fd05a546d868d2b6561    2021-08-13 09:14:29 +0000 UTC   tx: 131, uncles: 1
- block 13015942 has uncle: 0x020063e320774ebd7402dc5675f137925a013715f1d7ebbd52c9e1a2a77f99a9
- Block with hash 0x020063e320774ebd7402dc5675f137925a013715f1d7ebbd52c9e1a2a77f99a9 not found, downloading...
- block 13015941 has uncle: 0xed1d32f5ceacda53a128a1acc6aa1c631b78cacd0237955afbbf527a7a90e474
2021/08/13 11:14:46 New completed reorg found: Reorg 13015940_13015941_d2_b4: live: false, blocks 13015940 - 13015941, depth: 2, numBlocks: 4
2021/08/13 11:14:46 ReorgMonitor[eth1]: 13015877 .. 13015942 - 73 blocks

main: 0x5fab <- 0x4536 <- 0x8c1c
- 0xed1d: uncle of 0x4536
- 0x0200: uncle of 0x8c1c

Two depth-1 reorgs after one another, but detected as one depth-1 and one depth-2 reorg.
```

+

