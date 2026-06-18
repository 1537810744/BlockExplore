package errcode

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetMsg(t *testing.T) {
	tests := []struct {
		name string
		code int
		want string
	}{
		{"成功", CodeSuccess, "success"},
		{"参数错误", CodeBadRequest, "请求参数错误"},
		{"未找到", CodeNotFound, "资源不存在"},
		{"内部错误", CodeInternalError, "服务器内部错误"},
		{"数据库错误", CodeDBError, "数据库错误"},
		{"缓存错误", CodeCacheError, "缓存错误"},
		{"RPC 错误", CodeRPCError, "区块链 RPC 调用错误"},
		{"Kafka 错误", CodeKafkaError, "消息队列错误"},
		{"未知错误码", 999, "未知错误"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, GetMsg(tt.code))
		})
	}
}

func TestSuccess(t *testing.T) {
	data := map[string]int{"count": 42}
	resp := Success(data, "req-123")

	assert.NotNil(t, resp)
	assert.Equal(t, CodeSuccess, resp.Code)
	assert.Equal(t, "success", resp.Message)
	assert.Equal(t, "req-123", resp.RequestID)
	assert.Equal(t, data, resp.Data)
}

func TestError(t *testing.T) {
	resp := Error(CodeNotFound, "req-456")

	assert.NotNil(t, resp)
	assert.Equal(t, CodeNotFound, resp.Code)
	assert.Equal(t, "资源不存在", resp.Message)
	assert.Equal(t, "req-456", resp.RequestID)
	assert.Nil(t, resp.Data)
}

func TestErrorWithMsg(t *testing.T) {
	resp := ErrorWithMsg(CodeBadRequest, "chain 参数非法", "req-789")

	assert.NotNil(t, resp)
	assert.Equal(t, CodeBadRequest, resp.Code)
	assert.Equal(t, "chain 参数非法", resp.Message)
	assert.Equal(t, "req-789", resp.RequestID)
}

func TestResponse_JSONSerialization(t *testing.T) {
	// 验证 Response 的 JSON 序列化包含所有字段
	resp := Success("hello", "rid")
	assert.Equal(t, 200, resp.Code)
	assert.Equal(t, "success", resp.Message)
	assert.Equal(t, "hello", resp.Data)
	assert.Equal(t, "rid", resp.RequestID)
}
