// Package mq 封装 Kafka 消息队列的生产者和消费者
// 生产者：Sync Worker 将从链上拉取的原始区块数据发送到 Kafka
// 消费者：Block Processor 从 Kafka 消费数据，解析后写入数据库
package mq

import (
	"context"
	"encoding/json"
	"fmt"

	"blockexplore/internal/config"
	"blockexplore/pkg/logger"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// Producer Kafka 生产者
// 负责将消息发送到指定的 Kafka Topic
type Producer struct {
	writer *kafka.Writer // Kafka 写入器
	topic  string        // 目标 Topic
}

// BlockMessage 发送到 Kafka 的区块消息格式
// Sync Worker 将区块数据封装为此格式发送
type BlockMessage struct {
	Chain       string      `json:"chain"`        // 链标识: eth/btc/sol
	BlockNumber int64       `json:"block_number"` // 区块高度
	Data        interface{} `json:"data"`          // 原始区块数据
}

// NewProducer 创建 Kafka 生产者实例
// brokers: Kafka Broker 地址列表
// topic: 目标 Topic 名称
func NewProducer(brokers []string, topic string) *Producer {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokers...), // Broker 地址
		Topic:        topic,                  // 目标 Topic
		Balancer:     &kafka.LeastBytes{},   // 负载均衡策略：最少字节
		RequiredAcks: kafka.RequireOne,       // 确认机制：至少一个 Broker 确认
		Async:        false,                  // 同步写入，确保消息不丢失
	}

	logger.Info("Kafka 生产者已创建",
		zap.String("topic", topic),
		zap.Strings("brokers", brokers),
	)

	return &Producer{
		writer: writer,
		topic:  topic,
	}
}

// Send 发送消息到 Kafka
// ctx: 上下文（用于超时控制）
// msg: 要发送的区块消息
func (p *Producer) Send(ctx context.Context, msg BlockMessage) error {
	// 将消息序列化为 JSON
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}

	// 构建 Kafka 消息
	kafkaMsg := kafka.Message{
		Key:   []byte(fmt.Sprintf("%s-%d", msg.Chain, msg.BlockNumber)), // 消息键（用于分区）
		Value: data,                                                      // 消息值
	}

	// 发送消息
	if err := p.writer.WriteMessages(ctx, kafkaMsg); err != nil {
		return fmt.Errorf("发送 Kafka 消息失败: %w", err)
	}

	logger.Debug("消息已发送到 Kafka",
		zap.String("topic", p.topic),
		zap.String("chain", msg.Chain),
		zap.Int64("block_number", msg.BlockNumber),
	)

	return nil
}

// Close 关闭生产者
func (p *Producer) Close() error {
	if p.writer != nil {
		return p.writer.Close()
	}
	return nil
}

// NewETHProducer 创建以太坊区块数据的 Kafka 生产者
func NewETHProducer(cfg config.KafkaConfig) *Producer {
	return NewProducer(cfg.Brokers, cfg.ETHTopic)
}

// NewBTCProducer 创建比特币区块数据的 Kafka 生产者
func NewBTCProducer(cfg config.KafkaConfig) *Producer {
	return NewProducer(cfg.Brokers, cfg.RTCTopic)
}

// NewSOLProducer 创建 Solana 区块数据的 Kafka 生产者
func NewSOLProducer(cfg config.KafkaConfig) *Producer {
	return NewProducer(cfg.Brokers, cfg.SOLTopic)
}
