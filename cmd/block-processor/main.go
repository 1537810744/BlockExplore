// ============================================================
// block-processor 区块处理器入口文件
// ============================================================
// 这是区块处理器的 main 包。
//
// 该服务的职责：
//   1. 从 Kafka 消费原始区块数据（来自三个链的同步 Worker）
//   2. 解析区块和交易数据
//   3. 将解析后的数据写入 PostgreSQL 数据库
//
// 消费者组（Consumer Group）：
//   - 同一个消费者组内的消费者共享消息
//   - 每条消息只会被组内的一个消费者处理
//   - 这样可以实现负载均衡和高可用
//
// Go 语言基础知识:
//   - package main：可执行程序的入口包
//   - func main()：程序启动时自动调用的入口函数
//   - goroutine：Go 的轻量级线程，用 go 关键字启动
//   - channel：Go 的通道，用于 goroutine 之间的通信
//   - select：多路复用，同时等待多个 channel 操作
//   - defer：延迟执行，确保资源被正确释放
// ============================================================
package main

import (
	"context"       // 上下文，用于控制 goroutine 的生命周期
	"os"            // 操作系统相关功能
	"os/signal"     // 系统信号处理
	"syscall"       // 系统调用，定义信号常量

	"blockexplore/internal/config"              // 配置管理
	"blockexplore/internal/mq"                  // Kafka 消息队列
	"blockexplore/internal/repository"          // 数据访问层
	"blockexplore/internal/service/processor"   // 区块处理器
	"blockexplore/pkg/logger"                  // 日志

	"go.uber.org/zap"          // 日志库
	"gorm.io/driver/postgres"  // PostgreSQL 驱动
	"gorm.io/gorm"             // ORM 库
)

// main 函数是 block-processor 的入口点
func main() {
	// ============================================================
	// 第 1 步：加载配置
	// ============================================================
	cfg := config.Load()

	// ============================================================
	// 第 2 步：初始化日志
	// ============================================================
	logger.Init(cfg.Log.Level, cfg.Log.Format)
	logger.Info("block-processor 启动中...")

	// ============================================================
	// 第 3 步：连接数据库
	// ============================================================
	// block-processor 需要将解析后的区块数据写入数据库
	db, err := gorm.Open(postgres.Open(cfg.DB.DSN()), &gorm.Config{})
	if err != nil {
		logger.Fatal("连接数据库失败", zap.Error(err))
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(cfg.DB.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.DB.MaxIdleConns)
	logger.Info("数据库连接成功")

	// ============================================================
	// 第 4 步：创建 Repository 和 Processor
	// ============================================================
	// BlockRepo 负责区块数据的数据库操作
	blockRepo := repository.NewBlockRepo(db)
	// TxRepo 负责交易数据的数据库操作
	txRepo := repository.NewTxRepo(db)
	// BlockProcessor 负责处理 Kafka 消息，解析并保存到数据库
	blockProcessor := processor.NewBlockProcessor(blockRepo, txRepo)

	// ============================================================
	// 第 5 步：创建 Kafka 消费者
	// ============================================================
	// 消费三个链的 Topic：以太坊、比特币、Solana
	// 每个 Topic 对应一个消费者
	topics := []string{
		cfg.Kafka.ETHTopic, // 以太坊区块数据 Topic
		cfg.Kafka.RTCTopic, // 比特币区块数据 Topic
		cfg.Kafka.SOLTopic, // Solana 区块数据 Topic
	}

	// make([]*mq.Consumer, 0, len(topics)) 创建切片，初始长度 0，容量为 len(topics)
	// 切片是 Go 的动态数组，类似于 Python 的 list
	consumers := make([]*mq.Consumer, 0, len(topics))
	for _, topic := range topics {
		// range 遍历切片，返回索引和值
		// _ 表示忽略索引（我们不需要索引）
		consumer := mq.NewConsumer(cfg.Kafka.Brokers, topic, cfg.Kafka.ConsumerGroup)
		consumers = append(consumers, consumer)
	}

	// defer 确保程序退出时关闭所有消费者，释放资源
	// defer 语句会在函数返回前执行
	defer func() {
		for _, c := range consumers {
			c.Close()
		}
	}()

	// ============================================================
	// 第 6 步：创建可取消的上下文
	// ============================================================
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ============================================================
	// 第 7 步：监听系统信号
	// ============================================================
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 启动 goroutine 监听信号
	go func() {
		sig := <-sigChan
		logger.Info("收到关闭信号", zap.String("signal", sig.String()))
		cancel() // 取消上下文，通知所有消费者停止
	}()

	// ============================================================
	// 第 8 步：并发消费所有 Topic
	// ============================================================
	// ConsumeAll 会为每个消费者启动一个 goroutine，并发消费消息
	// 当任意一个消费者出错或 ctx 被取消时，函数返回
	logger.Info("开始消费 Kafka 消息", zap.Strings("topics", topics))
	err = mq.ConsumeAll(ctx, consumers, blockProcessor.Handle)
	if err != nil {
		logger.Fatal("block-processor 异常退出", zap.Error(err))
	}

	logger.Info("block-processor 已停止")
}
