package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBlock_TableName(t *testing.T) {
	b := Block{}
	assert.Equal(t, "blocks", b.TableName())
}

func TestTransaction_TableName(t *testing.T) {
	tx := Transaction{}
	assert.Equal(t, "transactions", tx.TableName())
}

func TestPriceHistory_TableName(t *testing.T) {
	p := PriceHistory{}
	assert.Equal(t, "price_history", p.TableName())
}

func TestAddress_TableName(t *testing.T) {
	a := Address{}
	assert.Equal(t, "addresses", a.TableName())
}

func TestBlock_Fields(t *testing.T) {
	slot := int64(123)
	b := Block{
		Chain:       "eth",
		BlockNumber: 100,
		BlockHash:   "0xabc",
		TxCount:     5,
		Slot:        &slot,
	}
	assert.Equal(t, "eth", b.Chain)
	assert.Equal(t, int64(100), b.BlockNumber)
	assert.Equal(t, 5, b.TxCount)
	assert.NotNil(t, b.Slot)
	assert.Equal(t, int64(123), *b.Slot)
}

func TestBlock_NilSlot(t *testing.T) {
	// ETH/BTC 没有 slot 概念，应为 nil
	b := Block{Chain: "btc"}
	assert.Nil(t, b.Slot)
}

func TestTransaction_Fields(t *testing.T) {
	nonce := int64(7)
	tx := Transaction{
		Chain:       "eth",
		TxHash:      "0xdeadbeef",
		BlockNumber: 100,
		FromAddr:    "0xfrom",
		ToAddr:      "0xto",
		Value:       "1.5",
		Nonce:       &nonce,
		Status:      1,
	}
	assert.Equal(t, "eth", tx.Chain)
	assert.Equal(t, "0xdeadbeef", tx.TxHash)
	assert.Equal(t, "1.5", tx.Value)
	assert.Equal(t, int16(1), tx.Status)
	assert.Equal(t, int64(7), *tx.Nonce)
}
