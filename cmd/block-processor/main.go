// block-processor 区块处理器入口
// 从 Kafka 消费原始区块数据，解析后写入 PostgreSQL
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"blockexplore/internal/config"
	"blockexplore/internal/mq"
	"blockexplore/internal/repository"
	"blockexplore/internal/service/processor"
	"blockexplore/pkg/logger"

	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// 1. 加载配置
	cfg := config.Load()

	// 2. 初始化日志
	logger.Init(cfg.Log.Level, cfg.Log.Format)
	logger.Info("block-processor 启动中...")

	// 3. 连接数据库
	db, err := gorm.Open(postgres.Open(cfg.DB.DSN()), &gorm.Config{})
	if err != nil {
		logger.Fatal("连接数据库失败", zap.Error(err))
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(cfg.DB.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.DB.MaxIdleConns)
	logger.Info("数据库连接成功")

	// 4. 创建 Repository 和 Processor
	blockRepo := repository.NewBlockRepo(db)
	txRepo := repository.NewTxRepo(db)
	blockProcessor := processor.NewBlockProcessor(blockRepo, txRepo)

	// 5. 创建 Kafka 消费者（消费三个链的 Topic）
	topics := []string{
		cfg.Kafka.ETHTopic,
		cfg.Kafka.RTCTopic,
		cfg.Kafka.SOLTopic,
	}

	consumers := make([]*mq.Consumer, 0, len(topics))
	for _, topic := range topics {
		consumer := mq.NewConsumer(cfg.Kafka.Brokers, topic, cfg.Kafka.ConsumerGroup)
		consumers = append(consumers, consumer)
	}
	// 关闭所有消费者
	defer func() {
		for _, c := range consumers {
			c.Close()
		}
	}()

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

	// 8. 并发消费所有 Topic
	logger.Info("开始消费 Kafka 消息", zap.Strings("topics", topics))
	err = mq.ConsumeAll(ctx, consumers, blockProcessor.Handle)
	if err != nil {
		logger.Fatal("block-processor 异常退出", zap.Error(err))
	}

	logger.Info("block-processor 已停止")
}
