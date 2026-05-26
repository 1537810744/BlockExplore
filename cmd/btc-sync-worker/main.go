// ============================================================
// btc-sync-worker 比特币同步 Worker 入口文件
// ============================================================
// 这是比特币区块同步 Worker 的 main 包。
//
// 该 Worker 的职责：
//   1. 定期从比特币全节点（通过 JSON-RPC 接口）拉取最新区块数据
//   2. 将区块和交易数据封装成消息
//   3. 发送到 Kafka 消息队列，供 block-processor 消费处理
//
// 比特币节点的特点：
//   - 使用 HTTP Basic Auth 认证（用户名/密码）
//   - 区块时间较长（约 10 分钟一个区块）
//   - 使用 UTXO 模型（不同于以太坊的账户模型）
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

// main 函数是 btc-sync-worker 的入口点
func main() {
	// ============================================================
	// 第 1 步：加载配置
	// ============================================================
	cfg := config.Load()

	// ============================================================
	// 第 2 步：初始化日志
	// ============================================================
	logger.Init(cfg.Log.Level, cfg.Log.Format)
	logger.Info("btc-sync-worker 启动中...")

	// ============================================================
	// 第 3 步：创建比特币 RPC 客户端
	// ============================================================
	// 比特币节点需要用户名和密码认证（HTTP Basic Auth）
	// 这与以太坊不同，以太坊通常不需要认证
	btcClient := client.NewBtcClient(cfg.BTC.RPCURL, cfg.BTC.RPCUser, cfg.BTC.RPCPassword)
	logger.Info("比特币 RPC 客户端已创建", zap.String("rpc_url", cfg.BTC.RPCURL))

	// ============================================================
	// 第 4 步：创建 Kafka 生产者
	// ============================================================
	// 生产者负责将区块数据发送到 Kafka 消息队列
	producer := mq.NewBTCProducer(cfg.Kafka)
	defer producer.Close() // defer 确保程序退出时关闭生产者

	// ============================================================
	// 第 5 步：创建同步 Worker
	// ============================================================
	worker := sync.NewBtcSyncWorker(btcClient, producer, cfg.BTC.SyncInterval)

	// ============================================================
	// 第 6 步：创建可取消的上下文
	// ============================================================
	// context.Background() 创建根上下文
	// context.WithCancel 创建可取消的上下文，返回上下文和取消函数
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ============================================================
	// 第 7 步：监听系统信号
	// ============================================================
	// make(chan os.Signal, 1) 创建信号 channel，缓冲区大小为 1
	// channel 是 Go 的通道，用于 goroutine 之间的通信
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// go func() { ... }() 启动 goroutine 监听信号
	go func() {
		sig := <-sigChan // <- 从 channel 读取数据，阻塞直到有信号到来
		logger.Info("收到关闭信号", zap.String("signal", sig.String()))
		cancel() // 取消上下文，通知 Worker 停止
	}()

	// ============================================================
	// 第 8 步：启动同步 Worker
	// ============================================================
	// worker.Run(ctx) 阻塞运行，持续同步区块直到 ctx 被取消
	if err := worker.Run(ctx); err != nil {
		logger.Fatal("btc-sync-worker 异常退出", zap.Error(err))
	}

	logger.Info("btc-sync-worker 已停止")
}
