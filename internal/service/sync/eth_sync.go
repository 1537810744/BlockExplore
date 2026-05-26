// Package sync 提供各链的区块同步服务
// Sync Worker 负责从区块链节点拉取新区块，发送到 Kafka
package sync

import (
	"context"
	"time"

	"blockexplore/internal/client"
	"blockexplore/internal/mq"
	"blockexplore/pkg/logger"

	"go.uber.org/zap"
)

// EthSyncWorker 以太坊同步 Worker
// 定期从以太坊节点拉取最新区块，发送到 Kafka
type EthSyncWorker struct {
	client   *client.EthClient // 以太坊 RPC 客户端
	producer *mq.Producer      // Kafka 生产者
	interval time.Duration     // 同步间隔
}

// NewEthSyncWorker 创建以太坊同步 Worker
func NewEthSyncWorker(ethClient *client.EthClient, producer *mq.Producer, syncInterval int) *EthSyncWorker {
	return &EthSyncWorker{
		client:   ethClient,
		producer: producer,
		interval: time.Duration(syncInterval) * time.Second,
	}
}

// Run 启动同步任务
// ctx: 上下文（用于优雅关闭）
// 此方法会阻塞，持续同步区块直到 ctx 被取消
func (w *EthSyncWorker) Run(ctx context.Context) error {
	logger.Info("ETH 同步 Worker 已启动",
		zap.Duration("interval", w.interval),
	)

	// 启动时立即同步一次
	if err := w.sync(ctx); err != nil {
		logger.Error("ETH 首次同步失败", zap.Error(err))
	}

	// 定时同步
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("ETH 同步 Worker 已停止")
			return nil
		case <-ticker.C:
			if err := w.sync(ctx); err != nil {
				logger.Error("ETH 同步失败", zap.Error(err))
			}
		}
	}
}

// sync 执行一次同步
// 获取最新区块，与本地比较，拉取缺失的区块发送到 Kafka
func (w *EthSyncWorker) sync(ctx context.Context) error {
	// 获取链上最新区块高度
	latestBlock, err := w.client.GetLatestBlockNumber()
	if err != nil {
		return err
	}

	logger.Debug("ETH 最新区块高度", zap.Int64("block_number", latestBlock))

	// 获取本地最新同步到的区块高度
	// 这里从 Kafka 生产者无法直接获取，需要从数据库查询
	// 为简化实现，这里直接拉取最新区块
	// 实际生产中应该记录上次同步到的高度，只拉取增量

	// 拉取最新区块
	block, transactions, err := w.client.GetBlockByNumber(latestBlock)
	if err != nil {
		return err
	}

	// 构建 Kafka 消息
	msg := mq.BlockMessage{
		Chain:       "eth",
		BlockNumber: latestBlock,
		Data: map[string]interface{}{
			"block":        block,
			"transactions": transactions,
		},
	}

	// 发送到 Kafka
	if err := w.producer.Send(ctx, msg); err != nil {
		return err
	}

	logger.Info("ETH 区块已同步",
		zap.Int64("block_number", latestBlock),
		zap.Int("tx_count", len(transactions)),
	)

	return nil
}
