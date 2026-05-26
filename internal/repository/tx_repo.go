package repository

import (
	"blockexplore/internal/model"

	"gorm.io/gorm"
)

// TxRepo 交易数据访问层
type TxRepo struct {
	db *gorm.DB
}

// NewTxRepo 创建交易数据访问层实例
func NewTxRepo(db *gorm.DB) *TxRepo {
	return &TxRepo{db: db}
}

// Create 批量创建交易记录
func (r *TxRepo) Create(txs []model.Transaction) error {
	if len(txs) == 0 {
		return nil
	}
	return r.db.CreateInBatches(txs, 100).Error
}

// CreateSingle 创建单个交易记录
func (r *TxRepo) CreateSingle(tx *model.Transaction) error {
	return r.db.Create(tx).Error
}

// GetByHash 根据链标识和交易哈希查询交易
func (r *TxRepo) GetByHash(chain, txHash string) (*model.Transaction, error) {
	var tx model.Transaction
	err := r.db.Where("chain = ? AND tx_hash = ?", chain, txHash).First(&tx).Error
	if err != nil {
		return nil, err
	}
	return &tx, nil
}

// GetByBlockNumber 获取指定区块内的所有交易（分页）
func (r *TxRepo) GetByBlockNumber(chain string, blockNumber int64, page, pageSize int) ([]model.Transaction, int64, error) {
	var txs []model.Transaction
	var total int64

	query := r.db.Where("chain = ? AND block_number = ?", chain, blockNumber)
	query.Model(&model.Transaction{}).Count(&total)

	offset := (page - 1) * pageSize
	err := query.Order("id ASC").
		Offset(offset).
		Limit(pageSize).
		Find(&txs).Error

	return txs, total, err
}

// GetByAddress 获取指定地址的交易记录（分页）
// 同时查询 from_addr 和 to_addr
func (r *TxRepo) GetByAddress(chain, address string, page, pageSize int) ([]model.Transaction, int64, error) {
	var txs []model.Transaction
	var total int64

	query := r.db.Where("chain = ? AND (from_addr = ? OR to_addr = ?)", chain, address, address)
	query.Model(&model.Transaction{}).Count(&total)

	offset := (page - 1) * pageSize
	err := query.Order("timestamp DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&txs).Error

	return txs, total, err
}

// GetLatestN 获取指定链的最新 N 条交易
func (r *TxRepo) GetLatestN(chain string, n int) ([]model.Transaction, error) {
	var txs []model.Transaction
	err := r.db.Where("chain = ?", chain).
		Order("timestamp DESC").
		Limit(n).
		Find(&txs).Error
	return txs, err
}
