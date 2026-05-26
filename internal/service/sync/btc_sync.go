package sync

import (
	"context"
	"time"

	"blockexplore/internal/client"
	"blockexplore/internal/mq"
	"blockexplore/pkg/logger"

	"go.uber.org/zap"
)

// BtcSyncWorker 比特币同步 Worker
type BtcSyncWorker struct {
	client   *client.BtcClient
	producer *mq.Producer
	interval time.Duration
}

// NewBtcSyncWorker 创建比特币同步 Worker
func NewBtcSyncWorker(btcClient *client.BtcClient, producer *mq.Producer, syncInterval int) *BtcSyncWorker {
	return &BtcSyncWorker{
		client:   btcClient,
		producer: producer,
		interval: time.Duration(syncInterval) * time.Second,
	}
}

// Run 启动比特币区块同步
func (w *BtcSyncWorker) Run(ctx context.Context) error {
	logger.Info("BTC 同步 Worker 已启动",
		zap.Duration("interval", w.interval),
	)

	// 启动时立即同步一次
	if err := w.sync(ctx); err != nil {
		logger.Error("BTC 首次同步失败", zap.Error(err))
	}

	// 定时同步
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("BTC 同步 Worker 已停止")
			return nil
		case <-ticker.C:
			if err := w.sync(ctx); err != nil {
				logger.Error("BTC 同步失败", zap.Error(err))
			}
		}
	}
}

// sync 执行一次比特币区块同步
func (w *BtcSyncWorker) sync(ctx context.Context) error {
	// 获取最新区块高度
	latestBlock, err := w.client.GetLatestBlockNumber()
	if err != nil {
		return err
	}

	logger.Debug("BTC 最新区块高度", zap.Int64("block_number", latestBlock))

	// 拉取最新区块详情
	block, transactions, err := w.client.GetBlockByNumber(latestBlock)
	if err != nil {
		return err
	}

	// 发送到 Kafka
	msg := mq.BlockMessage{
		Chain:       "btc",
		BlockNumber: latestBlock,
		Data: map[string]interface{}{
			"block":        block,
			"transactions": transactions,
		},
	}

	if err := w.producer.Send(ctx, msg); err != nil {
		return err
	}

	logger.Info("BTC 区块已同步",
		zap.Int64("block_number", latestBlock),
		zap.Int("tx_count", len(transactions)),
	)

	return nil
}
