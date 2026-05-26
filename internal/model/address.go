// ============================================================
// Address 地址表模型
// ============================================================
// 对应数据库中的 addresses 表，记录地址的交易统计信息。
//
// 地址是区块链上的账户标识：
//   - 以太坊地址：42 字符，0x 开头，例如 0x742d35Cc6634C0532925a3b844Bc9e7595f2bD18
//   - 比特币地址：以 1、3 或 bc1 开头
//   - Solana 地址：32-44 字符的 Base58 编码
//
// Go 语言基础知识:
//   - struct：结构体，类似于 Java 的 class
//   - gorm:"autoCreateTime"：插入记录时自动填充创建时间
//   - gorm:"autoUpdateTime"：更新记录时自动填充更新时间
// ============================================================
package model

import "time"

// ============================================================
// Address 地址表模型
// ============================================================
// 记录每个地址的余额、交易次数、首次/最近交易时间等统计信息
type Address struct {
	// 主键 ID，自增
	ID int64 `gorm:"primaryKey;autoIncrement" json:"id"`

	// 链标识：eth/btc/sol
	Chain string `gorm:"type:varchar(10);not null" json:"chain"`

	// 区块链地址
	Address string `gorm:"type:varchar(128);not null" json:"address"`

	// 当前余额
	// numeric(78,18) 支持非常大的数字，小数部分 18 位
	// 以太坊的最小单位是 Wei，1 ETH = 10^18 Wei
	Balance string `gorm:"type:numeric(78,18)" json:"balance"`

	// 交易总数（该地址参与的所有交易）
	TxCount int64 `gorm:"default:0" json:"tx_count"`

	// 首次交易时间（Unix 时间戳）
	FirstSeenAt int64 `json:"first_seen_at"`

	// 最近交易时间（Unix 时间戳）
	LastSeenAt int64 `json:"last_seen_at"`

	// 记录创建时间，GORM 插入时自动填充
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`

	// 记录更新时间，GORM 更新时自动填充
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName 方法指定 GORM 使用的表名
func (Address) TableName() string {
	return "addresses"
}
