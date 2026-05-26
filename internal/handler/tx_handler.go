// ============================================================
// TxHandler 交易相关的 HTTP 处理器
// ============================================================
// 处理交易查询和地址交易历史的 HTTP 请求。
//
// Go 语言基础知识:
//   - *gin.Context：Gin 框架的上下文
//   - c.Param("name")：获取路径参数
//   - c.Query("name")：获取查询参数
//   - c.DefaultQuery("name", "default")：获取查询参数，不存在时返回默认值
//   - c.JSON(status, data)：返回 JSON 响应
//   - strconv.Atoi：字符串转整数
//   - strconv.ParseInt：字符串转 int64
// ============================================================
package handler

import (
	"net/http"   // HTTP 状态码
	"strconv"    // 字符串转换

	"blockexplore/internal/service/query" // 查询服务
	"blockexplore/pkg/errcode"          // 错误码

	"github.com/gin-gonic/gin" // Gin Web 框架
)

// ============================================================
// TxHandler 交易相关的 HTTP 处理器
// ============================================================
type TxHandler struct {
	queryService *query.QueryService
}

// ============================================================
// NewTxHandler 创建交易处理器实例
// ============================================================
func NewTxHandler(queryService *query.QueryService) *TxHandler {
	return &TxHandler{queryService: queryService}
}

// ============================================================
// GetTransactionDetail 方法：获取交易详情
// ============================================================
// GET /api/v1/transactions/:hash?chain=eth
func (h *TxHandler) GetTransactionDetail(c *gin.Context) {
	requestID := c.GetString("request_id")

	// 获取路径参数（交易哈希）
	txHash := c.Param("hash")
	chain := c.DefaultQuery("chain", "eth")

	// 查询交易详情
	tx, err := h.queryService.GetTransactionDetail(chain, txHash)
	if err != nil {
		c.JSON(http.StatusNotFound, errcode.Error(errcode.CodeNotFound, requestID))
		return
	}

	c.JSON(http.StatusOK, errcode.Success(tx, requestID))
}

// ============================================================
// GetAddressTransactions 方法：获取地址的交易历史
// ============================================================
// GET /api/v1/addresses/:address/transactions?chain=eth&page=1&page_size=20
func (h *TxHandler) GetAddressTransactions(c *gin.Context) {
	requestID := c.GetString("request_id")

	// 获取路径参数（地址）
	address := c.Param("address")
	chain := c.DefaultQuery("chain", "eth")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	// 参数校验
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 20
	}

	// 查询地址交易历史
	result, err := h.queryService.GetAddressTransactions(chain, address, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errcode.Error(errcode.CodeDBError, requestID))
		return
	}

	c.JSON(http.StatusOK, errcode.Success(result, requestID))
}
