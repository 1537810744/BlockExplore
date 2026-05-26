package mq

import (
	"context"
	"encoding/json"
	"fmt"

	"blockexplore/pkg/logger"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// Consumer Kafka 消费者
// 负责从 Kafka Topic 消费消息并交给业务逻辑处理
type Consumer struct {
	reader *kafka.Reader // Kafka 读取器
	topic  string        // 消费的 Topic
}

// MessageHandler 消息处理函数类型
// 消费者收到消息后会调用此函数进行业务处理
type MessageHandler func(msg BlockMessage) error

// NewConsumer 创建 Kafka 消费者实例
// brokers: Kafka Broker 地址列表
// topic: 消费的 Topic 名称
// group: 消费者组名称（同一组内的消费者共享消息）
func NewConsumer(brokers []string, topic string, group string) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,       // Broker 地址列表
		Topic:    topic,         // 消费的 Topic
		GroupID:  group,         // 消费者组 ID
		MinBytes: 1,             // 最小拉取字节数
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

// Consume 开始消费消息
// ctx: 上下文（用于优雅关闭）
// handler: 消息处理函数
// 此方法会阻塞，直到 ctx 被取消
func (c *Consumer) Consume(ctx context.Context, handler MessageHandler) error {
	logger.Info("开始消费 Kafka 消息", zap.String("topic", c.topic))

	for {
		select {
		case <-ctx.Done():
			// 上下文取消，停止消费
			logger.Info("停止消费 Kafka 消息", zap.String("topic", c.topic))
			return nil
		default:
			// 读取消息
			kafkaMsg, err := c.reader.ReadMessage(ctx)
			if err != nil {
				// 检查是否是上下文取消导致的错误
				if ctx.Err() != nil {
					return nil
				}
				logger.Error("读取 Kafka 消息失败", zap.String("topic", c.topic), zap.Error(err))
				continue
			}

			// 反序列化消息
			var msg BlockMessage
			if err := json.Unmarshal(kafkaMsg.Value, &msg); err != nil {
				logger.Error("解析 Kafka 消息失败",
					zap.String("topic", c.topic),
					zap.ByteString("value", kafkaMsg.Value),
					zap.Error(err),
				)
				continue
			}

			// 调用业务处理函数
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

// Close 关闭消费者
func (c *Consumer) Close() error {
	if c.reader != nil {
		return c.reader.Close()
	}
	return nil
}

// NewBlockConsumer 创建区块数据消费者
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

// ConsumeAll 并发消费多个 Topic
// ctx: 上下文
// consumers: 消费者列表
// handler: 统一的消息处理函数
func ConsumeAll(ctx context.Context, consumers []*Consumer, handler MessageHandler) error {
	errChan := make(chan error, len(consumers))

	for _, consumer := range consumers {
		go func(c *Consumer) {
			if err := c.Consume(ctx, handler); err != nil {
				errChan <- fmt.Errorf("消费者 %s 出错: %w", c.topic, err)
			}
		}(consumer)
	}

	// 等待第一个错误或上下文取消
	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		return nil
	}
}
