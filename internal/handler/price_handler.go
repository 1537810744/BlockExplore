package handler

import (
	"net/http"
	"strconv"

	"blockexplore/internal/service/price"
	"blockexplore/pkg/errcode"

	"github.com/gin-gonic/gin"
)

// PriceHandler 价格相关的 HTTP 处理器
type PriceHandler struct {
	priceService *price.PriceService
}

// NewPriceHandler 创建价格处理器实例
func NewPriceHandler(priceService *price.PriceService) *PriceHandler {
	return &PriceHandler{priceService: priceService}
}

// GetCurrentPrice 获取当前价格
// GET /api/v1/price/:chain
func (h *PriceHandler) GetCurrentPrice(c *gin.Context) {
	requestID := c.GetString("request_id")

	chain := c.Param("chain")

	result, err := h.priceService.GetCurrentPrice(chain)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errcode.Error(errcode.CodeInternalError, requestID))
		return
	}

	c.JSON(http.StatusOK, errcode.Success(result, requestID))
}

// GetPriceHistory 获取价格历史
// GET /api/v1/price/:chain/history?start_time=xxx&end_time=xxx&limit=100
func (h *PriceHandler) GetPriceHistory(c *gin.Context) {
	requestID := c.GetString("request_id")

	chain := c.Param("chain")

	startTime, _ := strconv.ParseInt(c.Query("start_time"), 10, 64)
	endTime, _ := strconv.ParseInt(c.Query("end_time"), 10, 64)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))

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
