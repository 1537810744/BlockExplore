// ============================================================
// Package client 封装各区块链的 RPC 客户端
// ============================================================
// 该包提供了与区块链节点通信的客户端实现。
//
// RPC（远程过程调用）是与区块链节点通信的标准协议：
//   - 通过 HTTP POST 请求发送 JSON 格式的调用请求
//   - 节点返回 JSON 格式的响应
//   - 例如：获取最新区块高度、获取区块详情、获取交易详情等
//
// 以太坊 RPC 方法示例：
//   - eth_blockNumber：获取最新区块高度
//   - eth_getBlockByNumber：根据区块高度获取区块详情
//   - eth_getTransactionReceipt：获取交易回执
//
// Go 语言基础知识:
//   - package：包，Go 的模块化机制
//   - struct：结构体，用于定义数据结构
//   - func (c *EthClient) 方法名()：方法定义，c 是接收者
//   - *EthClient：指针类型，指向 EthClient 实例
//   - interface{}：空接口，可以持有任意类型的值
//   - error：Go 的错误类型，函数通过返回 error 来报告错误
//   - fmt.Errorf：格式化创建错误，%w 包装原始错误
//   - defer：延迟执行，确保资源被正确释放
//   - json.RawMessage：原始 JSON 字节，延迟解析
//
// ============================================================
package client

import (
	"bytes"         // 字节操作
	"encoding/json" // JSON 编解码
	"fmt"           // 格式化字符串
	"io"            // IO 操作
	"net/http"      // HTTP 客户端
	"strconv"       // 字符串转换
	"strings"       // 字符串操作
	"time"          // 时间处理

	"blockexplore/internal/model" // 数据模型
	"blockexplore/pkg/logger"     // 日志

	"go.uber.org/zap" // 日志库
)

// ============================================================
// EthClient 以太坊 RPC 客户端
// ============================================================
// 通过 JSON-RPC 协议与以太坊全节点通信
// JSON-RPC 是一种远程过程调用协议，使用 JSON 格式传输数据
type EthClient struct {
	rpcURL     string       // RPC 节点地址，例如 "http://localhost:8545"
	httpClient *http.Client // HTTP 客户端，复用连接池，避免每次请求都新建连接
}

// ============================================================
// NewEthClient 创建以太坊 RPC 客户端实例
// ============================================================
// 参数 rpcURL：以太坊节点的 RPC 地址
// 返回值：*EthClient 指针，指向新创建的客户端实例
func NewEthClient(rpcURL string) *EthClient {
	return &EthClient{
		rpcURL: rpcURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second, // 请求超时 30 秒
		},
	}
}

// ============================================================
// jsonRPCRequest JSON-RPC 请求结构体
// ============================================================
// 以太坊使用 JSON-RPC 2.0 协议通信
// 请求格式示例：
//
//	{
//	  "jsonrpc": "2.0",
//	  "method": "eth_blockNumber",
//	  "params": [],
//	  "id": 1
//	}
type jsonRPCRequest struct {
	JsonRPC string        `json:"jsonrpc"` // 协议版本，固定为 "2.0"
	Method  string        `json:"method"`  // 调用的方法名
	Params  []interface{} `json:"params"`  // 方法参数（空接口切片，可以存放任意类型）
	ID      int           `json:"id"`      // 请求 ID，用于匹配请求和响应
}

// ============================================================
// jsonRPCResponse JSON-RPC 响应结构体
// ============================================================
// 响应格式示例：
//
//	{
//	  "jsonrpc": "2.0",
//	  "result": "0x1a2b3c",
//	  "error": null,
//	  "id": 1
//	}
type jsonRPCResponse struct {
	JsonRPC string          `json:"jsonrpc"` // 协议版本
	Result  json.RawMessage `json:"result"`  // 返回结果（原始 JSON，延迟解析）
	Error   *RPCError       `json:"error"`   // 错误信息（nil 表示无错误）
	ID      int             `json:"id"`      // 请求 ID
}

// ============================================================
// RPCError RPC 调用错误
// ============================================================
// 当 RPC 调用失败时，节点会返回错误信息
type RPCError struct {
	Code    int    `json:"code"`    // 错误码
	Message string `json:"message"` // 错误消息
}

// Error 方法实现 error 接口
// Go 的接口是隐式实现的，只要实现了接口的所有方法，就自动实现了该接口
func (e *RPCError) Error() string {
	return fmt.Sprintf("RPC 错误 %d: %s", e.Code, e.Message)
}

// ============================================================
// call 方法：发送 JSON-RPC 请求
// ============================================================
// 这是所有 RPC 调用的基础方法，其他方法都通过它发送请求
// 参数 method：RPC 方法名，例如 "eth_blockNumber"
// 参数 params：方法参数（可变参数，可以传零个或多个）
// 返回值：响应结果的原始 JSON 和错误信息
func (c *EthClient) call(method string, params ...interface{}) (json.RawMessage, error) {
	// 构建请求体
	// ...interface{} 是可变参数，params 本身是一个切片
	reqBody := jsonRPCRequest{
		JsonRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      1,
	}

	// json.Marshal 将结构体序列化为 JSON 字节
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
		// %w 包装原始错误，保留错误链，方便调试
	}

	// 发送 HTTP POST 请求
	// bytes.NewReader 将字节切片包装为 io.Reader 接口
	resp, err := c.httpClient.Post(c.rpcURL, "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("发送 RPC 请求失败: %w", err)
	}
	// defer 延迟执行，在函数返回前关闭响应体，释放连接
	// 这是 Go 的最佳实践，防止资源泄漏
	defer resp.Body.Close()

	// 读取响应体
	// io.ReadAll 读取所有数据，返回字节切片
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

// ============================================================
// ethBlock 以太坊区块的 JSON 结构
// ============================================================
// 这是 RPC 返回的原始格式，十六进制值需要转换为十进制
type ethBlock struct {
	Number       string           `json:"number"`       // 区块高度（十六进制），例如 "0x1a2b3c"
	Hash         string           `json:"hash"`         // 区块哈希
	ParentHash   string           `json:"parentHash"`   // 父区块哈希
	Timestamp    string           `json:"timestamp"`    // 出块时间（十六进制 Unix 时间戳）
	Transactions []ethTransaction `json:"transactions"` // 交易列表
	GasUsed      string           `json:"gasUsed"`      // 已消耗 Gas（十六进制）
	GasLimit     string           `json:"gasLimit"`     // Gas 上限（十六进制）
	Size         string           `json:"size"`         // 区块大小（十六进制）
}

// ============================================================
// ethTransaction 以太坊交易的 JSON 结构
// ============================================================
type ethTransaction struct {
	Hash             string `json:"hash"`             // 交易哈希
	From             string `json:"from"`             // 发送方地址
	To               string `json:"to"`               // 接收方地址（合约创建时为空）
	Value            string `json:"value"`            // 转账金额（Wei，十六进制）
	GasPrice         string `json:"gasPrice"`         // Gas 价格（十六进制）
	Gas              string `json:"gas"`              // Gas 上限（十六进制）
	Nonce            string `json:"nonce"`            // 交易序号（十六进制）
	Input            string `json:"input"`            // 调用数据（合约调用时的参数）
	BlockNumber      string `json:"blockNumber"`      // 所在区块高度（十六进制）
	BlockHash        string `json:"blockHash"`        // 所在区块哈希
	TransactionIndex string `json:"transactionIndex"` // 交易在区块中的索引（十六进制）
}

// ============================================================
// GetLatestBlockNumber 方法：获取最新区块高度
// ============================================================
// 调用 eth_blockNumber 方法，返回十进制区块高度
// 返回值：int64 区块高度，error 错误信息
func (c *EthClient) GetLatestBlockNumber() (int64, error) {
	// 调用 RPC 方法，无参数
	result, err := c.call("eth_blockNumber")
	if err != nil {
		return 0, fmt.Errorf("获取最新区块高度失败: %w", err)
	}

	// 解析十六进制区块高度
	// RPC 返回的是十六进制字符串，如 "0x1a2b3c"
	var hexNum string
	if err := json.Unmarshal(result, &hexNum); err != nil {
		return 0, fmt.Errorf("解析区块高度失败: %w", err)
	}

	// 十六进制转十进制
	return hexToDecimal(hexNum)
}

// ============================================================
// GetBlockByNumber 方法：根据区块高度获取区块详情（含交易）
// ============================================================
// 参数 blockNumber：区块高度（十进制）
// 返回值：区块模型、交易列表、错误信息
func (c *EthClient) GetBlockByNumber(blockNumber int64) (*model.Block, []model.Transaction, error) {
	// 十进制转十六进制
	// %x 格式化为十六进制（小写），0x 前缀
	hexNum := fmt.Sprintf("0x%x", blockNumber)

	// 调用 eth_getBlockByNumber，第二个参数 true 表示返回完整交易信息
	// 如果传 false，只返回交易哈希列表
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
	// 将 RPC 返回的十六进制值转换为十进制
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
		Difficulty:  "", // 以太坊 2.0 后不再有难度字段
	}

	// 转换交易列表
	// make([]model.Transaction, 0, len(...)) 创建切片，长度 0，容量为交易数量
	transactions := make([]model.Transaction, 0, len(ethBlock.Transactions))
	for _, tx := range ethBlock.Transactions {
		// 解析交易金额（Wei 转 ETH 不在这里做，存储原始值）
		nonce := hexToDecimalDefault(tx.Nonce, 0)
		txn := model.Transaction{
			Chain:       "eth",
			TxHash:      tx.Hash,
			BlockNumber: blockNumber,
			FromAddr:    strings.ToLower(tx.From), // 统一转为小写
			ToAddr:      strings.ToLower(tx.To),
			Value:       hexToDecimalStr(tx.Value),
			GasPrice:    hexToDecimalStr(tx.GasPrice),
			GasUsed:     "0", // GasUsed 需要通过交易回执获取
			GasLimit:    hexToDecimalStr(tx.Gas),
			Nonce:       &nonce, // 取指针
			InputData:   tx.Input,
			Status:      1, // 默认成功，状态需要通过交易回执获取
			Timestamp:   hexToDecimalDefault(ethBlock.Timestamp, 0),
		}
		transactions = append(transactions, txn)
	}

	return block, transactions, nil
}

// ============================================================
// GetTransactionReceipt 方法：获取交易回执
// ============================================================
// 交易回执包含交易的实际执行结果：
//   - GasUsed：实际消耗的 Gas
//   - Status：交易状态（1=成功，0=失败）
//   - Logs：事件日志
func (c *EthClient) GetTransactionReceipt(txHash string) (map[string]interface{}, error) {
	result, err := c.call("eth_getTransactionReceipt", txHash)
	if err != nil {
		return nil, fmt.Errorf("获取交易回执失败: %w", err)
	}

	// 使用 map[string]interface{} 接收任意结构的 JSON
	var receipt map[string]interface{}
	if err := json.Unmarshal(result, &receipt); err != nil {
		return nil, fmt.Errorf("解析交易回执失败: %w", err)
	}

	return receipt, nil
}

// ============================================================
// 工具函数：十六进制转换
// ============================================================

// hexToDecimal 将十六进制字符串转换为十进制 int64
// 例如: "0x1a" -> 26
// strings.TrimPrefix 去掉字符串前缀
func hexToDecimal(hex string) (int64, error) {
	hex = strings.TrimPrefix(hex, "0x") // 去掉 "0x" 前缀
	if hex == "" {
		return 0, nil
	}
	// strconv.ParseInt 将字符串解析为整数
	// 参数：字符串、进制（16=十六进制）、位数（64 位）
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
// strconv.FormatInt 将整数格式化为字符串
func hexToDecimalStr(hex string) string {
	val, err := hexToDecimal(hex)
	if err != nil {
		return "0"
	}
	return strconv.FormatInt(val, 10) // 10 表示十进制
}

// hexToIntDefault 十六进制转 int，默认值
func hexToIntDefault(hex string, defaultVal int) int {
	val, err := hexToDecimal(hex)
	if err != nil {
		return defaultVal
	}
	return int(val)
}
