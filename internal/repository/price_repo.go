// ============================================================
// PriceRepo 价格数据访问层
// ============================================================
// 封装了价格历史相关的数据库操作。
//
// Go 语言基础知识:
//   - struct：结构体，用于定义数据结构
//   - *gorm.DB：GORM 数据库连接的指针
//   - error：Go 的错误类型
//   - 条件查询：Where 添加条件，支持链式调用
//   - 时间范围查询：timestamp >= ? AND timestamp <= ?
// ============================================================
package repository

import (
	"blockexplore/internal/model" // 数据模型

	"gorm.io/gorm" // GORM ORM 库
)

// ============================================================
// PriceRepo 价格数据访问层
// ============================================================
type PriceRepo struct {
	db *gorm.DB
}

// ============================================================
// NewPriceRepo 创建价格数据访问层实例
// ============================================================
func NewPriceRepo(db *gorm.DB) *PriceRepo {
	return &PriceRepo{db: db}
}

// ============================================================
// Create 方法：保存价格记录
// ============================================================
// 参数 price：价格历史记录指针
func (r *PriceRepo) Create(price *model.PriceHistory) error {
	return r.db.Create(price).Error
}

// ============================================================
// CreateBatch 方法：批量保存价格记录
// ============================================================
func (r *PriceRepo) CreateBatch(prices []model.PriceHistory) error {
	if len(prices) == 0 {
		return nil
	}
	return r.db.CreateInBatches(prices, 100).Error
}

// ============================================================
// GetLatestPrice 方法：获取指定链的最新价格
// ============================================================
// 按时间戳降序排列，取第一条
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

// ============================================================
// GetPriceHistory 方法：获取价格历史（用于绘制价格曲线）
// ============================================================
// 参数 chain：链标识
// 参数 startTime：开始时间（Unix 时间戳），0 表示不限制
// 参数 endTime：结束时间（Unix 时间戳），0 表示不限制
// 参数 limit：最大返回条数
func (r *PriceRepo) GetPriceHistory(chain string, startTime, endTime int64, limit int) ([]model.PriceHistory, error) {
	var prices []model.PriceHistory
	query := r.db.Where("chain = ?", chain)

	// 时间范围条件
	if startTime > 0 {
		query = query.Where("timestamp >= ?", startTime)
	}
	if endTime > 0 {
		query = query.Where("timestamp <= ?", endTime)
	}

	// 按时间升序排列，限制返回条数
	err := query.Order("timestamp ASC").
		Limit(limit).
		Find(&prices).Error

	return prices, err
}
