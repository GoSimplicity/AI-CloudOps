/*
 * MIT License
 *
 * Copyright (c) 2024 Bamboo
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 *
 */

package aws

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/aws/smithy-go"
	awshttp "github.com/aws/smithy-go/transport/http"
)

// ErrorCode AWS错误码
type ErrorCode string

const (
	// 通用错误码
	ErrCodeUnknown             ErrorCode = "UnknownError"
	ErrCodeAccessDenied        ErrorCode = "AccessDenied"
	ErrCodeThrottling          ErrorCode = "Throttling"
	ErrCodeInvalidParameters   ErrorCode = "InvalidParameters"
	ErrCodeInsufficientBalance ErrorCode = "InsufficientBalance"
	ErrCodeResourceNotFound    ErrorCode = "ResourceNotFound"
	ErrCodeOperationConflict   ErrorCode = "OperationConflict"
	ErrCodeQuotaExceeded       ErrorCode = "QuotaExceeded"
	ErrCodeInternalError       ErrorCode = "InternalError"
	ErrCodeServiceUnavailable  ErrorCode = "ServiceUnavailable"
	ErrCodeInvalidCredentials  ErrorCode = "InvalidCredentials"
)

// 预定义错误
var (
	ErrInvalidCredentials = errors.New("AWS访问密钥无效")
	ErrSDKNotInitialized  = errors.New("AWS SDK未初始化")
	ErrInvalidRegion      = errors.New("无效的AWS区域")
	ErrInstanceNotFound   = errors.New("EC2实例不存在")
	ErrVPCNotFound        = errors.New("VPC不存在")
	ErrPermissionDenied   = errors.New("权限不足")
)

// SDKError 封装AWS SDK错误
type SDKError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestId string `json:"requestId"`
	HTTPCode  int    `json:"httpCode"`
	Resource  string `json:"resource,omitempty"`
}

// NewSDKError 创建新的SDK错误
func NewSDKError(err error) *SDKError {
	if err == nil {
		return nil
	}

	sdkError := &SDKError{
		Code:    string(ErrCodeUnknown),
		Message: err.Error(),
	}

	// 解析AWS SDK v2错误格式
	var apiError smithy.APIError
	if errors.As(err, &apiError) {
		sdkError.Code = apiError.ErrorCode()
		sdkError.Message = apiError.ErrorMessage()
	}

	// 解析HTTP响应错误
	var httpResponseError *awshttp.ResponseError
	if errors.As(err, &httpResponseError) {
		sdkError.HTTPCode = httpResponseError.Response.StatusCode

		// 提取请求ID
		if requestId := httpResponseError.Response.Header.Get("x-amzn-RequestId"); requestId != "" {
			sdkError.RequestId = requestId
		} else if requestId := httpResponseError.Response.Header.Get("x-amz-request-id"); requestId != "" {
			sdkError.RequestId = requestId
		}
	}

	// 解析Operation错误中的资源信息
	var operationError *smithy.OperationError
	if errors.As(err, &operationError) {
		sdkError.Resource = operationError.Operation()
	}

	// 根据错误消息推断错误类型
	sdkError.inferErrorType()

	return sdkError
}

// inferErrorType 根据错误消息推断错误类型
func (e *SDKError) inferErrorType() {
	errorCode := strings.ToLower(e.Code)
	errorMessage := strings.ToLower(e.Message)

	// 根据AWS常见错误码进行分类
	switch {
	case strings.Contains(errorCode, "accessdenied") || strings.Contains(errorCode, "forbidden"):
		e.Code = string(ErrCodeAccessDenied)
	case strings.Contains(errorCode, "throttling") || strings.Contains(errorCode, "throttled"):
		e.Code = string(ErrCodeThrottling)
	case strings.Contains(errorCode, "invalidparameter") || strings.Contains(errorCode, "validationexception"):
		e.Code = string(ErrCodeInvalidParameters)
	case strings.Contains(errorCode, "notfound") || strings.Contains(errorCode, "doesnotexist"):
		e.Code = string(ErrCodeResourceNotFound)
	case strings.Contains(errorCode, "limitexceeded") || strings.Contains(errorCode, "quotaexceeded"):
		e.Code = string(ErrCodeQuotaExceeded)
	case strings.Contains(errorCode, "internalerror") || strings.Contains(errorCode, "internalservererror"):
		e.Code = string(ErrCodeInternalError)
	case strings.Contains(errorCode, "serviceunavailable") || strings.Contains(errorCode, "serviceexception"):
		e.Code = string(ErrCodeServiceUnavailable)
	case strings.Contains(errorCode, "invalidcredentials") || strings.Contains(errorCode, "signaturedoesnotmatch"):
		e.Code = string(ErrCodeInvalidCredentials)
	case strings.Contains(errorCode, "conflictexception") || strings.Contains(errorCode, "resourceinuse"):
		e.Code = string(ErrCodeOperationConflict)
	case strings.Contains(errorCode, "insufficientfunds") || strings.Contains(errorCode, "insufficientbalance"):
		e.Code = string(ErrCodeInsufficientBalance)
	}

	// 如果还是未知错误，尝试从消息中推断
	if e.Code == string(ErrCodeUnknown) {
		switch {
		case strings.Contains(errorMessage, "access denied") || strings.Contains(errorMessage, "permission"):
			e.Code = string(ErrCodeAccessDenied)
		case strings.Contains(errorMessage, "throttl") || strings.Contains(errorMessage, "rate limit"):
			e.Code = string(ErrCodeThrottling)
		case strings.Contains(errorMessage, "not found") || strings.Contains(errorMessage, "does not exist"):
			e.Code = string(ErrCodeResourceNotFound)
		case strings.Contains(errorMessage, "invalid") || strings.Contains(errorMessage, "validation"):
			e.Code = string(ErrCodeInvalidParameters)
		case strings.Contains(errorMessage, "limit") || strings.Contains(errorMessage, "quota"):
			e.Code = string(ErrCodeQuotaExceeded)
		case strings.Contains(errorMessage, "internal") || e.HTTPCode >= 500:
			e.Code = string(ErrCodeInternalError)
		case strings.Contains(errorMessage, "conflict") || strings.Contains(errorMessage, "in use"):
			e.Code = string(ErrCodeOperationConflict)
		}
	}
}

// Error 实现error接口
func (e *SDKError) Error() string {
	return fmt.Sprintf("AWS API错误: Code=%s, Message=%s, RequestId=%s, HTTPCode=%d",
		e.Code, e.Message, e.RequestId, e.HTTPCode)
}

// IsThrottling 检查是否为限流错误
func (e *SDKError) IsThrottling() bool {
	return e.Code == string(ErrCodeThrottling) ||
		strings.Contains(strings.ToLower(e.Code), "throttling") ||
		strings.Contains(strings.ToLower(e.Message), "rate limit")
}

// IsAccessDenied 检查是否为访问拒绝错误
func (e *SDKError) IsAccessDenied() bool {
	return e.Code == string(ErrCodeAccessDenied) ||
		strings.Contains(strings.ToLower(e.Message), "access denied") ||
		strings.Contains(strings.ToLower(e.Code), "forbidden") ||
		strings.Contains(strings.ToLower(e.Message), "permission")
}

// IsResourceNotFound 检查是否为资源不存在错误
func (e *SDKError) IsResourceNotFound() bool {
	return e.Code == string(ErrCodeResourceNotFound) ||
		strings.Contains(e.Code, "NotFound") ||
		strings.Contains(strings.ToLower(e.Message), "not found") ||
		strings.Contains(strings.ToLower(e.Message), "does not exist") ||
		e.HTTPCode == 404
}

// IsInternalError 检查是否为内部错误
func (e *SDKError) IsInternalError() bool {
	return e.Code == string(ErrCodeInternalError) ||
		strings.Contains(strings.ToLower(e.Code), "internal") ||
		e.HTTPCode >= 500
}

// IsQuotaExceeded 检查是否为配额超限错误
func (e *SDKError) IsQuotaExceeded() bool {
	return e.Code == string(ErrCodeQuotaExceeded) ||
		strings.Contains(strings.ToLower(e.Code), "quota") ||
		strings.Contains(strings.ToLower(e.Message), "quota exceeded") ||
		strings.Contains(strings.ToLower(e.Message), "limit exceeded")
}

// IsInvalidCredentials 检查是否为凭证错误
func (e *SDKError) IsInvalidCredentials() bool {
	return e.Code == string(ErrCodeInvalidCredentials) ||
		strings.Contains(strings.ToLower(e.Code), "credentials") ||
		strings.Contains(strings.ToLower(e.Message), "credentials") ||
		strings.Contains(strings.ToLower(e.Message), "access key") ||
		strings.Contains(strings.ToLower(e.Message), "signature")
}

// HandleError 处理SDK错误，转换为标准错误格式
func HandleError(err error) error {
	if err == nil {
		return nil
	}

	return NewSDKError(err)
}

// RetryableError 检查错误是否可重试
func RetryableError(err error) bool {
	if err == nil {
		return false
	}

	sdkErr, ok := err.(*SDKError)
	if !ok {
		// 对于非SDK错误，只有网络相关错误才重试
		return strings.Contains(strings.ToLower(err.Error()), "timeout") ||
			strings.Contains(strings.ToLower(err.Error()), "connection") ||
			strings.Contains(strings.ToLower(err.Error()), "network")
	}

	// 以下情况可以重试：
	// 1. 限流错误
	// 2. 内部错误（5xx）
	// 3. 服务不可用
	return sdkErr.IsThrottling() ||
		sdkErr.IsInternalError() ||
		sdkErr.Code == string(ErrCodeServiceUnavailable)
}

// GetErrorCode 从错误中提取错误码
func GetErrorCode(err error) string {
	if err == nil {
		return ""
	}

	if sdkErr, ok := err.(*SDKError); ok {
		return sdkErr.Code
	}

	return string(ErrCodeUnknown)
}

// GetRequestId 从错误中提取请求ID
func GetRequestId(err error) string {
	if err == nil {
		return ""
	}

	if sdkErr, ok := err.(*SDKError); ok {
		return sdkErr.RequestId
	}

	return ""
}

// FormatError 格式化错误信息，用于日志记录
func FormatError(err error, operation string) map[string]interface{} {
	if err == nil {
		return nil
	}

	result := map[string]interface{}{
		"operation": operation,
		"error":     err.Error(),
	}

	if sdkErr, ok := err.(*SDKError); ok {
		result["error_code"] = sdkErr.Code
		result["http_code"] = sdkErr.HTTPCode
		result["request_id"] = sdkErr.RequestId
		if sdkErr.Resource != "" {
			result["resource"] = sdkErr.Resource
		}
	}

	return result
}

// IsAWSError 检查是否为AWS API错误
func IsAWSError(err error) bool {
	if err == nil {
		return false
	}

	var apiError smithy.APIError
	return errors.As(err, &apiError)
}

// GetAWSErrorCode 获取AWS错误码
func GetAWSErrorCode(err error) string {
	if err == nil {
		return ""
	}

	var apiError smithy.APIError
	if errors.As(err, &apiError) {
		return apiError.ErrorCode()
	}

	return ""
}

// GetHTTPStatusCode 获取HTTP状态码
func GetHTTPStatusCode(err error) int {
	if err == nil {
		return 0
	}

	var httpResponseError *awshttp.ResponseError
	if errors.As(err, &httpResponseError) {
		return httpResponseError.Response.StatusCode
	}

	return 0
}

// IsHTTPError 检查是否为特定HTTP状态码错误
func IsHTTPError(err error, statusCode int) bool {
	return GetHTTPStatusCode(err) == statusCode
}

// IsNotFoundError 检查是否为404错误
func IsNotFoundError(err error) bool {
	return IsHTTPError(err, http.StatusNotFound) ||
		strings.Contains(strings.ToLower(GetAWSErrorCode(err)), "notfound")
}

// IsForbiddenError 检查是否为403错误
func IsForbiddenError(err error) bool {
	return IsHTTPError(err, http.StatusForbidden) ||
		strings.Contains(strings.ToLower(GetAWSErrorCode(err)), "forbidden") ||
		strings.Contains(strings.ToLower(GetAWSErrorCode(err)), "accessdenied")
}
