// ============================================================
// TxRepo 交易数据访问层
// ============================================================
// 封装了交易相关的数据库操作。
//
// Go 语言基础知识:
//   - struct：结构体，用于定义数据结构
//   - *gorm.DB：GORM 数据库连接的指针
//   - error：Go 的错误类型
//   - OR 条件：GORM 使用 OR() 方法或直接在 Where 中写 SQL
//   - 分页：Offset 跳过记录，Limit 限制数量
// ============================================================
package repository

import (
	"blockexplore/internal/model" // 数据模型

	"gorm.io/gorm" // GORM ORM 库
)

// ============================================================
// TxRepo 交易数据访问层
// ============================================================
type TxRepo struct {
	db *gorm.DB
}

// ============================================================
// NewTxRepo 创建交易数据访问层实例
// ============================================================
func NewTxRepo(db *gorm.DB) *TxRepo {
	return &TxRepo{db: db}
}

// ============================================================
// Create 方法：批量创建交易记录
// ============================================================
// 使用 CreateInBatches 批量插入，每批 100 条
func (r *TxRepo) Create(txs []model.Transaction) error {
	if len(txs) == 0 {
		return nil
	}
	return r.db.CreateInBatches(txs, 100).Error
}

// ============================================================
// CreateSingle 方法：创建单个交易记录
// ============================================================
func (r *TxRepo) CreateSingle(tx *model.Transaction) error {
	return r.db.Create(tx).Error
}

// ============================================================
// GetByHash 方法：根据链标识和交易哈希查询交易
// ============================================================
// 参数 chain：链标识（eth/btc/sol）
// 参数 txHash：交易哈希
func (r *TxRepo) GetByHash(chain, txHash string) (*model.Transaction, error) {
	var tx model.Transaction
	err := r.db.Where("chain = ? AND tx_hash = ?", chain, txHash).First(&tx).Error
	if err != nil {
		return nil, err
	}
	return &tx, nil
}

// ============================================================
// GetByBlockNumber 方法：获取指定区块内的所有交易（分页）
// ============================================================
// 参数 chain：链标识
// 参数 blockNumber：区块高度
// 参数 page：页码
// 参数 pageSize：每页数量
func (r *TxRepo) GetByBlockNumber(chain string, blockNumber int64, page, pageSize int) ([]model.Transaction, int64, error) {
	var txs []model.Transaction
	var total int64

	// 构建查询条件
	query := r.db.Where("chain = ? AND block_number = ?", chain, blockNumber)
	// 统计总数
	query.Model(&model.Transaction{}).Count(&total)

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.Order("id ASC"). // 按 ID 升序
					Offset(offset).
					Limit(pageSize).
					Find(&txs).Error

	return txs, total, err
}

// ============================================================
// GetByAddress 方法：获取指定地址的交易记录（分页）
// ============================================================
// 同时查询 from_addr 和 to_addr，即该地址作为发送方或接收方的交易
// 使用 OR 条件：from_addr = address OR to_addr = address
func (r *TxRepo) GetByAddress(chain, address string, page, pageSize int) ([]model.Transaction, int64, error) {
	var txs []model.Transaction
	var total int64

	// SQL: WHERE chain = ? AND (from_addr = ? OR to_addr = ?)
	query := r.db.Where("chain = ? AND (from_addr = ? OR to_addr = ?)", chain, address, address)
	query.Model(&model.Transaction{}).Count(&total)

	offset := (page - 1) * pageSize
	err := query.Order("timestamp DESC"). // 按时间降序，最新的在前
					Offset(offset).
					Limit(pageSize).
					Find(&txs).Error

	return txs, total, err
}

// ============================================================
// GetLatestN 方法：获取指定链的最新 N 条交易
// ============================================================
func (r *TxRepo) GetLatestN(chain string, n int) ([]model.Transaction, error) {
	var txs []model.Transaction
	err := r.db.Where("chain = ?", chain).
		Order("timestamp DESC").
		Limit(n).
		Find(&txs).Error
	return txs, err
}
