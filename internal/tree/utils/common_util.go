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
	"context"
	"errors"
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	pkgUtils "github.com/GoSimplicity/AI-CloudOps/pkg/base"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// ValidateAndSetPaginationDefaults 验证并设置分页参数的默认值
func ValidateAndSetPaginationDefaults(page, size *int) {
	if *page <= 0 {
		*page = 1
	}
	if *size <= 0 {
		*size = 10
	}
}

// ValidateID 验证ID是否有效
func ValidateID(id int) error {
	if id <= 0 {
		return errors.New("无效的ID")
	}
	return nil
}

// ValidateTreeNodeIDs 验证树节点ID列表
func ValidateTreeNodeIDs(treeNodeIDs []int) bool {
	return len(treeNodeIDs) > 0
}

// SetSSHDefaults 设置SSH连接的默认值
func SetSSHDefaults(port *int, username *string) {
	if *port == 0 {
		*port = 22
	}
	if *username == "" {
		*username = "root"
	}
}

// EncryptPassword 加密密码
func EncryptPassword(password string) (string, error) {
	if password == "" {
		return "", nil
	}

	encryptionKey := viper.GetString("tree.password_encryption_key")
	if encryptionKey == "" {
		return "", errors.New("未配置密码加密密钥")
	}
	if len(encryptionKey) != 32 {
		return "", errors.New("密码加密密钥长度必须为32字节")
	}

	return pkgUtils.EncryptSecretKey(password, []byte(encryptionKey))
}

// DecryptPassword 解密密码
func DecryptPassword(encryptedPassword string) (string, error) {
	if encryptedPassword == "" {
		return "", nil
	}

	encryptionKey := viper.GetString("tree.password_encryption_key")
	if encryptionKey == "" {
		return "", errors.New("未配置密码加密密钥")
	}
	if len(encryptionKey) != 32 {
		return "", errors.New("密码加密密钥长度必须为32字节")
	}

	return pkgUtils.DecryptSecretKey(encryptedPassword, []byte(encryptionKey))
}

// GetAvailableRegionsByProvider 根据云厂商获取可用区域列表（通过API动态获取）
func GetAvailableRegionsByProvider(ctx context.Context, provider model.CloudProvider, accessKey, secretKey string, logger *zap.Logger) ([]model.AvailableRegion, error) {
	switch provider {
	case model.ProviderAliyun:
		return getAliyunAvailableRegions(ctx, accessKey, secretKey, logger)
	case model.ProviderTencent:
		return getTencentAvailableRegions(ctx, accessKey, secretKey, logger)
	case model.ProviderAWS:
		return getAWSAvailableRegions(ctx, accessKey, secretKey, logger)
	case model.ProviderHuawei:
		return getHuaweiAvailableRegions(ctx, accessKey, secretKey, logger)
	case model.ProviderAzure:
		return getAzureAvailableRegions(ctx, accessKey, secretKey, logger)
	case model.ProviderGCP:
		return getGCPAvailableRegions(ctx, accessKey, secretKey, logger)
	default:
		return []model.AvailableRegion{}, fmt.Errorf("不支持的云厂商: %d", provider)
	}
}

// GetAvailableRegionsByProviderWithoutCredentials 获取云厂商可用区域列表（无需凭证，返回常用区域作为降级方案）
func GetAvailableRegionsByProviderWithoutCredentials(provider model.CloudProvider) []model.AvailableRegion {
	// 返回各云厂商的常用区域作为降级方案
	switch provider {
	case model.ProviderAliyun:
		return []model.AvailableRegion{
			{Region: "cn-hangzhou", RegionName: "华东1（杭州）", Available: true},
			{Region: "cn-shanghai", RegionName: "华东2（上海）", Available: true},
			{Region: "cn-beijing", RegionName: "华北2（北京）", Available: true},
			{Region: "cn-shenzhen", RegionName: "华南1（深圳）", Available: true},
			{Region: "cn-chengdu", RegionName: "西南1（成都）", Available: true},
			{Region: "cn-hongkong", RegionName: "中国香港", Available: true},
		}
	case model.ProviderTencent:
		return []model.AvailableRegion{
			{Region: "ap-beijing", RegionName: "华北地区（北京）", Available: true},
			{Region: "ap-shanghai", RegionName: "华东地区（上海）", Available: true},
			{Region: "ap-guangzhou", RegionName: "华南地区（广州）", Available: true},
			{Region: "ap-chengdu", RegionName: "西南地区（成都）", Available: true},
			{Region: "ap-hongkong", RegionName: "港澳台地区（中国香港）", Available: true},
		}
	case model.ProviderAWS:
		return []model.AvailableRegion{
			{Region: "us-east-1", RegionName: "US East (N. Virginia)", Available: true},
			{Region: "us-west-2", RegionName: "US West (Oregon)", Available: true},
			{Region: "eu-west-1", RegionName: "Europe (Ireland)", Available: true},
			{Region: "ap-southeast-1", RegionName: "Asia Pacific (Singapore)", Available: true},
			{Region: "ap-northeast-1", RegionName: "Asia Pacific (Tokyo)", Available: true},
		}
	case model.ProviderHuawei:
		return []model.AvailableRegion{
			{Region: "cn-north-1", RegionName: "华北-北京一", Available: true},
			{Region: "cn-east-3", RegionName: "华东-上海一", Available: true},
			{Region: "cn-south-1", RegionName: "华南-广州", Available: true},
		}
	case model.ProviderAzure:
		return []model.AvailableRegion{
			{Region: "eastus", RegionName: "East US", Available: true},
			{Region: "westus2", RegionName: "West US 2", Available: true},
			{Region: "westeurope", RegionName: "West Europe", Available: true},
		}
	case model.ProviderGCP:
		return []model.AvailableRegion{
			{Region: "us-central1", RegionName: "Iowa", Available: true},
			{Region: "us-west1", RegionName: "Oregon", Available: true},
			{Region: "europe-west1", RegionName: "Belgium", Available: true},
			{Region: "asia-east1", RegionName: "Taiwan", Available: true},
		}
	default:
		return []model.AvailableRegion{}
	}
}

// getAliyunAvailableRegions 获取阿里云可用区域列表
func getAliyunAvailableRegions(ctx context.Context, accessKey, secretKey string, logger *zap.Logger) ([]model.AvailableRegion, error) {
	// 使用任意区域创建客户端来获取区域列表（获取区域列表本身不依赖具体区域）
	client, err := NewAliyunClient(accessKey, secretKey, "cn-hangzhou", logger)
	if err != nil {
		logger.Error("创建阿里云客户端失败", zap.Error(err))
		// API调用失败时返回降级方案
		return GetAvailableRegionsByProviderWithoutCredentials(model.ProviderAliyun), fmt.Errorf("获取阿里云区域列表失败: %w", err)
	}

	regions, err := client.GetAvailableRegions(ctx)
	if err != nil {
		logger.Error("获取阿里云区域列表失败", zap.Error(err))
		// API调用失败时返回降级方案
		return GetAvailableRegionsByProviderWithoutCredentials(model.ProviderAliyun), fmt.Errorf("获取阿里云区域列表失败: %w", err)
	}

	return regions, nil
}

// getTencentAvailableRegions 获取腾讯云可用区域列表
func getTencentAvailableRegions(ctx context.Context, accessKey, secretKey string, logger *zap.Logger) ([]model.AvailableRegion, error) {
	logger.Warn("腾讯云区域获取功能暂未实现，返回默认区域列表")
	// TODO: 实现腾讯云SDK调用
	return GetAvailableRegionsByProviderWithoutCredentials(model.ProviderTencent), nil
}

// getAWSAvailableRegions 获取AWS可用区域列表
func getAWSAvailableRegions(ctx context.Context, accessKey, secretKey string, logger *zap.Logger) ([]model.AvailableRegion, error) {
	logger.Warn("AWS区域获取功能暂未实现，返回默认区域列表")
	// TODO: 实现AWS SDK调用
	return GetAvailableRegionsByProviderWithoutCredentials(model.ProviderAWS), nil
}

// getHuaweiAvailableRegions 获取华为云可用区域列表
func getHuaweiAvailableRegions(ctx context.Context, accessKey, secretKey string, logger *zap.Logger) ([]model.AvailableRegion, error) {
	logger.Warn("华为云区域获取功能暂未实现，返回默认区域列表")
	// TODO: 实现华为云SDK调用
	return GetAvailableRegionsByProviderWithoutCredentials(model.ProviderHuawei), nil
}

// getAzureAvailableRegions 获取Azure可用区域列表
func getAzureAvailableRegions(ctx context.Context, accessKey, secretKey string, logger *zap.Logger) ([]model.AvailableRegion, error) {
	logger.Warn("Azure区域获取功能暂未实现，返回默认区域列表")
	// TODO: 实现Azure SDK调用
	return GetAvailableRegionsByProviderWithoutCredentials(model.ProviderAzure), nil
}

// getGCPAvailableRegions 获取GCP可用区域列表
func getGCPAvailableRegions(ctx context.Context, accessKey, secretKey string, logger *zap.Logger) ([]model.AvailableRegion, error) {
	logger.Warn("GCP区域获取功能暂未实现，返回默认区域列表")
	// TODO: 实现GCP SDK调用
	return GetAvailableRegionsByProviderWithoutCredentials(model.ProviderGCP), nil
}

// GetProviderName 获取云厂商名称
func GetProviderName(provider model.CloudProvider) string {
	switch provider {
	case model.ProviderAliyun:
		return "阿里云"
	case model.ProviderTencent:
		return "腾讯云"
	case model.ProviderAWS:
		return "AWS"
	case model.ProviderHuawei:
		return "华为云"
	case model.ProviderAzure:
		return "Azure"
	case model.ProviderGCP:
		return "Google Cloud"
	default:
		return "未知"
	}
}
