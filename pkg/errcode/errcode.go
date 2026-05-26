// Package errcode 定义统一的错误码和错误响应格式
// 所有 API 返回都遵循统一格式: { code, message, data, request_id }
package errcode

// 错误码定义
// 约定: 200=成功, 4xx=客户端错误, 5xx=服务端错误
const (
	CodeSuccess       = 200 // 成功
	CodeBadRequest    = 400 // 请求参数错误
	CodeNotFound      = 404 // 资源不存在
	CodeInternalError = 500 // 服务器内部错误
	CodeDBError       = 501 // 数据库错误
	CodeCacheError    = 502 // 缓存错误
	CodeRPCError      = 503 // 区块链 RPC 调用错误
	CodeKafkaError    = 504 // 消息队列错误
)

// 错误消息映射
var codeMessages = map[int]string{
	CodeSuccess:       "success",
	CodeBadRequest:    "请求参数错误",
	CodeNotFound:      "资源不存在",
	CodeInternalError: "服务器内部错误",
	CodeDBError:       "数据库错误",
	CodeCacheError:    "缓存错误",
	CodeRPCError:      "区块链 RPC 调用错误",
	CodeKafkaError:    "消息队列错误",
}

// GetMsg 根据错误码获取对应的错误消息
func GetMsg(code int) string {
	msg, ok := codeMessages[code]
	if !ok {
		return "未知错误"
	}
	return msg
}

// Response 统一 API 响应结构体
type Response struct {
	Code      int         `json:"code"`       // 错误码
	Message   string      `json:"message"`    // 错误消息
	Data      interface{} `json:"data"`       // 响应数据
	RequestID string      `json:"request_id"` // 请求 ID（用于链路追踪）
}

// Success 构建成功响应
func Success(data interface{}, requestID string) *Response {
	return &Response{
		Code:      CodeSuccess,
		Message:   GetMsg(CodeSuccess),
		Data:      data,
		RequestID: requestID,
	}
}

// Error 构建错误响应
func Error(code int, requestID string) *Response {
	return &Response{
		Code:      code,
		Message:   GetMsg(code),
		Data:      nil,
		RequestID: requestID,
	}
}

// ErrorWithMsg 构建带自定义消息的错误响应
func ErrorWithMsg(code int, msg string, requestID string) *Response {
	return &Response{
		Code:      code,
		Message:   msg,
		Data:      nil,
		RequestID: requestID,
	}
}
