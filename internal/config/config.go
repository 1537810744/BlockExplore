// Package config 提供统一的配置管理功能
// 使用 Viper 库读取 .env 文件和环境变量
package config

import (
	"fmt"
	"log"

	"github.com/spf13/viper"
)

// Config 应用总配置结构体，包含所有子配置
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

// AppConfig 应用基础配置
type AppConfig struct {
	Name  string // 应用名称
	Env   string // 运行环境：development / production
	Debug bool   // 是否开启调试模式
}

// DBConfig PostgreSQL 数据库配置
type DBConfig struct {
	Host         string // 数据库主机地址
	Port         int    // 数据库端口
	User         string // 数据库用户名
	Password     string // 数据库密码
	DBName       string // 数据库名称
	SSLMode      string // SSL 模式
	MaxOpenConns int    // 最大打开连接数
	MaxIdleConns int    // 最大空闲连接数
}

// DSN 返回 PostgreSQL 连接字符串
func (c DBConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode)
}

// RedisConfig Redis 缓存配置
type RedisConfig struct {
	Host     string // Redis 主机地址
	Port     int    // Redis 端口
	Password string // Redis 密码（无密码留空）
	DB       int    // Redis 数据库编号（0-15）
	PoolSize int    // 连接池大小
}

// Addr 返回 Redis 地址: host:port
func (c RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// KafkaConfig Kafka 消息队列配置
type KafkaConfig struct {
	Brokers       []string // Kafka Broker 地址列表
	ETHTopic      string   // 以太坊区块数据 Topic
	RTCTopic      string   // 比特币区块数据 Topic
	SOLTopic      string   // Solana 区块数据 Topic
	ConsumerGroup string   // 消费者组名称
}

// ChainConfig 通用链配置（ETH/SOL）
type ChainConfig struct {
	RPCURL       string // RPC 节点 URL
	SyncInterval int    // 同步间隔（秒）
}

// BTCConfig 比特币链配置（需要额外的用户名密码认证）
type BTCConfig struct {
	RPCURL       string // RPC 节点 URL
	RPCUser      string // RPC 用户名
	RPCPassword  string // RPC 密码
	SyncInterval int    // 同步间隔（秒）
}

// PriceConfig 价格 API 配置
type PriceConfig struct {
	APIURL       string // 价格 API 地址
	SyncInterval int    // 同步间隔（秒）
}

// ServerConfig API 服务端口配置
type ServerConfig struct {
	QueryAPIPort  int // 查询 API 端口
	SearchAPIPort int // 搜索 API 端口
	PriceAPIPort  int // 价格 API 端口
}

// LogConfig 日志配置
type LogConfig struct {
	Level  string // 日志级别：debug/info/warn/error
	Format string // 日志格式：json/console
}

// Load 从环境变量和 .env 文件加载配置
// 优先级：环境变量 > .env 文件 > 默认值
func Load() *Config {
	// 设置配置文件名和路径
	viper.SetConfigName(".env") // 配置文件名（不带扩展名）
	viper.SetConfigType("env")  // 配置文件类型
	viper.AddConfigPath(".")    // 在当前目录查找
	viper.AddConfigPath("..")   // 也在上级目录查找（用于 cmd 子目录运行时）

	// 允许读取环境变量（自动覆盖配置文件中的同名配置）
	viper.AutomaticEnv()

	// 读取配置文件（如果文件不存在也不报错，因为有环境变量兜底）
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			log.Printf("警告: 读取配置文件失败: %v", err)
		}
	}

	// 设置所有默认值
	setDefaults()

	// 解析配置到结构体
	cfg := &Config{}
	cfg.App.Name = viper.GetString("APP_NAME")
	cfg.App.Env = viper.GetString("APP_ENV")
	cfg.App.Debug = viper.GetBool("APP_DEBUG")

	cfg.DB.Host = viper.GetString("DB_HOST")
	cfg.DB.Port = viper.GetInt("DB_PORT")
	cfg.DB.User = viper.GetString("DB_USER")
	cfg.DB.Password = viper.GetString("DB_PASSWORD")
	cfg.DB.DBName = viper.GetString("DB_NAME")
	cfg.DB.SSLMode = viper.GetString("DB_SSLMODE")
	cfg.DB.MaxOpenConns = viper.GetInt("DB_MAX_OPEN_CONNS")
	cfg.DB.MaxIdleConns = viper.GetInt("DB_MAX_IDLE_CONNS")

	cfg.Redis.Host = viper.GetString("REDIS_HOST")
	cfg.Redis.Port = viper.GetInt("REDIS_PORT")
	cfg.Redis.Password = viper.GetString("REDIS_PASSWORD")
	cfg.Redis.DB = viper.GetInt("REDIS_DB")
	cfg.Redis.PoolSize = viper.GetInt("REDIS_POOL_SIZE")

	cfg.Kafka.Brokers = viper.GetStringSlice("KAFKA_BROKERS")
	cfg.Kafka.ETHTopic = viper.GetString("KAFKA_ETH_TOPIC")
	cfg.Kafka.RTCTopic = viper.GetString("KAFKA_BTC_TOPIC")
	cfg.Kafka.SOLTopic = viper.GetString("KAFKA_SOL_TOPIC")
	cfg.Kafka.ConsumerGroup = viper.GetString("KAFKA_CONSUMER_GROUP")

	cfg.ETH.RPCURL = viper.GetString("ETH_RPC_URL")
	cfg.ETH.SyncInterval = viper.GetInt("ETH_SYNC_INTERVAL")

	cfg.BTC.RPCURL = viper.GetString("BTC_RPC_URL")
	cfg.BTC.RPCUser = viper.GetString("BTC_RPC_USER")
	cfg.BTC.RPCPassword = viper.GetString("BTC_RPC_PASSWORD")
	cfg.BTC.SyncInterval = viper.GetInt("BTC_SYNC_INTERVAL")

	cfg.SOL.RPCURL = viper.GetString("SOL_RPC_URL")
	cfg.SOL.SyncInterval = viper.GetInt("SOL_SYNC_INTERVAL")

	cfg.Price.APIURL = viper.GetString("PRICE_API_URL")
	cfg.Price.SyncInterval = viper.GetInt("PRICE_SYNC_INTERVAL")

	cfg.Server.QueryAPIPort = viper.GetInt("QUERY_API_PORT")
	cfg.Server.SearchAPIPort = viper.GetInt("SEARCH_API_PORT")
	cfg.Server.PriceAPIPort = viper.GetInt("PRICE_API_PORT")

	cfg.Log.Level = viper.GetString("LOG_LEVEL")
	cfg.Log.Format = viper.GetString("LOG_FORMAT")

	return cfg
}

// setDefaults 设置所有配置项的默认值
// 当配置文件和环境变量都没有设置时，使用这些默认值
func setDefaults() {
	// 应用配置默认值
	viper.SetDefault("APP_NAME", "blockexplore")
	viper.SetDefault("APP_ENV", "development")
	viper.SetDefault("APP_DEBUG", true)

	// 数据库配置默认值（Docker 内部使用 postgres 作为主机名）
	viper.SetDefault("DB_HOST", "postgres")
	viper.SetDefault("DB_PORT", 5432)
	viper.SetDefault("DB_USER", "blockexplore")
	viper.SetDefault("DB_PASSWORD", "blockexplore123")
	viper.SetDefault("DB_NAME", "blockexplore")
	viper.SetDefault("DB_SSLMODE", "disable")
	viper.SetDefault("DB_MAX_OPEN_CONNS", 100)
	viper.SetDefault("DB_MAX_IDLE_CONNS", 10)

	// Redis 配置默认值（Docker 内部使用 redis 作为主机名）
	viper.SetDefault("REDIS_HOST", "redis")
	viper.SetDefault("REDIS_PORT", 6379)
	viper.SetDefault("REDIS_PASSWORD", "")
	viper.SetDefault("REDIS_DB", 0)
	viper.SetDefault("REDIS_POOL_SIZE", 100)

	// Kafka 配置默认值
	viper.SetDefault("KAFKA_BROKERS", []string{"kafka:9092"})
	viper.SetDefault("KAFKA_ETH_TOPIC", "block.raw.eth")
	viper.SetDefault("KAFKA_BTC_TOPIC", "block.raw.btc")
	viper.SetDefault("KAFKA_SOL_TOPIC", "block.raw.sol")
	viper.SetDefault("KAFKA_CONSUMER_GROUP", "block-processor-group")

	// 各链同步间隔默认值（秒）
	// ETH: 约12秒一个区块
	// BTC: 约10分钟一个区块
	// SOL: 约0.4秒一个区块
	viper.SetDefault("ETH_SYNC_INTERVAL", 12)
	viper.SetDefault("BTC_SYNC_INTERVAL", 600)
	viper.SetDefault("SOL_SYNC_INTERVAL", 1)

	// 价格 API 默认值
	viper.SetDefault("PRICE_API_URL", "https://api.coingecko.com/api/v3")
	viper.SetDefault("PRICE_SYNC_INTERVAL", 30)

	// 服务端口默认值
	viper.SetDefault("QUERY_API_PORT", 8080)
	viper.SetDefault("SEARCH_API_PORT", 8081)
	viper.SetDefault("PRICE_API_PORT", 8082)

	// 日志配置默认值
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("LOG_FORMAT", "json")
}
