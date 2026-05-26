// ============================================================
// Package handler 提供 HTTP 请求处理器（Controller 层）
// ============================================================
// 接收 HTTP 请求，调用 Service 层处理，返回 JSON 响应。
//
// Handler 的职责：
//   - 解析请求参数（路径参数、查询参数、请求体）
//   - 参数校验
//   - 调用 Service 层处理业务逻辑
//   - 构建统一格式的 JSON 响应
//   - 返回 HTTP 状态码
//
// Go 语言基础知识:
//   - *gin.Context：Gin 框架的上下文，包含请求和响应信息
//   - c.Param("name")：获取路径参数，例如 /blocks/:block_number 中的 block_number
//   - c.Query("name")：获取查询参数，例如 ?chain=eth 中的 chain
//   - c.DefaultQuery("name", "default")：获取查询参数，不存在时返回默认值
//   - c.JSON(status, data)：返回 JSON 响应
//   - c.GetString("key")：从上下文中获取值（由中间件设置）
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
// BlockHandler 区块相关的 HTTP 处理器
// ============================================================
type BlockHandler struct {
	queryService *query.QueryService // 查询服务
}

// ============================================================
// NewBlockHandler 创建区块处理器实例
// ============================================================
func NewBlockHandler(queryService *query.QueryService) *BlockHandler {
	return &BlockHandler{queryService: queryService}
}

// ============================================================
// GetBlockList 方法：获取区块列表（分页）
// ============================================================
// GET /api/v1/blocks?chain=eth&page=1&page_size=20
func (h *BlockHandler) GetBlockList(c *gin.Context) {
	// 获取请求 ID（由 RequestID 中间件生成）
	requestID := c.GetString("request_id")

	// 解析查询参数
	// c.DefaultQuery 获取查询参数，不存在时返回默认值
	chain := c.DefaultQuery("chain", "eth") // 默认以太坊
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	// strconv.Atoi 将字符串转换为整数
	// _ 表示忽略错误（这里错误时会使用默认值 0，后面会校正）

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
		// 返回错误响应
		// http.StatusInternalServerError = 500
		c.JSON(http.StatusInternalServerError, errcode.Error(errcode.CodeDBError, requestID))
		return
	}

	// 返回成功响应
	// http.StatusOK = 200
	c.JSON(http.StatusOK, errcode.Success(result, requestID))
}

// ============================================================
// GetBlockDetail 方法：获取区块详情
// ============================================================
// GET /api/v1/blocks/:block_number?chain=eth
func (h *BlockHandler) GetBlockDetail(c *gin.Context) {
	requestID := c.GetString("request_id")

	// 解析路径参数
	// c.Param 获取路径参数，例如 /blocks/12345 中的 "12345"
	blockNumber, err := strconv.ParseInt(c.Param("block_number"), 10, 64)
	if err != nil {
		// 参数格式错误，返回 400
		c.JSON(http.StatusBadRequest, errcode.Error(errcode.CodeBadRequest, requestID))
		return
	}

	chain := c.DefaultQuery("chain", "eth")

	// 查询区块详情
	block, err := h.queryService.GetBlockDetail(chain, blockNumber)
	if err != nil {
		// 未找到，返回 404
		c.JSON(http.StatusNotFound, errcode.Error(errcode.CodeNotFound, requestID))
		return
	}

	c.JSON(http.StatusOK, errcode.Success(block, requestID))
}

// ============================================================
// GetBlockTransactions 方法：获取区块内的交易列表
// ============================================================
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
