// ============================================================
// sol-sync-worker Solana 同步 Worker 入口文件
// ============================================================
// 这是 Solana 区块同步 Worker 的 main 包。
//
// 该 Worker 的职责：
//   1. 定期从 Solana 验证节点（通过 JSON-RPC 接口）拉取最新区块数据
//   2. 将区块和交易数据封装成消息
//   3. 发送到 Kafka 消息队列，供 block-processor 消费处理
//
// Solana 的特点：
//   - 出块速度极快（约 0.4 秒一个区块）
//   - 使用 Slot（槽位号）而非 Block Number
//   - 交易费用极低，吞吐量高
//
// Go 语言基础知识:
//   - package main：可执行程序的入口包
//   - func main()：程序启动时自动调用的入口函数
//   - := 操作符：短变量声明，同时声明变量并赋值
//   - go func() { ... }()：启动 goroutine（轻量级线程）
//   - <-channel：从 channel 读取数据
//   - defer：延迟执行，确保资源被正确释放
// ============================================================
package main

import (
	"context"       // 上下文，用于控制 goroutine 的生命周期
	"os"            // 操作系统相关功能
	"os/signal"     // 系统信号处理
	"syscall"       // 系统调用，定义信号常量

	"blockexplore/internal/client"       // 区块链 RPC 客户端
	"blockexplore/internal/config"       // 配置管理
	"blockexplore/internal/mq"           // Kafka 消息队列
	"blockexplore/internal/service/sync" // 同步服务
	"blockexplore/pkg/logger"           // 日志

	"go.uber.org/zap" // 日志库
)

// main 函数是 sol-sync-worker 的入口点
func main() {
	// ============================================================
	// 第 1 步：加载配置
	// ============================================================
	cfg := config.Load()

	// ============================================================
	// 第 2 步：初始化日志
	// ============================================================
	logger.Init(cfg.Log.Level, cfg.Log.Format)
	logger.Info("sol-sync-worker 启动中...")

	// ============================================================
	// 第 3 步：创建 Solana RPC 客户端
	// ============================================================
	// Solana 节点通常不需要认证（公开 RPC）
	solClient := client.NewSolClient(cfg.SOL.RPCURL)
	logger.Info("Solana RPC 客户端已创建", zap.String("rpc_url", cfg.SOL.RPCURL))

	// ============================================================
	// 第 4 步：创建 Kafka 生产者
	// ============================================================
	producer := mq.NewSOLProducer(cfg.Kafka)
	defer producer.Close()

	// ============================================================
	// 第 5 步：创建同步 Worker
	// ============================================================
	worker := sync.NewSolSyncWorker(solClient, producer, cfg.SOL.SyncInterval)

	// ============================================================
	// 第 6 步：创建可取消的上下文
	// ============================================================
	// context.WithCancel 返回两个值：上下文和取消函数
	// 当调用 cancel() 时，ctx.Done() channel 会被关闭
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ============================================================
	// 第 7 步：监听系统信号
	// ============================================================
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 启动 goroutine 监听系统信号
	go func() {
		sig := <-sigChan
		logger.Info("收到关闭信号", zap.String("signal", sig.String()))
		cancel()
	}()

	// ============================================================
	// 第 8 步：启动同步 Worker
	// ============================================================
	if err := worker.Run(ctx); err != nil {
		logger.Fatal("sol-sync-worker 异常退出", zap.Error(err))
	}

	logger.Info("sol-sync-worker 已停止")
}
