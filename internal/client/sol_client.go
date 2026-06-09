// ============================================================
// SolClient Solana RPC 客户端
// ============================================================
// 通过 JSON-RPC 协议与 Solana 验证节点通信。
//
// Solana 的特点：
//   - 出块速度极快（约 0.4 秒一个区块）
//   - 使用 Slot（槽位号）而非 Block Number
//   - 交易费用极低（约 0.000005 SOL）
//   - 吞吐量高（理论 65,000 TPS）
//
// 常用 RPC 方法：
//   - getBlockHeight：获取最新区块高度
//   - getBlock：根据槽位号获取区块详情
//
// Go 语言基础知识:
//   - struct：结构体，用于定义数据结构
//   - *int64：指针类型，可以为 nil（表示空值）
//   - interface{}：空接口，可以持有任意类型的值
//   - strconv.FormatInt：将整数格式化为字符串
//   - append：向切片追加元素
// ============================================================
package client

import (
	"bytes"         // 字节操作
	"encoding/json" // JSON 编解码
	"fmt"           // 格式化字符串
	"io"            // IO 操作
	"net/http"      // HTTP 客户端
	"net/url"       // URL 解析
	"os"            // 环境变量
	"strconv"       // 字符串转换
	"time"          // 时间处理

	"blockexplore/internal/model" // 数据模型
	"blockexplore/pkg/logger"     // 日志

	"go.uber.org/zap" // 日志库
)

// ============================================================
// SolClient Solana RPC 客户端
// ============================================================
type SolClient struct {
	rpcURL     string       // RPC 节点地址
	httpClient *http.Client // HTTP 客户端
}

// ============================================================
// NewSolClient 创建 Solana RPC 客户端实例
// ============================================================
func NewSolClient(rpcURL string) *SolClient {
	logSolProxyInfo(rpcURL)
	return &SolClient{
		rpcURL: rpcURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// logSolProxyInfo 打印 SOL 客户端代理配置
func logSolProxyInfo(apiURL string) {
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
		zap.String("链", "SOL"),
		zap.String("代理状态", proxyStatus),
		zap.String("代理地址", proxyAddr),
		zap.String("API地址", apiURL),
	)
}

// ============================================================
// call 方法：发送 Solana JSON-RPC 请求
// ============================================================
// Solana 节点通常不需要认证（公开 RPC）
func (c *SolClient) call(method string, params ...interface{}) (json.RawMessage, error) {
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

	resp, err := c.httpClient.Post(c.rpcURL, "application/json", bytes.NewReader(bodyBytes))
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

// ============================================================
// solSlot Solana Slot 的 JSON 结构
// ============================================================
// Slot 是 Solana 的时间单位，类似于区块
type solSlot struct {
	Slot int64 `json:"slot"` // 槽位号
}

// ============================================================
// solBlock Solana 区块的 JSON 结构
// ============================================================
type solBlock struct {
	BlockHeight       int64            `json:"blockHeight"`       // 区块高度
	BlockTime         *int64           `json:"blockTime"`         // 出块时间（Unix 时间戳），可能为 nil
	Blockhash         string           `json:"blockhash"`         // 区块哈希
	ParentSlot        int64            `json:"parentSlot"`        // 父槽位号
	PreviousBlockhash string           `json:"previousBlockhash"` // 父区块哈希
	Transactions      []solTransaction `json:"transactions"`      // 交易列表
}

// ============================================================
// solTransaction Solana 交易的 JSON 结构
// ============================================================
// Solana 交易包含指令（Instructions），每个指令调用一个程序
type solTransaction struct {
	Transaction struct {
		Message struct {
			AccountKeys  []string `json:"accountKeys"`  // 账户地址列表
			Instructions []struct {
				ProgramId string `json:"programId"` // 程序 ID（被调用的程序）
			} `json:"instructions"` // 指令列表
		} `json:"message"`
		Signatures []string `json:"signatures"` // 签名列表（第一个是交易签名）
	} `json:"transaction"`
	Meta struct {
		Err           interface{} `json:"err"`           // 错误信息（null 表示成功）
		Fee           int64       `json:"fee"`           // 手续费（lamports，1 SOL = 10^9 lamports）
		PreBalances   []int64     `json:"preBalances"`   // 交易前余额
		PostBalances  []int64     `json:"postBalances"`  // 交易后余额
	} `json:"meta"`
}

// ============================================================
// GetLatestBlockNumber 方法：获取最新区块高度
// ============================================================
// 调用 getBlockHeight 方法，返回最新确认的区块高度
func (c *SolClient) GetLatestBlockNumber() (int64, error) {
	// commitment 参数指定确认级别：
	// - processed：最新处理的（可能回滚）
	// - confirmed：已确认的（推荐）
	// - finalized：最终确认的（最安全但最慢）
	result, err := c.call("getBlockHeight", map[string]string{"commitment": "confirmed"})
	if err != nil {
		return 0, fmt.Errorf("获取最新区块高度失败: %w", err)
	}

	var height int64
	if err := json.Unmarshal(result, &height); err != nil {
		return 0, fmt.Errorf("解析区块高度失败: %w", err)
	}

	return height, nil
}

// ============================================================
// GetBlockByNumber 方法：根据区块高度获取区块详情
// ============================================================
// Solana 使用槽位号(slot)来获取区块
func (c *SolClient) GetBlockByNumber(blockNumber int64) (*model.Block, []model.Transaction, error) {
	// 配置参数：
	// - encoding: json（返回 JSON 格式）
	// - transactionDetails: full（返回完整交易详情）
	// - rewards: false（不返回奖励信息）
	// - commitment: confirmed（已确认级别）
	// - maxSupportedTransactionVersion: 0（支持的交易版本）
	result, err := c.call("getBlock", blockNumber, map[string]interface{}{
		"encoding":                       "json",
		"transactionDetails":             "full",
		"rewards":                        false,
		"commitment":                     "confirmed",
		"maxSupportedTransactionVersion": 0,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("获取区块 %d 失败: %w", blockNumber, err)
	}

	var solBlock solBlock
	if err := json.Unmarshal(result, &solBlock); err != nil {
		return nil, nil, fmt.Errorf("解析区块数据失败: %w", err)
	}

	// 获取区块时间（可能为 nil）
	var timestamp int64
	if solBlock.BlockTime != nil {
		timestamp = *solBlock.BlockTime // 解引用指针，获取实际值
	}

	// 转换为我们的区块模型
	block := &model.Block{
		Chain:       "sol",
		BlockNumber: solBlock.BlockHeight,
		BlockHash:   solBlock.Blockhash,
		ParentHash:  solBlock.PreviousBlockhash,
		Timestamp:   timestamp,
		TxCount:     len(solBlock.Transactions),
		Slot:        &blockNumber, // 取指针
		Difficulty:  "0",          // Solana 没有难度概念
		GasUsed:     "0",
		GasLimit:    "0",
	}

	// 转换交易列表
	transactions := make([]model.Transaction, 0, len(solBlock.Transactions))
	for _, solTx := range solBlock.Transactions {
		// 获取交易签名（第一个签名作为交易哈希）
		txHash := ""
		if len(solTx.Transaction.Signatures) > 0 {
			txHash = solTx.Transaction.Signatures[0]
		}

		// 获取发送方和接收方地址
		// Solana 交易的第一个账户通常是发送方（fee payer）
		fromAddr := ""
		toAddr := ""
		preLen := len(solTx.Meta.PreBalances)
		postLen := len(solTx.Meta.PostBalances)
		if len(solTx.Transaction.Message.AccountKeys) > 0 {
			fromAddr = solTx.Transaction.Message.AccountKeys[0]
		}
		// 找到余额增加最多的账户作为接收方
		var value int64
		if preLen > 0 && postLen > 0 && preLen == postLen {
			maxDelta := int64(0)
			for i := 0; i < preLen && i < len(solTx.Transaction.Message.AccountKeys); i++ {
				delta := solTx.Meta.PostBalances[i] - solTx.Meta.PreBalances[i]
				if delta > maxDelta {
					maxDelta = delta
					toAddr = solTx.Transaction.Message.AccountKeys[i]
				}
			}
			if maxDelta > 0 {
				value = maxDelta
			} else {
				// 如果没有正向变化，取绝对值
				value = solTx.Meta.PreBalances[0] - solTx.Meta.PostBalances[0]
				if value < 0 {
					value = -value
				}
			}
		}
		if toAddr == "" && len(solTx.Transaction.Message.AccountKeys) > 1 {
			toAddr = solTx.Transaction.Message.AccountKeys[1]
		}

		// 判断交易状态
		status := int16(1) // 默认成功
		if solTx.Meta.Err != nil {
			status = 0 // 有错误则失败
		}

		// 构建交易模型
		txn := model.Transaction{
			Chain:       "sol",
			TxHash:      txHash,
			BlockNumber: solBlock.BlockHeight,
			FromAddr:    fromAddr,
			ToAddr:      toAddr,
			Value:       fmt.Sprintf("%.9f", float64(value)/1e9), // lamports 转 SOL
			GasUsed:     strconv.FormatInt(solTx.Meta.Fee, 10),
			Status:      status,
			Timestamp:   timestamp,
		}
		transactions = append(transactions, txn)
	}

	return block, transactions, nil
}
