package mq

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBlockMessage_Serialization(t *testing.T) {
	orig := BlockMessage{
		Chain:       "eth",
		BlockNumber: 123456,
		Data: map[string]interface{}{
			"block":        "0xabc",
			"transactions": []string{"0xtx1", "0xtx2"},
		},
	}

	data, err := json.Marshal(orig)
	assert.NoError(t, err)
	assert.NotEmpty(t, data)

	var decoded BlockMessage
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, "eth", decoded.Chain)
	assert.Equal(t, int64(123456), decoded.BlockNumber)
}

func TestBlockMessage_EmptyData(t *testing.T) {
	msg := BlockMessage{Chain: "btc", BlockNumber: 1}
	data, err := json.Marshal(msg)
	assert.NoError(t, err)

	var decoded BlockMessage
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, "btc", decoded.Chain)
	assert.Nil(t, decoded.Data)
}

func TestNewProducer_DoesNotConnect(t *testing.T) {
	// NewProducer 仅创建 writer，不立即连接，所以可以用任意地址
	p := NewProducer([]string{"localhost:9999"}, "test-topic")
	assert.NotNil(t, p)
	assert.Equal(t, "test-topic", p.topic)
	// Close 一个未使用的 writer 不应 panic
	assert.NoError(t, p.Close())
}
