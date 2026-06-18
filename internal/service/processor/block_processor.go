// ============================================================
// Package processor 提供区块处理器服务
// ============================================================
// 从 Kafka 消费原始区块数据，解析后写入 PostgreSQL。
//
// 处理流程：
//   1. 从 Kafka 接收 BlockMessage
//   2. 解析区块数据（反序列化 JSON）
//   3. 解析交易数据
//   4. 保存区块到数据库
//   5. 设置交易的外键关联
//   6. 批量保存交易到数据库
//
// Go 语言基础知识:
//   - json.Marshal：将结构体序列化为 JSON 字节
//   - json.Unmarshal：将 JSON 字节反序列化为结构体
//   - interface{}：空接口，可以持有任意类型的值
//   - map[string]interface{}：键为字符串的 map，值可以是任意类型
//   - range：遍历切片或 map
//   - append：向切片追加元素
//   - for i := range transactions：遍历切片，i 是索引
// ============================================================
package processor

import (
	"encoding/json" // JSON 编解码
	"fmt"           // 格式化字符串

	"blockexplore/internal/model" // 数据模型
	"blockexplore/internal/mq"    // Kafka 消息队列
	"blockexplore/pkg/logger"     // 日志

	"go.uber.org/zap" // 日志库
)

// ============================================================
// 依赖接口定义（用于解耦和单元测试 mock）
// ============================================================

// BlockWriter 区块写入接口
type BlockWriter interface {
	CreateSingle(block *model.Block) error
}

// TxWriter 交易写入接口
type TxWriter interface {
	Create(txs []model.Transaction) error
}

// ============================================================
// BlockProcessor 区块处理器
// ============================================================
// 从 Kafka 消费消息，解析区块和交易数据，写入数据库
type BlockProcessor struct {
	blockRepo BlockWriter // 区块数据访问层
	txRepo    TxWriter    // 交易数据访问层
}

// ============================================================
// NewBlockProcessor 创建区块处理器实例
// ============================================================
func NewBlockProcessor(blockRepo BlockWriter, txRepo TxWriter) *BlockProcessor {
	return &BlockProcessor{
		blockRepo: blockRepo,
		txRepo:    txRepo,
	}
}

// ============================================================
// Handle 方法：处理从 Kafka 消费到的区块消息
// ============================================================
// 参数 msg：Kafka 消息（包含链标识、区块高度、原始数据）
// 这个方法会被 Consumer 调用，作为消息处理函数
func (p *BlockProcessor) Handle(msg mq.BlockMessage) error {
	logger.Info("开始处理区块消息",
		zap.String("chain", msg.Chain),
		zap.Int64("block_number", msg.BlockNumber),
	)

	// ============================================================
	// 第 1 步：将原始数据反序列化为 map
	// ============================================================
	// msg.Data 是 interface{} 类型，需要先序列化为 JSON，再反序列化为 map
	dataBytes, err := json.Marshal(msg.Data)
	if err != nil {
		return fmt.Errorf("序列化区块数据失败: %w", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(dataBytes, &data); err != nil {
		return fmt.Errorf("解析区块数据失败: %w", err)
	}

	// ============================================================
	// 第 2 步：解析区块数据
	// ============================================================
	// data["block"] 是 interface{} 类型，需要再次序列化/反序列化
	blockData, err := json.Marshal(data["block"])
	if err != nil {
		return fmt.Errorf("解析区块失败: %w", err)
	}

	var block model.Block
	if err := json.Unmarshal(blockData, &block); err != nil {
		return fmt.Errorf("反序列化区块失败: %w", err)
	}

	// ============================================================
	// 第 3 步：解析交易数据
	// ============================================================
	txData, err := json.Marshal(data["transactions"])
	if err != nil {
		return fmt.Errorf("解析交易失败: %w", err)
	}

	var transactions []model.Transaction
	if err := json.Unmarshal(txData, &transactions); err != nil {
		return fmt.Errorf("反序列化交易失败: %w", err)
	}

	// ============================================================
	// 第 4 步：保存区块到数据库
	// ============================================================
	if err := p.blockRepo.CreateSingle(&block); err != nil {
		return fmt.Errorf("保存区块失败: %w", err)
	}

	// ============================================================
	// 第 5 步：设置交易的区块 ID（外键关联）
	// ============================================================
	// block.ID 是数据库自动生成的主键
	// 保存区块后，GORM 会自动填充 ID 字段
	// for i := range transactions 遍历切片，i 是索引
	// 我们需要修改切片元素，所以使用索引访问
	for i := range transactions {
		transactions[i].BlockID = block.ID // 设置外键
	}

	// ============================================================
	// 第 6 步：批量保存交易到数据库
	// ============================================================
	if len(transactions) > 0 {
		if err := p.txRepo.Create(transactions); err != nil {
			return fmt.Errorf("保存交易失败: %w", err)
		}
	}

	logger.Info("区块处理完成",
		zap.String("chain", msg.Chain),
		zap.Int64("block_number", msg.BlockNumber),
		zap.Int("tx_count", len(transactions)),
	)

	return nil
}
