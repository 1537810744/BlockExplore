package repository

import (
	"blockexplore/internal/model"

	"gorm.io/gorm"
)

// PriceRepo 价格数据访问层
type PriceRepo struct {
	db *gorm.DB
}

// NewPriceRepo 创建价格数据访问层实例
func NewPriceRepo(db *gorm.DB) *PriceRepo {
	return &PriceRepo{db: db}
}

// Create 保存价格记录
func (r *PriceRepo) Create(price *model.PriceHistory) error {
	return r.db.Create(price).Error
}

// CreateBatch 批量保存价格记录
func (r *PriceRepo) CreateBatch(prices []model.PriceHistory) error {
	if len(prices) == 0 {
		return nil
	}
	return r.db.CreateInBatches(prices, 100).Error
}

// GetLatestPrice 获取指定链的最新价格
func (r *PriceRepo) GetLatestPrice(chain string) (*model.PriceHistory, error) {
	var price model.PriceHistory
	err := r.db.Where("chain = ?", chain).
		Order("timestamp DESC").
		First(&price).Error
	if err != nil {
		return nil, err
	}
	return &price, nil
}

// GetPriceHistory 获取价格历史（用于绘制价格曲线）
// chain: 链标识
// startTime: 开始时间（Unix 时间戳）
// endTime: 结束时间（Unix 时间戳）
// limit: 最大返回条数
func (r *PriceRepo) GetPriceHistory(chain string, startTime, endTime int64, limit int) ([]model.PriceHistory, error) {
	var prices []model.PriceHistory
	query := r.db.Where("chain = ?", chain)

	if startTime > 0 {
		query = query.Where("timestamp >= ?", startTime)
	}
	if endTime > 0 {
		query = query.Where("timestamp <= ?", endTime)
	}

	err := query.Order("timestamp ASC").
		Limit(limit).
		Find(&prices).Error

	return prices, err
}
