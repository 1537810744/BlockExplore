// ============================================================
// PriceHandler 价格相关的 HTTP 处理器
// ============================================================
// 处理价格查询和价格历史的 HTTP 请求。
//
// Go 语言基础知识:
//   - *gin.Context：Gin 框架的上下文
//   - c.Param("name")：获取路径参数
//   - c.Query("name")：获取查询参数
//   - c.DefaultQuery("name", "default")：获取查询参数，不存在时返回默认值
//   - c.JSON(status, data)：返回 JSON 响应
//   - strconv.ParseInt：字符串转 int64
//   - strconv.Atoi：字符串转 int
// ============================================================
package handler

import (
	"net/http"   // HTTP 状态码
	"strconv"    // 字符串转换

	"blockexplore/internal/service/price" // 价格服务
	"blockexplore/pkg/errcode"          // 错误码

	"github.com/gin-gonic/gin" // Gin Web 框架
)

// ============================================================
// PriceHandler 价格相关的 HTTP 处理器
// ============================================================
type PriceHandler struct {
	priceService *price.PriceService
}

// ============================================================
// NewPriceHandler 创建价格处理器实例
// ============================================================
func NewPriceHandler(priceService *price.PriceService) *PriceHandler {
	return &PriceHandler{priceService: priceService}
}

// ============================================================
// GetCurrentPrice 方法：获取当前价格
// ============================================================
// GET /api/v1/price/:chain
func (h *PriceHandler) GetCurrentPrice(c *gin.Context) {
	requestID := c.GetString("request_id")

	// 获取路径参数（链标识）
	chain := c.Param("chain")

	result, err := h.priceService.GetCurrentPrice(chain)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errcode.Error(errcode.CodeInternalError, requestID))
		return
	}

	c.JSON(http.StatusOK, errcode.Success(result, requestID))
}

// ============================================================
// GetPriceHistory 方法：获取价格历史
// ============================================================
// GET /api/v1/price/:chain/history?start_time=xxx&end_time=xxx&limit=100
func (h *PriceHandler) GetPriceHistory(c *gin.Context) {
	requestID := c.GetString("request_id")

	chain := c.Param("chain")

	// 解析查询参数
	// c.Query 获取查询参数，返回字符串
	// strconv.ParseInt 将字符串转换为 int64
	startTime, _ := strconv.ParseInt(c.Query("start_time"), 10, 64)
	endTime, _ := strconv.ParseInt(c.Query("end_time"), 10, 64)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))

	// 参数校验
	if limit < 1 || limit > 1000 {
		limit = 100
	}

	result, err := h.priceService.GetPriceHistory(chain, startTime, endTime, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errcode.Error(errcode.CodeDBError, requestID))
		return
	}

	c.JSON(http.StatusOK, errcode.Success(result, requestID))
}
