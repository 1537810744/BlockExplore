// search-api 搜索 API 服务入口
// 提供统一搜索功能：支持地址/交易哈希/区块号检索
// 端口: 8081
package main

import (
	"fmt"

	"blockexplore/internal/config"
	"blockexplore/internal/handler"
	"blockexplore/internal/middleware"
	"blockexplore/internal/repository"
	"blockexplore/pkg/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// 1. 加载配置
	cfg := config.Load()

	// 2. 初始化日志
	logger.Init(cfg.Log.Level, cfg.Log.Format)
	logger.Info("search-api 服务启动中...")

	// 3. 连接数据库
	db, err := gorm.Open(postgres.Open(cfg.DB.DSN()), &gorm.Config{})
	if err != nil {
		logger.Fatal("连接数据库失败", zap.Error(err))
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(cfg.DB.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.DB.MaxIdleConns)
	logger.Info("数据库连接成功")

	// 4. 创建 Repository 和 Handler
	searchRepo := repository.NewSearchRepo(db)
	searchHandler := handler.NewSearchHandler(searchRepo)

	// 5. 初始化路由
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(middleware.RequestID())
	r.Use(middleware.CORS())
	r.Use(gin.Recovery())

	v1 := r.Group("/api/v1")
	{
		v1.GET("/search", searchHandler.Search)
	}

	// 6. 启动 HTTP 服务
	addr := fmt.Sprintf(":%d", cfg.Server.SearchAPIPort)
	logger.Info("search-api 已启动", zap.String("addr", addr))
	if err := r.Run(addr); err != nil {
		logger.Fatal("search-api 启动失败", zap.Error(err))
	}
}
