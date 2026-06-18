//go:build integration

// 集成测试：需要真实的 PostgreSQL 运行在 localhost:5432。
// 运行方式：
//   docker compose -f docker-compose.dev.yaml up -d
//   docker exec blockexplore-dev-postgres psql -U blockexplore -c "CREATE DATABASE blockexplore_test;"
//   go test -v -tags=integration ./internal/repository/

package repository

import (
	"context"
	"fmt"
	"os"
	"testing"

	"blockexplore/internal/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type BlockRepoIntegrationSuite struct {
	suite.Suite
	db   *gorm.DB
	repo *BlockRepo
}

func (s *BlockRepoIntegrationSuite) SetupSuite() {
	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "5432"
	}
	dsn := fmt.Sprintf("host=%s port=%s user=blockexplore password=blockexplore123 dbname=blockexplore_test sslmode=disable", host, port)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		s.T().Fatalf("无法连接测试数据库: %v", err)
	}
	if err := db.AutoMigrate(&model.Block{}, &model.Transaction{}); err != nil {
		s.T().Fatalf("无法建表: %v", err)
	}
	s.db = db
	s.repo = NewBlockRepo(db)
}

func (s *BlockRepoIntegrationSuite) TearDownSuite() {
	s.db.Exec("DROP TABLE IF EXISTS transactions")
	s.db.Exec("DROP TABLE IF EXISTS blocks")
}

func (s *BlockRepoIntegrationSuite) SetupTest() {
	s.db.Exec("DELETE FROM transactions")
	s.db.Exec("DELETE FROM blocks")
}

func (s *BlockRepoIntegrationSuite) TestCreateSingle() {
	block := &model.Block{
		Chain:       "eth",
		BlockNumber: 100,
		BlockHash:   "0xabc",
		Timestamp:   1718000000,
		TxCount:     1,
	}
	err := s.repo.CreateSingle(block)
	assert.NoError(s.T(), err)
	assert.NotZero(s.T(), block.ID)
}

func (s *BlockRepoIntegrationSuite) TestGetByChainAndNumber() {
	s.repo.CreateSingle(&model.Block{Chain: "eth", BlockNumber: 200, BlockHash: "0xdef", Timestamp: 1})

	got, err := s.repo.GetByChainAndNumber("eth", 200)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), int64(200), got.BlockNumber)
	assert.Equal(s.T(), "0xdef", got.BlockHash)
}

func (s *BlockRepoIntegrationSuite) TestGetList_Pagination() {
	for i := int64(1); i <= 25; i++ {
		s.repo.CreateSingle(&model.Block{Chain: "eth", BlockNumber: i, BlockHash: "0x" + fmt.Sprintf("%d", i), Timestamp: i})
	}

	blocks, total, err := s.repo.GetList("eth", 1, 10)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), int64(25), total)
	assert.Len(s.T(), blocks, 10)

	// 第 3 页应只剩 5 条
	blocks, _, err = s.repo.GetList("eth", 3, 10)
	assert.NoError(s.T(), err)
	assert.Len(s.T(), blocks, 5)
}

func (s *BlockRepoIntegrationSuite) TestGetLatest() {
	s.repo.CreateSingle(&model.Block{Chain: "btc", BlockNumber: 10, BlockHash: "0xa", Timestamp: 1})
	s.repo.CreateSingle(&model.Block{Chain: "btc", BlockNumber: 20, BlockHash: "0xb", Timestamp: 2})

	got, err := s.repo.GetLatest("btc")
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), int64(20), got.BlockNumber)
}

func (s *BlockRepoIntegrationSuite) TestTxRepo_GetByAddress() {
	txRepo := NewTxRepo(s.db)
	s.repo.CreateSingle(&model.Block{Chain: "eth", BlockNumber: 1, BlockHash: "0x1", Timestamp: 1})
	txRepo.Create([]model.Transaction{
		{Chain: "eth", TxHash: "0xtx1", BlockNumber: 1, FromAddr: "0xalice", ToAddr: "0xbob", Value: "0", Timestamp: 1},
		{Chain: "eth", TxHash: "0xtx2", BlockNumber: 1, FromAddr: "0xbob", ToAddr: "0xalice", Value: "0", Timestamp: 2},
	})

	txs, total, err := txRepo.GetByAddress("eth", "0xalice", 1, 10)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), int64(2), total)
	assert.Len(s.T(), txs, 2)
	_ = context.Background()
}

func TestBlockRepoIntegrationSuite(t *testing.T) {
	suite.Run(t, new(BlockRepoIntegrationSuite))
}
