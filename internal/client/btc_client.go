// ============================================================
// BtcClient 比特币 REST 客户端
// ============================================================
// 使用 Mempool.space 公开 API 获取比特币区块数据。
// 无需运行本地比特币节点。
package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"blockexplore/internal/model"
)

// BtcClient 比特币 REST 客户端
type BtcClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewBtcClient 创建比特币客户端（参数保持兼容，但不再使用）
func NewBtcClient(rpcURL, rpcUser, rpcPassword string) *BtcClient {
	return &BtcClient{
		baseURL:    "https://mempool.space/api",
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// get 发送 GET 请求并解析 JSON
func (c *BtcClient) get(path string, target interface{}) error {
	resp, err := c.httpClient.Get(c.baseURL + path)
	if err != nil {
		return fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(target)
}

// mempoolBlock Mempool.space 区块响应结构
type mempoolBlock struct {
	ID           string `json:"id"`
	Height       int64  `json:"height"`
	Timestamp    int64  `json:"timestamp"`
	PreviousHash string `json:"previousblockhash"`
	Size         int    `json:"size"`
	Weight       int    `json:"weight"`
	TxCount      int    `json:"tx_count"`
	Difficulty    int64  `json:"difficulty"`
}

// GetLatestBlockNumber 获取最新区块高度
func (c *BtcClient) GetLatestBlockNumber() (int64, error) {
	var height int64
	resp, err := c.httpClient.Get(c.baseURL + "/blocks/tip/height")
	if err != nil {
		return 0, fmt.Errorf("获取最新区块高度失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	height, err = strconv.ParseInt(string(body), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("解析区块高度失败: %w", err)
	}

	return height, nil
}

// GetBlockByNumber 根据区块高度获取区块详情和交易
func (c *BtcClient) GetBlockByNumber(blockNumber int64) (*model.Block, []model.Transaction, error) {
	// 第 1 步：通过高度获取区块哈希
	var blockHash string
	resp, err := c.httpClient.Get(fmt.Sprintf("%s/block-height/%d", c.baseURL, blockNumber))
	if err != nil {
		return nil, nil, fmt.Errorf("获取区块哈希失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}
	blockHash = string(body)

	// 第 2 步：获取区块详情
	var mBlock mempoolBlock
	if err := c.get(fmt.Sprintf("/block/%s", blockHash), &mBlock); err != nil {
		return nil, nil, fmt.Errorf("获取区块详情失败: %w", err)
	}

	block := &model.Block{
		Chain:       "btc",
		BlockNumber: blockNumber,
		BlockHash:   mBlock.ID,
		ParentHash:  mBlock.PreviousHash,
		Timestamp:   mBlock.Timestamp,
		TxCount:     mBlock.TxCount,
		SizeBytes:   mBlock.Size,
		Difficulty:  fmt.Sprintf("%d", mBlock.Difficulty),
	}

	// 第 3 步：获取区块内的交易 ID 列表
	var txIDs []string
	if err := c.get(fmt.Sprintf("/block/%s/txids", blockHash), &txIDs); err != nil {
		return nil, nil, fmt.Errorf("获取交易列表失败: %w", err)
	}

	// 只取前 20 笔交易的详情（避免请求过多）
	limit := len(txIDs)
	if limit > 20 {
		limit = 20
	}

	transactions := make([]model.Transaction, 0, limit)
	for i := 0; i < limit; i++ {
		tx, err := c.getTransactionDetail(txIDs[i], blockNumber, mBlock.Timestamp)
		if err != nil {
			continue
		}
		transactions = append(transactions, *tx)
	}

	return block, transactions, nil
}

// mempoolTx Mempool.space 交易响应结构
type mempoolTx struct {
	TxID     string          `json:"txid"`
	Version  int             `json:"version"`
	Size     int             `json:"size"`
	Weight   int             `json:"weight"`
	Fee      int64           `json:"fee"`
	Vin      []mempoolVin    `json:"vin"`
	Vout     []mempoolVout   `json:"vout"`
	Status   mempoolTxStatus `json:"status"`
}

type mempoolVin struct {
	TxID      string `json:"txid"`
	Vout      int    `json:"vout"`
	Prevout   *mempoolVout `json:"prevout"`
	Sequence  int64  `json:"sequence"`
}

type mempoolVout struct {
	ScriptPubKey struct {
		Asm       string `json:"asm"`
		Hex       string `json:"hex"`
		Type      string `json:"type"`
		Address   string `json:"address"`
	} `json:"scriptpubkey"`
	Value int64 `json:"value"` // 单位：聪 (satoshi)
}

type mempoolTxStatus struct {
	Confirmed   bool   `json:"confirmed"`
	BlockHeight int64  `json:"block_height"`
	BlockTime   int64  `json:"block_time"`
}

// getTransactionDetail 获取交易详情
func (c *BtcClient) getTransactionDetail(txID string, blockNumber int64, blockTime int64) (*model.Transaction, error) {
	var tx mempoolTx
	if err := c.get(fmt.Sprintf("/tx/%s", txID), &tx); err != nil {
		return nil, err
	}

	// 提取发送方地址（从第一个输入的 prevout 获取）
	fromAddr := ""
	if len(tx.Vin) > 0 {
		if tx.Vin[0].Prevout != nil && tx.Vin[0].Prevout.ScriptPubKey.Address != "" {
			fromAddr = tx.Vin[0].Prevout.ScriptPubKey.Address
		} else if tx.Vin[0].TxID == "" {
			fromAddr = "coinbase"
		}
	}

	// 提取接收方地址和金额（从第一个输出获取）
	toAddr := ""
	var value int64
	if len(tx.Vout) > 0 {
		toAddr = tx.Vout[0].ScriptPubKey.Address
		value = tx.Vout[0].Value
	}

	return &model.Transaction{
		Chain:       "btc",
		TxHash:      tx.TxID,
		BlockNumber: blockNumber,
		FromAddr:    fromAddr,
		ToAddr:      toAddr,
		Value:       fmt.Sprintf("%.8f", float64(value)/1e8), // 转换为 BTC
		GasUsed:     "0",
		GasPrice:    "0",
		Timestamp:   blockTime,
		Status:      1,
	}, nil
}
