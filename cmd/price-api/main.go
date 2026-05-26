// ============================================================
// price-api 服务入口文件
// ============================================================
// 这是"价格 API"微服务的 main 包，端口默认 8082。
//
// 该服务提供代币（ETH/BTC/SOL）的当前价格查询和历史价格曲线接口。
//
// 特色功能：使用 cron 定时任务，定期从 CoinGecko API 拉取最新价格，
// 保存到数据库并更新 Redis 缓存。
//
// Go 语言基础知识:
//   - package main：可执行程序的入口包
//   - func main()：程序启动时自动调用的入口函数
//   - goroutine：Go 的轻量级线程，用 go 关键字启动，如 go func() { ... }()
//   - channel：Go 的通道，用于 goroutine 之间的通信
//   - defer：延迟执行，常用于关闭资源
//   - interface{}：空接口，可以持有任意类型的值（类似 Java 的 Object）
// ============================================================
package main

import (
	"fmt"  // 格式化字符串
	"time" // 时间处理

	"blockexplore/internal/config"       // 配置管理
	"blockexplore/internal/handler"      // HTTP 处理器
	"blockexplore/internal/middleware"   // 中间件
	"blockexplore/internal/repository"   // 数据访问层
	"blockexplore/internal/service/price" // 价格服务
	"blockexplore/pkg/cache"            // Redis 缓存
	"blockexplore/pkg/logger"           // 日志

	"github.com/gin-gonic/gin"     // Gin Web 框架
	"github.com/robfig/cron/v3"    // cron 定时任务库，用于执行周期性任务
	"go.uber.org/zap"              // 日志库
	"gorm.io/driver/postgres"      // PostgreSQL 驱动
	"gorm.io/gorm"                 // ORM 库
)

// main 函数是 price-api 服务的入口点
func main() {
	// ============================================================
	// 第 1 步：加载配置
	// ============================================================
	cfg := config.Load()

	// ============================================================
	// 第 2 步：初始化日志
	// ============================================================
	logger.Init(cfg.Log.Level, cfg.Log.Format)
	logger.Info("price-api 服务启动中...")

	// ============================================================
	// 第 3 步：连接数据库
	// ============================================================
	// 数据库用于存储历史价格数据，方便绘制价格曲线
	db, err := gorm.Open(postgres.Open(cfg.DB.DSN()), &gorm.Config{})
	if err != nil {
		logger.Fatal("连接数据库失败", zap.Error(err))
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(cfg.DB.MaxOpenConns) // 最大连接数
	sqlDB.SetMaxIdleConns(cfg.DB.MaxIdleConns) // 空闲连接数
	logger.Info("数据库连接成功")

	// ============================================================
	// 第 4 步：初始化 Redis 缓存
	// ============================================================
	// 价格数据变化频繁，使用 Redis 缓存可以减轻数据库压力
	// 缓存策略：查询时先读缓存，缓存没有再读数据库
	cache.Init(cfg.Redis)
	redisClient := cache.GetClient()

	// ============================================================
	// 第 5 步：创建 PriceService
	// ============================================================
	// PriceService 负责价格相关的业务逻辑
	// 它依赖 PriceRepo（数据库）、RedisClient（缓存）和 API URL（外部价格源）
	priceRepo := repository.NewPriceRepo(db)
	priceService := price.NewPriceService(priceRepo, redisClient, cfg.Price.APIURL)

	// ============================================================
	// 第 6 步：启动定时价格同步
	// ============================================================
	// cron 是 Linux 的定时任务工具，这里使用 Go 版本的 cron 库
	// @every 30s 表示每 30 秒执行一次
	// cron.New() 创建一个新的 cron 调度器
	c := cron.New()

	// 构建 cron 表达式，例如: "@every 30s"
	syncInterval := fmt.Sprintf("@every %ds", cfg.Price.SyncInterval)

	// AddFunc 注册一个定时任务
	// func() { ... } 是一个匿名函数（闭包），类似于 JavaScript 的箭头函数
	c.AddFunc(syncInterval, func() {
		// SyncPrices 从 CoinGecko API 获取 ETH/BTC/SOL 的最新价格
		if err := priceService.SyncPrices(); err != nil {
			logger.Error("价格同步失败", zap.Error(err))
		}
	})

	// Start() 启动 cron 调度器（在后台 goroutine 中运行）
	// goroutine 是 Go 的轻量级线程，用 go 关键字启动
	c.Start()
	logger.Info("价格同步定时任务已启动",
		zap.Int("interval", cfg.Price.SyncInterval),
		zap.Duration("duration", time.Duration(cfg.Price.SyncInterval)*time.Second),
	)

	// ============================================================
	// 第 7 步：创建 Handler
	// ============================================================
	priceHandler := handler.NewPriceHandler(priceService)

	// ============================================================
	// 第 8 步：初始化路由
	// ============================================================
	gin.SetMode(gin.ReleaseMode) // 生产模式
	r := gin.New()               // 创建 Gin 引擎
	r.Use(middleware.RequestID()) // 请求 ID 中间件
	r.Use(middleware.CORS())      // 跨域中间件
	r.Use(gin.Recovery())        // panic 恢复中间件

	// 注册路由
	v1 := r.Group("/api/v1")
	{
		priceGroup := v1.Group("/price")
		{
			// :chain 是路径参数，例如 /api/v1/price/eth
			priceGroup.GET("/:chain", priceHandler.GetCurrentPrice)        // 获取当前价格
			priceGroup.GET("/:chain/history", priceHandler.GetPriceHistory) // 获取价格历史
		}
	}

	// ============================================================
	// 第 9 步：启动 HTTP 服务
	// ============================================================
	addr := fmt.Sprintf(":%d", cfg.Server.PriceAPIPort)
	logger.Info("price-api 已启动", zap.String("addr", addr))
	if err := r.Run(addr); err != nil {
		logger.Fatal("price-api 启动失败", zap.Error(err))
	}
}
