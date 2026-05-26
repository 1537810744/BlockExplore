// Package handler 提供 HTTP 请求处理器（Controller 层）
// 接收 HTTP 请求，调用 Service 层处理，返回 JSON 响应
package handler

import (
	"net/http"
	"strconv"

	"blockexplore/internal/service/query"
	"blockexplore/pkg/errcode"

	"github.com/gin-gonic/gin"
)

// BlockHandler 区块相关的 HTTP 处理器
type BlockHandler struct {
	queryService *query.QueryService // 查询服务
}

// NewBlockHandler 创建区块处理器实例
func NewBlockHandler(queryService *query.QueryService) *BlockHandler {
	return &BlockHandler{queryService: queryService}
}

// GetBlockList 获取区块列表（分页）
// GET /api/v1/blocks?chain=eth&page=1&page_size=20
func (h *BlockHandler) GetBlockList(c *gin.Context) {
	// 获取请求 ID（由中间件生成）
	requestID := c.GetString("request_id")

	// 解析查询参数
	chain := c.DefaultQuery("chain", "eth") // 默认以太坊
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	// 参数校验
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 20 // 限制每页最多 50 条
	}

	// 调用查询服务
	result, err := h.queryService.GetBlockList(chain, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errcode.Error(errcode.CodeDBError, requestID))
		return
	}

	c.JSON(http.StatusOK, errcode.Success(result, requestID))
}

// GetBlockDetail 获取区块详情
// GET /api/v1/blocks/:block_number?chain=eth
func (h *BlockHandler) GetBlockDetail(c *gin.Context) {
	requestID := c.GetString("request_id")

	// 解析路径参数
	blockNumber, err := strconv.ParseInt(c.Param("block_number"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errcode.Error(errcode.CodeBadRequest, requestID))
		return
	}

	chain := c.DefaultQuery("chain", "eth")

	// 查询区块详情
	block, err := h.queryService.GetBlockDetail(chain, blockNumber)
	if err != nil {
		c.JSON(http.StatusNotFound, errcode.Error(errcode.CodeNotFound, requestID))
		return
	}

	c.JSON(http.StatusOK, errcode.Success(block, requestID))
}

// GetBlockTransactions 获取区块内的交易列表
// GET /api/v1/blocks/:block_number/transactions?chain=eth&page=1&page_size=20
func (h *BlockHandler) GetBlockTransactions(c *gin.Context) {
	requestID := c.GetString("request_id")

	blockNumber, err := strconv.ParseInt(c.Param("block_number"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errcode.Error(errcode.CodeBadRequest, requestID))
		return
	}

	chain := c.DefaultQuery("chain", "eth")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 20
	}

	result, err := h.queryService.GetBlockTransactions(chain, blockNumber, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errcode.Error(errcode.CodeDBError, requestID))
		return
	}

	c.JSON(http.StatusOK, errcode.Success(result, requestID))
}
