package processor

import (
	"errors"
	"testing"

	"blockexplore/internal/model"
	"blockexplore/internal/mq"

	"github.com/stretchr/testify/assert"
)

type mockBlockWriter struct {
	called     bool
	gotBlock   *model.Block
	failWith   error
	assignID   int64 // 模拟 GORM 自动填充 ID
}

func (m *mockBlockWriter) CreateSingle(block *model.Block) error {
	m.called = true
	m.gotBlock = block
	if m.failWith != nil {
		return m.failWith
	}
	if m.assignID > 0 {
		block.ID = m.assignID
	}
	return nil
}

type mockTxWriter struct {
	called      bool
	gotTxs      []model.Transaction
	failWith    error
}

func (m *mockTxWriter) Create(txs []model.Transaction) error {
	m.called = true
	m.gotTxs = txs
	if m.failWith != nil {
		return m.failWith
	}
	return nil
}

func TestHandle_Success(t *testing.T) {
	bw := &mockBlockWriter{assignID: 42}
	tw := &mockTxWriter{}
	p := NewBlockProcessor(bw, tw)

	msg := mq.BlockMessage{
		Chain:       "eth",
		BlockNumber: 100,
		Data: map[string]interface{}{
			"block": model.Block{
				Chain:       "eth",
				BlockNumber: 100,
				BlockHash:   "0xabc",
			},
			"transactions": []model.Transaction{
				{Chain: "eth", TxHash: "0xtx1"},
				{Chain: "eth", TxHash: "0xtx2"},
			},
		},
	}

	err := p.Handle(msg)
	assert.NoError(t, err)
	assert.True(t, bw.called)
	assert.True(t, tw.called)
	assert.Equal(t, int64(42), bw.gotBlock.ID)
	// 交易应被设置 BlockID = block.ID
	assert.Len(t, tw.gotTxs, 2)
	assert.Equal(t, int64(42), tw.gotTxs[0].BlockID)
	assert.Equal(t, int64(42), tw.gotTxs[1].BlockID)
}

func TestHandle_NoTransactions(t *testing.T) {
	bw := &mockBlockWriter{assignID: 1}
	tw := &mockTxWriter{}
	p := NewBlockProcessor(bw, tw)

	msg := mq.BlockMessage{
		Chain:       "btc",
		BlockNumber: 1,
		Data: map[string]interface{}{
			"block":        model.Block{Chain: "btc", BlockNumber: 1},
			"transactions": []model.Transaction{},
		},
	}

	err := p.Handle(msg)
	assert.NoError(t, err)
	assert.True(t, bw.called)
	// 没有交易时不应调用 tx 写入
	assert.False(t, tw.called)
}

func TestHandle_BlockWriteFails(t *testing.T) {
	bw := &mockBlockWriter{failWith: errors.New("db down")}
	tw := &mockTxWriter{}
	p := NewBlockProcessor(bw, tw)

	msg := mq.BlockMessage{
		Chain:       "eth",
		BlockNumber: 1,
		Data: map[string]interface{}{
			"block":        model.Block{Chain: "eth"},
			"transactions": []model.Transaction{{TxHash: "0x1"}},
		},
	}

	err := p.Handle(msg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "保存区块失败")
	assert.False(t, tw.called, "区块写入失败时不应继续写交易")
}

func TestHandle_TxWriteFails(t *testing.T) {
	bw := &mockBlockWriter{assignID: 5}
	tw := &mockTxWriter{failWith: errors.New("tx insert error")}
	p := NewBlockProcessor(bw, tw)

	msg := mq.BlockMessage{
		Chain:       "eth",
		BlockNumber: 1,
		Data: map[string]interface{}{
			"block":        model.Block{Chain: "eth"},
			"transactions": []model.Transaction{{TxHash: "0x1"}},
		},
	}

	err := p.Handle(msg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "保存交易失败")
}

func TestHandle_InvalidData(t *testing.T) {
	p := NewBlockProcessor(&mockBlockWriter{}, &mockTxWriter{})

	msg := mq.BlockMessage{
		Chain:       "eth",
		BlockNumber: 1,
		Data:        "not a map", // 无法解析为 map[string]interface{}
	}

	err := p.Handle(msg)
	assert.Error(t, err)
}
