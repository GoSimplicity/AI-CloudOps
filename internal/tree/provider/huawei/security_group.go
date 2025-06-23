package provider

import (
	"context"
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/huawei"
	"go.uber.org/zap"
)

// 安全组管理相关方法
// ListSecurityGroups, GetSecurityGroup, CreateSecurityGroup, DeleteSecurityGroup 及相关辅助函数

func (h *HuaweiProviderImpl) ListSecurityGroups(ctx context.Context, region string, pageNumber, pageSize int) ([]*model.ResourceSecurityGroup, error) {
	if region == "" {
		return nil, fmt.Errorf("region cannot be empty")
	}
	if pageNumber <= 0 || pageSize <= 0 {
		return nil, fmt.Errorf("pageNumber and pageSize must be positive integers")
	}

	if h.securityGroupService == nil {
		return nil, fmt.Errorf("华为云SDK未初始化，请先调用InitializeProvider")
	}

	req := &huawei.ListSecurityGroupsRequest{
		Region:     region,
		PageNumber: pageNumber,
		PageSize:   pageSize,
	}

	resp, err := h.securityGroupService.ListSecurityGroups(ctx, req)
	if err != nil {
		h.logger.Error("failed to list security groups", zap.Error(err), zap.String("region", region))
		return nil, fmt.Errorf("list security groups failed: %w", err)
	}

	if resp == nil || len(resp.SecurityGroups) == 0 {
		return nil, nil
	}

	result := make([]*model.ResourceSecurityGroup, 0, len(resp.SecurityGroups))
	for _, sg := range resp.SecurityGroups {
		result = append(result, h.convertToResourceSecurityGroupFromList(sg, region))
	}

	return result, nil
}

// GetSecurityGroup 获取指定region下的安全组详情。
func (h *HuaweiProviderImpl) GetSecurityGroup(ctx context.Context, region string, securityGroupID string) (*model.ResourceSecurityGroup, error) {
	if region == "" || securityGroupID == "" {
		return nil, fmt.Errorf("region and securityGroupID cannot be empty")
	}

	if h.securityGroupService == nil {
		return nil, fmt.Errorf("华为云SDK未初始化，请先调用InitializeProvider")
	}

	sg, err := h.securityGroupService.GetSecurityGroupDetail(ctx, region, securityGroupID)
	if err != nil {
		h.logger.Error("failed to get security group detail", zap.Error(err), zap.String("securityGroupID", securityGroupID))
		return nil, fmt.Errorf("get security group detail failed: %w", err)
	}

	if sg == nil {
		return nil, fmt.Errorf("security group not found")
	}

	return h.convertToResourceSecurityGroupFromDetail(sg, region), nil
}

// CreateSecurityGroup 创建安全组。
func (h *HuaweiProviderImpl) CreateSecurityGroup(ctx context.Context, region string, config *model.CreateSecurityGroupReq) error {
	if region == "" {
		return fmt.Errorf("region cannot be empty")
	}
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	if h.securityGroupService == nil {
		return fmt.Errorf("华为云SDK未初始化，请先调用InitializeProvider")
	}

	req := &huawei.CreateSecurityGroupRequest{
		Region:            region,
		SecurityGroupName: config.SecurityGroupName,
		Description:       config.Description,
		VpcId:             config.VpcId,
		SecurityGroupType: config.SecurityGroupType,
		ResourceGroupId:   config.ResourceGroupId,
		Tags:              config.Tags,
	}

	_, err := h.securityGroupService.CreateSecurityGroup(ctx, req)
	if err != nil {
		h.logger.Error("failed to create security group", zap.Error(err), zap.String("region", region))
		return fmt.Errorf("create security group failed: %w", err)
	}

	return nil
}

func (h *HuaweiProviderImpl) DeleteSecurityGroup(ctx context.Context, region string, securityGroupID string) error {
	if region == "" || securityGroupID == "" {
		return fmt.Errorf("region and securityGroupID cannot be empty")
	}

	if h.securityGroupService == nil {
		return fmt.Errorf("华为云SDK未初始化，请先调用InitializeProvider")
	}

	err := h.securityGroupService.DeleteSecurityGroup(ctx, region, securityGroupID)
	if err != nil {
		h.logger.Error("failed to delete security group", zap.Error(err), zap.String("securityGroupID", securityGroupID))
		return fmt.Errorf("delete security group failed: %w", err)
	}

	return nil
}
