package database

import (
	"net/url"

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

// /*
//  * READ OPERATIONS
//  */
// func (s *DatabaseService) Address(address string) (addr addressdetail.AddressDetail, found bool) {
// 	err := s.DB.Get(&addr, "SELECT * FROM address WHERE address=$1", strings.ToLower(address))
// 	if err != nil {
// 		// fmt.Println("err:", err, addr)
// 		return addr, false
// 	}
// 	return addr, true
// }

// func (s *DatabaseService) Analysis(id int) (entry AnalysisEntry, err error) {
// 	err = s.DB.Get(&entry, "SELECT * FROM analysis WHERE id=$1", id)
// 	return entry, err
// }

// func (s *DatabaseService) AddressStatsForAnalysis(analysisId int) (entries []AnalysisAddressStatsEntryWithAddress, err error) {
// 	rows, err := s.DB.Queryx("SELECT * FROM analysis_address_stat INNER JOIN address ON (address.address = analysis_address_stat.address) WHERE Analysis_id=$1", analysisId)
// 	if err != nil {
// 		fmt.Println(err)
// 		return entries, err
// 	}

// 	entries = make([]AnalysisAddressStatsEntryWithAddress, 0)
// 	entryAddressMap := make(map[string]bool) // helper to find repeated entries (bug: concurrent DB connections save two stats concurrently)

// 	for rows.Next() {
// 		var row AnalysisAddressStatsEntryWithAddress
// 		err = rows.StructScan(&row)
// 		if err != nil {
// 			return entries, err
// 		}
// 		if entryAddressMap[row.Address] {
// 			fmt.Println("Getting stats: got repeated addressstat for", row.Address)
// 			continue
// 		}
// 		entries = append(entries, row)
// 		entryAddressMap[row.Address] = true
// 	}

// 	return entries, nil
// }

// /*
//  * WRITE OPERATIONS
//  */
// func (s *DatabaseService) AddBlock(block *types.Block) {
// 	// Check count
// 	var count int
// 	err := s.DB.QueryRow("SELECT COUNT(*) FROM block WHERE number = $1", block.Header().Number.Int64()).Scan(&count)
// 	if err != nil {
// 		panic(err)
// 	}
// 	if count > 0 { // already exists
// 		return
// 	}

// 	// Insert
// 	s.DB.MustExec("INSERT INTO block (Number, Time, NumTx, GasUsed, GasLimit) VALUES ($1, $2, $3, $4, $5)", block.Number().String(), block.Header().Time, len(block.Transactions()), block.GasUsed(), block.GasLimit())
// }

// func (s *DatabaseService) AddAddressStats(block *types.Block, analysisId int, addr core.AddressStats) {
// }

// func (s *DatabaseService) AddAnalysisResultToDatabase(analysis *core.Analysis) {
// 	// todo
// }
