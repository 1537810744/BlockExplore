// ============================================================
// SearchHandler 搜索相关的 HTTP 处理器
// ============================================================
// 处理统一搜索接口的 HTTP 请求。
//
// 搜索功能：根据用户输入的关键词，自动识别是区块高度、交易哈希还是地址，
// 然后从数据库中查询匹配的结果。
//
// Go 语言基础知识:
//   - *gin.Context：Gin 框架的上下文
//   - c.Query("name")：获取查询参数
//   - c.JSON(status, data)：返回 JSON 响应
//   - http.StatusBadRequest = 400：请求参数错误
//   - http.StatusNotFound = 404：资源不存在
//   - http.StatusInternalServerError = 500：服务器内部错误
//   - http.StatusOK = 200：成功
// ============================================================
package handler

import (
	"net/http" // HTTP 状态码

	"blockexplore/internal/repository" // 数据访问层
	"blockexplore/pkg/errcode"        // 错误码

	"github.com/gin-gonic/gin" // Gin Web 框架
)

// ============================================================
// SearchHandler 搜索相关的 HTTP 处理器
// ============================================================
type SearchHandler struct {
	searchRepo *repository.SearchRepo
}

// ============================================================
// NewSearchHandler 创建搜索处理器实例
// ============================================================
func NewSearchHandler(searchRepo *repository.SearchRepo) *SearchHandler {
	return &SearchHandler{searchRepo: searchRepo}
}

// ============================================================
// Search 方法：统一搜索接口
// ============================================================
// GET /api/v1/search?q=keyword
// 根据输入自动识别类型（区块/交易/地址）
func (h *SearchHandler) Search(c *gin.Context) {
	requestID := c.GetString("request_id")

	// 获取查询参数 q
	keyword := c.Query("q")
	if keyword == "" {
		// 搜索关键词为空，返回 400 错误
		// errcode.ErrorWithMsg 返回带自定义消息的错误响应
		c.JSON(http.StatusBadRequest, errcode.ErrorWithMsg(errcode.CodeBadRequest, "搜索关键词不能为空", requestID))
		return
	}

	// 调用 SearchRepo 进行搜索
	result, err := h.searchRepo.Search(keyword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errcode.Error(errcode.CodeDBError, requestID))
		return
	}

	// 未找到匹配结果
	if result == nil {
		c.JSON(http.StatusNotFound, errcode.ErrorWithMsg(errcode.CodeNotFound, "未找到匹配结果", requestID))
		return
	}

	// 返回搜索结果
	c.JSON(http.StatusOK, errcode.Success(result, requestID))
}
