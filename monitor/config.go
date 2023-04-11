package monitor

// Config defines attributes used by reorg monitor server and used for reading in
// environment variables and CLI flags.
type Config struct { // NOTE: ensure struct tags match flag names for each attribute
	EthereumJsonRpcURIs []string `mapstructure:"ethereum-jsonrpc-uris"`
	PostgresDSN         string   `mapstructure:"postgres-dsn"`
	ListenAddress       string   `mapstructure:"listen-address"`
	SimulateBlocks      bool     `mapstructure:"simulate-blocks"`
	MevGethURI          string   `mapstructure:"mev-geth-uri"`
	MaxBlocks           int      `mapstructure:"max-blocks"`
	EnableDebug         bool     `mapstructure:"debug"`
}
