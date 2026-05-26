// ============================================================
// Package mq 封装 Kafka 消息队列的生产者和消费者
// ============================================================
// 该文件实现了 Kafka 消费者。
//
// 消费者的工作流程：
//   1. 连接到 Kafka Broker
//   2. 订阅指定的 Topic
//   3. 循环读取消息
//   4. 将消息反序列化为 BlockMessage
//   5. 调用业务处理函数处理消息
//
// 消费者组（Consumer Group）：
//   - 同一个消费者组内的消费者共享消息
//   - 每条消息只会被组内的一个消费者处理
//   - 这样可以实现负载均衡和高可用
//
// Go 语言基础知识:
//   - goroutine：Go 的轻量级线程，用 go 关键字启动
//   - channel：Go 的通道，用于 goroutine 之间的通信
//   - select：多路复用，同时等待多个 channel 操作
//   - for { select { ... } }：Go 的经典事件循环模式
//   - ctx.Done()：返回一个 channel，当上下文被取消时会关闭
//   - func 类型：Go 中函数也是类型，可以作为参数传递
// ============================================================
package mq

import (
	"context"       // 上下文，用于控制 goroutine 的生命周期
	"encoding/json" // JSON 编解码
	"fmt"           // 格式化字符串

	"blockexplore/pkg/logger" // 日志

	"github.com/segmentio/kafka-go" // Kafka Go 客户端库
	"go.uber.org/zap"              // 日志库
)

// ============================================================
// Consumer Kafka 消费者
// ============================================================
// 负责从 Kafka Topic 消费消息并交给业务逻辑处理
type Consumer struct {
	reader *kafka.Reader // Kafka 读取器，用于读取消息
	topic  string        // 消费的 Topic 名称
}

// ============================================================
// MessageHandler 消息处理函数类型
// ============================================================
// 这是一个函数类型定义，类似于 Java 的接口
// 消费者收到消息后会调用此函数进行业务处理
// 函数参数是 BlockMessage，返回 error
type MessageHandler func(msg BlockMessage) error

// ============================================================
// NewConsumer 创建 Kafka 消费者实例
// ============================================================
// 参数 brokers：Kafka Broker 地址列表
// 参数 topic：消费的 Topic 名称
// 参数 group：消费者组名称（同一组内的消费者共享消息）
func NewConsumer(brokers []string, topic string, group string) *Consumer {
	// kafka.NewReader 创建 Kafka 读取器
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,       // Broker 地址列表
		Topic:    topic,         // 消费的 Topic
		GroupID:  group,         // 消费者组 ID
		MinBytes: 1,             // 最小拉取字节数（1 字节，尽快返回）
		MaxBytes: 10e6,          // 最大拉取字节数（10MB）
	})

	logger.Info("Kafka 消费者已创建",
		zap.String("topic", topic),
		zap.String("group", group),
		zap.Strings("brokers", brokers),
	)

	return &Consumer{
		reader: reader,
		topic:  topic,
	}
}

// ============================================================
// Consume 方法：开始消费消息
// ============================================================
// 参数 ctx：上下文（用于优雅关闭）
// 参数 handler：消息处理函数
// 此方法会阻塞，直到 ctx 被取消
func (c *Consumer) Consume(ctx context.Context, handler MessageHandler) error {
	logger.Info("开始消费 Kafka 消息", zap.String("topic", c.topic))

	// 无限循环，持续消费消息
	for {
		// select 语句用于多路复用，同时等待多个 channel 操作
		// 类似于 switch，但每个 case 都是 channel 操作
		select {
		case <-ctx.Done():
			// ctx.Done() 返回一个 channel
			// 当 ctx 被取消时，这个 channel 会被关闭
			// 从已关闭的 channel 读取会立即返回零值
			logger.Info("停止消费 Kafka 消息", zap.String("topic", c.topic))
			return nil
		default:
			// default 分支：如果没有其他 case 就绪，立即执行 default
			// 这样不会阻塞在 select 上，而是继续读取消息

			// 读取消息（阻塞直到有新消息或 ctx 被取消）
			kafkaMsg, err := c.reader.ReadMessage(ctx)
			if err != nil {
				// 检查是否是上下文取消导致的错误
				if ctx.Err() != nil {
					return nil // 正常关闭，返回 nil
				}
				logger.Error("读取 Kafka 消息失败", zap.String("topic", c.topic), zap.Error(err))
				continue // 读取失败，跳过这条消息，继续读取下一条
			}

			// 反序列化消息
			// kafkaMsg.Value 是消息的原始字节，需要反序列化为 BlockMessage
			var msg BlockMessage
			if err := json.Unmarshal(kafkaMsg.Value, &msg); err != nil {
				logger.Error("解析 Kafka 消息失败",
					zap.String("topic", c.topic),
					zap.ByteString("value", kafkaMsg.Value), // 记录原始消息内容，方便调试
					zap.Error(err),
				)
				continue // 解析失败，跳过这条消息
			}

			// 调用业务处理函数
			// handler 是外部传入的处理函数，实现了具体的业务逻辑
			if err := handler(msg); err != nil {
				logger.Error("处理消息失败",
					zap.String("topic", c.topic),
					zap.String("chain", msg.Chain),
					zap.Int64("block_number", msg.BlockNumber),
					zap.Error(err),
				)
				// 处理失败可以选择重试或跳过，这里选择继续
				continue
			}

			logger.Debug("消息处理成功",
				zap.String("topic", c.topic),
				zap.String("chain", msg.Chain),
				zap.Int64("block_number", msg.BlockNumber),
			)
		}
	}
}

// ============================================================
// Close 方法：关闭消费者
// ============================================================
func (c *Consumer) Close() error {
	if c.reader != nil {
		return c.reader.Close()
	}
	return nil
}

// ============================================================
// NewBlockConsumer 创建区块数据消费者
// ============================================================
// 消费所有链的区块数据 Topic
func NewBlockConsumer(cfg struct {
	Brokers       []string
	ConsumerGroup string
}, topics ...string) []*Consumer {
	consumers := make([]*Consumer, 0, len(topics))
	for _, topic := range topics {
		consumer := NewConsumer(cfg.Brokers, topic, cfg.ConsumerGroup)
		consumers = append(consumers, consumer)
	}
	return consumers
}

// ============================================================
// ConsumeAll 并发消费多个 Topic
// ============================================================
// 参数 ctx：上下文
// 参数 consumers：消费者列表
// 参数 handler：统一的消息处理函数
// 使用 goroutine 并发消费，任意一个出错则返回错误
func ConsumeAll(ctx context.Context, consumers []*Consumer, handler MessageHandler) error {
	// errChan 是一个带缓冲的 channel，用于接收错误
	// 缓冲区大小为消费者数量，防止 goroutine 阻塞
	errChan := make(chan error, len(consumers))

	// 为每个消费者启动一个 goroutine
	for _, consumer := range consumers {
		// go func(c *Consumer) { ... }(consumer) 启动 goroutine
		// 注意：这里传入 consumer 参数，避免闭包捕获循环变量的问题
		go func(c *Consumer) {
			if err := c.Consume(ctx, handler); err != nil {
				errChan <- fmt.Errorf("消费者 %s 出错: %w", c.topic, err)
			}
		}(consumer)
	}

	// 等待第一个错误或上下文取消
	// select 会阻塞，直到某个 case 就绪
	select {
	case err := <-errChan:
		// 从 errChan 读取到错误
		return err
	case <-ctx.Done():
		// 上下文被取消（正常关闭）
		return nil
	}
}
