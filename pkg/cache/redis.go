// Package cache 封装 Redis 缓存操作
package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"blockexplore/internal/config"
	"blockexplore/pkg/logger"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

// RedisClient Redis 客户端封装
type RedisClient struct {
	client *redis.Client // Redis 客户端实例
}

// 全局 Redis 客户端实例
var rdb *RedisClient

// Init 初始化 Redis 连接
// 传入 Redis 配置，创建连接池
func Init(cfg config.RedisConfig) {
	rdb = &RedisClient{
		client: redis.NewClient(&redis.Options{
			Addr:     cfg.Addr(),     // Redis 地址: host:port
			Password: cfg.Password,   // 密码（无密码留空）
			DB:       cfg.DB,         // 数据库编号
			PoolSize: cfg.PoolSize,   // 连接池大小
		}),
	}

	// 测试连接是否成功
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.client.Ping(ctx).Err(); err != nil {
		logger.Error("Redis 连接失败", zap.Error(err))
	} else {
		logger.Info("Redis 连接成功", zap.String("addr", cfg.Addr()))
	}
}

// GetClient 获取 Redis 客户端实例
func GetClient() *RedisClient {
	return rdb
}

// Set 设置缓存（带过期时间）
// key: 缓存键
// value: 缓存值（会被 JSON 序列化）
// expiration: 过期时间
func (r *RedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	// 将值序列化为 JSON 字节
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("序列化缓存值失败: %w", err)
	}
	return r.client.Set(ctx, key, data, expiration).Err()
}

// Get 获取缓存
// key: 缓存键
// dest: 目标对象（会被 JSON 反序列化填充）
func (r *RedisClient) Get(ctx context.Context, key string, dest interface{}) error {
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		return err // redis.Nil 表示键不存在
	}
	return json.Unmarshal(data, dest)
}

// Delete 删除缓存
func (r *RedisClient) Delete(ctx context.Context, keys ...string) error {
	return r.client.Del(ctx, keys...).Err()
}

// Exists 检查键是否存在
func (r *RedisClient) Exists(ctx context.Context, key string) (bool, error) {
	n, err := r.client.Exists(ctx, key).Result()
	return n > 0, err
}

// SetNX 仅在键不存在时设置（用于分布式锁等场景）
func (r *RedisClient) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return false, fmt.Errorf("序列化缓存值失败: %w", err)
	}
	return r.client.SetNX(ctx, key, data, expiration).Result()
}

// Incr 原子递增（用于计数器、限流等场景）
func (r *RedisClient) Incr(ctx context.Context, key string) (int64, error) {
	return r.client.Incr(ctx, key).Result()
}

// Expire 设置键的过期时间
func (r *RedisClient) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return r.client.Expire(ctx, key, expiration).Err()
}

// Close 关闭 Redis 连接
func (r *RedisClient) Close() error {
	if r.client != nil {
		return r.client.Close()
	}
	return nil
}
