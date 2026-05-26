package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"blockexplore/internal/model"
)

// BtcClient 比特币 RPC 客户端
// 通过 JSON-RPC 协议与比特币全节点通信
// 比特币节点使用 HTTP Basic Auth 认证
type BtcClient struct {
	rpcURL     string       // RPC 节点地址
	rpcUser    string       // RPC 用户名
	rpcPassword string      // RPC 密码
	httpClient *http.Client // HTTP 客户端
}

// NewBtcClient 创建比特币 RPC 客户端实例
func NewBtcClient(rpcURL, rpcUser, rpcPassword string) *BtcClient {
	return &BtcClient{
		rpcURL:      rpcURL,
		rpcUser:     rpcUser,
		rpcPassword: rpcPassword,
		httpClient: &http.Client{
			Timeout: 60 * time.Second, // 比特币同步间隔较长，超时设大些
		},
	}
}

// call 发送比特币 JSON-RPC 请求
// 比特币节点需要 HTTP Basic Auth 认证
func (c *BtcClient) call(method string, params ...interface{}) (json.RawMessage, error) {
	reqBody := jsonRPCRequest{
		JsonRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      1,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	// 创建 HTTP 请求
	req, err := http.NewRequest("POST", c.rpcURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置 Basic Auth 认证
	req.SetBasicAuth(c.rpcUser, c.rpcPassword)
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送 RPC 请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	var rpcResp jsonRPCResponse
	if err := json.Unmarshal(respBytes, &rpcResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if rpcResp.Error != nil {
		return nil, rpcResp.Error
	}

	return rpcResp.Result, nil
}

// btcBlock 比特币区块的 JSON 结构
type btcBlock struct {
	Hash              string   `json:"hash"`              // 区块哈希
	Height            int64    `json:"height"`             // 区块高度
	PreviousHash      string   `json:"previousblockhash"`  // 父区块哈希
	Time              int64    `json:"time"`               // 出块时间
	Tx                []string `json:"tx"`                 // 交易哈希列表（简化模式）
	Nonce             uint64   `json:"nonce"`              // 随机数
	Size              int      `json:"size"`               // 区块大小（字节）
	Weight            int      `json:"weight"`             // 区块权重
	Difficulty        float64  `json:"difficulty"`         // 难度值
	Confirmations     int      `json:"confirmations"`      // 确认数
}

// btcTransaction 比特币交易的 JSON 结构
type btcTransaction struct {
	TxID     string `json:"txid"`     // 交易 ID
	Hash     string `json:"hash"`     // 交易哈希
	Size     int    `json:"size"`     // 交易大小
	Vsize    int    `json:"vsize"`    // 虚拟大小
	Version  int    `json:"version"`  // 版本号
	LockTime int64  `json:"locktime"` // 锁定时间
	Vin      []btcVin   `json:"vin"`  // 输入列表
	Vout     []btcVout  `json:"vout"` // 输出列表
	BlockHash   string `json:"blockhash"`   // 所在区块哈希
	BlockHeight int64  `json:"blockheight"` // 所在区块高度
	Time        int64  `json:"time"`        // 交易时间
	BlockTime   int64  `json:"blocktime"`   // 区块时间
}

// btcVin 比特币交易输入
type btcVin struct {
	TxID      string `json:"txid"`      // 引用的交易 ID
	Vout      int    `json:"vout"`      // 引用的输出索引
	ScriptSig struct {
		Asm string `json:"asm"` // 脚本汇编
		Hex string `json:"hex"` // 脚本十六进制
	} `json:"scriptSig"`
	Sequence int64 `json:"sequence"` // 序列号
}

// btcVout 比特币交易输出
type btcVout struct {
	Value        float64 `json:"value"`        // 输出金额（BTC）
	N            int     `json:"n"`            // 输出索引
	ScriptPubKey struct {
		Asm       string   `json:"asm"`       // 脚本汇编
		Hex       string   `json:"hex"`       // 脚本十六进制
		ReqSigs   int      `json:"reqSigs"`   // 需要的签名数
		Type      string   `json:"type"`      // 脚本类型
		Addresses []string `json:"addresses"` // 地址列表
	} `json:"scriptPubKey"`
}

// GetLatestBlockNumber 获取最新区块高度
func (c *BtcClient) GetLatestBlockNumber() (int64, error) {
	result, err := c.call("getblockcount")
	if err != nil {
		return 0, fmt.Errorf("获取最新区块高度失败: %w", err)
	}

	var height int64
	if err := json.Unmarshal(result, &height); err != nil {
		return 0, fmt.Errorf("解析区块高度失败: %w", err)
	}

	return height, nil
}

// GetBlockByNumber 根据区块高度获取区块详情
func (c *BtcClient) GetBlockByNumber(blockNumber int64) (*model.Block, []model.Transaction, error) {
	// 先获取区块哈希
	hashResult, err := c.call("getblockhash", blockNumber)
	if err != nil {
		return nil, nil, fmt.Errorf("获取区块哈希失败: %w", err)
	}

	var blockHash string
	if err := json.Unmarshal(hashResult, &blockHash); err != nil {
		return nil, nil, fmt.Errorf("解析区块哈希失败: %w", err)
	}

	// 通过区块哈希获取区块详情（verbosity=2 返回完整交易）
	result, err := c.call("getblock", blockHash, 2)
	if err != nil {
		return nil, nil, fmt.Errorf("获取区块详情失败: %w", err)
	}

	var btcBlock btcBlock
	if err := json.Unmarshal(result, &btcBlock); err != nil {
		return nil, nil, fmt.Errorf("解析区块数据失败: %w", err)
	}

	// 转换为我们的区块模型
	block := &model.Block{
		Chain:       "btc",
		BlockNumber: blockNumber,
		BlockHash:   btcBlock.Hash,
		ParentHash:  btcBlock.PreviousHash,
		Timestamp:   btcBlock.Time,
		TxCount:     len(btcBlock.Tx),
		SizeBytes:   btcBlock.Size,
		Difficulty:  fmt.Sprintf("%.0f", btcBlock.Difficulty),
	}

	// 获取区块内的交易详情
	transactions := make([]model.Transaction, 0, len(btcBlock.Tx))
	for _, txHash := range btcBlock.Tx {
		txResult, err := c.call("getrawtransaction", txHash, true)
		if err != nil {
			continue // 跳过获取失败的交易
		}

		var btcTx btcTransaction
		if err := json.Unmarshal(txResult, &btcTx); err != nil {
			continue
		}

		// 提取发送方地址（从第一个输入的前序交易获取）
		fromAddr := ""
		if len(btcTx.Vin) > 0 && btcTx.Vin[0].TxID != "" {
			// 简化处理：实际需要查询前序交易的输出地址
			fromAddr = "coinbase"
		}

		// 提取接收方地址（从第一个输出获取）
		toAddr := ""
		var value float64
		if len(btcTx.Vout) > 0 {
			if len(btcTx.Vout[0].ScriptPubKey.Addresses) > 0 {
				toAddr = btcTx.Vout[0].ScriptPubKey.Addresses[0]
			}
			value = btcTx.Vout[0].Value
		}

		txn := model.Transaction{
			Chain:       "btc",
			TxHash:      btcTx.TxID,
			BlockNumber: blockNumber,
			FromAddr:    fromAddr,
			ToAddr:      toAddr,
			Value:       fmt.Sprintf("%.8f", value),
			Timestamp:   btcBlock.Time,
			Status:      1,
		}
		transactions = append(transactions, txn)
	}

	return block, transactions, nil
}
