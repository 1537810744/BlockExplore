package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestRequestID_GeneratesNewID(t *testing.T) {
	r := gin.New()
	r.Use(RequestID())
	r.GET("/test", func(c *gin.Context) {
		rid := c.GetString("request_id")
		assert.NotEmpty(t, rid)
		c.String(200, rid)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.NotEmpty(t, w.Body.String())
	// 响应头应包含 X-Request-ID
	assert.NotEmpty(t, w.Header().Get("X-Request-ID"))
}

func TestRequestID_ReusesIncomingHeader(t *testing.T) {
	r := gin.New()
	r.Use(RequestID())
	r.GET("/test", func(c *gin.Context) {
		c.String(200, c.GetString("request_id"))
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", "incoming-id-abc")
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "incoming-id-abc", w.Body.String())
	assert.Equal(t, "incoming-id-abc", w.Header().Get("X-Request-ID"))
}

func TestCORS_SetsHeaders(t *testing.T) {
	r := gin.New()
	r.Use(CORS())
	r.GET("/test", func(c *gin.Context) { c.String(200, "ok") })

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "GET")
	assert.NotEmpty(t, w.Header().Get("Access-Control-Max-Age"))
}

func TestCORS_PreflightOptions(t *testing.T) {
	r := gin.New()
	r.Use(CORS())
	r.GET("/test", func(c *gin.Context) { c.String(200, "ok") })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	r.ServeHTTP(w, req)

	// 预检请求应直接返回 204
	assert.Equal(t, 204, w.Code)
}

func TestRateLimiter_AllowsBurst(t *testing.T) {
	rl := NewRateLimiter(100, 5)

	// 突发容量 5，前 5 次应允许
	for i := 0; i < 5; i++ {
		assert.True(t, rl.allow("1.2.3.4"), "第 %d 次请求应被允许", i+1)
	}
	// 第 6 次应被拒绝（桶空了）
	assert.False(t, rl.allow("1.2.3.4"))
}

func TestRateLimiter_IsolatesByIP(t *testing.T) {
	rl := NewRateLimiter(100, 2)
	assert.True(t, rl.allow("1.1.1.1"))
	assert.True(t, rl.allow("1.1.1.1"))
	assert.False(t, rl.allow("1.1.1.1"))

	// 不同 IP 应有独立的桶
	assert.True(t, rl.allow("2.2.2.2"))
}

func TestRateLimiter_RejectsAndAborts(t *testing.T) {
	rl := NewRateLimiter(1, 1)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("request_id", "test")
		c.Next()
	})
	r.Use(rl.RateLimit())
	r.GET("/test", func(c *gin.Context) { c.String(200, "ok") })

	// 第 1 次允许
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, httptest.NewRequest("GET", "/test", nil))
	assert.Equal(t, 200, w1.Code)

	// 第 2 次被限流（桶容量 1）
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest("GET", "/test", nil))
	assert.Equal(t, http.StatusTooManyRequests, w2.Code)
}
