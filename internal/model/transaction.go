// ============================================================
// Package model 定义数据模型，使用 GORM 映射数据库表结构
// ============================================================
// 该文件定义了交易表的数据模型。
//
// Go 语言基础知识:
//   - struct：结构体，用于定义数据结构
//   - *int64：指针类型，可以为 nil，表示可选字段
//   - gorm:"..."：GORM 标签，定义数据库字段属性
//   - json:"..."：JSON 标签，定义 JSON 序列化时的字段名
//   - index：创建索引，加快查询速度
// ============================================================
package model

import "time"

// ============================================================
// Transaction 交易表模型
// ============================================================
// 对应数据库中的 transactions 表，存储各链的交易信息
// 每笔交易包含：链标识、交易哈希、发送方、接收方、金额等
type Transaction struct {
	// 主键 ID，自增
	ID int64 `gorm:"primaryKey;autoIncrement" json:"id"`

	// 链标识：eth/btc/sol
	Chain string `gorm:"type:varchar(10);not null" json:"chain"`

	// 交易哈希，唯一标识一笔交易
	// 以太坊交易哈希 66 字符（0x + 64 十六进制字符）
	// 比特币交易哈希 64 字符（纯十六进制）
	// Solana 交易签名 87-88 字符（Base58 编码）
	TxHash string `gorm:"type:varchar(128);not null" json:"tx_hash"`

	// 所在区块高度
	BlockNumber int64 `gorm:"not null" json:"block_number"`

	// 关联的区块表 ID（外键）
	// 通过这个字段可以关联查询区块信息
	BlockID int64 `json:"block_id"`

	// 发送方地址
	// index 标签会在这个字段上创建数据库索引，加快按地址查询的速度
	FromAddr string `gorm:"type:varchar(128);index" json:"from_addr"`

	// 接收方地址
	ToAddr string `gorm:"type:varchar(128);index" json:"to_addr"`

	// 转账金额
	// numeric(78,18) 表示最多 78 位数字，其中小数部分 18 位
	// 使用字符串存储是因为 Go 的 float64 精度不够，大金额会丢失精度
	Value string `gorm:"type:numeric(78,18)" json:"value"`

	// Gas 价格（仅以太坊使用）
	// Gas 价格越高，矿工越优先打包你的交易
	GasPrice string `gorm:"type:text" json:"gas_price"`

	// 实际消耗 Gas（仅以太坊使用）
	// 交易执行后实际消耗的 Gas 数量
	GasUsed string `gorm:"type:text" json:"gas_used"`

	// Gas 上限（仅以太坊使用）
	// 用户愿意为这笔交易支付的最大 Gas 数量
	GasLimit string `gorm:"type:text" json:"gas_limit"`

	// 交易序号（仅以太坊使用）
	// 每个账户的交易都有一个递增的序号，防止重放攻击
	Nonce *int64 `json:"nonce"`

	// 调用数据（仅以太坊使用）
	// 如果是合约调用，这里存储调用参数（calldata）
	InputData string `gorm:"type:text" json:"input_data"`

	// 交易状态：1=成功，0=失败
	Status int16 `gorm:"default:1" json:"status"`

	// 交易时间，Unix 时间戳
	Timestamp int64 `gorm:"not null" json:"timestamp"`

	// 记录创建时间，GORM 自动填充
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// TableName 方法指定 GORM 使用的表名
func (Transaction) TableName() string {
	return "transactions"
}
