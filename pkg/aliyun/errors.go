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

package aliyun

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/alibabacloud-go/tea/tea"
)

// ErrorCode 阿里云常见错误码
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
)

// SDKError 封装阿里云SDK错误
type SDKError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestId string `json:"requestId"`
	HTTPCode  int    `json:"httpCode"`
}

// NewSDKError 创建新的SDK错误
func NewSDKError(err error) *SDKError {
	if err == nil {
		return nil
	}

	// 尝试从tea SDK错误中提取详细信息
	teaErr, ok := err.(*tea.SDKError)
	if ok {
		code := tea.StringValue(teaErr.Code)
		message := tea.StringValue(teaErr.Message)
		requestId := ""
		data := tea.StringValue(teaErr.Data)

		// 尝试从Data字符串解析RequestId
		if data != "" {
			var dataMap map[string]interface{}
			if err := json.Unmarshal([]byte(data), &dataMap); err == nil {
				if reqID, exists := dataMap["RequestId"]; exists {
					if reqIDStr, isString := reqID.(string); isString {
						requestId = reqIDStr
					}
				}
			}
		}

		// 如果Code为空，尝试从错误消息中解析
		if code == "" && message != "" {
			// 阿里云错误消息通常包含特定格式的JSON
			if strings.Contains(message, "{") && strings.Contains(message, "}") {
				startIdx := strings.Index(message, "{")
				endIdx := strings.LastIndex(message, "}")
				if startIdx >= 0 && endIdx > startIdx {
					jsonStr := message[startIdx : endIdx+1]
					var errData map[string]interface{}
					if err := json.Unmarshal([]byte(jsonStr), &errData); err == nil {
						if c, ok := errData["Code"].(string); ok && c != "" {
							code = c
						}
						if m, ok := errData["Message"].(string); ok && m != "" {
							message = m
						}
						if r, ok := errData["RequestId"].(string); ok && r != "" {
							requestId = r
						}
					}
				}
			}
		}

		return &SDKError{
			Code:      code,
			Message:   message,
			RequestId: requestId,
			HTTPCode:  tea.IntValue(teaErr.StatusCode),
		}
	}

	// 处理非tea SDK错误
	return &SDKError{
		Code:    string(ErrCodeUnknown),
		Message: err.Error(),
	}
}

// Error 实现error接口
func (e *SDKError) Error() string {
	return fmt.Sprintf("阿里云API错误: Code=%s, Message=%s, RequestId=%s, HTTPCode=%d",
		e.Code, e.Message, e.RequestId, e.HTTPCode)
}

// IsThrottling 检查是否为限流错误
func (e *SDKError) IsThrottling() bool {
	return e.Code == string(ErrCodeThrottling) || strings.Contains(strings.ToLower(e.Code), "throttling")
}

// IsAccessDenied 检查是否为访问拒绝错误
func (e *SDKError) IsAccessDenied() bool {
	return e.Code == string(ErrCodeAccessDenied) || 
		strings.Contains(strings.ToLower(e.Message), "access denied") ||
		strings.Contains(strings.ToLower(e.Code), "forbidden")
}

// IsResourceNotFound 检查是否为资源不存在错误
func (e *SDKError) IsResourceNotFound() bool {
	return e.Code == string(ErrCodeResourceNotFound) || 
		strings.Contains(e.Code, "NotFound") ||
		strings.Contains(strings.ToLower(e.Message), "not found") ||
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
		strings.Contains(strings.ToLower(e.Message), "quota exceeded")
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
		sdkErr = NewSDKError(err)
	}
	
	// 可重试的错误类型：限流、服务器内部错误、服务不可用
	return sdkErr.IsThrottling() || 
		sdkErr.IsInternalError() || 
		sdkErr.Code == string(ErrCodeServiceUnavailable) ||
		sdkErr.HTTPCode >= 500 ||
		strings.Contains(strings.ToLower(sdkErr.Message), "try again")
}
