// Package processor 提供区块处理器服务
// 从 Kafka 消费原始区块数据，解析后写入 PostgreSQL
package processor

import (
	"encoding/json"
	"fmt"

	"blockexplore/internal/model"
	"blockexplore/internal/mq"
	"blockexplore/internal/repository"
	"blockexplore/pkg/logger"

	"go.uber.org/zap"
)

// BlockProcessor 区块处理器
// 从 Kafka 消费消息，解析区块和交易数据，写入数据库
type BlockProcessor struct {
	blockRepo *repository.BlockRepo // 区块数据访问层
	txRepo    *repository.TxRepo    // 交易数据访问层
}

// NewBlockProcessor 创建区块处理器实例
func NewBlockProcessor(blockRepo *repository.BlockRepo, txRepo *repository.TxRepo) *BlockProcessor {
	return &BlockProcessor{
		blockRepo: blockRepo,
		txRepo:    txRepo,
	}
}

// Handle 处理从 Kafka 消费到的区块消息
// msg: Kafka 消息（包含链标识、区块高度、原始数据）
func (p *BlockProcessor) Handle(msg mq.BlockMessage) error {
	logger.Info("开始处理区块消息",
		zap.String("chain", msg.Chain),
		zap.Int64("block_number", msg.BlockNumber),
	)

	// 将原始数据反序列化为 map
	dataBytes, err := json.Marshal(msg.Data)
	if err != nil {
		return fmt.Errorf("序列化区块数据失败: %w", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(dataBytes, &data); err != nil {
		return fmt.Errorf("解析区块数据失败: %w", err)
	}

	// 解析区块数据
	blockData, err := json.Marshal(data["block"])
	if err != nil {
		return fmt.Errorf("解析区块失败: %w", err)
	}

	var block model.Block
	if err := json.Unmarshal(blockData, &block); err != nil {
		return fmt.Errorf("反序列化区块失败: %w", err)
	}

	// 解析交易数据
	txData, err := json.Marshal(data["transactions"])
	if err != nil {
		return fmt.Errorf("解析交易失败: %w", err)
	}

	var transactions []model.Transaction
	if err := json.Unmarshal(txData, &transactions); err != nil {
		return fmt.Errorf("反序列化交易失败: %w", err)
	}

	// 保存区块到数据库
	if err := p.blockRepo.CreateSingle(&block); err != nil {
		return fmt.Errorf("保存区块失败: %w", err)
	}

	// 设置交易的区块 ID（外键关联）
	for i := range transactions {
		transactions[i].BlockID = block.ID
	}

	// 批量保存交易到数据库
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
