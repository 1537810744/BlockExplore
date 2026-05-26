package model

import "time"

// PriceHistory 价格历史表模型
// 对应数据库中的 price_history 表，记录各链原生代币的历史价格
type PriceHistory struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`        // 主键 ID
	Chain     string    `gorm:"type:varchar(10);not null" json:"chain"`    // 链标识: eth/btc/sol
	Symbol    string    `gorm:"type:varchar(10);not null" json:"symbol"`   // 代币符号: ETH/BTC/SOL
	PriceUSD  string    `gorm:"type:text" json:"price_usd"`               // 美元价格
	Timestamp int64     `gorm:"not null" json:"timestamp"`                 // 价格时间（Unix 时间戳）
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`          // 记录创建时间
}

// TableName 指定 GORM 使用的表名
func (PriceHistory) TableName() string {
	return "price_history"
}
