// ============================================================
// Package query 提供查询服务
// ============================================================
// 封装区块和交易的查询逻辑，支持 Redis 缓存。
//
// 缓存策略（Cache-Aside 模式）：
//   1. 查询时先读 Redis 缓存
//   2. 缓存命中则直接返回
//   3. 缓存未命中则查询数据库
//   4. 查询结果写入缓存（设置过期时间）
//   5. 返回结果
//
// 为什么使用缓存？
//   - 减少数据库压力：热点数据从缓存读取，不走数据库
//   - 提高响应速度：Redis 内存读取比数据库快 10-100 倍
//   - 支持高并发：Redis 单机支持 10 万+ QPS
//
// Go 语言基础知识:
//   - struct：结构体，用于定义数据结构
//   - context.Context：上下文，用于超时控制
//   - time.Duration：时间间隔类型
//   - fmt.Sprintf：格式化字符串
//   - error：Go 的错误类型
//   - interface{}：空接口，可以持有任意类型的值
// ============================================================
package query

import (
	"context"    // 上下文
	"fmt"        // 格式化字符串
	"time"       // 时间处理

	"blockexplore/internal/model"       // 数据模型
	"blockexplore/pkg/logger"          // 日志

	"go.uber.org/zap" // 日志库
)

// ============================================================
// 依赖接口定义（用于解耦和单元测试 mock）
// ============================================================
// 通过接口而非具体结构体声明依赖，遵循 Go 的"隐式接口"设计。
// *repository.BlockRepo / *repository.TxRepo / *cache.RedisClient
// 都天然满足这些接口，无需修改它们。

// BlockRepository 区块数据访问接口
type BlockRepository interface {
	GetList(chain string, page, pageSize int) ([]model.Block, int64, error)
	GetByChainAndNumber(chain string, blockNumber int64) (*model.Block, error)
}

// TxRepository 交易数据访问接口
type TxRepository interface {
	GetByBlockNumber(chain string, blockNumber int64, page, pageSize int) ([]model.Transaction, int64, error)
	GetByHash(chain, txHash string) (*model.Transaction, error)
	GetByAddress(chain, address string, page, pageSize int) ([]model.Transaction, int64, error)
}

// Cacher 缓存接口
type Cacher interface {
	Get(ctx context.Context, key string, dest interface{}) error
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
}

// ============================================================
// QueryService 查询服务
// ============================================================
// 提供区块和交易的查询功能，优先读取 Redis 缓存
type QueryService struct {
	blockRepo BlockRepository // 区块数据访问层
	txRepo    TxRepository    // 交易数据访问层
	cache     Cacher          // Redis 缓存客户端
}

// ============================================================
// NewQueryService 创建查询服务实例
// ============================================================
// 参数接受接口类型，可传入真实的 *repository.BlockRepo 等，也可传入测试 mock
func NewQueryService(blockRepo BlockRepository, txRepo TxRepository, redisClient Cacher) *QueryService {
	return &QueryService{
		blockRepo: blockRepo,
		txRepo:    txRepo,
		cache:     redisClient,
	}
}

// ============================================================
// BlockListResponse 区块列表响应
// ============================================================
type BlockListResponse struct {
	Chain      string        `json:"chain"`      // 链标识
	Blocks     []model.Block `json:"blocks"`      // 区块列表
	Pagination Pagination    `json:"pagination"`  // 分页信息
}

// ============================================================
// Pagination 分页信息
// ============================================================
type Pagination struct {
	Page     int   `json:"page"`      // 当前页码
	PageSize int   `json:"page_size"` // 每页数量
	Total    int64 `json:"total"`     // 总记录数
}

// ============================================================
// TxListResponse 交易列表响应
// ============================================================
type TxListResponse struct {
	Chain        string              `json:"chain"`        // 链标识
	Transactions []model.Transaction `json:"transactions"` // 交易列表
	Pagination   Pagination          `json:"pagination"`   // 分页信息
}

// ============================================================
// GetBlockList 方法：获取区块列表（分页）
// ============================================================
// 实现 Cache-Aside 缓存模式
func (s *QueryService) GetBlockList(chain string, page, pageSize int) (*BlockListResponse, error) {
	// 构建缓存键
	// 格式: "blocks:eth:1:20" 表示以太坊第 1 页每页 20 条
	cacheKey := fmt.Sprintf("blocks:%s:%d:%d", chain, page, pageSize)

	// 尝试从缓存读取
	var result BlockListResponse
	if s.cache != nil {
		// s.cache.Get 尝试从 Redis 读取并反序列化
		if err := s.cache.Get(context.Background(), cacheKey, &result); err == nil {
			logger.Debug("命中区块列表缓存", zap.String("key", cacheKey))
			return &result, nil // 缓存命中，直接返回
		}
	}

	// 缓存未命中，查询数据库
	blocks, total, err := s.blockRepo.GetList(chain, page, pageSize)
	if err != nil {
		return nil, err
	}

	// 构建响应
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
	// 30 * time.Second 表示 30 秒后自动删除
	if s.cache != nil {
		if err := s.cache.Set(context.Background(), cacheKey, &result, 30*time.Second); err != nil {
			logger.Warn("写入区块列表缓存失败", zap.Error(err))
			// 缓存写入失败不影响业务，只记录日志
		}
	}

	return &result, nil
}

// ============================================================
// GetBlockDetail 方法：获取区块详情
// ============================================================
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

	// 写入缓存（60 秒过期）
	if s.cache != nil {
		s.cache.Set(context.Background(), cacheKey, blockPtr, 60*time.Second)
	}

	return blockPtr, nil
}

// ============================================================
// GetBlockTransactions 方法：获取区块内的交易列表
// ============================================================
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

// ============================================================
// GetTransactionDetail 方法：获取交易详情
// ============================================================
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

// ============================================================
// GetAddressTransactions 方法：获取地址的交易历史
// ============================================================
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
