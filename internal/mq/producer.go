// ============================================================
// Package mq 封装 Kafka 消息队列的生产者和消费者
// ============================================================
// 该包提供了 Kafka 消息队列的操作封装。
//
// Kafka 是什么？
//   - Apache Kafka 是一个分布式流处理平台
//   - 用于在不同服务之间传递消息
//   - 类似于一个"消息管道"，生产者往里写消息，消费者从里面读消息
//
// 本项目中 Kafka 的作用：
//   - 生产者（Sync Worker）：从区块链节点拉取区块数据，发送到 Kafka
//   - 消费者（Block Processor）：从 Kafka 读取区块数据，解析后写入数据库
//
// 为什么使用 Kafka？
//   - 解耦：Sync Worker 和 Block Processor 互不依赖
//   - 削峰：区块数据量大时，Kafka 可以缓冲消息，防止数据库压力过大
//   - 可靠：消息持久化，即使消费者挂了也不会丢失
//
// Go 语言基础知识:
//   - package：包，Go 的模块化机制
//   - struct：结构体，用于定义数据结构
//   - context.Context：上下文，用于控制超时和取消
//   - interface{}：空接口，可以持有任意类型的值
//   - json.Marshal：将结构体序列化为 JSON 字节
//   - fmt.Errorf：格式化创建错误，%w 包装原始错误
//   - defer：延迟执行，确保资源被正确释放
// ============================================================
package mq

import (
	"context"       // 上下文，用于超时控制
	"encoding/json" // JSON 编解码
	"fmt"           // 格式化字符串

	"blockexplore/internal/config" // 配置管理
	"blockexplore/pkg/logger"     // 日志

	"github.com/segmentio/kafka-go" // Kafka Go 客户端库
	"go.uber.org/zap"              // 日志库
)

// ============================================================
// Producer Kafka 生产者
// ============================================================
// 负责将消息发送到指定的 Kafka Topic
// Topic 是 Kafka 中消息的分类，类似于邮件的收件箱
type Producer struct {
	writer *kafka.Writer // Kafka 写入器，用于发送消息
	topic  string        // 目标 Topic 名称
}

// ============================================================
// BlockMessage 发送到 Kafka 的区块消息格式
// ============================================================
// Sync Worker 将区块数据封装为此格式发送到 Kafka
// Block Processor 从 Kafka 读取消息后，反序列化为这个格式
type BlockMessage struct {
	Chain       string      `json:"chain"`        // 链标识: eth/btc/sol
	BlockNumber int64       `json:"block_number"` // 区块高度
	Data        interface{} `json:"data"`          // 原始区块数据（使用空接口，可以存放任意类型）
}

// ============================================================
// NewProducer 创建 Kafka 生产者实例
// ============================================================
// 参数 brokers：Kafka Broker 地址列表，例如 ["kafka:9092"]
// 参数 topic：目标 Topic 名称，例如 "block.raw.eth"
func NewProducer(brokers []string, topic string) *Producer {
	// kafka.NewWriter 创建 Kafka 写入器
	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokers...), // Broker 地址，... 展开切片
		Topic:        topic,                  // 目标 Topic
		Balancer:     &kafka.LeastBytes{},   // 负载均衡策略：选择字节数最少的分区
		RequiredAcks: kafka.RequireOne,       // 确认机制：至少一个 Broker 确认收到
		Async:        false,                  // 同步写入，确保消息不丢失（异步写入更快但可能丢消息）
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

// ============================================================
// Send 方法：发送消息到 Kafka
// ============================================================
// 参数 ctx：上下文，用于超时控制
// 参数 msg：要发送的区块消息
func (p *Producer) Send(ctx context.Context, msg BlockMessage) error {
	// 将消息序列化为 JSON 字节
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}

	// 构建 Kafka 消息
	// Key 用于决定消息发送到哪个分区（相同 Key 的消息会发送到同一分区）
	// 这里使用 "链标识-区块高度" 作为 Key
	kafkaMsg := kafka.Message{
		Key:   []byte(fmt.Sprintf("%s-%d", msg.Chain, msg.BlockNumber)),
		Value: data,
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

// ============================================================
// Close 方法：关闭生产者
// ============================================================
// 释放资源，关闭与 Kafka Broker 的连接
func (p *Producer) Close() error {
	if p.writer != nil {
		return p.writer.Close()
	}
	return nil
}

// ============================================================
// 便捷创建函数
// ============================================================

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
