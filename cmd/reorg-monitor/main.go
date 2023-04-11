package main

import (
	"fmt"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/flashbots/reorg-monitor/analysis"
	"github.com/flashbots/reorg-monitor/database"
	"github.com/flashbots/reorg-monitor/monitor"
	flashbotsrpc "github.com/metachris/flashbots-rpc"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"log"
	"strings"
)

const (
	AppName = "reorg-monitor"

	// default values for CLI flags
	defaultMaxBlocks      = 200
	defaultSimulateBlocks = false
	defaultEnableDebug    = false

	// flag related constants
	// NOTE: Be sure to match the flag with its associated struct tag in config for `monitor.Config`
	flagEnableDebug = "debug"
	usageDebug      = "enable debug mode for additional logging"

	flagEthereumJsonRpcURIs  = "ethereum-jsonrpc-uris"
	usageEthereumJsonRpcURIs = "comma separated list of Ethereum JSON-RPC URIs used for monitoring network"

	flagListenAddress  = "listen-address"
	usageListenAddress = "TCP network address used to launch monitoring web server that displays block status information"

	flagMaxBlocks  = "max-blocks"
	usageMaxBlocks = "maximum number of blocks to hold in memory"

	flagMevGethURI  = "mev-geth-uri"
	usageMevGethURI = "URI for mev-geth used for block simulation"

	flagPostgresDSN  = "postgres-dsn"
	usagePostgresDSN = "data source name (DSN) for PostgreSQL database used for storing reorged blocks"

	flagSimulateBlocks  = "simulate-blocks"
	usageSimulateBlocks = "toggles block simulation and updates database with response metadata if enabled"
)

var (
	ColorGreen = "\033[1;32m%s\033[0m"

	version = "dev" // is set during build process

	db                   *database.DatabaseService
	rpc                  *flashbotsrpc.FlashbotsRPC
	callBundlePrivKey, _ = crypto.GenerateKey()
)

func main() {
	if err := MonitorCmd().Execute(); err != nil {
		log.Fatal(err)
	}
}

func handleReorg(simulateBlocks bool, db *database.DatabaseService, reorg *analysis.Reorg) {
	log.Println(reorg.String())
	fmt.Println("- common parent:    ", reorg.CommonParent.Hash)
	fmt.Println("- first block after:", reorg.FirstBlockAfterReorg.Hash)
	for chainKey, chain := range reorg.Chains {
		if chainKey == reorg.MainChainHash {
			fmt.Printf("- mainchain l=%d: ", len(chain))
		} else {
			fmt.Printf("- sidechain l=%d: ", len(chain))
		}

		for _, block := range chain {
			fmt.Printf("%s ", block.Hash)
		}
		fmt.Print("\n")
	}

	if db != nil {
		entry := database.NewReorgEntry(reorg)
		err := db.AddReorgEntry(entry)
		if err != nil {
			log.Printf("error at db.AddReorgEntry: %+v\n", err)
		}

		for _, block := range reorg.BlocksInvolved {
			blockEntry := database.NewBlockEntry(block, reorg)

			// If block has no transactions, then it has 0 miner value (no need to simulate)
			if simulateBlocks && len(block.Block.Transactions()) > 0 {
				res, err := rpc.FlashbotsSimulateBlock(callBundlePrivKey, block.Block, 0)
				if err != nil {
					log.Println("error: sim failed of block", block.Hash, "-", err)
				} else {
					log.Printf("- sim of block %s: CoinbaseDiff=%20s, GasFees=%20s, EthSentToCoinbase=%20s\n", block.Hash, res.CoinbaseDiff, res.GasFees, res.EthSentToCoinbase)
					blockEntry.UpdateWitCallBundleResponse(res)
				}
			}

			err := db.AddBlockEntry(blockEntry)
			if err != nil {
				log.Println("error at db.AddBlockEntry:", err)
			}
		}
	}

	if reorg.NumReplacedBlocks > 1 {
		fmt.Println(reorg.MermaidSyntax())
	}
	fmt.Println("")
}

// MonitorCmd defines top level command to instantiate and run reorg monitoring server based on
// input environment variables and command line flags
func MonitorCmd() *cobra.Command {
	conf := new(monitor.Config)

	cmd := &cobra.Command{
		Use:          AppName,
		SilenceUsage: true,
		Short:        "Monitor for re-orgs on Ethereum network across execution layer (EL) and consensus layer (CL)",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Load environment variables and cli flags into configuration structure using viper
			v := viper.New()
			// Replace dashes with underscores to support reading in environment variables
			v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
			if err := v.BindPFlags(cmd.PersistentFlags()); err != nil {
				return err
			}
			v.AutomaticEnv()

			return v.Unmarshal(conf)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			log.Printf("initializing %s [version: %s]", AppName, version)

			if len(conf.EthereumJsonRpcURIs) == 0 {
				return fmt.Errorf("ethereum json-rpc endpoints not found")
			}

			if conf.SimulateBlocks {
				if conf.MevGethURI == "" {
					return fmt.Errorf("block simulation enabled but mev-geth URI not found")
				}

				rpc = flashbotsrpc.NewFlashbotsRPC(conf.MevGethURI)
				rpc.Debug = conf.EnableDebug
				log.Printf("Using mev-geth node at %s for block simulations", conf.MevGethURI)
			}

			if conf.PostgresDSN != "" {
				var err error
				db, err = database.NewDatabaseService(conf.PostgresDSN)
				endpoint := strings.Split(conf.PostgresDSN, "@")[1]
				if err != nil {
					return fmt.Errorf("error initializing database service with endpoint %s - %v", endpoint, err)
				}
				log.Println("Connected to database at", "uri", endpoint)
			}

			// Channel to receive reorgs from monitor
			reorgChan := make(chan *analysis.Reorg)

			// Setup and start the monitor
			mon := monitor.NewReorgMonitor(conf.EthereumJsonRpcURIs, reorgChan, true, conf.MaxBlocks)
			if mon.ConnectClients() == 0 {
				return fmt.Errorf("%s could not connect to any clients", AppName)
			}

			if conf.ListenAddress != "" {
				log.Printf("Starting webserver on %s\n", conf.ListenAddress)
				ws := monitor.NewMonitorWebserver(mon, conf.ListenAddress)
				go ws.ListenAndServe()
			}

			// In the background, subscribe to new blocks and listen for updates
			go mon.SubscribeAndListen()

			// Wait for reorgs
			for reorg := range reorgChan {
				handleReorg(conf.SimulateBlocks, db, reorg)
			}
			return nil
		},
	}

	cmd.PersistentFlags().StringSliceVar(&conf.EthereumJsonRpcURIs, flagEthereumJsonRpcURIs, nil, usageEthereumJsonRpcURIs)
	cmd.PersistentFlags().StringVar(&conf.PostgresDSN, flagPostgresDSN, "", usagePostgresDSN)
	cmd.PersistentFlags().StringVar(&conf.ListenAddress, flagListenAddress, "", usageListenAddress)
	cmd.PersistentFlags().BoolVar(&conf.SimulateBlocks, flagSimulateBlocks, defaultSimulateBlocks, usageSimulateBlocks)
	cmd.PersistentFlags().StringVar(&conf.MevGethURI, flagMevGethURI, "", usageMevGethURI)
	cmd.PersistentFlags().IntVar(&conf.MaxBlocks, flagMaxBlocks, defaultMaxBlocks, usageMaxBlocks)
	cmd.PersistentFlags().BoolVar(&conf.EnableDebug, flagEnableDebug, defaultEnableDebug, usageDebug)
	return cmd
}
