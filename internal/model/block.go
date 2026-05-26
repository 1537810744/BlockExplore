// Package model 定义数据模型，使用 GORM 映射数据库表结构
package model

import "time"

// Block 区块表模型
// 对应数据库中的 blocks 表，存储各链的区块信息
type Block struct {
	ID          int64     `gorm:"primaryKey;autoIncrement" json:"id"`          // 主键 ID
	Chain       string    `gorm:"type:varchar(10);not null" json:"chain"`      // 链标识: eth/btc/sol
	BlockNumber int64     `gorm:"not null" json:"block_number"`                // 区块高度
	BlockHash   string    `gorm:"type:varchar(128);not null" json:"block_hash"` // 区块哈希
	ParentHash  string    `gorm:"type:varchar(128)" json:"parent_hash"`        // 父区块哈希
	Timestamp   int64     `gorm:"not null" json:"timestamp"`                   // 出块时间（Unix 时间戳）
	TxCount     int       `gorm:"default:0" json:"tx_count"`                   // 区块内交易数量
	GasUsed     string    `gorm:"type:text" json:"gas_used"`                   // 已消耗 Gas（ETH/SOL）
	GasLimit    string    `gorm:"type:text" json:"gas_limit"`                   // Gas 上限（ETH）
	SizeBytes   int       `json:"size_bytes"`                                  // 区块大小（字节，BTC）
	Difficulty  string    `gorm:"type:text" json:"difficulty"`                  // 难度值（BTC）
	Slot        *int64    `json:"slot"`                                        // 槽位号（SOL）
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`            // 记录创建时间
}

// TableName 指定 GORM 使用的表名
func (Block) TableName() string {
	return "blocks"
}
