// Package query 提供查询服务
// 封装区块和交易的查询逻辑，支持缓存
package query

import (
	"context"
	"fmt"
	"time"

	"blockexplore/internal/model"
	"blockexplore/internal/repository"
	"blockexplore/pkg/cache"
	"blockexplore/pkg/logger"

	"go.uber.org/zap"
)

// QueryService 查询服务
// 提供区块和交易的查询功能，优先读取 Redis 缓存
type QueryService struct {
	blockRepo *repository.BlockRepo
	txRepo    *repository.TxRepo
	cache     *cache.RedisClient
}

// NewQueryService 创建查询服务实例
func NewQueryService(blockRepo *repository.BlockRepo, txRepo *repository.TxRepo, redisClient *cache.RedisClient) *QueryService {
	return &QueryService{
		blockRepo: blockRepo,
		txRepo:    txRepo,
		cache:     redisClient,
	}
}

// BlockListResponse 区块列表响应
type BlockListResponse struct {
	Chain      string        `json:"chain"`      // 链标识
	Blocks     []model.Block `json:"blocks"`      // 区块列表
	Pagination Pagination    `json:"pagination"`  // 分页信息
}

// Pagination 分页信息
type Pagination struct {
	Page     int   `json:"page"`      // 当前页码
	PageSize int   `json:"page_size"` // 每页数量
	Total    int64 `json:"total"`     // 总记录数
}

// TxListResponse 交易列表响应
type TxListResponse struct {
	Chain        string              `json:"chain"`        // 链标识
	Transactions []model.Transaction `json:"transactions"` // 交易列表
	Pagination   Pagination          `json:"pagination"`   // 分页信息
}

// GetBlockList 获取区块列表（分页）
func (s *QueryService) GetBlockList(chain string, page, pageSize int) (*BlockListResponse, error) {
	// 构建缓存键
	cacheKey := fmt.Sprintf("blocks:%s:%d:%d", chain, page, pageSize)

	// 尝试从缓存读取
	var result BlockListResponse
	if s.cache != nil {
		if err := s.cache.Get(context.Background(), cacheKey, &result); err == nil {
			logger.Debug("命中区块列表缓存", zap.String("key", cacheKey))
			return &result, nil
		}
	}

	// 缓存未命中，查询数据库
	blocks, total, err := s.blockRepo.GetList(chain, page, pageSize)
	if err != nil {
		return nil, err
	}

	result = BlockListResponse{
		Chain:  chain,
		Blocks: blocks,
		Pagination: Pagination{
			Page:     page,
			PageSize: pageSize,
			Total:    total,
		},
	}

	// 写入缓存（过期时间 30 秒）
	if s.cache != nil {
		if err := s.cache.Set(context.Background(), cacheKey, &result, 30*time.Second); err != nil {
			logger.Warn("写入区块列表缓存失败", zap.Error(err))
		}
	}

	return &result, nil
}

// GetBlockDetail 获取区块详情
func (s *QueryService) GetBlockDetail(chain string, blockNumber int64) (*model.Block, error) {
	// 尝试从缓存读取
	cacheKey := fmt.Sprintf("block:%s:%d", chain, blockNumber)
	var block model.Block
	if s.cache != nil {
		if err := s.cache.Get(context.Background(), cacheKey, &block); err == nil {
			return &block, nil
		}
	}

	// 查询数据库
	blockPtr, err := s.blockRepo.GetByChainAndNumber(chain, blockNumber)
	if err != nil {
		return nil, err
	}

	// 写入缓存
	if s.cache != nil {
		s.cache.Set(context.Background(), cacheKey, blockPtr, 60*time.Second)
	}

	return blockPtr, nil
}

// GetBlockTransactions 获取区块内的交易列表
func (s *QueryService) GetBlockTransactions(chain string, blockNumber int64, page, pageSize int) (*TxListResponse, error) {
	txs, total, err := s.txRepo.GetByBlockNumber(chain, blockNumber, page, pageSize)
	if err != nil {
		return nil, err
	}

	return &TxListResponse{
		Chain:        chain,
		Transactions: txs,
		Pagination: Pagination{
			Page:     page,
			PageSize: pageSize,
			Total:    total,
		},
	}, nil
}

// GetTransactionDetail 获取交易详情
func (s *QueryService) GetTransactionDetail(chain, txHash string) (*model.Transaction, error) {
	// 尝试从缓存读取
	cacheKey := fmt.Sprintf("tx:%s:%s", chain, txHash)
	var tx model.Transaction
	if s.cache != nil {
		if err := s.cache.Get(context.Background(), cacheKey, &tx); err == nil {
			return &tx, nil
		}
	}

	// 查询数据库
	txPtr, err := s.txRepo.GetByHash(chain, txHash)
	if err != nil {
		return nil, err
	}

	// 写入缓存
	if s.cache != nil {
		s.cache.Set(context.Background(), cacheKey, txPtr, 60*time.Second)
	}

	return txPtr, nil
}

// GetAddressTransactions 获取地址的交易历史
func (s *QueryService) GetAddressTransactions(chain, address string, page, pageSize int) (*TxListResponse, error) {
	txs, total, err := s.txRepo.GetByAddress(chain, address, page, pageSize)
	if err != nil {
		return nil, err
	}

	return &TxListResponse{
		Chain:        chain,
		Transactions: txs,
		Pagination: Pagination{
			Page:     page,
			PageSize: pageSize,
			Total:    total,
		},
	}, nil
}
