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

package utils

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// LabelOption 通用返回结构体，用于前后端交互的数据格式
type LabelOption struct {
	Label    string         `json:"label"`
	Value    string         `json:"value"`
	Children []*LabelOption `json:"children"`
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

type OperationData struct {
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
	Data    interface{} `json:"data"`    // 响应数据
	Message string      `json:"message"` // 反馈信息
	Type    string      `json:"type"`    // 消息类型
}

// 定义操作成功和失败的常量状态码
const (
	StatusError   = 1 // 操作失败
	StatusSuccess = 0 // 操作成功
)

// ApiData 通用的返回函数，用于标准化API响应格式
// 参数：
// - c: gin 上下文
// - code: 状态码
// - data: 返回的数据
// - message: 返回的消息
func ApiData(c *gin.Context, code int, data interface{}, message string) {
	c.JSON(http.StatusOK, ApiResponse{
		Code:    code,
		Data:    data,
		Message: message,
		Type:    "",
	})
}

// Success 操作成功的返回
func Success(c *gin.Context) {
	ApiData(c, StatusSuccess, map[string]interface{}{}, "操作成功")
}

// SuccessWithMessage 带消息的操作成功返回
func SuccessWithMessage(c *gin.Context, message string) {
	ApiData(c, StatusSuccess, map[string]interface{}{}, message)
}

// SuccessWithData 带数据的操作成功返回
func SuccessWithData(c *gin.Context, data interface{}) {
	ApiData(c, StatusSuccess, data, "请求成功")
}

// SuccessWithDetails 带详细数据和消息的操作成功返回
func SuccessWithDetails(c *gin.Context, data interface{}, message string) {
	ApiData(c, StatusSuccess, data, message)
}

// Error 操作失败的返回
func Error(c *gin.Context) {
	ApiData(c, StatusError, map[string]interface{}{}, "操作失败")
}

// ErrorWithMessage 带消息的操作失败返回
func ErrorWithMessage(c *gin.Context, message string) {
	ApiData(c, StatusError, map[string]interface{}{}, message)
}

// ErrorWithDetails 带详细数据和消息的操作失败返回
func ErrorWithDetails(c *gin.Context, data interface{}, message string) {
	ApiData(c, StatusError, data, message)
}

// BadRequest 参数错误的返回，使用HTTP 400状态码
func BadRequest(c *gin.Context, code int, data interface{}, message string) {
	c.JSON(http.StatusBadRequest, ApiResponse{
		Code:    code,
		Data:    data,
		Message: message,
		Type:    "",
	})
}

// Forbidden 无权限的返回，使用HTTP 403状态码
func Forbidden(c *gin.Context, code int, data interface{}, message string) {
	c.JSON(http.StatusForbidden, ApiResponse{
		Code:    code,
		Data:    data,
		Message: message,
		Type:    "",
	})
}

// Unauthorized 未认证的返回，使用HTTP 401状态码
func Unauthorized(c *gin.Context, code int, data interface{}, message string) {
	c.JSON(http.StatusUnauthorized, ApiResponse{
		Code:    code,
		Data:    data,
		Message: message,
		Type:    "",
	})
}

// InternalServerError 服务器内部错误的返回，使用HTTP 500状态码
func InternalServerError(c *gin.Context, code int, data interface{}, message string) {
	c.JSON(http.StatusInternalServerError, ApiResponse{
		Code:    code,
		Data:    data,
		Message: message,
		Type:    "",
	})
}

// BadRequestError 参数错误的失败返回
func BadRequestError(c *gin.Context, message string) {
	BadRequest(c, StatusError, map[string]interface{}{}, message)
}

// BadRequestWithDetails 带详细数据和消息的参数错误返回
func BadRequestWithDetails(c *gin.Context, data interface{}, message string) {
	BadRequest(c, StatusError, data, message)
}

// UnauthorizedErrorWithDetails 带详细数据和消息的未认证返回
func UnauthorizedErrorWithDetails(c *gin.Context, data interface{}, message string) {
	Unauthorized(c, StatusError, data, message)
}

// ForbiddenError 无权限的失败返回
func ForbiddenError(c *gin.Context, message string) {
	Forbidden(c, StatusError, map[string]interface{}{}, message)
}

// InternalServerErrorWithDetails 带详细数据和消息的服务器内部错误返回
func InternalServerErrorWithDetails(c *gin.Context, data interface{}, message string) {
	InternalServerError(c, StatusError, data, message)
}

// PostWithJsonString 发送带 JSON 数据的 POST 请求
func PostWithJsonString(l *zap.Logger, funcName string, timeout int, url string, jsonStr string, paramsMap map[string]string, headerMap map[string]string) ([]byte, error) {
	client := &http.Client{Timeout: time.Duration(timeout) * time.Second}
	reader := bytes.NewReader([]byte(jsonStr))

	req, err := http.NewRequest("POST", url, reader)
	if err != nil {
		l.Error(fmt.Sprintf("[PostWithJsonString.NewRequest.error][funcName:%s][url:%s][err:%v]", funcName, url, err))
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	// 添加URL参数
	q := req.URL.Query()
	for k, v := range paramsMap {
		q.Add(k, v)
	}
	req.URL.RawQuery = q.Encode()

	// 添加Header
	for k, v := range headerMap {
		req.Header.Add(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		l.Error(fmt.Sprintf("[PostWithJsonString.Do.error][funcName:%s][url:%s][err:%v]", funcName, url, err))
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := ioutil.ReadAll(resp.Body) // 忽略错误，记录原始响应体
		l.Error(fmt.Sprintf("[PostWithJsonString.StatusCode.notOK][funcName:%s][url:%s][code:%d][resp_body:%s]", funcName, url, resp.StatusCode, string(bodyBytes)))
		return nil, fmt.Errorf("server returned HTTP status %s", resp.Status)
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		l.Error(fmt.Sprintf("[PostWithJsonString.ReadBody.error][funcName:%s][url:%s][err:%v]", funcName, url, err))
		return nil, err
	}

	return bodyBytes, nil
}

func DeleteWithId(l *zap.Logger, funcName string, timeout int, url string, paramsMap map[string]string, headerMap map[string]string) ([]byte, error) {
	client := &http.Client{}

	reader := bytes.NewReader([]byte(""))
	client.Timeout = time.Duration(timeout) * time.Second
	req, err := http.NewRequest("DELETE", url, reader)
	if err != nil {
		l.Error(fmt.Sprintf("[DeleteWithId.http.NewRequest.error][funcName:%+v][url:%v][err:%v]", funcName, url, err))
		return nil, err
	}

	// 添加URL参数
	q := req.URL.Query()
	for k, v := range paramsMap {
		q.Add(k, v)
	}
	req.URL.RawQuery = q.Encode()

	// 添加Header
	for k, v := range headerMap {
		req.Header.Add(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		l.Error(fmt.Sprintf("[DeleteWithId.request.error][funcName:%+v][url:%v][err:%v]", funcName, url, err))
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		l.Error(fmt.Sprintf("[DeleteWithId.StatusCode.not2xx.error][funcName:%+v][url:%v][code:%v][resp_body_text:%v]", funcName, url, resp.StatusCode, string(bodyBytes)))
		return nil, errors.New(fmt.Sprintf("server returned HTTP status %s", resp.Status))
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		l.Error(fmt.Sprintf("[DeleteWithId.readbody.error][funcName:%+v][url:%v][err:%v]", funcName, url, err))
		return nil, err
	}

	return bodyBytes, nil
}

// HandleRequest 用于统一处理请求绑定和响应
func HandleRequest(ctx *gin.Context, req interface{}, action func() (interface{}, error)) {
	if req != nil {
		// 如果提供了绑定对象，执行数据绑定
		if err := ctx.ShouldBind(req); err != nil {
			BadRequestWithDetails(ctx, err.Error(), "绑定数据失败")
			return
		}
	}

	// 执行主要业务逻辑
	data, err := action()
	if err != nil {
		ErrorWithMessage(ctx, err.Error())
		return
	}

	// 返回成功响应，若有数据则包含数据，否则仅返回成功状态
	if data != nil {
		SuccessWithData(ctx, data)
	} else {
		Success(ctx)
	}
}

// GetParamID 从查询参数中解析 ID，并进行类型转换
func GetParamID(ctx *gin.Context) (int, error) {
	id := ctx.Param("id")
	if id == "" {
		return 0, fmt.Errorf("缺少 'id' 参数")
	}
	paramID, err := strconv.Atoi(id)
	if err != nil {
		return 0, fmt.Errorf("'id' 非整数")
	}
	return paramID, nil
}

// GetQueryID 从query参数中解析 ID，并进行类型转换
func GetQueryID(ctx *gin.Context) (int, error) {
	id := ctx.Query("id")
	if id == "" {
		return 0, fmt.Errorf("缺少 'id' 参数")
	}
	paramID, err := strconv.Atoi(id)
	if err != nil {
		return 0, fmt.Errorf("'id' 非整数")
	}
	return paramID, nil
}

// GetParamName 从查询参数中解析 Name，并进行类型转换
func GetParamName(ctx *gin.Context) (string, error) {
	name := ctx.Param("name")
	if name == "" {
		return "", fmt.Errorf("缺少 'name' 参数")
	}

	return name, nil
}
