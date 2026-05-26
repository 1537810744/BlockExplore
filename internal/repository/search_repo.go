package repository

import (
	"regexp"
	"strconv"

	"blockexplore/internal/model"

	"gorm.io/gorm"
)

// SearchRepo 搜索数据访问层
// 提供统一搜索功能，根据输入自动识别类型（区块/交易/地址）
type SearchRepo struct {
	db *gorm.DB
}

// NewSearchRepo 创建搜索数据访问层实例
func NewSearchRepo(db *gorm.DB) *SearchRepo {
	return &SearchRepo{db: db}
}

// SearchResult 搜索结果结构体
type SearchResult struct {
	Type string      `json:"type"` // 结果类型: block/transaction/address
	Data interface{} `json:"data"` // 结果数据
}

// 根据输入内容的格式判断搜索类型
// 匹配规则：
//   - 42字符，0x开头 → 以太坊交易哈希
//   - 64字符，非0x开头 → 比特币交易哈希
//   - 87-88字符，Base58 → Solana 签名
//   - 0x + 40字符 → 以太坊地址
//   - 1/3/bc1开头 → 比特币地址
//   - 纯数字 → 区块高度

// 判断是否是纯数字（区块高度）
var isNumberRegex = regexp.MustCompile(`^\d+$`)

// Search 统一搜索入口
// keyword: 搜索关键词
// 返回: 搜索结果（包含类型和数据）
func (r *SearchRepo) Search(keyword string) (*SearchResult, error) {
	// 1. 判断是否是区块高度（纯数字）
	if isNumberRegex.MatchString(keyword) {
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

	// 2. 判断是否是 0x 开头（以太坊交易哈希或地址）
	if len(keyword) == 66 && keyword[:2] == "0x" {
		// 42字符（0x + 40）是地址，66字符（0x + 64）是交易哈希
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

	// 3. 判断是否是比特币地址（1/3/bc1开头）
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

	// 4. 判断是否是 Solana 签名或地址（Base58，87-88字符或32-44字符）
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
