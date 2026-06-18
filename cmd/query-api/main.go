// ============================================================
// query-api 服务入口文件
// ============================================================
// 这是"查询 API"微服务的 main 包，程序从这里开始执行。
//
// 该服务提供区块、交易、地址的 RESTful 查询接口，端口默认 8080。
//
// 启动流程:
//   1. 加载配置（从 .env 文件读取）
//   2. 初始化日志系统
//   3. 连接 PostgreSQL 数据库
//   4. 初始化 Redis 缓存
//   5. 创建各层实例（Repository -> Service -> Handler）
//   6. 注册路由
//   7. 启动 HTTP 服务，监听指定端口
//
// Go 语言基础知识:
//   - package main 表示这是一个可执行程序（不是库）
//   - func main() 是程序入口函数，类似于 Java/C 的 main 函数
//   - import 用于导入其他包（模块），类似于 Python 的 import
//   - := 是短变量声明，自动推断类型，例如 name := "张三" 等同于 var name string = "张三"
//   - if err != nil 是 Go 的惯用错误处理模式
// ============================================================
package main // main 包表示这是一个可执行程序

import (
	"fmt" // fmt 包用于格式化字符串，类似 Python 的 format()

	// ---- 以下是我们自己写的内部包 ----
	"blockexplore/internal/config"       // 配置管理，读取 .env 文件
	"blockexplore/internal/handler"      // HTTP 请求处理器（类似 Spring 的 Controller）
	"blockexplore/internal/repository"   // 数据访问层（类似 Spring 的 DAO/Repository）
	"blockexplore/internal/router"       // 路由注册，定义 URL 和处理器的映射
	"blockexplore/internal/service/price" // 价格业务逻辑层
	"blockexplore/internal/service/query" // 查询业务逻辑层
	"blockexplore/pkg/cache"            // Redis 缓存封装
	"blockexplore/pkg/logger"           // 日志封装

	// ---- 以下是第三方库 ----
	"go.uber.org/zap"           // Uber 开发的高性能日志库
	"gorm.io/driver/postgres"   // GORM 的 PostgreSQL 驱动
	"gorm.io/gorm"              // GORM ORM 库，用于操作数据库（类似 Java 的 Hibernate）
)

// main 函数是程序的入口点
// Go 程序启动时会自动调用 main 包中的 main 函数
func main() {
	// ============================================================
	// 第 1 步：加载配置
	// ============================================================
	// config.Load() 会读取 .env 文件和环境变量，返回配置结构体
	// := 是短变量声明，Go 会自动推断 cfg 的类型为 *config.Config
	cfg := config.Load()

	// ============================================================
	// 第 2 步：初始化日志系统
	// ============================================================
	// zap 是 Uber 开发的高性能日志库，比标准库 log 快很多
	// Init 函数接收日志级别和格式参数
	logger.Init(cfg.Log.Level, cfg.Log.Format)
	logger.Info("query-api 服务启动中...") // Info 级别日志，用于记录正常运行信息

	// ============================================================
	// 第 3 步：连接 PostgreSQL 数据库
	// ============================================================
	// GORM 是 Go 语言最流行的 ORM 库
	// gorm.Open() 打开数据库连接，第一个参数是数据库驱动，第二个是 GORM 配置
	// cfg.DB.DSN() 返回数据库连接字符串，格式如: "host=localhost port=5432 user=xxx password=xxx dbname=xxx sslmode=disable"
	db, err := gorm.Open(postgres.Open(cfg.DB.DSN()), &gorm.Config{})
	if err != nil {
		// logger.Fatal 会记录错误日志并终止程序（os.Exit(1)）
		// zap.Error(err) 将错误对象添加到日志字段中
		logger.Fatal("连接数据库失败", zap.Error(err))
	}

	// 获取底层的 sql.DB 对象，用于配置连接池参数
	// db.DB() 返回 GORM 底层使用的 *sql.DB 对象
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(cfg.DB.MaxOpenConns) // 设置最大打开连接数（并发连接上限）
	sqlDB.SetMaxIdleConns(cfg.DB.MaxIdleConns) // 设置最大空闲连接数（空闲时保持的连接数）
	logger.Info("数据库连接成功")

	// ============================================================
	// 第 4 步：初始化 Redis 缓存
	// ============================================================
	// Redis 是一个内存数据库，用于缓存热点数据，减轻数据库压力
	// cache.Init() 创建 Redis 客户端连接
	cache.Init(cfg.Redis)
	redisClient := cache.GetClient() // 获取 Redis 客户端实例

	// ============================================================
	// 第 5 步：创建各层实例（依赖注入）
	// ============================================================
	// 这里采用分层架构：Repository（数据访问） -> Service（业务逻辑） -> Handler（请求处理）
	// 类似于 Java Spring 的 DAO -> Service -> Controller

	// 创建 Repository 层（数据访问层，负责与数据库交互）
	blockRepo := repository.NewBlockRepo(db) // 区块数据访问
	txRepo := repository.NewTxRepo(db)       // 交易数据访问
	priceRepo := repository.NewPriceRepo(db) // 价格数据访问

	// 创建 Service 层（业务逻辑层，处理业务逻辑）
	// QueryService 依赖 blockRepo 和 txRepo，以及 Redis 缓存
	queryService := query.NewQueryService(blockRepo, txRepo, redisClient)
	// PriceService 依赖 priceRepo、Redis 缓存和 CoinGecko API 地址
	// query-api 作为网关也提供价格查询，避免 nil 导致 panic
	priceService := price.NewPriceService(priceRepo, redisClient, cfg.Price.APIURL)

	// 创建 Handler 层（请求处理层，接收 HTTP 请求并返回响应）
	blockHandler := handler.NewBlockHandler(queryService)   // 区块相关接口
	txHandler := handler.NewTxHandler(queryService)         // 交易相关接口
	searchHandler := handler.NewSearchHandler(repository.NewSearchRepo(db)) // 搜索接口
	priceHandler := handler.NewPriceHandler(priceService)   // 价格接口（query-api 作为网关统一提供服务）

	// ============================================================
	// 第 6 步：初始化路由
	// ============================================================
	// 路由定义了 URL 路径和处理函数的映射关系
	// 例如: GET /api/v1/blocks -> blockHandler.GetBlockList
	r := router.Setup(blockHandler, txHandler, searchHandler, priceHandler)

	// ============================================================
	// 第 7 步：启动 HTTP 服务
	// ============================================================
	// fmt.Sprintf 格式化字符串，生成监听地址如 ":8080"
	addr := fmt.Sprintf(":%d", cfg.Server.QueryAPIPort)
	logger.Info("query-api 已启动", zap.String("addr", addr))

	// r.Run() 启动 HTTP 服务，监听指定端口
	// 这是一个阻塞调用，会一直运行直到程序被终止
	if err := r.Run(addr); err != nil {
		logger.Fatal("query-api 启动失败", zap.Error(err))
	}
}
