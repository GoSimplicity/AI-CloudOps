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

package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gorm.io/plugin/soft_delete"
)

// Model 通用基础模型
type Model struct {
	ID        int                   `json:"id" gorm:"primaryKey;autoIncrement;comment:主键ID"`
	CreatedAt time.Time             `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt time.Time             `json:"updated_at" gorm:"autoUpdateTime;comment:更新时间"`
	DeletedAt soft_delete.DeletedAt `json:"deleted_at" gorm:"index;comment:删除时间"`
}

// ListReq 通用列表请求
type ListReq struct {
	Page   int    `json:"page" form:"page" binding:"omitempty,min=1"`
	Size   int    `json:"size" form:"size" binding:"omitempty,min=10,max=100"`
	Search string `json:"search" form:"search" binding:"omitempty"`
}

// ListResp 通用列表响应
type ListResp[T any] struct {
	Items []T   `json:"items"` // 数据列表
	Total int64 `json:"total"` // 总数
}

// StringList 用于存储字符串数组，支持多种数据库格式
type StringList []string

// Scan 实现 sql.Scanner 接口
func (s *StringList) Scan(val interface{}) error {
	if val == nil {
		*s = StringList{}
		return nil
	}

	var str string
	switch v := val.(type) {
	case []byte:
		str = string(v)
	case string:
		str = v
	default:
		return fmt.Errorf("cannot scan %T into StringList", val)
	}

	str = strings.TrimSpace(str)
	if str == "" || str == "[]" || str == "null" {
		*s = StringList{}
		return nil
	}

	// 优先尝试 JSON 解析
	var arr []string
	if err := json.Unmarshal([]byte(str), &arr); err == nil {
		*s = StringList(arr)
		return nil
	}

	// 兼容其他格式，逗号或竖线分割
	cleanStr := strings.Trim(str, `"'`)
	if cleanStr == "" {
		*s = StringList{}
		return nil
	}

	// 处理逗号分割
	if strings.Contains(cleanStr, ",") {
		parts := strings.Split(cleanStr, ",")
		result := make([]string, 0, len(parts))
		for _, part := range parts {
			if trimmed := strings.TrimSpace(strings.Trim(part, `"'`)); trimmed != "" {
				result = append(result, trimmed)
			}
		}
		*s = StringList(result)
		return nil
	}

	// 处理竖线分割
	if strings.Contains(cleanStr, "|") {
		parts := strings.Split(cleanStr, "|")
		result := make([]string, 0, len(parts))
		for _, part := range parts {
			if trimmed := strings.TrimSpace(strings.Trim(part, `"'`)); trimmed != "" {
				result = append(result, trimmed)
			}
		}
		*s = StringList(result)
		return nil
	}

	// 单元素
	*s = StringList{cleanStr}
	return nil
}

// Value 实现 driver.Valuer 接口
func (s StringList) Value() (driver.Value, error) {
	if len(s) == 0 {
		return "[]", nil
	}
	b, err := json.Marshal([]string(s))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal StringList: %w", err)
	}
	return string(b), nil
}

// MarshalJSON 实现 json.Marshaler 接口
func (s StringList) MarshalJSON() ([]byte, error) {
	return json.Marshal([]string(s))
}

// UnmarshalJSON 实现 json.Unmarshaler 接口
func (s *StringList) UnmarshalJSON(data []byte) error {
	var arr []string
	if err := json.Unmarshal(data, &arr); err != nil {
		return fmt.Errorf("failed to unmarshal StringList: %w", err)
	}
	*s = StringList(arr)
	return nil
}

// JSONMap 通用 JSON 字典类型
type JSONMap map[string]interface{}

// Value 实现 driver.Valuer 接口
func (m JSONMap) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSONMap: %w", err)
	}
	return string(b), nil
}

// Scan 实现 sql.Scanner 接口
func (m *JSONMap) Scan(val interface{}) error {
	if val == nil {
		*m = nil
		return nil
	}

	var data []byte
	switch v := val.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into JSONMap", val)
	}

	if len(data) == 0 {
		*m = nil
		return nil
	}

	return json.Unmarshal(data, m)
}

// KeyValue 用于存储单个键值对
type KeyValue struct {
	Key   string `json:"key" gorm:"size:128;index;comment:键"` // 键
	Value string `json:"value" gorm:"size:256;comment:值"`     // 值
}

// ToMap 转为 map[string]string
func (kv *KeyValue) ToMap() map[string]string {
	if kv == nil {
		return make(map[string]string)
	}
	return map[string]string{kv.Key: kv.Value}
}

// FromMap 从 map[string]string 填充 KeyValue（取第一个键值对）
func (kv *KeyValue) FromMap(m map[string]string) {
	if kv == nil || len(m) == 0 {
		return
	}
	for k, v := range m {
		kv.Key = k
		kv.Value = v
		return // 只取第一个
	}
}

// KeyValueList 表示一组键值对，通常用于存储标签(tags)等场景
type KeyValueList []KeyValue

// Value 实现 driver.Valuer 接口，用于数据库存储
func (kvl KeyValueList) Value() (driver.Value, error) {
	if len(kvl) == 0 {
		return "[]", nil
	}
	b, err := json.Marshal(kvl)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal KeyValueList: %w", err)
	}
	return string(b), nil
}

// Scan 实现 sql.Scanner 接口，用于数据库读取
func (kvl *KeyValueList) Scan(val interface{}) error {
	if val == nil {
		*kvl = KeyValueList{}
		return nil
	}

	var str string
	switch v := val.(type) {
	case []byte:
		str = string(v)
	case string:
		str = v
	default:
		return fmt.Errorf("cannot scan %T into KeyValueList", val)
	}

	str = strings.TrimSpace(str)
	if str == "" || str == "[]" || str == "null" {
		*kvl = KeyValueList{}
		return nil
	}

	var result []KeyValue
	if err := json.Unmarshal([]byte(str), &result); err != nil {
		return fmt.Errorf("failed to unmarshal KeyValueList: %w", err)
	}

	*kvl = KeyValueList(result)
	return nil
}

// MarshalJSON 实现 json.Marshaler 接口
func (kvl KeyValueList) MarshalJSON() ([]byte, error) {
	return json.Marshal([]KeyValue(kvl))
}

// UnmarshalJSON 实现 json.Unmarshaler 接口
func (kvl *KeyValueList) UnmarshalJSON(data []byte) error {
	var arr []KeyValue
	if err := json.Unmarshal(data, &arr); err != nil {
		return fmt.Errorf("failed to unmarshal KeyValueList: %w", err)
	}
	*kvl = KeyValueList(arr)
	return nil
}

// ToMap 转为 map[string]string，常用于标签查询和处理
func (kvl KeyValueList) ToMap() map[string]string {
	m := make(map[string]string, len(kvl))
	for _, kv := range kvl {
		if kv.Key != "" {
			m[kv.Key] = kv.Value
		}
	}
	return m
}

// FromMap 从 map[string]string 填充 KeyValueList，常用于从配置或查询结果构建标签
func (kvl *KeyValueList) FromMap(m map[string]string) {
	if kvl == nil {
		return
	}
	*kvl = make(KeyValueList, 0, len(m))
	for k, v := range m {
		*kvl = append(*kvl, KeyValue{Key: k, Value: v})
	}
}

// AddTag 添加标签，如果键已存在则更新值
func (kvl *KeyValueList) AddTag(key, value string) {
	if kvl == nil || key == "" {
		return
	}

	// 查找是否已存在
	for i, kv := range *kvl {
		if kv.Key == key {
			(*kvl)[i].Value = value
			return
		}
	}

	// 不存在则添加
	*kvl = append(*kvl, KeyValue{Key: key, Value: value})
}

// RemoveTag 删除指定键的标签
func (kvl *KeyValueList) RemoveTag(key string) {
	if kvl == nil {
		return
	}

	for i, kv := range *kvl {
		if kv.Key == key {
			*kvl = append((*kvl)[:i], (*kvl)[i+1:]...)
			return
		}
	}
}

// GetTag 获取指定键的标签值
func (kvl KeyValueList) GetTag(key string) (string, bool) {
	for _, kv := range kvl {
		if kv.Key == key {
			return kv.Value, true
		}
	}
	return "", false
}

// HasTag 判断是否包含指定键的标签
func (kvl KeyValueList) HasTag(key string) bool {
	_, exists := kvl.GetTag(key)
	return exists
}

// FilterByKey 根据键前缀过滤标签
func (kvl KeyValueList) FilterByKey(prefix string) KeyValueList {
	var result KeyValueList
	for _, kv := range kvl {
		if strings.HasPrefix(kv.Key, prefix) {
			result = append(result, kv)
		}
	}
	return result
}

// Keys 获取所有标签键
func (kvl KeyValueList) Keys() []string {
	keys := make([]string, 0, len(kvl))
	for _, kv := range kvl {
		keys = append(keys, kv.Key)
	}
	return keys
}

// IsEmpty 判断标签列表是否为空
func (kvl KeyValueList) IsEmpty() bool {
	return len(kvl) == 0
}

// String 返回标签的字符串表示，格式: {key1=value1, key2=value2}
func (kvl KeyValueList) String() string {
	if len(kvl) == 0 {
		return "{}"
	}
	pairs := make([]string, 0, len(kvl))
	for _, kv := range kvl {
		pairs = append(pairs, fmt.Sprintf("%s=%s", kv.Key, kv.Value))
	}
	return "{" + strings.Join(pairs, ", ") + "}"
}
