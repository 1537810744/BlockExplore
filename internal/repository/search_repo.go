// ============================================================
// SearchRepo 搜索数据访问层
// ============================================================
// 提供统一搜索功能，根据输入自动识别类型（区块/交易/地址）。
//
// 搜索逻辑：
//   - 纯数字 → 区块高度
//   - 0x 开头 + 64 字符 → 以太坊交易哈希
//   - 0x 开头 + 40 字符 → 以太坊地址
//   - 1/3/bc1 开头 → 比特币地址
//   - Base58 编码 → Solana 交易签名或地址
//
// Go 语言基础知识:
//   - regexp.MustCompile：编译正则表达式，Must 表示如果编译失败会 panic
//   - strconv.ParseInt：字符串转整数
//   - interface{}：空接口，可以持有任意类型的值
//   - var isNumberRegex = regexp.MustCompile(...)：包级变量，程序启动时编译一次
//   - keyword[:2]：切片操作，取前 2 个字符
//   - keyword[0]：取第一个字符
// ============================================================
package repository

import (
	"regexp"  // 正则表达式
	"strconv" // 字符串转换

	"blockexplore/internal/model" // 数据模型

	"gorm.io/gorm" // GORM ORM 库
)

// ============================================================
// SearchRepo 搜索数据访问层
// ============================================================
type SearchRepo struct {
	db *gorm.DB
}

// ============================================================
// NewSearchRepo 创建搜索数据访问层实例
// ============================================================
func NewSearchRepo(db *gorm.DB) *SearchRepo {
	return &SearchRepo{db: db}
}

// ============================================================
// SearchResult 搜索结果结构体
// ============================================================
type SearchResult struct {
	Type string      `json:"type"` // 结果类型: block/transaction/address
	Data interface{} `json:"data"` // 结果数据（可以是任意类型）
}

// ============================================================
// 匹配规则说明
// ============================================================
// 根据输入内容的格式判断搜索类型：
//   - 42 字符，0x 开头 → 以太坊地址
//   - 66 字符，0x 开头 → 以太坊交易哈希
//   - 64 字符，非 0x 开头 → 比特币交易哈希
//   - 87-88 字符，Base58 → Solana 签名
//   - 1/3/bc1 开头 → 比特币地址
//   - 纯数字 → 区块高度

// isNumberRegex 判断是否是纯数字（区块高度）
// ^ 表示字符串开头，\d+ 表示一个或多个数字，$ 表示字符串结尾
var isNumberRegex = regexp.MustCompile(`^\d+$`)

// ============================================================
// Search 方法：统一搜索入口
// ============================================================
// 参数 keyword：搜索关键词
// 返回值：搜索结果（包含类型和数据），未找到返回 nil
func (r *SearchRepo) Search(keyword string) (*SearchResult, error) {
	// ============================================================
	// 第 1 步：判断是否是区块高度（纯数字）
	// ============================================================
	if isNumberRegex.MatchString(keyword) {
		// strconv.ParseInt 将字符串解析为 int64
		// 参数：字符串、进制（10=十进制）、位数（64 位）
		blockNumber, err := strconv.ParseInt(keyword, 10, 64)
		if err == nil {
			// 尝试在三条链中查找
			for _, chain := range []string{"eth", "btc", "sol"} {
				var block model.Block
				if err := r.db.Where("chain = ? AND block_number = ?", chain, blockNumber).First(&block).Error; err == nil {
					return &SearchResult{Type: "block", Data: block}, nil
				}
			}
		}
	}

	// ============================================================
	// 第 2 步：判断是否是 0x 开头（以太坊交易哈希或地址）
	// ============================================================
	// 以太坊地址：42 字符（0x + 40 十六进制字符）
	// 以太坊交易哈希：66 字符（0x + 64 十六进制字符）
	if len(keyword) == 66 && keyword[:2] == "0x" {
		// 先查交易
		var tx model.Transaction
		if err := r.db.Where("chain = 'eth' AND tx_hash = ?", keyword).First(&tx).Error; err == nil {
			return &SearchResult{Type: "transaction", Data: tx}, nil
		}
		// 再查地址
		var addr model.Address
		if err := r.db.Where("chain = 'eth' AND address = ?", keyword).First(&addr).Error; err == nil {
			return &SearchResult{Type: "address", Data: addr}, nil
		}
	}

	// 0x 开头但不是 66 字符，可能是 42 字符的地址
	if len(keyword) == 42 && keyword[:2] == "0x" {
		var addr model.Address
		if err := r.db.Where("chain = 'eth' AND address = ?", keyword).First(&addr).Error; err == nil {
			return &SearchResult{Type: "address", Data: addr}, nil
		}
	}

	// ============================================================
	// 第 3 步：判断是否是比特币地址（1/3/bc1 开头）
	// ============================================================
	// 比特币地址格式：
	//   - P2PKH：以 1 开头，例如 1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa
	//   - P2SH：以 3 开头，例如 3J98t1WpEZ73CNmQviecrnyiWrnqRhWNLy
	//   - Bech32：以 bc1 开头，例如 bc1qar0srrr7xfkvy5l643lydnw9re59gtzzwf5mdq
	if len(keyword) > 0 && (keyword[0] == '1' || keyword[0] == '3' || keyword[:3] == "bc1") {
		var addr model.Address
		if err := r.db.Where("chain = 'btc' AND address = ?", keyword).First(&addr).Error; err == nil {
			return &SearchResult{Type: "address", Data: addr}, nil
		}
		// 也可能是比特币交易哈希
		var tx model.Transaction
		if err := r.db.Where("chain = 'btc' AND tx_hash = ?", keyword).First(&tx).Error; err == nil {
			return &SearchResult{Type: "transaction", Data: tx}, nil
		}
	}

	// ============================================================
	// 第 4 步：判断是否是 Solana 签名或地址
	// ============================================================
	// Solana 交易签名：87-88 字符的 Base58 编码
	// Solana 地址：32-44 字符的 Base58 编码
	if len(keyword) >= 32 && len(keyword) <= 88 {
		// 先查交易
		var tx model.Transaction
		if err := r.db.Where("chain = 'sol' AND tx_hash = ?", keyword).First(&tx).Error; err == nil {
			return &SearchResult{Type: "transaction", Data: tx}, nil
		}
		// 再查地址
		var addr model.Address
		if err := r.db.Where("chain = 'sol' AND address = ?", keyword).First(&addr).Error; err == nil {
			return &SearchResult{Type: "address", Data: addr}, nil
		}
	}

	return nil, nil // 未找到匹配结果
}
