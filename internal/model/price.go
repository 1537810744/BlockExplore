// ============================================================
// PriceHistory 价格历史表模型
// ============================================================
// 对应数据库中的 price_history 表，记录各链原生代币的历史价格。
//
// 用途：
//   - 绘制价格曲线图
//   - 查询历史价格
//   - 价格趋势分析
//
// Go 语言基础知识:
//   - struct：结构体，用于定义数据模型
//   - gorm 标签：定义数据库字段属性
//   - json 标签：定义 JSON 序列化时的字段名
// ============================================================
package model

import "time"

// ============================================================
// PriceHistory 价格历史表模型
// ============================================================
// 记录 ETH、BTC、SOL 的历史价格数据
// 数据来源：CoinGecko API（免费的加密货币价格 API）
type PriceHistory struct {
	// 主键 ID，自增
	ID int64 `gorm:"primaryKey;autoIncrement" json:"id"`

	// 链标识：eth/btc/sol
	Chain string `gorm:"type:varchar(10);not null" json:"chain"`

	// 代币符号：ETH/BTC/SOL
	Symbol string `gorm:"type:varchar(10);not null" json:"symbol"`

	// 美元价格
	// 使用 text 类型存储，因为价格可能有小数，精度要求高
	PriceUSD string `gorm:"type:text" json:"price_usd"`

	// 价格时间，Unix 时间戳
	// 记录这个价格是什么时候获取的
	Timestamp int64 `gorm:"not null" json:"timestamp"`

	// 记录创建时间，GORM 自动填充
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// TableName 方法指定 GORM 使用的表名
func (PriceHistory) TableName() string {
	return "price_history"
}
