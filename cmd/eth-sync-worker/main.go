// ============================================================
// eth-sync-worker 以太坊同步 Worker 入口文件
// ============================================================
// 这是以太坊区块同步 Worker 的 main 包。
//
// 该 Worker 的职责：
//   1. 定期从以太坊全节点（通过 RPC 接口）拉取最新区块数据
//   2. 将区块和交易数据封装成消息
//   3. 发送到 Kafka 消息队列，供 block-processor 消费处理
//
// 为什么使用 Worker 模式？
//   - 区块链同步是耗时操作，如果在 API 服务中同步会阻塞请求处理
//   - 使用独立的 Worker 进程，可以异步同步，不影响 API 响应速度
//   - 多个 Worker 可以并行运行，提高同步效率
//
// Go 语言基础知识:
//   - goroutine：Go 的轻量级线程，用 go 关键字启动
//   - context：Go 的上下文，用于控制 goroutine 的生命周期
//   - channel：Go 的通道，用于 goroutine 之间的通信
//   - signal.Notify：监听系统信号（如 Ctrl+C），实现优雅关闭
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

// main 函数是 eth-sync-worker 的入口点
func main() {
	// ============================================================
	// 第 1 步：加载配置
	// ============================================================
	cfg := config.Load()

	// ============================================================
	// 第 2 步：初始化日志
	// ============================================================
	logger.Init(cfg.Log.Level, cfg.Log.Format)
	logger.Info("eth-sync-worker 启动中...")

	// ============================================================
	// 第 3 步：创建以太坊 RPC 客户端
	// ============================================================
	// RPC（远程过程调用）是与以太坊节点通信的协议
	// 通过 HTTP 请求调用以太坊节点的方法，如 eth_blockNumber、eth_getBlockByNumber
	ethClient := client.NewEthClient(cfg.ETH.RPCURL)
	logger.Info("以太坊 RPC 客户端已创建", zap.String("rpc_url", cfg.ETH.RPCURL))

	// ============================================================
	// 第 4 步：创建 Kafka 生产者
	// ============================================================
	// Kafka 是分布式消息队列，用于在服务之间传递消息
	// 生产者（Producer）负责发送消息到 Kafka
	// 消费者（Consumer）从 Kafka 读取消息
	// 这里创建的是以太坊区块数据的生产者
	producer := mq.NewETHProducer(cfg.Kafka)
	defer producer.Close() // defer 确保程序退出时关闭生产者，释放资源

	// ============================================================
	// 第 5 步：创建同步 Worker
	// ============================================================
	// EthSyncWorker 负责从以太坊节点拉取区块数据并发送到 Kafka
	worker := sync.NewEthSyncWorker(ethClient, producer, cfg.ETH.SyncInterval)

	// ============================================================
	// 第 6 步：创建可取消的上下文
	// ============================================================
	// context.WithCancel 创建一个可取消的上下文
	// 当调用 cancel() 函数时，所有使用这个上下文的 goroutine 都会收到取消信号
	// 这是 Go 实现优雅关闭的标准模式
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // 确保程序退出时取消上下文

	// ============================================================
	// 第 7 步：监听系统信号，实现优雅关闭
	// ============================================================
	// 当用户按下 Ctrl+C（SIGINT）或系统发送 SIGTERM 信号时，程序应该优雅关闭
	// 优雅关闭：先停止接收新任务，等待正在处理的任务完成，然后退出
	sigChan := make(chan os.Signal, 1)                    // 创建信号通道，缓冲区大小为 1
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM) // 注册要监听的信号

	// 启动一个 goroutine 来监听信号
	// goroutine 是 Go 的轻量级线程，用 go 关键字启动
	go func() {
		sig := <-sigChan // 阻塞等待信号，<- 是从 channel 读取数据的操作符
		logger.Info("收到关闭信号", zap.String("signal", sig.String()))
		cancel() // 取消上下文，通知 Worker 停止
	}()

	// ============================================================
	// 第 8 步：启动同步 Worker（阻塞运行）
	// ============================================================
	// worker.Run(ctx) 会持续运行，直到 ctx 被取消
	// 这是一个阻塞调用，main 函数会一直等待在这里
	if err := worker.Run(ctx); err != nil {
		logger.Fatal("eth-sync-worker 异常退出", zap.Error(err))
	}

	logger.Info("eth-sync-worker 已停止")
}
