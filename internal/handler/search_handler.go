package handler

import (
	"net/http"

	"blockexplore/internal/repository"
	"blockexplore/pkg/errcode"

	"github.com/gin-gonic/gin"
)

// SearchHandler 搜索相关的 HTTP 处理器
type SearchHandler struct {
	searchRepo *repository.SearchRepo
}

// NewSearchHandler 创建搜索处理器实例
func NewSearchHandler(searchRepo *repository.SearchRepo) *SearchHandler {
	return &SearchHandler{searchRepo: searchRepo}
}

// Search 统一搜索接口
// GET /api/v1/search?q=keyword
// 根据输入自动识别类型（区块/交易/地址）
func (h *SearchHandler) Search(c *gin.Context) {
	requestID := c.GetString("request_id")

	keyword := c.Query("q")
	if keyword == "" {
		c.JSON(http.StatusBadRequest, errcode.ErrorWithMsg(errcode.CodeBadRequest, "搜索关键词不能为空", requestID))
		return
	}

	result, err := h.searchRepo.Search(keyword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errcode.Error(errcode.CodeDBError, requestID))
		return
	}

	if result == nil {
		c.JSON(http.StatusNotFound, errcode.ErrorWithMsg(errcode.CodeNotFound, "未找到匹配结果", requestID))
		return
	}

	c.JSON(http.StatusOK, errcode.Success(result, requestID))
}
