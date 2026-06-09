// ============================================================
// Package config 提供统一的配置管理功能
// ============================================================
// 该包负责从 .env 文件和环境变量中读取配置，供整个应用使用。
//
// 配置优先级：环境变量 > .env 文件 > 默认值
//
// 使用的第三方库：
//   - github.com/spf13/viper：配置管理库，支持多种配置格式（JSON、YAML、ENV 等）
//
// Go 语言基础知识:
//   - package：Go 的包管理机制，每个目录是一个包
//   - struct：结构体，类似于 Java 的 class，用于定义数据结构
//   - mapstructure：结构体标签，用于配置映射
//   - func (c DBConfig) DSN()：方法定义，c 是接收者（receiver），类似于 this
//   - viper.GetString()：从配置中获取字符串值
// ============================================================
package config

import (
	"fmt"  // 格式化字符串
	"log"  // 标准日志库

	"github.com/spf13/viper" // 配置管理库，由 spf13 开发
)

// ============================================================
// Config 应用总配置结构体
// ============================================================
// 包含所有子配置：应用、数据库、Redis、Kafka、各链配置等
// mapstructure:",squash" 表示将嵌套结构体的字段展平到父结构体中
type Config struct {
	App    AppConfig    `mapstructure:",squash"` // 应用基础配置
	DB     DBConfig     `mapstructure:",squash"` // 数据库配置
	Redis  RedisConfig  `mapstructure:",squash"` // Redis 缓存配置
	Kafka  KafkaConfig  `mapstructure:",squash"` // Kafka 消息队列配置
	ETH    ChainConfig  `mapstructure:",squash"` // 以太坊链配置
	BTC    BTCConfig    `mapstructure:",squash"` // 比特币链配置
	SOL    ChainConfig  `mapstructure:",squash"` // Solana 链配置
	Price  PriceConfig  `mapstructure:",squash"` // 价格 API 配置
	Server ServerConfig `mapstructure:",squash"` // API 服务端口配置
	Log    LogConfig    `mapstructure:",squash"` // 日志配置
}

// ============================================================
// AppConfig 应用基础配置
// ============================================================
type AppConfig struct {
	Name  string // 应用名称，例如 "blockexplore"
	Env   string // 运行环境：development（开发）/ production（生产）
	Debug bool   // 是否开启调试模式（true 会输出更多日志）
}

// ============================================================
// DBConfig PostgreSQL 数据库配置
// ============================================================
// PostgreSQL 是一个功能强大的开源关系型数据库
type DBConfig struct {
	Host         string // 数据库主机地址，例如 "localhost" 或 Docker 服务名 "postgres"
	Port         int    // 数据库端口，默认 5432
	User         string // 数据库用户名
	Password     string // 数据库密码
	DBName       string // 数据库名称
	SSLMode      string // SSL 模式："disable"（不加密）/ "require"（要求加密）
	MaxOpenConns int    // 最大打开连接数（并发连接上限）
	MaxIdleConns int    // 最大空闲连接数（空闲时保持的连接数）
}

// DSN 方法返回 PostgreSQL 连接字符串
// DSN = Data Source Name（数据源名称）
// 格式: "host=localhost port=5432 user=xxx password=xxx dbname=xxx sslmode=disable"
// 方法定义语法：func (接收者类型) 方法名(参数) 返回值 { ... }
// 接收者 c 类似于 Java 的 this，表示当前实例
func (c DBConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode)
}

// ============================================================
// RedisConfig Redis 缓存配置
// ============================================================
// Redis 是一个高性能的内存数据库，常用于缓存和消息队列
type RedisConfig struct {
	Host     string // Redis 主机地址
	Port     int    // Redis 端口，默认 6379
	Password string // Redis 密码（无密码留空）
	DB       int    // Redis 数据库编号（0-15，共 16 个数据库）
	PoolSize int    // 连接池大小（同时保持的最大连接数）
}

// Addr 方法返回 Redis 地址，格式: "host:port"
func (c RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// ============================================================
// KafkaConfig Kafka 消息队列配置
// ============================================================
// Kafka 是分布式消息队列，用于在服务之间传递消息
// Topic 是消息的分类，类似于邮件的收件箱
type KafkaConfig struct {
	Brokers       []string // Kafka Broker 地址列表，例如 ["kafka:9092"]
	ETHTopic      string   // 以太坊区块数据 Topic，例如 "block.raw.eth"
	RTCTopic      string   // 比特币区块数据 Topic，例如 "block.raw.btc"
	SOLTopic      string   // Solana 区块数据 Topic，例如 "block.raw.sol"
	ConsumerGroup string   // 消费者组名称，同一组内的消费者共享消息
}

// ============================================================
// ChainConfig 通用链配置（ETH/SOL）
// ============================================================
// 以太坊和 Solana 的配置结构相同，共用这个结构体
type ChainConfig struct {
	RPCURL       string // RPC 节点 URL，例如 "http://localhost:8545"
	SyncInterval int    // 同步间隔（秒），每隔多少秒拉取一次新区块
}

// ============================================================
// BTCConfig 比特币链配置
// ============================================================
// 比特币节点需要额外的用户名密码认证（HTTP Basic Auth）
type BTCConfig struct {
	RPCURL       string // RPC 节点 URL
	RPCUser      string // RPC 用户名
	RPCPassword  string // RPC 密码
	SyncInterval int    // 同步间隔（秒）
}

// ============================================================
// PriceConfig 价格 API 配置
// ============================================================
type PriceConfig struct {
	APIURL       string // 价格 API 地址，例如 "https://api.coingecko.com/api/v3"
	SyncInterval int    // 价格同步间隔（秒）
}

// ============================================================
// ServerConfig API 服务端口配置
// ============================================================
type ServerConfig struct {
	QueryAPIPort  int // 查询 API 端口，默认 8080
	SearchAPIPort int // 搜索 API 端口，默认 8081
	PriceAPIPort  int // 价格 API 端口，默认 8082
}

// ============================================================
// LogConfig 日志配置
// ============================================================
type LogConfig struct {
	Level  string // 日志级别：debug（调试）/ info（信息）/ warn（警告）/ error（错误）
	Format string // 日志格式：json（JSON 格式，适合生产环境）/ console（控制台格式，适合开发）
}

// ============================================================
// Load 函数：从环境变量和 .env 文件加载配置
// ============================================================
// 优先级：环境变量 > .env 文件 > 默认值
// 返回值：*Config 指针类型，Go 中指针传递避免拷贝整个结构体
func Load() *Config {
	// ---- 设置配置文件名和路径 ----
	viper.SetConfigName(".env") // 配置文件名（不带扩展名）
	viper.SetConfigType("env")  // 配置文件类型为 .env 格式
	viper.AddConfigPath(".")    // 在当前目录查找配置文件
	viper.AddConfigPath("..")   // 也在上级目录查找（用于 cmd 子目录运行时）

	// ---- 允许读取环境变量 ----
	// AutomaticEnv() 会自动读取环境变量，覆盖配置文件中的同名配置
	// 例如：环境变量 DB_HOST 会覆盖 .env 文件中的 DB_HOST
	viper.AutomaticEnv()

	// ---- 读取配置文件 ----
	// 如果文件不存在也不报错，因为有环境变量和默认值兜底
	if err := viper.ReadInConfig(); err != nil {
		// 类型断言：检查错误是否是 ConfigFileNotFoundError 类型
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			log.Printf("警告: 读取配置文件失败: %v", err)
		}
	}

	// ---- 设置所有默认值 ----
	setDefaults()

	// ---- 解析配置到结构体 ----
	// viper.GetString() 从配置中获取字符串值
	// 如果配置中没有设置，返回默认值
	cfg := &Config{}

	// 应用配置
	cfg.App.Name = viper.GetString("APP_NAME")
	cfg.App.Env = viper.GetString("APP_ENV")
	cfg.App.Debug = viper.GetBool("APP_DEBUG")

	// 数据库配置
	cfg.DB.Host = viper.GetString("DB_HOST")
	cfg.DB.Port = viper.GetInt("DB_PORT")
	cfg.DB.User = viper.GetString("DB_USER")
	cfg.DB.Password = viper.GetString("DB_PASSWORD")
	cfg.DB.DBName = viper.GetString("DB_NAME")
	cfg.DB.SSLMode = viper.GetString("DB_SSLMODE")
	cfg.DB.MaxOpenConns = viper.GetInt("DB_MAX_OPEN_CONNS")
	cfg.DB.MaxIdleConns = viper.GetInt("DB_MAX_IDLE_CONNS")

	// Redis 配置
	cfg.Redis.Host = viper.GetString("REDIS_HOST")
	cfg.Redis.Port = viper.GetInt("REDIS_PORT")
	cfg.Redis.Password = viper.GetString("REDIS_PASSWORD")
	cfg.Redis.DB = viper.GetInt("REDIS_DB")
	cfg.Redis.PoolSize = viper.GetInt("REDIS_POOL_SIZE")

	// Kafka 配置
	cfg.Kafka.Brokers = viper.GetStringSlice("KAFKA_BROKERS") // GetStringSlice 获取字符串切片
	cfg.Kafka.ETHTopic = viper.GetString("KAFKA_ETH_TOPIC")
	cfg.Kafka.RTCTopic = viper.GetString("KAFKA_BTC_TOPIC")
	cfg.Kafka.SOLTopic = viper.GetString("KAFKA_SOL_TOPIC")
	cfg.Kafka.ConsumerGroup = viper.GetString("KAFKA_CONSUMER_GROUP")

	// 以太坊配置
	cfg.ETH.RPCURL = viper.GetString("ETH_RPC_URL")
	cfg.ETH.SyncInterval = viper.GetInt("ETH_SYNC_INTERVAL")

	// 比特币配置
	cfg.BTC.RPCURL = viper.GetString("BTC_RPC_URL")
	cfg.BTC.RPCUser = viper.GetString("BTC_RPC_USER")
	cfg.BTC.RPCPassword = viper.GetString("BTC_RPC_PASSWORD")
	cfg.BTC.SyncInterval = viper.GetInt("BTC_SYNC_INTERVAL")

	// Solana 配置
	cfg.SOL.RPCURL = viper.GetString("SOL_RPC_URL")
	cfg.SOL.SyncInterval = viper.GetInt("SOL_SYNC_INTERVAL")

	// 价格 API 配置
	cfg.Price.APIURL = viper.GetString("PRICE_API_URL")
	cfg.Price.SyncInterval = viper.GetInt("PRICE_SYNC_INTERVAL")

	// 服务端口配置
	cfg.Server.QueryAPIPort = viper.GetInt("QUERY_API_PORT")
	cfg.Server.SearchAPIPort = viper.GetInt("SEARCH_API_PORT")
	cfg.Server.PriceAPIPort = viper.GetInt("PRICE_API_PORT")

	// 日志配置
	cfg.Log.Level = viper.GetString("LOG_LEVEL")
	cfg.Log.Format = viper.GetString("LOG_FORMAT")

	return cfg
}

// ============================================================
// setDefaults 函数：设置所有配置项的默认值
// ============================================================
// 当配置文件和环境变量都没有设置时，使用这些默认值
func setDefaults() {
	// ---- 应用配置默认值 ----
	viper.SetDefault("APP_NAME", "blockexplore")
	viper.SetDefault("APP_ENV", "development")
	viper.SetDefault("APP_DEBUG", true)

	// ---- 数据库配置默认值 ----
	// Docker 内部使用 "postgres" 作为主机名（Docker 服务名）
	viper.SetDefault("DB_HOST", "postgres")
	viper.SetDefault("DB_PORT", 5432) // PostgreSQL 默认端口
	viper.SetDefault("DB_USER", "blockexplore")
	viper.SetDefault("DB_PASSWORD", "blockexplore123")
	viper.SetDefault("DB_NAME", "blockexplore")
	viper.SetDefault("DB_SSLMODE", "disable")
	viper.SetDefault("DB_MAX_OPEN_CONNS", 100) // 最大连接数
	viper.SetDefault("DB_MAX_IDLE_CONNS", 10)  // 空闲连接数

	// ---- Redis 配置默认值 ----
	// Docker 内部使用 "redis" 作为主机名
	viper.SetDefault("REDIS_HOST", "redis")
	viper.SetDefault("REDIS_PORT", 6379) // Redis 默认端口
	viper.SetDefault("REDIS_PASSWORD", "")
	viper.SetDefault("REDIS_DB", 0)       // 使用 0 号数据库
	viper.SetDefault("REDIS_POOL_SIZE", 100)

	// ---- Kafka 配置默认值 ----
	viper.SetDefault("KAFKA_BROKERS", []string{"kafka:9092"})
	viper.SetDefault("KAFKA_ETH_TOPIC", "block.raw.eth")
	viper.SetDefault("KAFKA_BTC_TOPIC", "block.raw.btc")
	viper.SetDefault("KAFKA_SOL_TOPIC", "block.raw.sol")
	viper.SetDefault("KAFKA_CONSUMER_GROUP", "block-processor-group")

	// ---- 各链同步间隔默认值（秒）----
	// ETH: 约 12 秒一个区块
	// BTC: 约 10 分钟一个区块
	// SOL: 约 0.4 秒一个区块，但公开 RPC 限流，5 秒比较安全
	viper.SetDefault("ETH_SYNC_INTERVAL", 12)
	viper.SetDefault("BTC_SYNC_INTERVAL", 600)
	viper.SetDefault("SOL_SYNC_INTERVAL", 5)

	// ---- 价格 API 默认值 ----
	// CoinGecko 免费 API 限流严格，120 秒比较安全
	viper.SetDefault("PRICE_API_URL", "https://api.coingecko.com/api/v3")
	viper.SetDefault("PRICE_SYNC_INTERVAL", 120)

	// ---- 服务端口默认值 ----
	viper.SetDefault("QUERY_API_PORT", 8080)
	viper.SetDefault("SEARCH_API_PORT", 8081)
	viper.SetDefault("PRICE_API_PORT", 8082)

	// ---- 日志配置默认值 ----
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("LOG_FORMAT", "json")
}
