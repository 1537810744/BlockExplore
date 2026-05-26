// eth-sync-worker 以太坊同步 Worker 入口
// 从以太坊全节点拉取区块/交易数据，发送到 Kafka
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"blockexplore/internal/client"
	"blockexplore/internal/config"
	"blockexplore/internal/mq"
	"blockexplore/internal/service/sync"
	"blockexplore/pkg/logger"

	"go.uber.org/zap"
)

func main() {
	// 1. 加载配置
	cfg := config.Load()

	// 2. 初始化日志
	logger.Init(cfg.Log.Level, cfg.Log.Format)
	logger.Info("eth-sync-worker 启动中...")

	// 3. 创建以太坊 RPC 客户端
	ethClient := client.NewEthClient(cfg.ETH.RPCURL)
	logger.Info("以太坊 RPC 客户端已创建", zap.String("rpc_url", cfg.ETH.RPCURL))

	// 4. 创建 Kafka 生产者
	producer := mq.NewETHProducer(cfg.Kafka)
	defer producer.Close()

	// 5. 创建同步 Worker
	worker := sync.NewEthSyncWorker(ethClient, producer, cfg.ETH.SyncInterval)

	// 6. 创建可取消的上下文（用于优雅关闭）
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 7. 监听系统信号，实现优雅关闭
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		logger.Info("收到关闭信号", zap.String("signal", sig.String()))
		cancel() // 取消上下文，通知 Worker 停止
	}()

	// 8. 启动同步 Worker（阻塞运行）
	if err := worker.Run(ctx); err != nil {
		logger.Fatal("eth-sync-worker 异常退出", zap.Error(err))
	}

	logger.Info("eth-sync-worker 已停止")
}
