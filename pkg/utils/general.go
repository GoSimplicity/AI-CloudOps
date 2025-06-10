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
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net"
	"os/exec"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

func ConvertToIntList(stringList []string) ([]int, error) {
	intList := make([]int, 0, len(stringList))
	for _, idStr := range stringList {
		id, err := strconv.Atoi(strings.TrimSpace(idStr)) // 去除空白并转换为整数
		if err != nil {
			return nil, fmt.Errorf("无法解析 leafNodeId: '%s' 为整数", idStr)
		}
		intList = append(intList, id)
	}

	return intList, nil
}

// IsType 判断两个值是否是相同类型
func IsType(value1, value2 interface{}) bool {
	return reflect.TypeOf(value1) == reflect.TypeOf(value2)
}

// GetDefaultValue 返回值的默认值（零值）
func GetDefaultValue(value interface{}) interface{} {
	v := reflect.ValueOf(value)
	if !v.IsValid() {
		return nil
	}

	// 创建零值的副本并返回
	return reflect.Zero(v.Type()).Interface()
}

// GetMax 返回两个数值中的最大值，支持 int, float64 等常见类型
func GetMax(value1, value2 interface{}) (interface{}, error) {
	switch v1 := value1.(type) {
	case int:
		v2, ok := value2.(int)
		if !ok {
			return nil, errors.New("类型不匹配")
		}
		if v1 > v2 {
			return v1, nil
		}
		return v2, nil
	case float64:
		v2, ok := value2.(float64)
		if !ok {
			return nil, errors.New("类型不匹配")
		}
		return math.Max(v1, v2), nil
	default:
		return nil, errors.New("不支持的类型")
	}
}

// GetMin 返回两个数值中的最小值，支持 int, float64 等常见类型
func GetMin(value1, value2 interface{}) (interface{}, error) {
	switch v1 := value1.(type) {
	case int:
		v2, ok := value2.(int)
		if !ok {
			return nil, errors.New("类型不匹配")
		}
		if v1 < v2 {
			return v1, nil
		}
		return v2, nil
	case float64:
		v2, ok := value2.(float64)
		if !ok {
			return nil, errors.New("类型不匹配")
		}
		return math.Min(v1, v2), nil
	default:
		return nil, errors.New("不支持的类型")
	}
}

// ToUpperCase 将字符串转换为大写
func ToUpperCase(str string) string {
	return strings.ToUpper(str)
}

// ToLowerCase 将字符串转换为小写
func ToLowerCase(str string) string {
	return strings.ToLower(str)
}

// TrimSpaces 去掉字符串的前后空格
func TrimSpaces(str string) string {
	return strings.TrimSpace(str)
}

// IsSameDay 判断两个日期是否为同一天
func IsSameDay(t1, t2 time.Time) bool {
	y1, m1, d1 := t1.Date()
	y2, m2, d2 := t2.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

// DaysBetween 计算两个日期之间的天数
func DaysBetween(t1, t2 time.Time) int {
	days := t2.Sub(t1).Hours() / 24
	return int(math.Abs(days))
}

// IsValidEmail 简单检查一个字符串是否是有效的电子邮件格式
func IsValidEmail(email string) bool {
	return strings.Contains(email, "@") && strings.Contains(email, ".")
}

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

// AesEncrypt AES加密
func AesEncrypt(data []byte, key []byte) ([]byte, error) {
	// 创建加密实例
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// 填充数据
	blockSize := block.BlockSize()
	padding := blockSize - len(data)%blockSize
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	data = append(data, padText...)

	// 加密
	encrypted := make([]byte, len(data))
	mode := cipher.NewCBCEncrypter(block, key[:blockSize])
	mode.CryptBlocks(encrypted, data)

	return encrypted, nil
}

// AesDecrypt AES解密
func AesDecrypt(encrypted []byte, key []byte) ([]byte, error) {
	// 创建解密实例
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// 解密
	blockSize := block.BlockSize()
	mode := cipher.NewCBCDecrypter(block, key[:blockSize])
	decrypted := make([]byte, len(encrypted))
	mode.CryptBlocks(decrypted, encrypted)

	// 去除填充
	padding := int(decrypted[len(decrypted)-1])
	decrypted = decrypted[:len(decrypted)-padding]

	return decrypted, nil
}

// Base64Encode 将字节数组转换为 base64 编码的字符串
func Base64Encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// Base64Decode 将 base64 编码的字符串转换为字节数组
func Base64Decode(encoded string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(encoded)
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

// buildTextResult 构建标准的文本返回结果
func buildTextResult(text string) *mcp.CallToolResult {
	// 检查空字符串
	if text == "" {
		text = "{}"
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Type: "text",
				Text: text,
			},
		},
	}
}

// TextResult 将任意类型转换为 mcp.CallToolResult
func TextResult[T any](item T) (*mcp.CallToolResult, error) {
	switch v := any(item).(type) {
	case []byte:
		return buildTextResult(string(v)), nil
	case string:
		return buildTextResult(v), nil
	case []string:
		// 优化：预分配容量
		contents := make([]mcp.Content, 0, len(v))
		for _, s := range v {
			contents = append(contents, &mcp.TextContent{
				Type: "text",
				Text: s,
			})
		}
		return &mcp.CallToolResult{Content: contents}, nil
	case *mcp.CallToolResult:
		return v, nil
	default:
		bytes, err := json.Marshal(item)
		if err != nil {
			return nil, fmt.Errorf("无法将项目序列化为JSON: %v", err)
		}
		return buildTextResult(string(bytes)), nil
	}
}
