// ============================================================
// search-api 服务入口文件
// ============================================================
// 这是"搜索 API"微服务的 main 包。
//
// 该服务提供统一搜索功能：支持地址/交易哈希/区块号检索，端口默认 8081。
//
// 搜索逻辑：根据用户输入的关键词，自动识别是区块高度、交易哈希还是地址，
// 然后从数据库中查询匹配的结果。
//
// Go 语言基础知识:
//   - package main：表示这是一个可独立运行的程序
//   - import (...)：导入其他包，Go 会自动查找 GOPATH 或 go.mod 中定义的模块路径
//   - :=  操作符：短变量声明，同时声明变量并赋值，类型由右侧值自动推断
//   - if err != nil：Go 的标准错误处理模式，Go 没有 try-catch，通过返回值处理错误
//   - defer：延迟执行语句，在函数返回前执行，常用于资源清理（如关闭连接）
// ============================================================
package main

import (
	"fmt" // 格式化字符串

	"blockexplore/internal/config"      // 配置管理
	"blockexplore/internal/handler"     // HTTP 处理器
	"blockexplore/internal/middleware"  // 中间件（CORS、请求ID等）
	"blockexplore/internal/repository"  // 数据访问层
	"blockexplore/pkg/logger"          // 日志

	"github.com/gin-gonic/gin"    // Gin Web 框架，类似 Python 的 Flask
	"go.uber.org/zap"             // 高性能日志库
	"gorm.io/driver/postgres"     // PostgreSQL 驱动
	"gorm.io/gorm"                // ORM 库
)

// main 函数是 search-api 服务的入口点
func main() {
	// ============================================================
	// 第 1 步：加载配置
	// ============================================================
	// config.Load() 从 .env 文件和环境变量中读取配置
	cfg := config.Load()

	// ============================================================
	// 第 2 步：初始化日志
	// ============================================================
	// 日志级别从低到高：debug < info < warn < error
	// 设置为 info 时，debug 日志不会输出
	logger.Init(cfg.Log.Level, cfg.Log.Format)
	logger.Info("search-api 服务启动中...")

	// ============================================================
	// 第 3 步：连接数据库
	// ============================================================
	// GORM 的 Open 函数建立数据库连接池
	// 连接池：预先创建多个数据库连接，复用这些连接，避免每次查询都新建连接
	db, err := gorm.Open(postgres.Open(cfg.DB.DSN()), &gorm.Config{})
	if err != nil {
		logger.Fatal("连接数据库失败", zap.Error(err))
	}

	// 配置数据库连接池
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(cfg.DB.MaxOpenConns) // 连接池最大连接数
	sqlDB.SetMaxIdleConns(cfg.DB.MaxIdleConns) // 空闲连接数
	logger.Info("数据库连接成功")

	// ============================================================
	// 第 4 步：创建 Repository 和 Handler
	// ============================================================
	// SearchRepo 负责搜索相关的数据库操作
	searchRepo := repository.NewSearchRepo(db)
	// SearchHandler 负责处理搜索相关的 HTTP 请求
	searchHandler := handler.NewSearchHandler(searchRepo)

	// ============================================================
	// 第 5 步：初始化路由
	// ============================================================
	// gin.SetMode 设置 Gin 的运行模式
	// gin.ReleaseMode：生产模式，不输出调试信息，性能更好
	gin.SetMode(gin.ReleaseMode)

	// gin.New() 创建一个新的 Gin 引擎实例
	// Gin 是 Go 语言最流行的 Web 框架，类似于 Python 的 Flask
	r := gin.New()

	// Use() 注册中间件，中间件会在每个请求到达 Handler 之前执行
	// 多个中间件按注册顺序依次执行
	r.Use(middleware.RequestID()) // 为每个请求生成唯一 ID，用于日志追踪
	r.Use(middleware.CORS())      // 处理跨域请求（浏览器安全策略）
	r.Use(gin.Recovery())        // 捕获 panic 异常，防止程序崩溃

	// ============================================================
	// 第 6 步：注册路由
	// ============================================================
	// Group() 创建路由组，所有以 /api/v1 开头的路由都归这个组管理
	// 这样可以统一添加前缀，方便版本管理
	v1 := r.Group("/api/v1")
	{
		// GET() 注册 GET 请求的路由
		// /api/v1/search?q=xxx -> searchHandler.Search
		v1.GET("/search", searchHandler.Search)
	}

	// ============================================================
	// 第 7 步：启动 HTTP 服务
	// ============================================================
	// fmt.Sprintf 将端口号格式化为 ":8081" 这样的字符串
	addr := fmt.Sprintf(":%d", cfg.Server.SearchAPIPort)
	logger.Info("search-api 已启动", zap.String("addr", addr))

	// Run() 启动 HTTP 服务，监听指定端口
	// 这是一个阻塞调用，程序会一直运行在这里
	if err := r.Run(addr); err != nil {
		logger.Fatal("search-api 启动失败", zap.Error(err))
	}
}
