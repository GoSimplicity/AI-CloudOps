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

// VPC管理相关方法
// ListVpcs, GetVpc, CreateVpc, DeleteVpc 及相关辅助函数

// ListVPCs 获取指定region下的VPC列表，支持分页。
func (a *AWSProviderImpl) ListVPCs(ctx context.Context, region string, pageNumber, pageSize int) ([]*model.ResourceVpc, int64, error) {
	if region == "" {
		return nil, 0, fmt.Errorf("region cannot be empty")
	}
	if pageNumber <= 0 || pageSize <= 0 {
		return nil, 0, fmt.Errorf("pageNumber and pageSize must be positive integers")
	}

	if err := a.ensureServicesInitialized(); err != nil {
		return nil, 0, fmt.Errorf("AWS服务未初始化: %w", err)
	}

	if a.VpcService == nil {
		return nil, 0, fmt.Errorf("AWS VPC SDK未初始化，请先调用InitializeProvider")
	}

	req := &aws.ListVpcsRequest{
		Region: region,
		Page:   pageNumber,
		Size:   pageSize,
	}

	resp, total, err := a.VpcService.ListVpcs(ctx, req)
	if err != nil {
		a.logger.Error("failed to list VPCs", zap.Error(err), zap.String("region", region))
		return nil, 0, fmt.Errorf("list VPCs failed: %w", err)
	}

	if resp == nil || len(resp.Vpcs) == 0 {
		return nil, 0, nil
	}

	result := make([]*model.ResourceVpc, 0, len(resp.Vpcs))
	for _, vpc := range resp.Vpcs {
		result = append(result, a.convertToResourceVpcFromListVpc(vpc, region))
	}

	return result, total, nil
}

// GetVPC 获取指定region下的VPC详情。
func (a *AWSProviderImpl) GetVPC(ctx context.Context, region string, vpcID string) (*model.ResourceVpc, error) {
	if region == "" || vpcID == "" {
		return nil, fmt.Errorf("region and vpcID cannot be empty")
	}

	if err := a.ensureServicesInitialized(); err != nil {
		return nil, fmt.Errorf("AWS服务未初始化: %w", err)
	}

	if a.VpcService == nil {
		return nil, fmt.Errorf("AWS VPC SDK未初始化，请先调用InitializeProvider")
	}

	vpcDetail, err := a.VpcService.GetVpcDetail(ctx, region, vpcID)
	if err != nil {
		a.logger.Error("failed to get VPC detail", zap.Error(err), zap.String("vpcID", vpcID))
		return nil, fmt.Errorf("get VPC detail failed: %w", err)
	}

	if vpcDetail == nil {
		return nil, fmt.Errorf("VPC not found")
	}

	return a.convertToResourceVpcFromDetail(vpcDetail, region), nil
}

// CreateVPC 创建VPC，支持指定配置。
func (a *AWSProviderImpl) CreateVPC(ctx context.Context, region string, config *model.CreateVpcResourceReq) (*model.ResourceVpc, error) {
	if region == "" {
		return nil, fmt.Errorf("region cannot be empty")
	}
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if err := a.ensureServicesInitialized(); err != nil {
		return nil, fmt.Errorf("AWS服务未初始化: %w", err)
	}

	if a.VpcService == nil {
		return nil, fmt.Errorf("AWS VPC SDK未初始化，请先调用InitializeProvider")
	}

	// 准备标签
	tags := make(map[string]string)
	if config.VpcName != "" {
		tags["Name"] = config.VpcName
	}
	if config.Description != "" {
		tags["Description"] = config.Description
	}

	req := &aws.CreateVpcRequest{
		Region:           region,
		VpcName:          config.VpcName,
		CidrBlock:        config.CidrBlock,
		Description:      config.Description,
		AvailabilityZone: config.ZoneId,
		SubnetName:       config.VSwitchName,
		SubnetCidrBlock:  config.VSwitchCidrBlock,
		Tags:             tags,
	}

	resp, err := a.VpcService.CreateVPC(ctx, req)
	if err != nil {
		a.logger.Error("failed to create VPC", zap.Error(err), zap.String("region", region))
		return nil, fmt.Errorf("create VPC failed: %w", err)
	}

	if resp == nil {
		return nil, fmt.Errorf("create VPC response is nil")
	}

	// 获取创建的VPC详情
	vpcDetail, err := a.VpcService.GetVpcDetail(ctx, region, resp.VpcId)
	if err != nil {
		a.logger.Warn("failed to get created VPC detail", zap.Error(err), zap.String("vpcId", resp.VpcId))
		// 返回基本信息
		return &model.ResourceVpc{
			InstanceName: config.VpcName,
			InstanceId:   resp.VpcId,
			Provider:     model.CloudProviderAWS,
			RegionId:     region,
			VpcId:        resp.VpcId,
			VpcName:      config.VpcName,
			CidrBlock:    config.CidrBlock,
			Description:  config.Description,
			LastSyncTime: time.Now(),
		}, nil
	}

	return a.convertToResourceVpcFromDetail(vpcDetail, region), nil
}

// DeleteVPC 删除指定region下的VPC。
func (a *AWSProviderImpl) DeleteVPC(ctx context.Context, region string, vpcID string) error {
	if region == "" || vpcID == "" {
		return fmt.Errorf("region and vpcID cannot be empty")
	}

	if err := a.ensureServicesInitialized(); err != nil {
		return fmt.Errorf("AWS服务未初始化: %w", err)
	}

	if a.VpcService == nil {
		return fmt.Errorf("AWS VPC SDK未初始化，请先调用InitializeProvider")
	}

	err := a.VpcService.DeleteVPC(ctx, region, vpcID)
	if err != nil {
		a.logger.Error("failed to delete VPC", zap.Error(err), zap.String("vpcID", vpcID))
		return fmt.Errorf("delete VPC failed: %w", err)
	}

	return nil
}

// 辅助方法：数据转换

// convertToResourceVpcFromListVpc 将AWS VPC转换为ResourceVpc（列表模式）
func (a *AWSProviderImpl) convertToResourceVpcFromListVpc(vpc types.Vpc, region string) *model.ResourceVpc {
	var tags []string
	vpcName := ""
	description := ""

	for _, tag := range vpc.Tags {
		if *tag.Key == "Name" {
			vpcName = *tag.Value
		} else if *tag.Key == "Description" {
			description = *tag.Value
		}
		tags = append(tags, fmt.Sprintf("%s=%s", *tag.Key, *tag.Value))
	}

	// AWS VPC的IPv6 CIDR
	ipv6CidrBlock := ""
	if len(vpc.Ipv6CidrBlockAssociationSet) > 0 {
		ipv6CidrBlock = *vpc.Ipv6CidrBlockAssociationSet[0].Ipv6CidrBlock
	}

	// 检查是否为默认VPC
	isDefault := vpc.IsDefault != nil && *vpc.IsDefault

	return &model.ResourceVpc{
		InstanceName:    vpcName,
		InstanceId:      *vpc.VpcId,
		Provider:        model.CloudProviderAWS,
		RegionId:        region,
		VpcId:           *vpc.VpcId,
		Status:          string(vpc.State),
		CreationTime:    "", // AWS VPC没有直接的创建时间字段
		Description:     description,
		LastSyncTime:    time.Now(),
		Tags:            model.StringList(tags),
		VpcName:         vpcName,
		CidrBlock:       *vpc.CidrBlock,
		Ipv6CidrBlock:   ipv6CidrBlock,
		VSwitchIds:      model.StringList([]string{}), // 子网ID需要额外查询
		IsDefault:       isDefault,
		ResourceGroupId: "", // AWS使用不同的资源组织方式
	}
}

// convertToResourceVpcFromDetail 将AWS VPC转换为ResourceVpc（详情模式）
func (a *AWSProviderImpl) convertToResourceVpcFromDetail(vpc *types.Vpc, region string) *model.ResourceVpc {
	if vpc == nil {
		return nil
	}
	return a.convertToResourceVpcFromListVpc(*vpc, region)
}
