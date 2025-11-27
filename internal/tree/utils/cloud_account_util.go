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
	"go.uber.org/zap"
)

// VerifyAliyunCredentials 验证阿里云凭证
func VerifyAliyunCredentials(ctx context.Context, req *model.VerifyCloudCredentialsReq, logger *zap.Logger) error {
	client, err := NewAliyunClient(req.AccessKey, req.SecretKey, req.Region, logger)
	if err != nil {
		logger.Error("创建阿里云客户端失败", zap.Error(err))
		return err
	}

	if err := client.VerifyCredentials(ctx); err != nil {
		logger.Error("验证阿里云凭证失败", zap.Error(err))
		return err
	}

	return nil
}

// VerifyTencentCredentials 验证腾讯云凭证
func VerifyTencentCredentials(ctx context.Context, req *model.VerifyCloudCredentialsReq, logger *zap.Logger) error {
	logger.Warn("腾讯云凭证验证功能暂未实现")
	return fmt.Errorf("腾讯云凭证验证功能暂未实现")
}

// VerifyAWSCredentials 验证AWS凭证
func VerifyAWSCredentials(ctx context.Context, req *model.VerifyCloudCredentialsReq, logger *zap.Logger) error {
	logger.Warn("AWS凭证验证功能暂未实现")
	return fmt.Errorf("AWS凭证验证功能暂未实现")
}

// VerifyHuaweiCredentials 验证华为云凭证
func VerifyHuaweiCredentials(ctx context.Context, req *model.VerifyCloudCredentialsReq, logger *zap.Logger) error {
	logger.Warn("华为云凭证验证功能暂未实现")
	return fmt.Errorf("华为云凭证验证功能暂未实现")
}

// VerifyAzureCredentials 验证Azure凭证
func VerifyAzureCredentials(ctx context.Context, req *model.VerifyCloudCredentialsReq, logger *zap.Logger) error {
	logger.Warn("Azure凭证验证功能暂未实现")
	return fmt.Errorf("Azure凭证验证功能暂未实现")
}

// VerifyGCPCredentials 验证GCP凭证
func VerifyGCPCredentials(ctx context.Context, req *model.VerifyCloudCredentialsReq, logger *zap.Logger) error {
	logger.Warn("GCP凭证验证功能暂未实现")
	return fmt.Errorf("GCP凭证验证功能暂未实现")
}

// GetDefaultRegion 从云账户的区域列表中获取默认区域
// 返回默认区域的Region字符串，如果没有找到默认区域则返回第一个区域
// 如果区域列表为空，返回空字符串和错误
func GetDefaultRegion(regions []*model.CloudAccountRegion) (string, error) {
	if len(regions) == 0 {
		return "", errors.New("云账户没有配置区域信息")
	}

	// 查找默认区域
	for _, region := range regions {
		if region.IsDefault {
			return region.Region, nil
		}
	}

	// 如果没有找到默认区域，返回第一个区域
	return regions[0].Region, nil
}

// SanitizeCloudAccount 清理云账户敏感信息（双重保险）
// 虽然AccessKey和SecretKey已经设置了json:"-"标签，但这个方法提供额外的安全保障
func SanitizeCloudAccount(account *model.CloudAccount) {
	if account == nil {
		return
	}
	// 清空敏感信息
	account.AccessKey = ""
	account.SecretKey = ""
}

// SanitizeCloudAccounts 批量清理云账户敏感信息
func SanitizeCloudAccounts(accounts []*model.CloudAccount) {
	for _, account := range accounts {
		SanitizeCloudAccount(account)
	}
}

// ValidateAndNormalizeRegions 验证和规范化区域列表
// 检查区域是否为空、是否重复、默认区域是否唯一
// 如果没有指定默认区域，会将第一个区域设置为默认
// 返回规范化后的区域列表（新切片），不修改传入的切片
func ValidateAndNormalizeRegions(regions []model.CreateCloudAccountRegionItem) ([]model.CreateCloudAccountRegionItem, error) {
	// 验证区域列表不为空
	if len(regions) == 0 {
		return nil, errors.New("必须至少指定一个区域")
	}

	// 创建规范化后的区域列表副本
	normalized := make([]model.CreateCloudAccountRegionItem, len(regions))
	copy(normalized, regions)

	// 检查是否有重复的区域
	regionMap := make(map[string]bool)
	var defaultCount int
	for i := range normalized {
		// 检查重复
		if regionMap[normalized[i].Region] {
			return nil, fmt.Errorf("区域 %s 重复", normalized[i].Region)
		}
		regionMap[normalized[i].Region] = true

		// 统计默认区域数量
		if normalized[i].IsDefault {
			defaultCount++
		}
	}

	// 确保只有一个默认区域
	if defaultCount == 0 {
		// 如果没有指定默认区域，则设置第一个为默认
		normalized[0].IsDefault = true
	} else if defaultCount > 1 {
		return nil, errors.New("只能设置一个默认区域")
	}

	return normalized, nil
}

// ExportAsJSON 导出为 JSON 格式
func ExportAsJSON(accounts []*model.CloudAccount) interface{} {
	// 为了安全，不导出敏感信息（AccessKey、SecretKey）
	exportAccounts := make([]model.ExportAccount, 0, len(accounts))
	for _, account := range accounts {
		regions := make([]model.ExportRegion, 0, len(account.Regions))
		for _, region := range account.Regions {
			regions = append(regions, model.ExportRegion{
				Region:      region.Region,
				RegionName:  region.RegionName,
				IsDefault:   region.IsDefault,
				Description: region.Description,
			})
		}

		exportAccounts = append(exportAccounts, model.ExportAccount{
			ID:           account.ID,
			Name:         account.Name,
			Provider:     account.Provider,
			ProviderName: GetProviderName(account.Provider),
			AccountID:    account.AccountID,
			AccountName:  account.AccountName,
			AccountAlias: account.AccountAlias,
			Description:  account.Description,
			Status:       int8(account.Status),
			Regions:      regions,
			CreatedAt:    account.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return exportAccounts
}

// ExportAsCSV 导出为 CSV 格式
func ExportAsCSV(accounts []*model.CloudAccount) [][]string {
	csvData := [][]string{
		{"ID", "名称", "云厂商", "账号ID", "账号名称", "账号别名", "描述", "状态", "区域列表", "创建时间"},
	}

	for _, account := range accounts {
		regions := ""
		for i, region := range account.Regions {
			if i > 0 {
				regions += ";"
			}
			regions += fmt.Sprintf("%s(%s)", region.Region, region.RegionName)
		}

		status := "禁用"
		if account.Status == model.CloudAccountEnabled {
			status = "启用"
		}

		csvData = append(csvData, []string{
			fmt.Sprintf("%d", account.ID),
			account.Name,
			GetProviderName(account.Provider),
			account.AccountID,
			account.AccountName,
			account.AccountAlias,
			account.Description,
			status,
			regions,
			account.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return csvData
}
