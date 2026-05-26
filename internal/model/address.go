package model

import "time"

// Address 地址表模型
// 对应数据库中的 addresses 表，记录地址的交易统计信息
type Address struct {
	ID           int64     `gorm:"primaryKey;autoIncrement" json:"id"`           // 主键 ID
	Chain        string    `gorm:"type:varchar(10);not null" json:"chain"`       // 链标识: eth/btc/sol
	Address      string    `gorm:"type:varchar(128);not null" json:"address"`    // 区块链地址
	Balance      string    `gorm:"type:numeric(78,18)" json:"balance"`           // 当前余额
	TxCount      int64     `gorm:"default:0" json:"tx_count"`                    // 交易总数
	FirstSeenAt  int64     `json:"first_seen_at"`                                // 首次交易时间
	LastSeenAt   int64     `json:"last_seen_at"`                                 // 最近交易时间
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`             // 记录创建时间
	UpdatedAt    time.Time `gorm:"autoUpdateTime" json:"updated_at"`             // 记录更新时间
}

// TableName 指定 GORM 使用的表名
func (Address) TableName() string {
	return "addresses"
}
