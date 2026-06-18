package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"blockexplore/internal/model"
	"blockexplore/internal/service/query"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ---- 在 handler 测试包内实现 query 包的接口 ----

type hMockBlockRepo struct {
	blocks []model.Block
	total  int64
	err    error
}

func (m *hMockBlockRepo) GetList(chain string, page, pageSize int) ([]model.Block, int64, error) {
	return m.blocks, m.total, m.err
}
func (m *hMockBlockRepo) GetByChainAndNumber(chain string, blockNumber int64) (*model.Block, error) {
	if len(m.blocks) > 0 {
		return &m.blocks[0], nil
	}
	return nil, errNotFound
}

type hMockTxRepo struct {
	txs   []model.Transaction
	total int64
	err   error
}

func (m *hMockTxRepo) GetByBlockNumber(chain string, blockNumber int64, page, pageSize int) ([]model.Transaction, int64, error) {
	return m.txs, m.total, m.err
}
func (m *hMockTxRepo) GetByHash(chain, txHash string) (*model.Transaction, error) {
	if len(m.txs) > 0 {
		return &m.txs[0], nil
	}
	return nil, errNotFound
}
func (m *hMockTxRepo) GetByAddress(chain, address string, page, pageSize int) ([]model.Transaction, int64, error) {
	return m.txs, m.total, m.err
}

type hMockCache struct{}

func (h *hMockCache) Get(ctx context.Context, key string, dest interface{}) error {
	return errNotFound // 永远缓存未命中
}
func (h *hMockCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return nil
}

var errNotFound = errNotFoundErr{}

type errNotFoundErr struct{}

func (errNotFoundErr) Error() string { return "not found" }

// ---- 测试用例 ----

func setupRouter() *gin.Engine {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("request_id", "test-rid")
		c.Next()
	})
	return r
}

func TestBlockHandler_GetBlockList_Success(t *testing.T) {
	repo := &hMockBlockRepo{
		blocks: []model.Block{{ID: 1, Chain: "eth", BlockNumber: 100, BlockHash: "0xabc"}},
		total:  1,
	}
	svc := query.NewQueryService(repo, &hMockTxRepo{}, &hMockCache{})
	h := NewBlockHandler(svc)

	r := setupRouter()
	r.GET("/api/v1/blocks", h.GetBlockList)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/blocks?chain=eth&page=1&page_size=20", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(200), resp["code"])
	assert.Equal(t, "test-rid", resp["request_id"])
	data := resp["data"].(map[string]interface{})
	assert.NotNil(t, data["blocks"])
}

func TestBlockHandler_GetBlockList_DefaultsApplied(t *testing.T) {
	repo := &hMockBlockRepo{blocks: []model.Block{}, total: 0}
	svc := query.NewQueryService(repo, &hMockTxRepo{}, &hMockCache{})
	h := NewBlockHandler(svc)

	r := setupRouter()
	r.GET("/api/v1/blocks", h.GetBlockList)

	// 不传 page/page_size，应使用默认值，不报错
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/blocks", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestBlockHandler_GetBlockDetail_Success(t *testing.T) {
	repo := &hMockBlockRepo{
		blocks: []model.Block{{ID: 1, Chain: "eth", BlockNumber: 100, BlockHash: "0xabc"}},
	}
	svc := query.NewQueryService(repo, &hMockTxRepo{}, &hMockCache{})
	h := NewBlockHandler(svc)

	r := setupRouter()
	r.GET("/api/v1/blocks/:block_number", h.GetBlockDetail)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/blocks/100?chain=eth", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestBlockHandler_GetBlockDetail_InvalidNumber(t *testing.T) {
	svc := query.NewQueryService(&hMockBlockRepo{}, &hMockTxRepo{}, &hMockCache{})
	h := NewBlockHandler(svc)

	r := setupRouter()
	r.GET("/api/v1/blocks/:block_number", h.GetBlockDetail)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/blocks/notanumber", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(400), resp["code"])
}

func TestBlockHandler_GetBlockDetail_NotFound(t *testing.T) {
	repo := &hMockBlockRepo{blocks: nil} // GetByChainAndNumber 返回 errNotFound
	svc := query.NewQueryService(repo, &hMockTxRepo{}, &hMockCache{})
	h := NewBlockHandler(svc)

	r := setupRouter()
	r.GET("/api/v1/blocks/:block_number", h.GetBlockDetail)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/blocks/999?chain=eth", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestBlockHandler_GetBlockTransactions_Success(t *testing.T) {
	txRepo := &hMockTxRepo{
		txs:   []model.Transaction{{ID: 1, TxHash: "0xtx"}},
		total: 1,
	}
	svc := query.NewQueryService(&hMockBlockRepo{}, txRepo, &hMockCache{})
	h := NewBlockHandler(svc)

	r := setupRouter()
	r.GET("/api/v1/blocks/:block_number/transactions", h.GetBlockTransactions)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/blocks/100/transactions?chain=eth", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
