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
	"reflect"
	"strings"
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

// ProbeTCP80 只探测该地址的 TCP 80 是否接受连接，不是 ICMP。
// 发布容器丢掉全部 capability 后，系统 ping 需要 CAP_NET_RAW。
func ProbeTCP80(ipAddr string) bool {
	return probeTCP(ipAddr, "80", 3*time.Second)
}

// Deprecated: 使用 ProbeTCP80。本函数不是 ICMP，只探测 TCP 80。
func Ping(ipAddr string) bool {
	return ProbeTCP80(ipAddr)
}

func probeTCP(ipAddr, port string, timeout time.Duration) bool {
	if strings.TrimSpace(ipAddr) == "" {
		return false
	}
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(ipAddr, port), timeout)
	if err != nil {
		log.Printf("tcp probe %s:%s 失败: %v", ipAddr, port, err)
		return false
	}
	_ = conn.Close()
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

func ValidateUniqueResource[T any](ctx context.Context, getResourceFunc func(context.Context, interface{}) (T, error), newResource T, id interface{}) error {
	existingResource, err := getResourceFunc(ctx, id)
	if err != nil {
		return fmt.Errorf("获取资源失败: %w", err)
	}

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

func NewBusinessError(code error, message string) error {
	return &BusinessError{
		Code:    code,
		Message: message,
	}
}
