package provider

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/aws"
	awsSDK "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

// 区域相关方法
// 完全通过AWS SDK动态获取区域信息，不使用任何硬编码

// ListRegions 获取可用区域列表
func (a *AWSProviderImpl) ListRegions(ctx context.Context) ([]*model.RegionResp, error) {
	// 检查缓存是否有效
	if a.cachedRegions != nil && time.Since(a.regionsCacheTime) < time.Hour {
		a.logger.Debug("使用缓存的区域列表", zap.Int("count", len(a.cachedRegions)))
		return a.cachedRegions, nil
	}

	// 缓存失效，重新从AWS SDK获取区域列表
	regions, err := a.getAWSRegionsFromSDK()
	if err != nil {
		a.logger.Error("获取AWS区域列表失败", zap.Error(err))
		return nil, err
	}

	// 更新缓存
	a.cachedRegions = regions
	a.regionsCacheTime = time.Now()
	return regions, nil
}

// RefreshRegions 刷新区域缓存
func (a *AWSProviderImpl) RefreshRegions(ctx context.Context) error {
	a.logger.Info("开始刷新AWS区域缓存")

	regions, err := a.getAWSRegionsFromSDK()
	if err != nil {
		a.logger.Error("刷新AWS区域列表失败", zap.Error(err))
		return fmt.Errorf("刷新区域列表失败: %w", err)
	}

	a.cachedRegions = regions
	a.regionsCacheTime = time.Now()
	a.logger.Info("AWS区域缓存刷新成功", zap.Int("count", len(regions)))
	return nil
}

// SyncRegionsWithCredentials 使用新凭证同步区域
func (a *AWSProviderImpl) SyncRegionsWithCredentials(ctx context.Context, accessKey, secretKey string) error {
	a.logger.Info("开始使用新凭证同步AWS区域")

	// 使用新凭证创建临时SDK
	tempSDK := aws.NewSDK(accessKey, secretKey)
	
	// 获取区域列表
	regions, err := a.getRegionsFromSDK(ctx, tempSDK)
	if err != nil {
		a.logger.Error("使用新凭证获取区域列表失败", zap.Error(err))
		return fmt.Errorf("使用提供的凭证无法获取AWS区域列表: %w", err)
	}

	if len(regions) == 0 {
		return fmt.Errorf("使用提供的凭证无法访问任何AWS区域，请检查AKSK权限")
	}

	// 验证区域可访问性
	validRegions, err := a.validateRegionsAccess(ctx, tempSDK, regions)
	if err != nil {
		a.logger.Error("验证区域可访问性失败", zap.Error(err))
		return fmt.Errorf("验证区域可访问性失败: %w", err)
	}

	// 更新SDK和服务
	a.sdk = tempSDK
	a.EC2Service = aws.NewEC2Service(tempSDK)
	a.VpcService = aws.NewVpcService(tempSDK)
	a.SecurityGroupService = aws.NewSecurityGroupService(tempSDK)
	a.EBSService = aws.NewEBSService(tempSDK)

	// 更新缓存
	a.cachedRegions = validRegions
	a.regionsCacheTime = time.Now()

	a.logger.Info("使用新凭证同步AWS区域成功",
		zap.Int("可访问区域数", len(validRegions)),
		zap.Strings("区域列表", a.getRegionIDs(validRegions)))

	return nil
}

// getAWSRegionsFromSDK 从当前SDK获取区域列表
func (a *AWSProviderImpl) getAWSRegionsFromSDK() ([]*model.RegionResp, error) {
	if a.sdk == nil {
		return nil, fmt.Errorf("AWS SDK未初始化")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return a.getRegionsFromSDK(ctx, a.sdk)
}

// getRegionsFromSDK 使用指定SDK获取区域列表
func (a *AWSProviderImpl) getRegionsFromSDK(ctx context.Context, sdk *aws.SDK) ([]*model.RegionResp, error) {
	a.logger.Info("开始从AWS SDK获取区域列表")

	// 使用默认区域创建EC2客户端来获取所有区域
	defaultRegion := "us-east-1" // AWS全球服务的默认区域
	client, err := sdk.CreateEC2Client(ctx, defaultRegion)
	if err != nil {
		return nil, fmt.Errorf("创建EC2客户端失败: %w", err)
	}

	// 调用DescribeRegions API
	input := &ec2.DescribeRegionsInput{
		AllRegions: awsSDK.Bool(true), // 获取所有区域，包括已选择加入的区域
	}

	output, err := client.DescribeRegions(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("调用DescribeRegions API失败: %w", err)
	}

	if len(output.Regions) == 0 {
		return nil, fmt.Errorf("未获取到任何AWS区域")
	}

	// 转换为内部模型
	regions := make([]*model.RegionResp, 0, len(output.Regions))
	for _, region := range output.Regions {
		if region.RegionName == nil {
			continue
		}

		regionID := *region.RegionName
		
		// 获取区域的本地化名称，如果没有则使用OptInStatus作为描述
		localName := regionID
		if region.OptInStatus != nil {
			localName = fmt.Sprintf("%s (%s)", regionID, string(*region.OptInStatus))
		}

		// 构建区域端点
		endpoint := fmt.Sprintf(a.config.Endpoints.EC2Template, regionID)

		regions = append(regions, &model.RegionResp{
			RegionId:       regionID,
			LocalName:      localName,
			RegionEndpoint: endpoint,
		})

		a.logger.Debug("发现AWS区域", 
			zap.String("regionId", regionID),
			zap.String("localName", localName),
			zap.String("endpoint", endpoint))
	}

	a.logger.Info("成功获取AWS区域列表", zap.Int("count", len(regions)))
	return regions, nil
}

// validateRegionsAccess 验证区域可访问性
func (a *AWSProviderImpl) validateRegionsAccess(ctx context.Context, sdk *aws.SDK, regions []*model.RegionResp) ([]*model.RegionResp, error) {
	a.logger.Info("开始验证区域可访问性", zap.Int("regions", len(regions)))

	validRegions := make([]*model.RegionResp, 0, len(regions))
	
	// 并发验证区域访问
	type regionResult struct {
		region *model.RegionResp
		valid  bool
		error  error
	}

	resultChan := make(chan regionResult, len(regions))
	semaphore := make(chan struct{}, 10) // 限制并发数

	for _, region := range regions {
		go func(r *model.RegionResp) {
			semaphore <- struct{}{} // 获取信号量
			defer func() { <-semaphore }() // 释放信号量

			valid, err := a.validateRegionAccess(ctx, sdk, r.RegionId)
			resultChan <- regionResult{
				region: r,
				valid:  valid,
				error:  err,
			}
		}(region)
	}

	// 收集结果
	for i := 0; i < len(regions); i++ {
		result := <-resultChan
		if result.valid {
			validRegions = append(validRegions, result.region)
			a.logger.Debug("区域验证成功", zap.String("regionId", result.region.RegionId))
		} else {
			a.logger.Debug("区域验证失败", 
				zap.String("regionId", result.region.RegionId),
				zap.Error(result.error))
		}
	}

	a.logger.Info("区域可访问性验证完成", 
		zap.Int("total", len(regions)),
		zap.Int("valid", len(validRegions)))

	return validRegions, nil
}

// validateRegionAccess 验证单个区域的可访问性
func (a *AWSProviderImpl) validateRegionAccess(ctx context.Context, sdk *aws.SDK, regionID string) (bool, error) {
	// 创建区域特定的客户端
	client, err := sdk.CreateEC2Client(ctx, regionID)
	if err != nil {
		return false, fmt.Errorf("创建区域客户端失败: %w", err)
	}

	// 使用超时上下文
	validateCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// 尝试调用一个简单的API来验证访问权限
	input := &ec2.DescribeAvailabilityZonesInput{}
	_, err = client.DescribeAvailabilityZones(validateCtx, input)
	if err != nil {
		return false, fmt.Errorf("验证区域访问权限失败: %w", err)
	}

	return true, nil
}

// getRegionIDs 获取区域ID列表
func (a *AWSProviderImpl) getRegionIDs(regions []*model.RegionResp) []string {
	var regionIDs []string
	for _, region := range regions {
		regionIDs = append(regionIDs, region.RegionId)
	}
	return regionIDs
}

// validateAWSRegion 验证AWS区域名称格式
func (a *AWSProviderImpl) validateAWSRegion(region string) bool {
	if region == "" {
		return false
	}

	// 通过调用AWS API验证区域是否存在
	regions, err := a.getAWSRegionsFromSDK()
	if err != nil {
		a.logger.Error("验证区域时获取区域列表失败", zap.Error(err))
		return false
	}

	for _, r := range regions {
		if r.RegionId == region {
			return true
		}
	}

	return false
}

// getKnownRegions 获取已知的可访问区域
func (a *AWSProviderImpl) getKnownRegions() []AWSRegionInfo {
	var regions []AWSRegionInfo
	
	// 从发现的区域中获取可访问的区域
	for _, regionInfo := range a.discoveredRegions {
		if regionInfo.IsAccessible {
			regions = append(regions, *regionInfo)
		}
	}

	// 如果没有发现的区域，尝试从缓存中获取
	if len(regions) == 0 && a.cachedRegions != nil {
		for _, region := range a.cachedRegions {
			regions = append(regions, AWSRegionInfo{
				RegionID:     region.RegionId,
				LocalName:    region.LocalName,
				IsAccessible: true,
				LastChecked:  time.Now(),
			})
		}
	}

	return regions
}

// probeRegion 探测区域可用性（使用EC2 API）
func (a *AWSProviderImpl) probeRegion(regionID string) bool {
	if a.sdk == nil {
		a.logger.Debug("区域探测失败：SDK未初始化", zap.String("regionId", regionID))
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	valid, err := a.validateRegionAccess(ctx, a.sdk, regionID)
	if err != nil {
		a.logger.Debug("区域探测失败", zap.String("regionId", regionID), zap.Error(err))
		return false
	}

	return valid
}