package query

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"blockexplore/internal/model"

	"github.com/stretchr/testify/assert"
)

// ============================================================
// 手写 Mock 实现（满足 query 包定义的接口）
// ============================================================

type mockBlockRepo struct {
	blocks       []model.Block
	total        int64
	err          error
	getByIDErr   error
	calledList   bool
	calledSingle bool
}

func (m *mockBlockRepo) GetList(chain string, page, pageSize int) ([]model.Block, int64, error) {
	m.calledList = true
	return m.blocks, m.total, m.err
}

func (m *mockBlockRepo) GetByChainAndNumber(chain string, blockNumber int64) (*model.Block, error) {
	if m.getByIDErr != nil {
		return nil, m.getByIDErr
	}
	if len(m.blocks) > 0 {
		return &m.blocks[0], nil
	}
	return nil, errors.New("not found")
}

type mockTxRepo struct {
	txs   []model.Transaction
	total int64
	err   error
}

func (m *mockTxRepo) GetByBlockNumber(chain string, blockNumber int64, page, pageSize int) ([]model.Transaction, int64, error) {
	return m.txs, m.total, m.err
}
func (m *mockTxRepo) GetByHash(chain, txHash string) (*model.Transaction, error) {
	if len(m.txs) > 0 {
		return &m.txs[0], nil
	}
	return nil, errors.New("not found")
}
func (m *mockTxRepo) GetByAddress(chain, address string, page, pageSize int) ([]model.Transaction, int64, error) {
	return m.txs, m.total, m.err
}

// mockCache 可控制命中/未命中
type mockCache struct {
	store      map[string][]byte
	getErr     error
	setErr     error
	setCalled  bool
	lastKey    string
	lastTTL    time.Duration
}

func newMockCache() *mockCache {
	return &mockCache{store: make(map[string][]byte)}
}
func (m *mockCache) Get(ctx context.Context, key string, dest interface{}) error {
	if m.getErr != nil {
		return m.getErr
	}
	data, ok := m.store[key]
	if !ok {
		return errors.New("redis: nil")
	}
	// 简单处理：只存原始字节，Get 时直接写回（测试用）
	return json.Unmarshal(data, dest)
}
func (m *mockCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	m.setCalled = true
	m.lastKey = key
	m.lastTTL = expiration
	if m.setErr != nil {
		return m.setErr
	}
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	m.store[key] = data
	return nil
}

// ============================================================
// 测试用例
// ============================================================

func TestGetBlockList_CacheMiss_HitsDB_WritesCache(t *testing.T) {
	repo := &mockBlockRepo{
		blocks: []model.Block{{ID: 1, Chain: "eth", BlockNumber: 100}},
		total:  1,
	}
	cache := newMockCache()
	svc := NewQueryService(repo, &mockTxRepo{}, cache)

	result, err := svc.GetBlockList("eth", 1, 20)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "eth", result.Chain)
	assert.Len(t, result.Blocks, 1)
	assert.Equal(t, int64(1), result.Pagination.Total)
	assert.True(t, repo.calledList)
	// 应该写入缓存
	assert.True(t, cache.setCalled)
	assert.Equal(t, "blocks:eth:1:20", cache.lastKey)
	assert.Equal(t, 30*time.Second, cache.lastTTL)
}

func TestGetBlockList_CacheHit_SkipsDB(t *testing.T) {
	// repo 故意带错误：如果缓存没命中去查 DB，就会触发这个错误
	repo := &mockBlockRepo{err: errors.New("db should not be called")}
	cache := newMockCache()

	// 直接预填缓存（不经过 GetBlockList，避免触发 DB）
	preloaded := &BlockListResponse{
		Chain:  "eth",
		Blocks: []model.Block{{ID: 1, Chain: "eth", BlockNumber: 100}},
		Pagination: Pagination{Page: 1, PageSize: 20, Total: 1},
	}
	_ = cache.Set(context.Background(), "blocks:eth:1:20", preloaded, 30*time.Second)

	svc := NewQueryService(repo, &mockTxRepo{}, cache)

	// 应命中缓存，不查 DB（repo.err 不会被触发）
	result, err := svc.GetBlockList("eth", 1, 20)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, repo.calledList, "缓存命中时不应调用 DB")
	assert.Equal(t, "eth", result.Chain)
	assert.Len(t, result.Blocks, 1)
}

func TestGetBlockList_DBError(t *testing.T) {
	repo := &mockBlockRepo{err: errors.New("connection refused")}
	svc := NewQueryService(repo, &mockTxRepo{}, newMockCache())

	result, err := svc.GetBlockList("eth", 1, 20)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "connection refused")
}

func TestGetBlockList_NilCache_NoPanic(t *testing.T) {
	repo := &mockBlockRepo{
		blocks: []model.Block{{ID: 1}},
		total:  1,
	}
	// cache 传 nil，不应 panic，应直接查 DB
	svc := NewQueryService(repo, &mockTxRepo{}, nil)
	result, err := svc.GetBlockList("eth", 1, 20)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Blocks, 1)
}

func TestGetBlockTransactions(t *testing.T) {
	txRepo := &mockTxRepo{
		txs:   []model.Transaction{{ID: 1, TxHash: "0xtx"}},
		total: 1,
	}
	svc := NewQueryService(&mockBlockRepo{}, txRepo, newMockCache())

	result, err := svc.GetBlockTransactions("eth", 100, 1, 20)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "eth", result.Chain)
	assert.Len(t, result.Transactions, 1)
	assert.Equal(t, int64(1), result.Pagination.Total)
}

func TestGetTransactionDetail(t *testing.T) {
	txRepo := &mockTxRepo{txs: []model.Transaction{{ID: 1, TxHash: "0xabc"}}}
	svc := NewQueryService(&mockBlockRepo{}, txRepo, newMockCache())

	tx, err := svc.GetTransactionDetail("eth", "0xabc")
	assert.NoError(t, err)
	assert.NotNil(t, tx)
	assert.Equal(t, "0xabc", tx.TxHash)
}

func TestGetAddressTransactions(t *testing.T) {
	txRepo := &mockTxRepo{
		txs:   []model.Transaction{{ID: 1, FromAddr: "0xaddr"}},
		total: 1,
	}
	svc := NewQueryService(&mockBlockRepo{}, txRepo, newMockCache())

	result, err := svc.GetAddressTransactions("eth", "0xaddr", 1, 20)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Transactions, 1)
}
