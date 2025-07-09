package provider

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

// AWS转换器辅助函数
// 通用的数据转换和解析方法

// formatAWSTime 格式化AWS时间为ISO8601格式
func (a *AWSProviderImpl) formatAWSTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339)
}

// parsePortRange 解析端口范围字符串
func (a *AWSProviderImpl) parsePortRange(portRange string) (fromPort, toPort int32, err error) {
	if portRange == "" || portRange == "-1/-1" {
		return -1, -1, nil
	}

	// 处理单个端口
	if !strings.Contains(portRange, "/") && !strings.Contains(portRange, "-") {
		port, err := strconv.Atoi(portRange)
		if err != nil {
			return 0, 0, err
		}
		return int32(port), int32(port), nil
	}

	// 处理范围格式 "80/80" 或 "80-443"
	var parts []string
	if strings.Contains(portRange, "/") {
		parts = strings.Split(portRange, "/")
	} else if strings.Contains(portRange, "-") {
		parts = strings.Split(portRange, "-")
	}

	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid port range format: %s", portRange)
	}

	from, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, err
	}

	to, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, err
	}

	return int32(from), int32(to), nil
}

// formatTags 格式化标签为字符串数组
func (a *AWSProviderImpl) formatTags(tags map[string]string) model.StringList {
	var result []string
	for key, value := range tags {
		result = append(result, fmt.Sprintf("%s=%s", key, value))
	}
	return model.StringList(result)
}

// extractTagValue 从标签中提取指定键的值
func (a *AWSProviderImpl) extractTagValue(tags map[string]*string, key string) string {
	if tags == nil {
		return ""
	}

	if value, exists := tags[key]; exists && value != nil {
		return *value
	}

	return ""
}

// mapAWSVolumeTypeToCategory 直接返回AWS卷类型
func (a *AWSProviderImpl) mapAWSVolumeTypeToCategory(volumeType string) string {
	return volumeType
}

// mapCategoryToAWSVolumeType 直接返回统一类型（或AWS类型）
func (a *AWSProviderImpl) mapCategoryToAWSVolumeType(category string) string {
	return category
}

// generateDefaultDevice 返回空字符串，让AWS自动分配
func (a *AWSProviderImpl) generateDefaultDevice(instanceType string) string {
	return ""
}

// calculateCostEstimate 计算资源成本估算（仅供参考）
func (a *AWSProviderImpl) calculateCostEstimate(instanceType string, volumeSize int, region string) float64 {
	// 这里应该基于AWS定价API或预设的价格表计算
	// 当前返回一个简单的估算值
	baseCost := 0.1 // 每GB每小时基础成本

	switch {
	case strings.HasPrefix(instanceType, "t3"):
		baseCost = 0.05
	case strings.HasPrefix(instanceType, "m5"):
		baseCost = 0.08
	case strings.HasPrefix(instanceType, "c5"):
		baseCost = 0.10
	case strings.HasPrefix(instanceType, "r5"):
		baseCost = 0.12
	}

	return baseCost * float64(volumeSize)
}
