package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"go.uber.org/zap"
)

// 安全组管理相关方法
// ListSecurityGroups, GetSecurityGroup, CreateSecurityGroup, DeleteSecurityGroup 及相关辅助函数

// ListSecurityGroups 获取指定region下的安全组列表，支持分页。
func (a *AWSProviderImpl) ListSecurityGroups(ctx context.Context, region string, pageNumber, pageSize int) ([]*model.ResourceSecurityGroup, int64, error) {
	if region == "" {
		return nil, 0, fmt.Errorf("region cannot be empty")
	}
	if pageNumber <= 0 || pageSize <= 0 {
		return nil, 0, fmt.Errorf("pageNumber and pageSize must be positive integers")
	}

	if err := a.ensureServicesInitialized(); err != nil {
		return nil, 0, fmt.Errorf("AWS服务未初始化: %w", err)
	}

	if a.SecurityGroupService == nil {
		return nil, 0, fmt.Errorf("AWS安全组SDK未初始化，请先调用InitializeProvider")
	}

	req := &aws.ListSecurityGroupsRequest{
		Region:     region,
		PageNumber: pageNumber,
		PageSize:   pageSize,
	}

	resp, total, err := a.SecurityGroupService.ListSecurityGroups(ctx, req)
	if err != nil {
		a.logger.Error("failed to list security groups", zap.Error(err), zap.String("region", region))
		return nil, 0, fmt.Errorf("list security groups failed: %w", err)
	}

	if resp == nil || len(resp.SecurityGroups) == 0 {
		return nil, 0, nil
	}

	result := make([]*model.ResourceSecurityGroup, 0, len(resp.SecurityGroups))
	for _, sg := range resp.SecurityGroups {
		result = append(result, a.convertToResourceSecurityGroupFromList(sg, region))
	}

	return result, total, nil
}

// GetSecurityGroup 获取指定region下的安全组详情。
func (a *AWSProviderImpl) GetSecurityGroup(ctx context.Context, region string, securityGroupID string) (*model.ResourceSecurityGroup, error) {
	if region == "" || securityGroupID == "" {
		return nil, fmt.Errorf("region and securityGroupID cannot be empty")
	}

	if err := a.ensureServicesInitialized(); err != nil {
		return nil, fmt.Errorf("AWS服务未初始化: %w", err)
	}

	if a.SecurityGroupService == nil {
		return nil, fmt.Errorf("AWS安全组SDK未初始化，请先调用InitializeProvider")
	}

	sg, err := a.SecurityGroupService.GetSecurityGroupDetail(ctx, region, securityGroupID)
	if err != nil {
		a.logger.Error("failed to get security group detail", zap.Error(err), zap.String("securityGroupID", securityGroupID))
		return nil, fmt.Errorf("get security group detail failed: %w", err)
	}

	if sg == nil {
		return nil, fmt.Errorf("security group not found")
	}

	return a.convertToResourceSecurityGroupFromDetail(sg, region), nil
}

// CreateSecurityGroup 创建安全组。
func (a *AWSProviderImpl) CreateSecurityGroup(ctx context.Context, region string, config *model.CreateSecurityGroupReq) (*model.ResourceSecurityGroup, error) {
	if region == "" {
		return nil, fmt.Errorf("region cannot be empty")
	}
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	if config.SecurityGroupName == "" {
		return nil, fmt.Errorf("security group name cannot be empty")
	}
	if config.VpcId == "" {
		return nil, fmt.Errorf("vpcID cannot be empty")
	}

	if err := a.ensureServicesInitialized(); err != nil {
		return nil, fmt.Errorf("AWS服务未初始化: %w", err)
	}

	if a.SecurityGroupService == nil {
		return nil, fmt.Errorf("AWS安全组SDK未初始化，请先调用InitializeProvider")
	}

	req := &aws.CreateSecurityGroupRequest{
		Region:            region,
		SecurityGroupName: config.SecurityGroupName,
		Description:       config.Description,
		VpcId:             config.VpcId,
		Tags:              config.Tags,
	}

	// 确保有Name标签
	if req.Tags == nil {
		req.Tags = make(map[string]string)
	}
	if _, exists := req.Tags["Name"]; !exists {
		req.Tags["Name"] = config.SecurityGroupName
	}

	resp, err := a.SecurityGroupService.CreateSecurityGroup(ctx, req)
	if err != nil {
		a.logger.Error("failed to create security group", zap.Error(err), zap.String("region", region))
		return nil, fmt.Errorf("create security group failed: %w", err)
	}

	if resp == nil {
		return nil, fmt.Errorf("create security group response is nil")
	}

	// 获取创建的安全组详情
	sg, err := a.SecurityGroupService.GetSecurityGroupDetail(ctx, region, resp.SecurityGroupId)
	if err != nil {
		a.logger.Warn("failed to get created security group detail", zap.Error(err), zap.String("securityGroupId", resp.SecurityGroupId))
		// 返回基本信息
		return &model.ResourceSecurityGroup{
			InstanceName:      config.SecurityGroupName,
			InstanceId:        resp.SecurityGroupId,
			Provider:          model.CloudProviderAWS,
			RegionId:          region,
			SecurityGroupName: config.SecurityGroupName,
			VpcId:             config.VpcId,
			Description:       config.Description,
			LastSyncTime:      time.Now(),
		}, nil
	}

	return a.convertToResourceSecurityGroupFromDetail(sg, region), nil
}

// DeleteSecurityGroup 删除指定region下的安全组。
func (a *AWSProviderImpl) DeleteSecurityGroup(ctx context.Context, region string, securityGroupID string) error {
	if region == "" || securityGroupID == "" {
		return fmt.Errorf("region and securityGroupID cannot be empty")
	}

	if err := a.ensureServicesInitialized(); err != nil {
		return fmt.Errorf("AWS服务未初始化: %w", err)
	}

	if a.SecurityGroupService == nil {
		return fmt.Errorf("AWS安全组SDK未初始化，请先调用InitializeProvider")
	}

	err := a.SecurityGroupService.DeleteSecurityGroup(ctx, region, securityGroupID)
	if err != nil {
		a.logger.Error("failed to delete security group", zap.Error(err), zap.String("securityGroupID", securityGroupID))
		return fmt.Errorf("delete security group failed: %w", err)
	}

	return nil
}

// 辅助方法：数据转换

// convertToResourceSecurityGroupFromList 将AWS安全组转换为ResourceSecurityGroup（列表模式）
func (a *AWSProviderImpl) convertToResourceSecurityGroupFromList(sg types.SecurityGroup, region string) *model.ResourceSecurityGroup {
	var tags []string
	sgName := ""
	description := ""

	for _, tag := range sg.Tags {
		if *tag.Key == "Name" {
			sgName = *tag.Value
		} else if *tag.Key == "Description" {
			description = *tag.Value
		}
		tags = append(tags, fmt.Sprintf("%s=%s", *tag.Key, *tag.Value))
	}

	if sgName == "" {
		sgName = *sg.GroupName
	}
	if description == "" && sg.Description != nil {
		description = *sg.Description
	}

	return &model.ResourceSecurityGroup{
		InstanceName:      sgName,
		InstanceId:        *sg.GroupId,
		Provider:          model.CloudProviderAWS,
		RegionId:          region,
		SecurityGroupName: sgName,
		VpcId:             *sg.VpcId,
		Description:       description,
		CreationTime:      "", // AWS安全组没有直接的创建时间字段
		LastSyncTime:      time.Now(),
		Tags:              model.StringList(tags),
		SecurityGroupType: "normal", // AWS默认类型
	}
}

// convertToResourceSecurityGroupFromDetail 将AWS安全组转换为ResourceSecurityGroup（详情模式）
func (a *AWSProviderImpl) convertToResourceSecurityGroupFromDetail(sg *types.SecurityGroup, region string) *model.ResourceSecurityGroup {
	if sg == nil {
		return nil
	}
	return a.convertToResourceSecurityGroupFromList(*sg, region)
}
