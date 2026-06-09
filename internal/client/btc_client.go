// ============================================================
// BtcClient 比特币 REST 客户端
// ============================================================
// 使用 BlockCypher 公开 API 获取比特币区块数据。
// 无需运行本地比特币节点。
package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"blockexplore/internal/model"
	"blockexplore/pkg/logger"

	"go.uber.org/zap"
)

// BtcClient 比特币 REST 客户端
type BtcClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewBtcClient 创建比特币客户端（参数保持兼容，但不再使用）
func NewBtcClient(rpcURL, rpcUser, rpcPassword string) *BtcClient {
	baseURL := "https://api.blockcypher.com/v1/btc/main"
	logBtcProxyInfo(baseURL)
	return &BtcClient{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// logBtcProxyInfo 打印 BTC 客户端代理配置
func logBtcProxyInfo(apiURL string) {
	httpProxy := os.Getenv("HTTP_PROXY")
	httpsProxy := os.Getenv("HTTPS_PROXY")
	if httpProxy == "" {
		httpProxy = os.Getenv("http_proxy")
	}
	if httpsProxy == "" {
		httpsProxy = os.Getenv("https_proxy")
	}

	proxyStatus := "未使用代理"
	proxyAddr := "无"

	if httpsProxy != "" {
		proxyStatus = "使用代理"
		if u, err := url.Parse(httpsProxy); err == nil {
			proxyAddr = u.Host
		} else {
			proxyAddr = httpsProxy
		}
	} else if httpProxy != "" {
		proxyStatus = "使用代理"
		if u, err := url.Parse(httpProxy); err == nil {
			proxyAddr = u.Host
		} else {
			proxyAddr = httpProxy
		}
	}

	logger.Info("客户端初始化",
		zap.String("链", "BTC"),
		zap.String("代理状态", proxyStatus),
		zap.String("代理地址", proxyAddr),
		zap.String("API地址", apiURL),
	)
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

// blockCypherBlock BlockCypher 区块响应结构
type blockCypherBlock struct {
	Hash         string   `json:"hash"`
	Height       int64    `json:"height"`
	Time         string   `json:"time"`
	PrevBlock    string   `json:"prev_block"`
	Size         int      `json:"size"`
	Weight       int      `json:"weight"`
	TxCount      int      `json:"n_tx"`
	Difficulty    float64  `json:"difficulty"`
	TotalBTCSent float64  `json:"total_btcsent"`
	TxIDs        []string `json:"txids"`
}

// blockCypherTx BlockCypher 交易响应结构
type blockCypherTx struct {
	Hash      string             `json:"hash"`
	Total     int64              `json:"total"`      // 总输出金额（聪）
	Fees      int64              `json:"fees"`       // 手续费（聪）
	Size      int                `json:"size"`
	Confirmed string             `json:"confirmed"`  // 确认时间
	BlockHeight int64            `json:"block_height"`
	Inputs    []blockCypherInput  `json:"inputs"`
	Outputs   []blockCypherOutput `json:"outputs"`
}

type blockCypherInput struct {
	Addresses []string `json:"addresses"`
	PrevHash  string   `json:"prev_hash"`
	OutputIndex int    `json:"output_index"`
}

type blockCypherOutput struct {
	Addresses []string `json:"addresses"`
	Value     int64    `json:"value"` // 单位：聪 (satoshi)
	Spent     bool     `json:"spent"`
}

// GetLatestBlockNumber 获取最新区块高度
func (c *BtcClient) GetLatestBlockNumber() (int64, error) {
	var info struct {
		Height int64 `json:"height"`
	}
	if err := c.get("", &info); err != nil {
		return 0, fmt.Errorf("获取最新区块高度失败: %w", err)
	}
	return info.Height, nil
}

// GetBlockByNumber 根据区块高度获取区块详情和交易
func (c *BtcClient) GetBlockByNumber(blockNumber int64) (*model.Block, []model.Transaction, error) {
	// 获取区块详情（包含前 20 笔交易哈希）
	var bBlock blockCypherBlock
	if err := c.get(fmt.Sprintf("/blocks/%d?txstart=0&limit=20", blockNumber), &bBlock); err != nil {
		return nil, nil, fmt.Errorf("获取区块详情失败: %w", err)
	}

	// 解析时间
	blockTime := parseBlockCypherTime(bBlock.Time)

	block := &model.Block{
		Chain:       "btc",
		BlockNumber: blockNumber,
		BlockHash:   bBlock.Hash,
		ParentHash:  bBlock.PrevBlock,
		Timestamp:   blockTime,
		TxCount:     bBlock.TxCount,
		SizeBytes:   bBlock.Size,
		Difficulty:  fmt.Sprintf("%.0f", bBlock.Difficulty),
	}

	// 获取交易详情（最多 20 笔）
	limit := len(bBlock.TxIDs)
	if limit > 20 {
		limit = 20
	}

	transactions := make([]model.Transaction, 0, limit)
	for i := 0; i < limit; i++ {
		tx, err := c.getTransactionDetail(bBlock.TxIDs[i], blockNumber, blockTime)
		if err != nil {
			// 记录错误但继续处理
			fmt.Printf("获取交易 %s 失败: %v\n", bBlock.TxIDs[i], err)
			continue
		}
		transactions = append(transactions, *tx)
	}

	return block, transactions, nil
}

// getTransactionDetail 获取交易详情
func (c *BtcClient) getTransactionDetail(txHash string, blockNumber int64, blockTime int64) (*model.Transaction, error) {
	var tx blockCypherTx
	if err := c.get(fmt.Sprintf("/txs/%s", txHash), &tx); err != nil {
		return nil, err
	}

	// 提取发送方地址（从第一个输入获取）
	fromAddr := ""
	if len(tx.Inputs) > 0 && len(tx.Inputs[0].Addresses) > 0 {
		fromAddr = tx.Inputs[0].Addresses[0]
	}

	// 提取接收方地址和金额（从第一个输出获取）
	toAddr := ""
	var value int64
	if len(tx.Outputs) > 0 {
		if len(tx.Outputs[0].Addresses) > 0 {
			toAddr = tx.Outputs[0].Addresses[0]
		}
		value = tx.Outputs[0].Value
	}

	return &model.Transaction{
		Chain:       "btc",
		TxHash:      tx.Hash,
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

// parseBlockCypherTime 解析 BlockCypher 时间格式
// 格式: "2024-01-15T10:30:00.000Z"
func parseBlockCypherTime(timeStr string) int64 {
	if timeStr == "" {
		return 0
	}
	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return 0
	}
	return t.Unix()
}
