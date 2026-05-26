// Package router 提供路由注册
// 使用 Gin 框架的路由组管理所有 API 路由
package router

import (
	"blockexplore/internal/handler"
	"blockexplore/internal/middleware"

	"github.com/gin-gonic/gin"
)

// Setup 初始化路由
// 注册所有 API 路由和中间件
func Setup(
	blockHandler *handler.BlockHandler,
	txHandler *handler.TxHandler,
	searchHandler *handler.SearchHandler,
	priceHandler *handler.PriceHandler,
) *gin.Engine {
	// 创建 Gin 引擎（生产模式）
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	// 注册全局中间件
	r.Use(middleware.RequestID()) // 请求 ID（最先执行）
	r.Use(middleware.CORS())      // 跨域支持
	r.Use(gin.Recovery())        // panic 恢复

	// 创建限流器（每秒 100 请求，突发容量 200）
	limiter := middleware.NewRateLimiter(100, 200)
	r.Use(limiter.RateLimit())

	// 健康检查接口
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// API v1 路由组
	v1 := r.Group("/api/v1")
	{
		// 区块相关接口
		blocks := v1.Group("/blocks")
		{
			blocks.GET("", blockHandler.GetBlockList)                              // 区块列表
			blocks.GET("/:block_number", blockHandler.GetBlockDetail)              // 区块详情
			blocks.GET("/:block_number/transactions", blockHandler.GetBlockTransactions) // 区块内交易
		}

		// 交易相关接口
		transactions := v1.Group("/transactions")
		{
			transactions.GET("/:hash", txHandler.GetTransactionDetail) // 交易详情
		}

		// 地址相关接口
		addresses := v1.Group("/addresses")
		{
			addresses.GET("/:address/transactions", txHandler.GetAddressTransactions) // 地址交易历史
		}

		// 搜索接口
		v1.GET("/search", searchHandler.Search)

		// 价格接口
		price := v1.Group("/price")
		{
			price.GET("/:chain", priceHandler.GetCurrentPrice)        // 当前价格
			price.GET("/:chain/history", priceHandler.GetPriceHistory) // 价格历史
		}
	}

	return r
}
