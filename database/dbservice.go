package database

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/flashbots/reorg-monitor/analysis"
	"github.com/jmoiron/sqlx"
)

const driverName = "postgres"

type DatabaseService struct {
	DB *sqlx.DB
}

func NewDatabaseService(dsn string) (*DatabaseService, error) {
	db, err := sqlx.Connect(driverName, dsn)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(Schema)
	if err != nil {
		return nil, err
	}

	return &DatabaseService{
		DB: db,
	}, nil
}

func (s *DatabaseService) Reset() {
	s.DB.MustExec(`DROP TABLE "reorg_summary";`)
	s.DB.MustExec(`DROP TABLE "reorg_block";`)
	s.DB.MustExec(Schema)
}

func (s *DatabaseService) Close() {
	s.DB.Close()
}

func (s *DatabaseService) ReorgEntry(key string) (entry ReorgEntry, err error) {
	err = s.DB.Get(&entry, "SELECT * FROM reorg_summary WHERE Key=$1", key)
	return entry, err
}

func (s *DatabaseService) BlockEntry(hash common.Hash) (entry BlockEntry, err error) {
	err = s.DB.Get(&entry, "SELECT * FROM reorg_block WHERE BlockHash=$1", strings.ToLower(hash.Hex()))
	return entry, err
}

func (s *DatabaseService) AddReorgEntry(entry ReorgEntry) error {
	var count int
	err := s.DB.QueryRow("SELECT COUNT(*) FROM reorg_summary WHERE Key = $1", entry.Key).Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 { // already exists
		return fmt.Errorf("a reorg with key %s already exists", entry.Key)
	}

	// Insert
	_, err = s.DB.Exec("INSERT INTO reorg_summary (Key, SeenLive, StartBlockNumber, EndBlockNumber, Depth, NumChains, NumBlocksInvolved, NumBlocksReplaced, MermaidSyntax) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)", entry.Key, entry.SeenLive, entry.StartBlockNumber, entry.EndBlockNumber, entry.Depth, entry.NumChains, entry.NumBlocksInvolved, entry.NumBlocksReplaced, entry.MermaidSyntax)
	return err
}

func (s *DatabaseService) AddBlockEntry(entry BlockEntry) error {
	var count int
	err := s.DB.QueryRow("SELECT COUNT(*) FROM reorg_block WHERE Reorg_Key = $1 AND BlockHash = $2", entry.Reorg_Key, strings.ToLower(entry.BlockHash)).Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 { // already exists
		return fmt.Errorf("a block with hash %s for reorg %s already exists", entry.BlockHash, entry.Reorg_Key)
	}

	// Insert
	_, err = s.DB.Exec("INSERT INTO reorg_block (Reorg_Key, Origin, NodeUri, BlockNumber, BlockHash, ParentHash, BlockTimestamp, CoinbaseAddress, Difficulty, NumUncles, NumTx, IsPartOfReorg, IsMainChain, IsFirst, MevGeth_CoinbaseDiffEth, MevGeth_CoinbaseDiffWei, MevGeth_GasFeesWei, MevGeth_EthSentToCoinbaseWei, MevGeth_EthSentToCoinbase) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)",
		entry.Reorg_Key, entry.Origin, entry.NodeUri, entry.BlockNumber, entry.BlockHash, entry.ParentHash, entry.BlockTimestamp, entry.CoinbaseAddress, entry.Difficulty, entry.NumUncles, entry.NumTx, entry.IsPartOfReorg, entry.IsMainChain, entry.IsFirst, entry.MevGeth_CoinbaseDiffEth, entry.MevGeth_CoinbaseDiffWei, entry.MevGeth_GasFeesWei, entry.MevGeth_EthSentToCoinbaseWei, entry.MevGeth_EthSentToCoinbase)
	return err
}

func (s *DatabaseService) AddReorgWithBlocks(reorg *analysis.Reorg) error {
	// First add the reorg summary
	err := s.AddReorgEntry(NewReorgEntry(reorg))
	if err != nil {
		return err
	}

	// Then add all the blocks
	for _, block := range reorg.BlocksInvolved {
		err := s.AddBlockEntry(NewBlockEntry(block, reorg))
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *DatabaseService) DeleteReorgWithBlocks(entry ReorgEntry) error {
	_, err := s.DB.Exec("DELETE FROM reorg_block WHERE Reorg_Key=$1", entry.Key)
	if err != nil {
		return err
	}
	_, err = s.DB.Exec("DELETE FROM reorg_summary WHERE id=$1", entry.Id)
	return err
}

func (s *DatabaseService) DeleteReorgEntry(entry ReorgEntry) error {
	_, err := s.DB.Exec("DELETE FROM reorg_summary WHERE id=$1", entry.Id)
	return err
}

func (s *DatabaseService) DeleteBlockEntry(entry BlockEntry) error {
	_, err := s.DB.Exec("DELETE FROM reorg_block WHERE id=$1", entry.Id)
	return err
}
