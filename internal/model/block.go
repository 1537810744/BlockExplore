// ============================================================
// Package model 定义数据模型，使用 GORM 映射数据库表结构
// ============================================================
// 该包定义了所有数据库表对应的结构体（Model）。
//
// GORM 是 Go 语言的 ORM（对象关系映射）库，可以将结构体映射到数据库表。
// 类似于 Java 的 Hibernate 或 Python 的 SQLAlchemy。
//
// GORM 标签说明：
//   - gorm:"primaryKey"：标记为主键
//   - gorm:"autoIncrement"：自增
//   - gorm:"type:varchar(10)"：指定数据库字段类型
//   - gorm:"not null"：不允许为空
//   - gorm:"default:0"：默认值
//   - gorm:"autoCreateTime"：创建时自动填充时间
//   - json:"field_name"：JSON 序列化时的字段名
//
// Go 语言基础知识:
//   - struct：结构体，类似于 Java 的 class
//   - 反引号 `：用于定义标签（tag），标签是键值对，用于给字段附加元信息
//   - *int64：指针类型，指针可以为 nil（表示空值），普通类型不能为 nil
//   - time.Time：Go 标准库的时间类型
// ============================================================
package model

import "time" // 导入时间包

// ============================================================
// Block 区块表模型
// ============================================================
// 对应数据库中的 blocks 表，存储各链的区块信息
// 每个区块包含：链标识、区块高度、区块哈希、父区块哈希、出块时间等
type Block struct {
	// 主键 ID，自增，GORM 会自动管理
	ID int64 `gorm:"primaryKey;autoIncrement" json:"id"`

	// 链标识：eth（以太坊）/ btc（比特币）/ sol（Solana）
	// varchar(10) 表示最大长度 10 的字符串
	Chain string `gorm:"type:varchar(10);not null" json:"chain"`

	// 区块高度（区块编号），从 0 开始递增
	// 比特币创世区块高度为 0，以太坊创世区块高度为 0
	BlockNumber int64 `gorm:"not null" json:"block_number"`

	// 区块哈希，唯一标识一个区块
	// 以太坊哈希 66 字符（0x + 64 十六进制字符）
	// 比特币哈希 64 字符（纯十六进制）
	BlockHash string `gorm:"type:varchar(128);not null" json:"block_hash"`

	// 父区块哈希，用于链接区块形成链
	// 每个区块都包含前一个区块的哈希，形成区块链
	ParentHash string `gorm:"type:varchar(128)" json:"parent_hash"`

	// 出块时间，Unix 时间戳（从 1970-01-01 00:00:00 UTC 开始的秒数）
	Timestamp int64 `gorm:"not null" json:"timestamp"`

	// 区块内的交易数量
	TxCount int `gorm:"default:0" json:"tx_count"`

	// 已消耗 Gas（仅以太坊/Solana 使用）
	// Gas 是以太坊的计费单位，类似于手机话费
	GasUsed string `gorm:"type:text" json:"gas_used"`

	// Gas 上限（仅以太坊使用）
	// 每个区块有 Gas 上限，限制区块内交易的总 Gas 消耗
	GasLimit string `gorm:"type:text" json:"gas_limit"`

	// 区块大小（字节，仅比特币使用）
	SizeBytes int `json:"size_bytes"`

	// 难度值（仅比特币使用）
	// 比特币的挖矿难度，越高越难挖到区块
	Difficulty string `gorm:"type:text" json:"difficulty"`

	// 槽位号（仅 Solana 使用）
	// *int64 是指针类型，可以为 nil（表示非 Solana 链时为空）
	Slot *int64 `json:"slot"`

	// 记录创建时间，GORM 会在插入时自动填充
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// TableName 方法指定 GORM 使用的表名
// 如果不实现这个方法，GORM 会默认使用结构体名的复数形式（blocks）
// 这里显式指定表名为 "blocks"
func (Block) TableName() string {
	return "blocks"
}
