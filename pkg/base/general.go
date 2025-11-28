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

package base

import (
	"context"
	"fmt"
	"log"
	"net"
	"os/exec"
	"reflect"
	"runtime"
	"time"
)

// MapToStringSlice 将 map 转换为 []string，要求偶数个元素，key和值依次排列
func MapToStringSlice(inputMap map[string]string) ([]string, error) {
	if inputMap == nil {
		return []string{}, nil
	}

	var result []string
	for key, value := range inputMap {
		result = append(result, key, value)
	}

	// 确保结果长度为偶数
	if len(result)%2 != 0 {
		return nil, fmt.Errorf("转换后的字符串切片长度为奇数，不符合键值对要求")
	}

	return result, nil
}

// StringSliceToMap 将 []string 转换为 map[string]string，要求输入长度为偶数，奇数索引为 key，偶数索引为 value
func StringSliceToMap(inputSlice []string) (map[string]string, error) {
	if len(inputSlice)%2 != 0 {
		return nil, fmt.Errorf("输入的字符串切片长度必须为偶数，实际长度为 %d", len(inputSlice))
	}

	result := make(map[string]string)
	for i := 0; i < len(inputSlice); i += 2 {
		key := inputSlice[i]
		value := inputSlice[i+1]
		result[key] = value
	}

	return result, nil
}

// Ping 检查指定的 IP 地址是否可达
func Ping(ipAddr string) bool {
	// 检查IP地址是否为空
	if ipAddr == "" {
		return false
	}

	// 使用系统命令执行ping操作
	var cmd *exec.Cmd
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 根据操作系统选择不同的ping命令参数
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "ping", "-n", "1", "-w", "3000", ipAddr)
	} else {
		cmd = exec.CommandContext(ctx, "ping", "-c", "1", "-W", "3", ipAddr)
	}

	// 执行命令并捕获输出
	output, err := cmd.CombinedOutput()
	if err != nil {
		// 记录错误信息和输出以便调试
		log.Printf("ping %s 失败: %v, 输出: %s", ipAddr, err, string(output))
		return false
	}

	return true
}

func GetLocalIPs() ([]string, error) {
	var ips []string
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	for _, i := range interfaces {
		addrs, err := i.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			// 过滤掉回环地址和 IPv6 地址
			if ip != nil && !ip.IsLoopback() && ip.To4() != nil {
				ips = append(ips, ip.String())
			}
		}
	}

	return ips, nil
}

// ValidateUniqueResource 验证是否存在相同的资源
func ValidateUniqueResource[T any](ctx context.Context, getResourceFunc func(context.Context, interface{}) (T, error), newResource T, id interface{}) error {
	// 获取已存在的资源
	existingResource, err := getResourceFunc(ctx, id)
	if err != nil {
		return fmt.Errorf("获取资源失败: %w", err)
	}

	// 使用比较两个资源是否相同
	if reflect.DeepEqual(existingResource, newResource) {
		return fmt.Errorf("资源已存在")
	}

	return nil
}

// GetAge 计算资源创建时间到现在的时间差，返回易读格式
func GetAge(creationTime time.Time) string {
	duration := time.Since(creationTime)

	if duration < time.Minute {
		return fmt.Sprintf("%ds", int(duration.Seconds()))
	}

	if duration < time.Hour {
		return fmt.Sprintf("%dm", int(duration.Minutes()))
	}

	if duration < 24*time.Hour {
		return fmt.Sprintf("%dh", int(duration.Hours()))
	}

	days := int(duration.Hours() / 24)
	return fmt.Sprintf("%dd", days)
}

// BusinessError 业务错误结构体
type BusinessError struct {
	Code    error
	Message string
}

func (e *BusinessError) Error() string {
	return e.Message
}

// NewBusinessError 创建业务错误
func NewBusinessError(code error, message string) error {
	return &BusinessError{
		Code:    code,
		Message: message,
	}
}
