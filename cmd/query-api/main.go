// query-api 查询 API 服务入口
// 提供区块、交易、地址的 RESTful 查询接口
// 端口: 8080
package main

import (
	"fmt"

	"blockexplore/internal/config"
	"blockexplore/internal/handler"
	"blockexplore/internal/repository"
	"blockexplore/internal/router"
	"blockexplore/internal/service/query"
	"blockexplore/pkg/cache"
	"blockexplore/pkg/logger"

	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// 1. 加载配置
	cfg := config.Load()

	// 2. 初始化日志
	logger.Init(cfg.Log.Level, cfg.Log.Format)
	logger.Info("query-api 服务启动中...")

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

	// 5. 创建 Repository
	blockRepo := repository.NewBlockRepo(db)
	txRepo := repository.NewTxRepo(db)

	// 6. 创建 Service
	queryService := query.NewQueryService(blockRepo, txRepo, redisClient)

	// 7. 创建 Handler
	blockHandler := handler.NewBlockHandler(queryService)
	txHandler := handler.NewTxHandler(queryService)
	searchHandler := handler.NewSearchHandler(repository.NewSearchRepo(db))
	priceHandler := handler.NewPriceHandler(nil) // price-api 单独服务

	// 8. 初始化路由
	r := router.Setup(blockHandler, txHandler, searchHandler, priceHandler)

	// 9. 启动 HTTP 服务
	addr := fmt.Sprintf(":%d", cfg.Server.QueryAPIPort)
	logger.Info("query-api 已启动", zap.String("addr", addr))
	if err := r.Run(addr); err != nil {
		logger.Fatal("query-api 启动失败", zap.Error(err))
	}
}
