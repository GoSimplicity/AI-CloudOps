package apiresponse

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// LabelOption 通用返回结构体，用于前后端交互的数据格式
type LabelOption struct {
	Label    string         `json:"label"`
	Value    string         `json:"value"`
	Children []*LabelOption `json:"children"`
}

type KeyValueItem struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type K8sBatchRequest struct {
	Cluster string           `json:"cluster"`
	Items   []K8sRequestItem `json:"items"`
}

type K8sRequestItem struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

type K8sObjectRequest struct {
	Cluster   string `json:"cluster"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

type OperationResult struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

type SelectOption struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type KeyValuePair struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type SelectOptionInt struct {
	Label string `json:"label"`
	Value int    `json:"value"`
}

type SilenceResponse struct {
	Status string `json:"status"`
	Data   struct {
		SilenceID string `json:"silence_id"`
	} `json:"data"`
}

// ApiResponse 通用的API响应结构体
type ApiResponse struct {
	Code    int         `json:"code"`    // 状态码，表示业务逻辑的状态，而非HTTP状态码
	Result  interface{} `json:"result"`  // 响应数据
	Message string      `json:"message"` // 反馈信息
	Type    string      `json:"type"`    // 消息类型
}

// 定义操作成功和失败的常量状态码
const (
	StatusError   = 1 // 操作失败
	StatusSuccess = 0 // 操作成功
)

// ApiResult 通用的返回函数，用于标准化API响应格式
// 参数：
// - c: gin 上下文
// - code: 状态码
// - data: 返回的数据
// - message: 返回的消息
func ApiResult(c *gin.Context, code int, data interface{}, message string) {
	c.JSON(http.StatusOK, ApiResponse{
		Code:    code,
		Result:  data,
		Message: message,
		Type:    "",
	})
	return
}

// Success 操作成功的返回
func Success(c *gin.Context) {
	ApiResult(c, StatusSuccess, map[string]interface{}{}, "操作成功")
	return
}

// SuccessWithMessage 带消息的操作成功返回
func SuccessWithMessage(c *gin.Context, message string) {
	ApiResult(c, StatusSuccess, map[string]interface{}{}, message)
	return
}

// SuccessWithData 带数据的操作成功返回
func SuccessWithData(c *gin.Context, data interface{}) {
	ApiResult(c, StatusSuccess, data, "请求成功")
	return
}

// SuccessWithDetails 带详细数据和消息的操作成功返回
func SuccessWithDetails(c *gin.Context, data interface{}, message string) {
	ApiResult(c, StatusSuccess, data, message)
	return
}

// Error 操作失败的返回
func Error(c *gin.Context) {
	ApiResult(c, StatusError, map[string]interface{}{}, "操作失败")
	return
}

// ErrorWithMessage 带消息的操作失败返回
func ErrorWithMessage(c *gin.Context, message string) {
	ApiResult(c, StatusError, map[string]interface{}{}, message)
	return
}

// ErrorWithDetails 带详细数据和消息的操作失败返回
func ErrorWithDetails(c *gin.Context, data interface{}, message string) {
	ApiResult(c, StatusError, data, message)
	return
}

// BadRequest 参数错误的返回，使用HTTP 400状态码
func BadRequest(c *gin.Context, code int, data interface{}, message string) {
	c.JSON(http.StatusBadRequest, ApiResponse{
		Code:    code,
		Result:  data,
		Message: message,
		Type:    "",
	})
	return
}

// Forbidden 无权限的返回，使用HTTP 403状态码
func Forbidden(c *gin.Context, code int, data interface{}, message string) {
	c.JSON(http.StatusForbidden, ApiResponse{
		Code:    code,
		Result:  data,
		Message: message,
		Type:    "",
	})
	return
}

// Unauthorized 未认证的返回，使用HTTP 401状态码
func Unauthorized(c *gin.Context, code int, data interface{}, message string) {
	c.JSON(http.StatusUnauthorized, ApiResponse{
		Code:    code,
		Result:  data,
		Message: message,
		Type:    "",
	})
	return
}

// InternalServerError 服务器内部错误的返回，使用HTTP 500状态码
func InternalServerError(c *gin.Context, code int, data interface{}, message string) {
	c.JSON(http.StatusInternalServerError, ApiResponse{
		Code:    code,
		Result:  data,
		Message: message,
		Type:    "",
	})
	return
}

// BadRequestError 参数错误的失败返回
func BadRequestError(c *gin.Context, message string) {
	BadRequest(c, StatusError, map[string]interface{}{}, message)
	return
}

// BadRequestWithDetails 带详细数据和消息的参数错误返回
func BadRequestWithDetails(c *gin.Context, data interface{}, message string) {
	BadRequest(c, StatusError, data, message)
	return
}

// UnauthorizedErrorWithDetails 带详细数据和消息的未认证返回
func UnauthorizedErrorWithDetails(c *gin.Context, data interface{}, message string) {
	Unauthorized(c, StatusError, data, message)
	return
}

// ForbiddenError 无权限的失败返回
func ForbiddenError(c *gin.Context, message string) {
	Forbidden(c, StatusError, map[string]interface{}{}, message)
	return
}

// InternalServerErrorWithDetails 带详细数据和消息的服务器内部错误返回
func InternalServerErrorWithDetails(c *gin.Context, data interface{}, message string) {
	InternalServerError(c, StatusError, data, message)
	return
}
