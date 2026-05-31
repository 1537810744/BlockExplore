// ============================================================
// Package sync 提供各链的区块同步服务
// ============================================================
package sync

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"blockexplore/internal/client"
	"blockexplore/internal/model"
	"blockexplore/internal/mq"
	"blockexplore/pkg/logger"

	"go.uber.org/zap"
)

// EthSyncWorker 以太坊同步 Worker
type EthSyncWorker struct {
	client   *client.EthClient
	producer *mq.Producer
	interval time.Duration
}

// NewEthSyncWorker 创建以太坊同步 Worker
func NewEthSyncWorker(ethClient *client.EthClient, producer *mq.Producer, syncInterval int) *EthSyncWorker {
	return &EthSyncWorker{
		client:   ethClient,
		producer: producer,
		interval: time.Duration(syncInterval) * time.Second,
	}
}

// Run 方法：启动同步任务（阻塞运行直到 ctx 取消）
func (w *EthSyncWorker) Run(ctx context.Context) error {
	logger.Info("ETH 同步 Worker 已启动", zap.Duration("interval", w.interval))

	if err := w.sync(ctx); err != nil {
		logger.Error("ETH 首次同步失败", zap.Error(err))
	}

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

// sync 方法：执行一次同步
func (w *EthSyncWorker) sync(ctx context.Context) error {
	latestBlock, err := w.client.GetLatestBlockNumber()
	if err != nil {
		return err
	}

	block, transactions, err := w.client.GetBlockByNumber(latestBlock)
	if err != nil {
		return err
	}

	// 并发获取交易回执，填充 gas_used 和 status
	w.fillReceipts(ctx, transactions)

	msg := mq.BlockMessage{
		Chain:       "eth",
		BlockNumber: latestBlock,
		Data: map[string]interface{}{
			"block":        block,
			"transactions": transactions,
		},
	}

	if err := w.producer.Send(ctx, msg); err != nil {
		return err
	}

	logger.Info("ETH 区块已同步",
		zap.Int64("block_number", latestBlock),
		zap.Int("tx_count", len(transactions)),
	)

	return nil
}

// fillReceipts 并发获取交易回执，填充 gas_used 和 status
func (w *EthSyncWorker) fillReceipts(ctx context.Context, txs []model.Transaction) {
	if len(txs) == 0 {
		return
	}

	// 限制并发数为 10，避免 RPC 节点限流
	sem := make(chan struct{}, 10)
	var wg sync.WaitGroup

	for i := range txs {
		wg.Add(1)
		sem <- struct{}{} // 获取信号量
		go func(idx int) {
			defer wg.Done()
			defer func() { <-sem }() // 释放信号量

			receipt, err := w.client.GetTransactionReceipt(txs[idx].TxHash)
			if err != nil {
				logger.Debug("获取交易回执失败", zap.String("tx_hash", txs[idx].TxHash), zap.Error(err))
				return
			}

			// 解析 gasUsed
			if gasUsed, ok := receipt["gasUsed"].(string); ok {
				txs[idx].GasUsed = hexToDecimalStr(gasUsed)
			}

			// 解析 status
			if status, ok := receipt["status"].(string); ok {
				val, _ := hexToDecimal(status)
				txs[idx].Status = int16(val)
			}
		}(i)
	}

	wg.Wait()
}

// 十六进制转十进制字符串
func hexToDecimalStr(hex string) string {
	hex = strings.TrimPrefix(hex, "0x")
	if hex == "" {
		return "0"
	}
	val, err := strconv.ParseInt(hex, 16, 64)
	if err != nil {
		return "0"
	}
	return strconv.FormatInt(val, 10)
}

func hexToDecimal(hex string) (int64, error) {
	hex = strings.TrimPrefix(hex, "0x")
	if hex == "" {
		return 0, nil
	}
	return strconv.ParseInt(hex, 16, 64)
}

// hexToDecimalDefault 十六进制转十进制，失败返回默认值
func hexToDecimalDefault(hex string, defaultVal int64) int64 {
	val, err := hexToDecimal(hex)
	if err != nil {
		return defaultVal
	}
	return val
}

func hexToIntDefault(hex string, defaultVal int) int {
	return int(hexToDecimalDefault(hex, int64(defaultVal)))
}

func fmtHexToDecimalStr(hex string) string {
	return fmt.Sprintf("%d", hexToDecimalDefault(hex, 0))
}
