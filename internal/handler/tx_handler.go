package handler

import (
	"net/http"
	"strconv"

	"blockexplore/internal/service/query"
	"blockexplore/pkg/errcode"

	"github.com/gin-gonic/gin"
)

// TxHandler 交易相关的 HTTP 处理器
type TxHandler struct {
	queryService *query.QueryService
}

// NewTxHandler 创建交易处理器实例
func NewTxHandler(queryService *query.QueryService) *TxHandler {
	return &TxHandler{queryService: queryService}
}

// GetTransactionDetail 获取交易详情
// GET /api/v1/transactions/:hash?chain=eth
func (h *TxHandler) GetTransactionDetail(c *gin.Context) {
	requestID := c.GetString("request_id")

	txHash := c.Param("hash")
	chain := c.DefaultQuery("chain", "eth")

	tx, err := h.queryService.GetTransactionDetail(chain, txHash)
	if err != nil {
		c.JSON(http.StatusNotFound, errcode.Error(errcode.CodeNotFound, requestID))
		return
	}

	c.JSON(http.StatusOK, errcode.Success(tx, requestID))
}

// GetAddressTransactions 获取地址的交易历史
// GET /api/v1/addresses/:address/transactions?chain=eth&page=1&page_size=20
func (h *TxHandler) GetAddressTransactions(c *gin.Context) {
	requestID := c.GetString("request_id")

	address := c.Param("address")
	chain := c.DefaultQuery("chain", "eth")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 20
	}

	result, err := h.queryService.GetAddressTransactions(chain, address, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errcode.Error(errcode.CodeDBError, requestID))
		return
	}

	c.JSON(http.StatusOK, errcode.Success(result, requestID))
}
