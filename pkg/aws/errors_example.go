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
	"context"
	"fmt"

	"go.uber.org/zap"
)

// ErrorHandlingExample 展示AWS错误处理的示例
func ErrorHandlingExample() {
	// 创建示例SDK和服务
	sdk := NewSDK("test-access-key", "test-secret-key")
	ec2Service := NewEC2Service(sdk)

	// 示例：处理创建实例时的错误
	req := &CreateInstanceRequest{
		Region:       "us-east-1",
		ImageId:      "ami-12345678",
		InstanceType: "t3.micro",
		MinCount:     1,
		MaxCount:     1,
	}

	_, err := ec2Service.CreateInstance(context.Background(), req)
	if err != nil {
		// 使用改进的错误处理
		handleAWSError(err, "CreateInstance")
	}
}

// handleAWSError 展示如何处理AWS错误
func handleAWSError(err error, operation string) {
	// 转换为SDK错误
	sdkErr := NewSDKError(err)
	if sdkErr == nil {
		return
	}

	// 记录详细的错误信息
	logger, _ := zap.NewDevelopment()
	logger.Error("AWS操作失败",
		zap.String("operation", operation),
		zap.String("error_code", sdkErr.Code),
		zap.String("message", sdkErr.Message),
		zap.String("request_id", sdkErr.RequestId),
		zap.Int("http_code", sdkErr.HTTPCode),
		zap.String("resource", sdkErr.Resource),
	)

	// 根据错误类型进行不同的处理
	switch {
	case sdkErr.IsThrottling():
		fmt.Printf("遇到限流错误，请稍后重试。请求ID: %s\n", sdkErr.RequestId)
		// 实现重试逻辑
	case sdkErr.IsAccessDenied():
		fmt.Printf("权限不足，请检查IAM权限。错误码: %s\n", sdkErr.Code)
		// 提示用户检查权限
	case sdkErr.IsResourceNotFound():
		fmt.Printf("资源不存在，请检查资源ID。错误码: %s\n", sdkErr.Code)
		// 处理资源不存在的情况
	case sdkErr.IsQuotaExceeded():
		fmt.Printf("配额超限，请联系AWS支持增加限制。错误码: %s\n", sdkErr.Code)
		// 处理配额超限
	case sdkErr.IsInternalError():
		fmt.Printf("AWS内部错误，请稍后重试。HTTP状态码: %d\n", sdkErr.HTTPCode)
		// 实现重试逻辑
	default:
		fmt.Printf("未知错误: %s\n", sdkErr.Error())
	}

	// 检查是否可重试
	if RetryableError(err) {
		fmt.Println("这是一个可重试的错误")
		// 实现重试逻辑
	}

	// 获取格式化的错误信息用于日志记录
	errorInfo := FormatError(err, operation)
	fmt.Printf("格式化错误信息: %+v\n", errorInfo)
}

// 演示具体的错误检查功能
func ExampleErrorChecking(err error) {
	if err == nil {
		return
	}

	// 检查是否为AWS API错误
	if IsAWSError(err) {
		fmt.Printf("这是AWS API错误，错误码: %s\n", GetAWSErrorCode(err))
	}

	// 检查HTTP状态码
	if httpCode := GetHTTPStatusCode(err); httpCode > 0 {
		fmt.Printf("HTTP状态码: %d\n", httpCode)
	}

	// 检查特定的错误类型
	if IsNotFoundError(err) {
		fmt.Println("资源未找到 (404)")
	}

	if IsForbiddenError(err) {
		fmt.Println("权限被拒绝 (403)")
	}

	if IsHTTPError(err, 500) {
		fmt.Println("服务器内部错误 (500)")
	}
}
