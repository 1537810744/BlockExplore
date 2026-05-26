// ============================================================
// Package repository 提供数据访问层，封装数据库操作
// ============================================================
// 该包是数据访问层（DAO），负责与数据库交互。
//
// 分层架构：
//   - Handler（控制器层）：接收 HTTP 请求，调用 Service
//   - Service（业务逻辑层）：处理业务逻辑，调用 Repository
//   - Repository（数据访问层）：执行数据库查询，返回结果
//
// 为什么要分层？
//   - 解耦：业务逻辑和数据访问分离，方便维护
//   - 可测试：可以 Mock Repository 层，方便单元测试
//   - 复用：同一个 Repository 方法可以被多个 Service 调用
//
// Go 语言基础知识:
//   - package：包，Go 的模块化机制
//   - struct：结构体，用于定义数据结构
//   - *gorm.DB：GORM 数据库连接的指针
//   - error：Go 的错误类型，函数通过返回 error 来报告错误
//   - .Error：GORM 的错误属性，操作失败时会设置
//   - append：向切片追加元素
// ============================================================
package repository

import (
	"blockexplore/internal/model" // 数据模型

	"gorm.io/gorm" // GORM ORM 库
)

// ============================================================
// BlockRepo 区块数据访问层
// ============================================================
// 封装了区块相关的数据库操作
type BlockRepo struct {
	db *gorm.DB // 数据库连接实例
}

// ============================================================
// NewBlockRepo 创建区块数据访问层实例
// ============================================================
// 参数 db：GORM 数据库连接
// 返回值：*BlockRepo 指针
func NewBlockRepo(db *gorm.DB) *BlockRepo {
	return &BlockRepo{db: db}
}

// ============================================================
// Create 方法：批量创建区块记录
// ============================================================
// 使用 GORM 的 CreateInBatches 批量插入，提高写入性能
// 参数 blocks：区块切片
// 参数 100：每批插入 100 条记录
func (r *BlockRepo) Create(blocks []model.Block) error {
	if len(blocks) == 0 {
		return nil // 空切片直接返回，不执行数据库操作
	}
	return r.db.CreateInBatches(blocks, 100).Error
}

// ============================================================
// CreateSingle 方法：创建单个区块记录
// ============================================================
// 参数 block：区块指针
// GORM 会自动填充 ID、CreatedAt 等字段
func (r *BlockRepo) CreateSingle(block *model.Block) error {
	return r.db.Create(block).Error
}

// ============================================================
// GetByChainAndNumber 方法：根据链标识和区块高度查询区块
// ============================================================
// 参数 chain：链标识（eth/btc/sol）
// 参数 blockNumber：区块高度
// 返回值：区块指针和错误信息
func (r *BlockRepo) GetByChainAndNumber(chain string, blockNumber int64) (*model.Block, error) {
	var block model.Block
	// Where 添加查询条件，First 查询第一条记录
	// ? 是参数占位符，防止 SQL 注入
	err := r.db.Where("chain = ? AND block_number = ?", chain, blockNumber).First(&block).Error
	if err != nil {
		return nil, err
	}
	return &block, nil
}

// ============================================================
// GetLatest 方法：获取指定链的最新区块
// ============================================================
// 按区块高度降序排列，取第一条
func (r *BlockRepo) GetLatest(chain string) (*model.Block, error) {
	var block model.Block
	err := r.db.Where("chain = ?", chain).
		Order("block_number DESC"). // 降序排列
		First(&block).Error        // 取第一条
	if err != nil {
		return nil, err
	}
	return &block, nil
}

// ============================================================
// GetList 方法：获取区块列表（分页）
// ============================================================
// 参数 chain：链标识
// 参数 page：页码（从 1 开始）
// 参数 pageSize：每页数量
// 返回值：区块切片、总数、错误信息
func (r *BlockRepo) GetList(chain string, page, pageSize int) ([]model.Block, int64, error) {
	var blocks []model.Block
	var total int64

	// 查询总数
	// Model 指定模型，Count 统计总数
	r.db.Model(&model.Block{}).Where("chain = ?", chain).Count(&total)

	// 分页查询
	// offset = (page - 1) * pageSize，跳过前面的记录
	offset := (page - 1) * pageSize
	err := r.db.Where("chain = ?", chain).
		Order("block_number DESC"). // 按区块高度降序，最新的在前
		Offset(offset).             // 跳过 offset 条记录
		Limit(pageSize).            // 最多返回 pageSize 条记录
		Find(&blocks).Error         // 查询结果填充到 blocks

	return blocks, total, err
}

// ============================================================
// GetLatestN 方法：获取指定链的最新 N 个区块
// ============================================================
// 参数 chain：链标识
// 参数 n：数量
func (r *BlockRepo) GetLatestN(chain string, n int) ([]model.Block, error) {
	var blocks []model.Block
	err := r.db.Where("chain = ?", chain).
		Order("block_number DESC").
		Limit(n).
		Find(&blocks).Error
	return blocks, err
}
