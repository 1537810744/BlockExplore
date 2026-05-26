// Package repository 提供数据访问层，封装数据库操作
// 所有数据库查询都在这一层完成，上层服务通过接口调用
package repository

import (
	"blockexplore/internal/model"

	"gorm.io/gorm"
)

// BlockRepo 区块数据访问层
type BlockRepo struct {
	db *gorm.DB // 数据库连接实例
}

// NewBlockRepo 创建区块数据访问层实例
func NewBlockRepo(db *gorm.DB) *BlockRepo {
	return &BlockRepo{db: db}
}

// Create 批量创建区块记录
// 使用 GORM 的 CreateInBatches 批量插入，提高写入性能
func (r *BlockRepo) Create(blocks []model.Block) error {
	if len(blocks) == 0 {
		return nil
	}
	return r.db.CreateInBatches(blocks, 100).Error // 每批 100 条
}

// CreateSingle 创建单个区块记录
func (r *BlockRepo) CreateSingle(block *model.Block) error {
	return r.db.Create(block).Error
}

// GetByChainAndNumber 根据链标识和区块高度查询区块
func (r *BlockRepo) GetByChainAndNumber(chain string, blockNumber int64) (*model.Block, error) {
	var block model.Block
	err := r.db.Where("chain = ? AND block_number = ?", chain, blockNumber).First(&block).Error
	if err != nil {
		return nil, err
	}
	return &block, nil
}

// GetLatest 获取指定链的最新区块
func (r *BlockRepo) GetLatest(chain string) (*model.Block, error) {
	var block model.Block
	err := r.db.Where("chain = ?", chain).
		Order("block_number DESC").
		First(&block).Error
	if err != nil {
		return nil, err
	}
	return &block, nil
}

// GetList 获取区块列表（分页）
// chain: 链标识
// page: 页码（从 1 开始）
// pageSize: 每页数量
func (r *BlockRepo) GetList(chain string, page, pageSize int) ([]model.Block, int64, error) {
	var blocks []model.Block
	var total int64

	// 查询总数
	r.db.Model(&model.Block{}).Where("chain = ?", chain).Count(&total)

	// 分页查询（按区块高度降序，最新的在前）
	offset := (page - 1) * pageSize
	err := r.db.Where("chain = ?", chain).
		Order("block_number DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&blocks).Error

	return blocks, total, err
}

// GetLatestN 获取指定链的最新 N 个区块
func (r *BlockRepo) GetLatestN(chain string, n int) ([]model.Block, error) {
	var blocks []model.Block
	err := r.db.Where("chain = ?", chain).
		Order("block_number DESC").
		Limit(n).
		Find(&blocks).Error
	return blocks, err
}
