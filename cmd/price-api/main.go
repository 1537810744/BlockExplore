// price-api 价格 API 服务入口
// 提供代币价格查询和历史价格曲线接口
// 端口: 8082
package main

import (
	"fmt"
	"time"

	"blockexplore/internal/config"
	"blockexplore/internal/handler"
	"blockexplore/internal/middleware"
	"blockexplore/internal/repository"
	"blockexplore/internal/service/price"
	"blockexplore/pkg/cache"
	"blockexplore/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// 1. 加载配置
	cfg := config.Load()

	// 2. 初始化日志
	logger.Init(cfg.Log.Level, cfg.Log.Format)
	logger.Info("price-api 服务启动中...")

	// 3. 连接数据库
	db, err := gorm.Open(postgres.Open(cfg.DB.DSN()), &gorm.Config{})
	if err != nil {
		logger.Fatal("连接数据库失败", zap.Error(err))
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(cfg.DB.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.DB.MaxIdleConns)
	logger.Info("数据库连接成功")

	// 4. 初始化 Redis
	cache.Init(cfg.Redis)
	redisClient := cache.GetClient()

	// 5. 创建 Service
	priceRepo := repository.NewPriceRepo(db)
	priceService := price.NewPriceService(priceRepo, redisClient, cfg.Price.APIURL)

	// 6. 启动定时价格同步（使用 cron 表达式）
	c := cron.New()
	syncInterval := fmt.Sprintf("@every %ds", cfg.Price.SyncInterval)
	c.AddFunc(syncInterval, func() {
		if err := priceService.SyncPrices(); err != nil {
			logger.Error("价格同步失败", zap.Error(err))
		}
	})
	c.Start()
	logger.Info("价格同步定时任务已启动",
		zap.Int("interval", cfg.Price.SyncInterval),
		zap.Duration("duration", time.Duration(cfg.Price.SyncInterval)*time.Second),
	)

	// 7. 创建 Handler
	priceHandler := handler.NewPriceHandler(priceService)

	// 8. 初始化路由
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(middleware.RequestID())
	r.Use(middleware.CORS())
	r.Use(gin.Recovery())

	v1 := r.Group("/api/v1")
	{
		priceGroup := v1.Group("/price")
		{
			priceGroup.GET("/:chain", priceHandler.GetCurrentPrice)
			priceGroup.GET("/:chain/history", priceHandler.GetPriceHistory)
		}
	}

	// 9. 启动 HTTP 服务
	addr := fmt.Sprintf(":%d", cfg.Server.PriceAPIPort)
	logger.Info("price-api 已启动", zap.String("addr", addr))
	if err := r.Run(addr); err != nil {
		logger.Fatal("price-api 启动失败", zap.Error(err))
	}
}
