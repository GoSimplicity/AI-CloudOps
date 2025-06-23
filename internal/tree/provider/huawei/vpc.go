package provider

import (
	"context"
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/huawei"
	vpcv3model "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/vpc/v3/model"
	"go.uber.org/zap"
)

// VPC管理相关方法
// ListVPCs, GetVPC, CreateVPC, DeleteVPC, GetZonesByVpc, getSubnetsByVpc 及相关辅助函数

// ListVPCs 获取指定region下的VPC列表，支持分页。
func (h *HuaweiProviderImpl) ListVPCs(ctx context.Context, region string, pageNumber, pageSize int) ([]*model.ResourceVpc, error) {
	if region == "" {
		return nil, fmt.Errorf("region cannot be empty")
	}
	if pageNumber <= 0 || pageSize <= 0 {
		return nil, fmt.Errorf("pageNumber and pageSize must be positive integers")
	}

	if h.vpcService == nil {
		return nil, fmt.Errorf("华为云SDK未初始化，请先调用InitializeProvider")
	}

	req := &huawei.ListVpcsRequest{
		Region: region,
		Page:   pageNumber,
		Size:   pageSize,
	}

	resp, err := h.vpcService.ListVpcs(ctx, req)
	if err != nil {
		h.logger.Error("failed to list VPCs", zap.Error(err), zap.String("region", region))
		return nil, fmt.Errorf("list VPCs failed: %w", err)
	}

	if resp == nil || len(resp.Vpcs) == 0 {
		return nil, nil
	}

	result := make([]*model.ResourceVpc, 0, len(resp.Vpcs))
	for _, vpcData := range resp.Vpcs {
		result = append(result, h.convertToResourceVpcFromListVpc(vpcData, region))
	}

	return result, nil
}

func (h *HuaweiProviderImpl) GetVPC(ctx context.Context, region string, vpcID string) (*model.ResourceVpc, error) {
	if region == "" || vpcID == "" {
		return nil, fmt.Errorf("region and vpcID cannot be empty")
	}

	if h.vpcService == nil {
		return nil, fmt.Errorf("华为云SDK未初始化，请先调用InitializeProvider")
	}

	vpcDetail, err := h.vpcService.GetVpcDetail(ctx, region, vpcID)
	if err != nil {
		h.logger.Error("failed to get VPC detail", zap.Error(err), zap.String("vpcID", vpcID))
		return nil, fmt.Errorf("get VPC detail failed: %w", err)
	}

	if vpcDetail == nil {
		return nil, fmt.Errorf("VPC not found")
	}

	return h.convertToResourceVpcFromDetail(vpcDetail, region), nil
}

func (h *HuaweiProviderImpl) CreateVPC(ctx context.Context, region string, config *model.CreateVpcResourceReq) error {
	if region == "" {
		return fmt.Errorf("region cannot be empty")
	}
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	if h.vpcService == nil {
		return fmt.Errorf("华为云SDK未初始化，请先调用InitializeProvider")
	}

	req := &huawei.CreateVpcRequest{
		Region:          region,
		VpcName:         config.VpcName,
		CidrBlock:       config.CidrBlock,
		Description:     config.Description,
		ZoneId:          config.ZoneId,
		SubnetName:      config.VSwitchName,      // 华为云使用SubnetName
		SubnetCidrBlock: config.VSwitchCidrBlock, // 华为云使用SubnetCidrBlock
	}

	_, err := h.vpcService.CreateVPC(ctx, req)
	if err != nil {
		h.logger.Error("failed to create VPC", zap.Error(err), zap.String("region", region))
		return fmt.Errorf("create VPC failed: %w", err)
	}

	return nil
}

func (h *HuaweiProviderImpl) DeleteVPC(ctx context.Context, region string, vpcID string) error {
	if region == "" || vpcID == "" {
		return fmt.Errorf("region and vpcID cannot be empty")
	}

	if h.vpcService == nil {
		return fmt.Errorf("华为云SDK未初始化，请先调用InitializeProvider")
	}

	err := h.vpcService.DeleteVPC(ctx, region, vpcID)
	if err != nil {
		h.logger.Error("failed to delete VPC", zap.Error(err), zap.String("vpcID", vpcID))
		return fmt.Errorf("delete VPC failed: %w", err)
	}

	return nil
}

// 获取指定VPC下的可用区信息
func (h *HuaweiProviderImpl) GetZonesByVpc(ctx context.Context, region string, vpcId string) ([]*model.ZoneResp, error) {
	if region == "" || vpcId == "" {
		return nil, fmt.Errorf("region and vpcId cannot be empty")
	}
	if h.vpcService == nil {
		return nil, fmt.Errorf("华为云SDK未初始化，请先调用InitializeProvider")
	}
	h.logger.Debug("开始获取VPC可用区", zap.String("region", region), zap.String("vpcId", vpcId))
	vpcDetail, err := h.vpcService.GetVpcDetail(ctx, region, vpcId)
	if err != nil {
		h.logger.Error("获取VPC详情失败", zap.Error(err), zap.String("vpcId", vpcId))
		return nil, fmt.Errorf("获取VPC详情失败: %w", err)
	}
	if vpcDetail == nil {
		return nil, fmt.Errorf("VPC不存在: %s", vpcId)
	}
	subnets, err := h.getSubnetsByVpc(ctx, region, vpcId)
	if err != nil {
		h.logger.Error("获取VPC子网列表失败", zap.Error(err), zap.String("vpcId", vpcId))
		return nil, fmt.Errorf("获取VPC子网列表失败: %w", err)
	}
	zoneMap := make(map[string]*model.ZoneResp)
	for _, subnet := range subnets {
		if subnet.ZoneId != "" {
			zoneMap[subnet.ZoneId] = &model.ZoneResp{
				ZoneId:    subnet.ZoneId,
				LocalName: subnet.ZoneId, // 可根据需要本地化
			}
		}
	}
	var zones []*model.ZoneResp
	for _, zone := range zoneMap {
		zones = append(zones, zone)
	}
	h.logger.Info("获取VPC可用区成功", zap.String("vpcId", vpcId), zap.Int("zoneCount", len(zones)), zap.String("region", region))
	return zones, nil
}

// getSubnetsByVpc 获取指定VPC下的子网列表。
func (h *HuaweiProviderImpl) getSubnetsByVpc(ctx context.Context, region string, vpcId string) ([]*model.ResourceSubnet, error) {
	if region == "" || vpcId == "" {
		return nil, fmt.Errorf("region and vpcId cannot be empty")
	}
	if h.sdk == nil {
		return nil, fmt.Errorf("华为云SDK未初始化，请先调用InitializeProvider")
	}

	// 优先用 ListClouddcnSubnets
	client, err := h.sdk.CreateVpcClient(region, "")
	if err == nil {
		listReq := &vpcv3model.ListClouddcnSubnetsRequest{VpcId: &vpcId}
		resp, err := client.ListClouddcnSubnets(listReq)
		if err == nil && resp.ClouddcnSubnets != nil {
			var result []*model.ResourceSubnet
			for _, subnet := range *resp.ClouddcnSubnets {
				tags := make([]string, 0, len(subnet.Tags))
				for _, tag := range subnet.Tags {
					if tag.Value != nil {
						if v, ok := (*tag.Value).(string); ok {
							tags = append(tags, v)
						}
					}
				}
				creationTime := ""
				if subnet.CreatedAt != nil {
					creationTime = subnet.CreatedAt.String()
				}
				result = append(result, &model.ResourceSubnet{
					SubnetId:     subnet.Id,
					SubnetName:   subnet.Name,
					VpcId:        subnet.VpcId,
					Provider:     model.CloudProviderHuawei,
					RegionId:     region,
					ZoneId:       subnet.AvailabilityZone,
					CidrBlock:    subnet.Cidr,
					CreationTime: creationTime,
					Description:  subnet.Description,
					Tags:         model.StringList(tags),
				})
			}
			return result, nil
		}
	}

	// 兜底：通过 VPC 详情 CloudResources 获取子网类型
	vpcDetail, err := h.vpcService.GetVpcDetail(ctx, region, vpcId)
	if err != nil {
		return nil, err
	}
	var subnets []*model.ResourceSubnet
	if vpcDetail.CloudResources != nil {
		for _, resource := range vpcDetail.CloudResources {
			if resource.ResourceType == "virsubnet" {
				subnets = append(subnets, &model.ResourceSubnet{
					SubnetId: "", // 无法获取具体ID
					VpcId:    vpcId,
					Provider: model.CloudProviderHuawei,
					RegionId: region,
					Status:   "Available",
				})
			}
		}
	}
	return subnets, nil
}
