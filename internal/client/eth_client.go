// Package client 封装各区块链的 RPC 客户端
package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"blockexplore/internal/model"
	"blockexplore/pkg/logger"

	"go.uber.org/zap"
)

// EthClient 以太坊 RPC 客户端
// 通过 JSON-RPC 协议与以太坊全节点通信
type EthClient struct {
	rpcURL     string       // RPC 节点地址
	httpClient *http.Client // HTTP 客户端（复用连接）
}

// NewEthClient 创建以太坊 RPC 客户端实例
func NewEthClient(rpcURL string) *EthClient {
	return &EthClient{
		rpcURL: rpcURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second, // 请求超时 30 秒
		},
	}
}

// jsonRPCRequest JSON-RPC 请求结构体
// 以太坊使用 JSON-RPC 2.0 协议通信
type jsonRPCRequest struct {
	JsonRPC string        `json:"jsonrpc"` // 协议版本，固定为 "2.0"
	Method  string        `json:"method"`  // 调用的方法名
	Params  []interface{} `json:"params"`  // 方法参数
	ID      int           `json:"id"`      // 请求 ID
}

// jsonRPCResponse JSON-RPC 响应结构体
type jsonRPCResponse struct {
	JsonRPC string          `json:"jsonrpc"` // 协议版本
	Result  json.RawMessage `json:"result"`  // 返回结果
	Error   *RPCError       `json:"error"`   // 错误信息
	ID      int             `json:"id"`      // 请求 ID
}

// RPCError RPC 调用错误
type RPCError struct {
	Code    int    `json:"code"`    // 错误码
	Message string `json:"message"` // 错误消息
}

func (e *RPCError) Error() string {
	return fmt.Sprintf("RPC 错误 %d: %s", e.Code, e.Message)
}

// call 发送 JSON-RPC 请求
// method: RPC 方法名
// params: 方法参数
// 返回: 响应结果的原始 JSON
func (c *EthClient) call(method string, params ...interface{}) (json.RawMessage, error) {
	// 构建请求体
	reqBody := jsonRPCRequest{
		JsonRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      1,
	}

	// 序列化为 JSON
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	// 发送 HTTP POST 请求
	resp, err := c.httpClient.Post(c.rpcURL, "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("发送 RPC 请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 解析响应
	var rpcResp jsonRPCResponse
	if err := json.Unmarshal(respBytes, &rpcResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	// 检查是否有 RPC 错误
	if rpcResp.Error != nil {
		return nil, rpcResp.Error
	}

	return rpcResp.Result, nil
}

// ethBlock 以太坊区块的 JSON 结构（RPC 返回的原始格式）
type ethBlock struct {
	Number       string `json:"number"`        // 区块高度（十六进制）
	Hash         string `json:"hash"`           // 区块哈希
	ParentHash   string `json:"parentHash"`     // 父区块哈希
	Timestamp    string `json:"timestamp"`      // 出块时间（十六进制）
	Transactions []ethTransaction `json:"transactions"` // 交易列表
	GasUsed      string `json:"gasUsed"`        // 已消耗 Gas
	GasLimit     string `json:"gasLimit"`       // Gas 上限
	Size         string `json:"size"`           // 区块大小
}

// ethTransaction 以太坊交易的 JSON 结构
type ethTransaction struct {
	Hash        string `json:"hash"`        // 交易哈希
	From        string `json:"from"`        // 发送方地址
	To          string `json:"to"`          // 接收方地址（合约创建时为空）
	Value       string `json:"value"`       // 转账金额（Wei，十六进制）
	GasPrice    string `json:"gasPrice"`    // Gas 价格
	Gas         string `json:"gas"`         // Gas 上限
	Nonce       string `json:"nonce"`       // 交易序号
	Input       string `json:"input"`       // 调用数据
	BlockNumber string `json:"blockNumber"` // 所在区块高度
	BlockHash   string `json:"blockHash"`   // 所在区块哈希
	TransactionIndex string `json:"transactionIndex"` // 交易在区块中的索引
}

// GetLatestBlockNumber 获取最新区块高度
// 调用 eth_blockNumber 方法，返回十进制区块高度
func (c *EthClient) GetLatestBlockNumber() (int64, error) {
	result, err := c.call("eth_blockNumber")
	if err != nil {
		return 0, fmt.Errorf("获取最新区块高度失败: %w", err)
	}

	// 解析十六进制区块高度
	var hexNum string
	if err := json.Unmarshal(result, &hexNum); err != nil {
		return 0, fmt.Errorf("解析区块高度失败: %w", err)
	}

	// 十六进制转十进制
	return hexToDecimal(hexNum)
}

// GetBlockByNumber 根据区块高度获取区块详情（含交易）
// blockNumber: 区块高度
// 返回: 区块模型和交易列表
func (c *EthClient) GetBlockByNumber(blockNumber int64) (*model.Block, []model.Transaction, error) {
	// 十进制转十六进制
	hexNum := fmt.Sprintf("0x%x", blockNumber)

	// 调用 eth_getBlockByNumber，第二个参数 true 表示返回完整交易信息
	result, err := c.call("eth_getBlockByNumber", hexNum, true)
	if err != nil {
		return nil, nil, fmt.Errorf("获取区块 %d 失败: %w", blockNumber, err)
	}

	// 解析区块数据
	var ethBlock ethBlock
	if err := json.Unmarshal(result, &ethBlock); err != nil {
		return nil, nil, fmt.Errorf("解析区块数据失败: %w", err)
	}

	// 转换为我们的区块模型
	block := &model.Block{
		Chain:       "eth",
		BlockNumber: blockNumber,
		BlockHash:   ethBlock.Hash,
		ParentHash:  ethBlock.ParentHash,
		Timestamp:   hexToDecimalDefault(ethBlock.Timestamp, 0),
		TxCount:     len(ethBlock.Transactions),
		GasUsed:     hexToDecimalStr(ethBlock.GasUsed),
		GasLimit:    hexToDecimalStr(ethBlock.GasLimit),
		SizeBytes:   hexToIntDefault(ethBlock.Size, 0),
		Difficulty:  "",
	}

	// 转换交易列表
	transactions := make([]model.Transaction, 0, len(ethBlock.Transactions))
	for _, tx := range ethBlock.Transactions {
		// 解析交易金额（Wei 转 ETH 不在这里做，存储原始值）
		nonce := hexToDecimalDefault(tx.Nonce, 0)
		txn := model.Transaction{
			Chain:       "eth",
			TxHash:      tx.Hash,
			BlockNumber: blockNumber,
			FromAddr:    strings.ToLower(tx.From),
			ToAddr:      strings.ToLower(tx.To),
			Value:       hexToDecimalStr(tx.Value),
			GasPrice:    hexToDecimalStr(tx.GasPrice),
			GasUsed:     "0", // GasUsed 需要通过交易回执获取
			GasLimit:    hexToDecimalStr(tx.Gas),
			Nonce:       &nonce,
			InputData:   tx.Input,
			Status:      1, // 默认成功，状态需要通过交易回执获取
			Timestamp:   hexToDecimalDefault(ethBlock.Timestamp, 0),
		}
		transactions = append(transactions, txn)
	}

	return block, transactions, nil
}

// GetTransactionReceipt 获取交易回执（用于获取实际 GasUsed 和交易状态）
func (c *EthClient) GetTransactionReceipt(txHash string) (map[string]interface{}, error) {
	result, err := c.call("eth_getTransactionReceipt", txHash)
	if err != nil {
		return nil, fmt.Errorf("获取交易回执失败: %w", err)
	}

	var receipt map[string]interface{}
	if err := json.Unmarshal(result, &receipt); err != nil {
		return nil, fmt.Errorf("解析交易回执失败: %w", err)
	}

	return receipt, nil
}

// hexToDecimal 十六进制字符串转十进制 int64
// 例如: "0x1a" -> 26
func hexToDecimal(hex string) (int64, error) {
	hex = strings.TrimPrefix(hex, "0x")
	if hex == "" {
		return 0, nil
	}
	return strconv.ParseInt(hex, 16, 64)
}

// hexToDecimalDefault 十六进制转十进制，失败时返回默认值
func hexToDecimalDefault(hex string, defaultVal int64) int64 {
	val, err := hexToDecimal(hex)
	if err != nil {
		logger.Warn("十六进制转换失败", zap.String("hex", hex), zap.Error(err))
		return defaultVal
	}
	return val
}

// hexToDecimalStr 十六进制转十进制字符串
func hexToDecimalStr(hex string) string {
	val, err := hexToDecimal(hex)
	if err != nil {
		return "0"
	}
	return strconv.FormatInt(val, 10)
}

// hexToIntDefault 十六进制转 int，默认值
func hexToIntDefault(hex string, defaultVal int) int {
	val, err := hexToDecimal(hex)
	if err != nil {
		return defaultVal
	}
	return int(val)
}
