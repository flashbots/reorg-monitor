package database

import (
	"net/url"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
)

type PostgresConfig struct {
	User       string
	Password   string
	Host       string
	Name       string
	DisableTLS bool
}

func NewDatabaseConnection(cfg PostgresConfig) *sqlx.DB {
	sslMode := "require"
	if cfg.DisableTLS {
		sslMode = "disable"
	}

	q := make(url.Values)
	q.Set("sslmode", sslMode)
	q.Set("timezone", "utc")

	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(cfg.User, cfg.Password),
		Host:     cfg.Host,
		Path:     cfg.Name,
		RawQuery: q.Encode(),
	}

	db := sqlx.MustConnect("postgres", u.String())
	db.MustExec(Schema)
	return db
}

type DatabaseService struct {
	DB *sqlx.DB
}

func NewDatabaseService(cfg PostgresConfig) *DatabaseService {
	db := NewDatabaseConnection(cfg)
	return &DatabaseService{
		DB: db,
	}
}

func (s *DatabaseService) Reset() {
	s.DB.MustExec(`DROP TABLE "blocks_with_earnings";`)
	s.DB.MustExec(Schema)
}

func (s *DatabaseService) Close() {
	s.DB.Close()
}

func (s *DatabaseService) ReorgEntry(key string) (entry ReorgEntry, err error) {
	err = s.DB.Get(&entry, "SELECT * FROM reorgs WHERE Key=$1", key)
	return entry, err
}

func (s *DatabaseService) BlockEntry(hash common.Hash) (entry BlockEntry, err error) {
	err = s.DB.Get(&entry, "SELECT * FROM blocks_with_earnings WHERE BlockHash=$1", strings.ToLower(hash.Hex()))
	return entry, err
}

func (s *DatabaseService) AddReorgEntry(entry ReorgEntry) error {
	var count int
	err := s.DB.QueryRow("SELECT COUNT(*) FROM reorgs WHERE Key = $1", entry.Key).Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 { // already exists
		return nil
	}

	// Insert
	_, err = s.DB.Exec("INSERT INTO reorgs (Key, NodeUri, SeenLive, StartBlockNumber, EndBlockNumber, Depth, NumBlocksInvolved, NumBlocksReplaced, MermaidSyntax) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)", entry.Key, entry.NodeUri, entry.SeenLive, entry.StartBlockNumber, entry.EndBlockNumber, entry.Depth, entry.NumBlocksInvolved, entry.NumBlocksReplaced, entry.MermaidSyntax)
	return err
}

func (s *DatabaseService) AddBlockEntry(entry BlockEntry) error {
	var count int
	err := s.DB.QueryRow("SELECT COUNT(*) FROM blocks_with_earnings WHERE BlockHash = $1", strings.ToLower(entry.BlockHash)).Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 { // already exists
		return nil
	}

	// Insert
	_, err = s.DB.Exec("INSERT INTO blocks_with_earnings (Reorg_Key, BlockNumber, BlockHash, ParentHash, BlockTimestamp, CoinbaseAddress, Difficulty, NumUncles, NumTx, IsPartOfReorg, IsMainChain, IsUncle, IsChild, MevGeth_CoinbaseDiffEth, MevGeth_CoinbaseDiffWei, MevGeth_GasFeesWei, MevGeth_EthSentToCoinbaseWei) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)",
		entry.Reorg_Key, entry.BlockNumber, entry.BlockHash, entry.ParentHash, entry.BlockTimestamp, entry.CoinbaseAddress, entry.Difficulty, entry.NumUncles, entry.NumTx, entry.IsPartOfReorg, entry.IsMainChain, entry.IsUncle, entry.IsChild, entry.MevGeth_CoinbaseDiffEth, entry.MevGeth_CoinbaseDiffWei, entry.MevGeth_GasFeesWei, entry.MevGeth_EthSentToCoinbaseWei)
	return err
}
