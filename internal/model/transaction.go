package model

import "time"

// Transaction 交易表模型
// 对应数据库中的 transactions 表，存储各链的交易信息
type Transaction struct {
	ID          int64     `gorm:"primaryKey;autoIncrement" json:"id"`           // 主键 ID
	Chain       string    `gorm:"type:varchar(10);not null" json:"chain"`       // 链标识: eth/btc/sol
	TxHash      string    `gorm:"type:varchar(128);not null" json:"tx_hash"`    // 交易哈希
	BlockNumber int64     `gorm:"not null" json:"block_number"`                 // 所在区块高度
	BlockID     int64     `json:"block_id"`                                     // 关联的区块表 ID（外键）
	FromAddr    string    `gorm:"type:varchar(128);index" json:"from_addr"`     // 发送方地址
	ToAddr      string    `gorm:"type:varchar(128);index" json:"to_addr"`       // 接收方地址
	Value       string    `gorm:"type:numeric(78,18)" json:"value"`             // 转账金额
	GasPrice    string    `gorm:"type:text" json:"gas_price"`                   // Gas 价格（ETH）
	GasUsed     string    `gorm:"type:text" json:"gas_used"`                    // 实际消耗 Gas（ETH）
	GasLimit    string    `gorm:"type:text" json:"gas_limit"`                   // Gas 上限（ETH）
	Nonce       *int64    `json:"nonce"`                                        // 交易序号（ETH）
	InputData   string    `gorm:"type:text" json:"input_data"`                  // 调用数据（ETH calldata）
	Status      int16     `gorm:"default:1" json:"status"`                      // 交易状态：1=成功 0=失败
	Timestamp   int64     `gorm:"not null" json:"timestamp"`                    // 交易时间（Unix 时间戳）
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`             // 记录创建时间
}

// TableName 指定 GORM 使用的表名
func (Transaction) TableName() string {
	return "transactions"
}
