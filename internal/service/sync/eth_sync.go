// ============================================================
// Package sync 提供各链的区块同步服务
// ============================================================
// 该包实现了区块链区块的同步功能。
//
// 同步流程：
//   1. 定时从区块链节点获取最新区块高度
//   2. 与本地已同步的高度对比，获取缺失的区块
//   3. 将区块数据发送到 Kafka 消息队列
//   4. Block Processor 从 Kafka 消费并写入数据库
//
// Go 语言基础知识:
//   - time.Ticker：定时器，每隔固定时间触发一次
//   - time.NewTicker：创建定时器
//   - ticker.C：定时器的 channel，每隔固定时间会收到一个值
//   - select：多路复用，同时等待多个 channel 操作
//   - defer：延迟执行，确保资源被正确释放
//   - context.Context：上下文，用于控制 goroutine 的生命周期
//   - context.Done()：返回一个 channel，当上下文被取消时会关闭
// ============================================================
package sync

import (
	"context"   // 上下文，用于控制 goroutine 的生命周期
	"time"      // 时间处理

	"blockexplore/internal/client"  // 区块链 RPC 客户端
	"blockexplore/internal/mq"      // Kafka 消息队列
	"blockexplore/pkg/logger"       // 日志

	"go.uber.org/zap" // 日志库
)

// ============================================================
// EthSyncWorker 以太坊同步 Worker
// ============================================================
// 定期从以太坊节点拉取最新区块，发送到 Kafka
type EthSyncWorker struct {
	client   *client.EthClient // 以太坊 RPC 客户端
	producer *mq.Producer      // Kafka 生产者
	interval time.Duration     // 同步间隔
}

// ============================================================
// NewEthSyncWorker 创建以太坊同步 Worker
// ============================================================
// 参数 ethClient：以太坊 RPC 客户端
// 参数 producer：Kafka 生产者
// 参数 syncInterval：同步间隔（秒）
func NewEthSyncWorker(ethClient *client.EthClient, producer *mq.Producer, syncInterval int) *EthSyncWorker {
	return &EthSyncWorker{
		client:   ethClient,
		producer: producer,
		// time.Duration(syncInterval) * time.Second 将秒数转换为 Duration 类型
		interval: time.Duration(syncInterval) * time.Second,
	}
}

// ============================================================
// Run 方法：启动同步任务
// ============================================================
// 参数 ctx：上下文（用于优雅关闭）
// 此方法会阻塞，持续同步区块直到 ctx 被取消
func (w *EthSyncWorker) Run(ctx context.Context) error {
	logger.Info("ETH 同步 Worker 已启动",
		zap.Duration("interval", w.interval),
	)

	// 启动时立即同步一次
	if err := w.sync(ctx); err != nil {
		logger.Error("ETH 首次同步失败", zap.Error(err))
	}

	// time.NewTicker 创建定时器
	// 每隔 w.interval 时间，ticker.C channel 会收到一个值
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop() // 确保定时器被释放

	// 无限循环，持续同步
	for {
		// select 语句用于多路复用
		select {
		case <-ctx.Done():
			// 上下文被取消，停止同步
			logger.Info("ETH 同步 Worker 已停止")
			return nil
		case <-ticker.C:
			// 定时器触发，执行同步
			if err := w.sync(ctx); err != nil {
				logger.Error("ETH 同步失败", zap.Error(err))
			}
		}
	}
}

// ============================================================
// sync 方法：执行一次同步
// ============================================================
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
