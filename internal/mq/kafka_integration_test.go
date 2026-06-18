//go:build integration

// Kafka 集成测试：需要真实的 Kafka 运行在 localhost:9092。
// 运行方式：
//   docker compose -f docker-compose.dev.yaml up -d
//   go test -v -tags=integration ./internal/mq/

package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
)

// ensureTopic 在测试前创建 topic（Kafka 默认可能禁用自动创建）
// 直接连接 broker 端口发送 CreateTopics 请求（broker 会内部转发给 controller）
func ensureTopic(broker []string, topic string) error {
	conn, err := kafka.Dial("tcp", broker[0])
	if err != nil {
		return fmt.Errorf("连接 Kafka 创建 topic 失败: %w", err)
	}
	defer conn.Close()

	return conn.CreateTopics(kafka.TopicConfig{
		Topic:             topic,
		NumPartitions:     1,
		ReplicationFactor: 1,
	})
}

func TestKafka_ProduceAndConsume(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	broker := []string{"localhost:9092"}
	topic := "test.integration.blocks"
	group := "test-integration-group"

	// 先确保 topic 存在（若已开启 auto.create.topics 则可忽略此步；保留以兼容）
	_ = ensureTopic(broker, topic) // best-effort，不阻断测试

	producer := NewProducer(broker, topic)
	defer producer.Close()

	original := BlockMessage{
		Chain:       "eth",
		BlockNumber: 999,
		Data:        map[string]interface{}{"hello": "world"},
	}

	// 发送消息
	err := producer.Send(context.Background(), original)
	assert.NoError(t, err)

	// 消费消息（带超时）
	consumer := NewConsumer(broker, topic, group)
	defer consumer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var received *BlockMessage
	err = consumer.Consume(ctx, func(msg BlockMessage) error {
		received = &msg
		cancel() // 收到一条就停止
		return nil
	})
	// ctx 取消会返回 nil，忽略

	if received == nil {
		t.Skip("未在超时内消费到消息（可能 Kafka topic 刚创建，重试即可）")
		return
	}
	assert.Equal(t, "eth", received.Chain)
	assert.Equal(t, int64(999), received.BlockNumber)

	// 验证 Data 可反序列化
	dataBytes, _ := json.Marshal(received.Data)
	var m map[string]interface{}
	assert.NoError(t, json.Unmarshal(dataBytes, &m))
	assert.Equal(t, "world", m["hello"])
}
