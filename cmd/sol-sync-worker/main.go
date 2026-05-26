// sol-sync-worker Solana 同步 Worker 入口
// 从 Solana RPC 拉取区块/交易数据，发送到 Kafka
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
	logger.Info("sol-sync-worker 启动中...")

	// 3. 创建 Solana RPC 客户端
	solClient := client.NewSolClient(cfg.SOL.RPCURL)
	logger.Info("Solana RPC 客户端已创建", zap.String("rpc_url", cfg.SOL.RPCURL))

	// 4. 创建 Kafka 生产者
	producer := mq.NewSOLProducer(cfg.Kafka)
	defer producer.Close()

	// 5. 创建同步 Worker
	worker := sync.NewSolSyncWorker(solClient, producer, cfg.SOL.SyncInterval)

	// 6. 创建可取消的上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 7. 监听系统信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		logger.Info("收到关闭信号", zap.String("signal", sig.String()))
		cancel()
	}()

	// 8. 启动同步 Worker
	if err := worker.Run(ctx); err != nil {
		logger.Fatal("sol-sync-worker 异常退出", zap.Error(err))
	}

	logger.Info("sol-sync-worker 已停止")
}
