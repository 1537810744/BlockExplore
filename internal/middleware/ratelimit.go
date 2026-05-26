package middleware

import (
	"net/http"
	"sync"
	"time"

	"blockexplore/pkg/errcode"

	"github.com/gin-gonic/gin"
)

// RateLimiter 令牌桶限流器
// 使用内存实现的简单令牌桶算法
// 生产环境建议使用 Redis 实现分布式限流
type RateLimiter struct {
	rate       int           // 每秒生成的令牌数
	bucketSize int           // 桶容量
	buckets    map[string]*bucket // 客户端令牌桶
	mu         sync.Mutex    // 互斥锁
}

// bucket 单个令牌桶
type bucket struct {
	tokens    int       // 当前令牌数
	lastTime  time.Time // 上次更新时间
}

// NewRateLimiter 创建限流器实例
// rate: 每秒允许的请求数
// bucketSize: 突发请求容量
func NewRateLimiter(rate, bucketSize int) *RateLimiter {
	rl := &RateLimiter{
		rate:       rate,
		bucketSize: bucketSize,
		buckets:    make(map[string]*bucket),
	}

	// 定期清理过期的令牌桶（每分钟）
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			rl.cleanup()
		}
	}()

	return rl
}

// RateLimit 限流中间件
// 根据客户端 IP 进行限流
func (rl *RateLimiter) RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取客户端 IP 作为限流 key
		key := c.ClientIP()

		// 检查是否有可用令牌
		if !rl.allow(key) {
			c.JSON(http.StatusTooManyRequests, errcode.ErrorWithMsg(
				http.StatusTooManyRequests,
				"请求过于频繁，请稍后再试",
				c.GetString("request_id"),
			))
			c.Abort()
			return
		}

		c.Next()
	}
}

// allow 检查是否允许请求
func (rl *RateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	b, exists := rl.buckets[key]
	if !exists {
		// 首次请求，创建新的令牌桶
		rl.buckets[key] = &bucket{
			tokens:   rl.bucketSize - 1, // 消耗一个令牌
			lastTime: time.Now(),
		}
		return true
	}

	// 计算应该添加的令牌数
	now := time.Now()
	elapsed := now.Sub(b.lastTime)
	tokensToAdd := int(elapsed.Seconds()) * rl.rate

	// 更新令牌数（不超过桶容量）
	b.tokens += tokensToAdd
	if b.tokens > rl.bucketSize {
		b.tokens = rl.bucketSize
	}
	b.lastTime = now

	// 检查是否有可用令牌
	if b.tokens <= 0 {
		return false
	}

	// 消耗一个令牌
	b.tokens--
	return true
}

// cleanup 清理长时间未使用的令牌桶
func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	threshold := time.Now().Add(-5 * time.Minute)
	for key, b := range rl.buckets {
		if b.lastTime.Before(threshold) {
			delete(rl.buckets, key)
		}
	}
}
