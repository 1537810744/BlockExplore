package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"blockexplore/internal/model"
)

// SolClient Solana RPC 客户端
// 通过 JSON-RPC 协议与 Solana 验证节点通信
type SolClient struct {
	rpcURL     string       // RPC 节点地址
	httpClient *http.Client // HTTP 客户端
}

// NewSolClient 创建 Solana RPC 客户端实例
func NewSolClient(rpcURL string) *SolClient {
	return &SolClient{
		rpcURL: rpcURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// call 发送 Solana JSON-RPC 请求
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

// solSlot Solana Slot 的 JSON 结构
type solSlot struct {
	Slot int64 `json:"slot"` // 槽位号
}

// solBlock Solana 区块的 JSON 结构
type solBlock struct {
	BlockHeight   int64                    `json:"blockHeight"`   // 区块高度
	BlockTime     *int64                   `json:"blockTime"`     // 出块时间（Unix 时间戳）
	Blockhash     string                   `json:"blockhash"`     // 区块哈希
	ParentSlot    int64                    `json:"parentSlot"`    // 父槽位号
	PreviousBlockhash string               `json:"previousBlockhash"` // 父区块哈希
	Transactions  []solTransaction         `json:"transactions"`  // 交易列表
}

// solTransaction Solana 交易的 JSON 结构
type solTransaction struct {
	Transaction struct {
		Message struct {
			AccountKeys []string `json:"accountKeys"` // 账户地址列表
			Instructions []struct {
				ProgramId string `json:"programId"` // 程序 ID
			} `json:"instructions"` // 指令列表
		} `json:"message"`
		Signatures []string `json:"signatures"` // 签名列表（第一个是交易签名）
	} `json:"transaction"`
	Meta struct {
		Err           interface{} `json:"err"`           // 错误信息（null 表示成功）
		Fee           int64       `json:"fee"`           // 手续费（lamports）
		PreBalances   []int64     `json:"preBalances"`   // 交易前余额
		PostBalances  []int64     `json:"postBalances"`  // 交易后余额
	} `json:"meta"`
}

// GetLatestBlockNumber 获取最新区块高度
func (c *SolClient) GetLatestBlockNumber() (int64, error) {
	// 获取最新确认的区块高度
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

// GetBlockByNumber 根据区块高度获取区块详情
func (c *SolClient) GetBlockByNumber(blockNumber int64) (*model.Block, []model.Transaction, error) {
	// Solana 使用槽位号(slot)来获取区块
	// 配置: 返回完整交易详情，交易版本为 0
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

	// 获取区块时间
	var timestamp int64
	if solBlock.BlockTime != nil {
		timestamp = *solBlock.BlockTime
	}

	// 转换为我们的区块模型
	block := &model.Block{
		Chain:       "sol",
		BlockNumber: solBlock.BlockHeight,
		BlockHash:   solBlock.Blockhash,
		ParentHash:  solBlock.PreviousBlockhash,
		Timestamp:   timestamp,
		TxCount:     len(solBlock.Transactions),
		Slot:        &blockNumber,
		Difficulty:  "0",
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
		fromAddr := ""
		toAddr := ""
		if len(solTx.Transaction.Message.AccountKeys) > 0 {
			fromAddr = solTx.Transaction.Message.AccountKeys[0]
		}
		if len(solTx.Transaction.Message.AccountKeys) > 1 {
			toAddr = solTx.Transaction.Message.AccountKeys[1]
		}

		// 计算转账金额（余额差值）
		var value int64
		if len(solTx.Meta.PreBalances) > 0 && len(solTx.Meta.PostBalances) > 0 {
			value = solTx.Meta.PostBalances[0] - solTx.Meta.PreBalances[0]
		}

		// 判断交易状态
		status := int16(1) // 默认成功
		if solTx.Meta.Err != nil {
			status = 0 // 有错误则失败
		}

		txn := model.Transaction{
			Chain:       "sol",
			TxHash:      txHash,
			BlockNumber: solBlock.BlockHeight,
			FromAddr:    fromAddr,
			ToAddr:      toAddr,
			Value:       strconv.FormatInt(value, 10),
			GasUsed:     strconv.FormatInt(solTx.Meta.Fee, 10),
			Status:      status,
			Timestamp:   timestamp,
		}
		transactions = append(transactions, txn)
	}

	return block, transactions, nil
}
