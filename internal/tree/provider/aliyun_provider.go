package provider

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/aliyun"
	ecs "github.com/alibabacloud-go/ecs-20140526/v2/client"
	"github.com/alibabacloud-go/tea/tea"
	vpc "github.com/alibabacloud-go/vpc-20160428/v2/client"
	"go.uber.org/zap"
)

type AliyunProviderImpl struct {
	logger               *zap.Logger
	sdk                  *aliyun.SDK
	ecsService           *aliyun.EcsService
	vpcService           *aliyun.VpcService
	diskService          *aliyun.DiskService
	securityGroupService *aliyun.SecurityGroupService
}

func NewAliyunProvider(logger *zap.Logger) *AliyunProviderImpl {
	accessKeyId := os.Getenv("ALIYUN_ACCESS_KEY_ID")
	accessKeySecret := os.Getenv("ALIYUN_ACCESS_KEY_SECRET")

	// 检查必要的环境变量
	if accessKeyId == "" || accessKeySecret == "" {
		logger.Error("ALIYUN_ACCESS_KEY_ID and ALIYUN_ACCESS_KEY_SECRET environment variables are required")
		return nil
	}

	sdk := aliyun.NewSDK(logger, accessKeyId, accessKeySecret)

	return &AliyunProviderImpl{
		logger:               logger,
		sdk:                  sdk,
		ecsService:           aliyun.NewEcsService(sdk),
		vpcService:           aliyun.NewVpcService(sdk),
		diskService:          aliyun.NewDiskService(sdk),
		securityGroupService: aliyun.NewSecurityGroupService(sdk),
	}
}

// CreateInstance 创建ECS实例
func (a *AliyunProviderImpl) CreateInstance(ctx context.Context, region string, config *model.CreateEcsResourceReq) error {
	if region == "" {
		return fmt.Errorf("region cannot be empty")
	}
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	req := &aliyun.CreateInstanceRequest{
		Region:             region,
		ZoneId:             config.ZoneId,
		ImageId:            config.ImageId,
		InstanceType:       config.InstanceType,
		SecurityGroupIds:   config.SecurityGroupIds,
		VSwitchId:          config.VSwitchId,
		InstanceName:       config.InstanceName,
		Hostname:           config.Hostname,
		Password:           config.Password,
		Description:        config.Description,
		Amount:             config.Amount,
		DryRun:             config.DryRun,
		InstanceChargeType: string(config.InstanceChargeType),
		SystemDiskCategory: config.SystemDiskCategory,
		SystemDiskSize:     config.SystemDiskSize,
		DataDiskCategory:   config.DataDiskCategory,
		DataDiskSize:       config.DataDiskSize,
	}

	_, err := a.ecsService.CreateInstance(ctx, req)
	if err != nil {
		a.logger.Error("failed to create instance", zap.Error(err), zap.String("region", region))
		return fmt.Errorf("create instance failed: %w", err)
	}

	return nil
}

// StartInstance 启动ECS实例
func (a *AliyunProviderImpl) StartInstance(ctx context.Context, region string, instanceID string) error {
	if region == "" || instanceID == "" {
		return fmt.Errorf("region and instanceID cannot be empty")
	}

	err := a.ecsService.StartInstance(ctx, region, instanceID)
	if err != nil {
		a.logger.Error("failed to start instance", zap.Error(err), zap.String("instanceID", instanceID))
		return fmt.Errorf("start instance failed: %w", err)
	}

	return nil
}

// StopInstance 停止ECS实例
func (a *AliyunProviderImpl) StopInstance(ctx context.Context, region string, instanceID string) error {
	if region == "" || instanceID == "" {
		return fmt.Errorf("region and instanceID cannot be empty")
	}

	err := a.ecsService.StopInstance(ctx, region, instanceID, false)
	if err != nil {
		a.logger.Error("failed to stop instance", zap.Error(err), zap.String("instanceID", instanceID))
		return fmt.Errorf("stop instance failed: %w", err)
	}

	return nil
}

// RestartInstance 重启ECS实例
func (a *AliyunProviderImpl) RestartInstance(ctx context.Context, region string, instanceID string) error {
	if region == "" || instanceID == "" {
		return fmt.Errorf("region and instanceID cannot be empty")
	}

	err := a.ecsService.RestartInstance(ctx, region, instanceID)
	if err != nil {
		a.logger.Error("failed to restart instance", zap.Error(err), zap.String("instanceID", instanceID))
		return fmt.Errorf("restart instance failed: %w", err)
	}

	return nil
}

// DeleteInstance 删除ECS实例
func (a *AliyunProviderImpl) DeleteInstance(ctx context.Context, region string, instanceID string) error {
	if region == "" || instanceID == "" {
		return fmt.Errorf("region and instanceID cannot be empty")
	}

	err := a.ecsService.DeleteInstance(ctx, region, instanceID, true)
	if err != nil {
		a.logger.Error("failed to delete instance", zap.Error(err), zap.String("instanceID", instanceID))
		return fmt.Errorf("delete instance failed: %w", err)
	}

	return nil
}

// ListInstances 列出ECS实例
func (a *AliyunProviderImpl) ListInstances(ctx context.Context, region string, page int, size int) ([]*model.ResourceEcs, int64, error) {
	if region == "" {
		return nil, 0, fmt.Errorf("region cannot be empty")
	}
	if page <= 0 || size <= 0 {
		return nil, 0, fmt.Errorf("page and size must be positive integers")
	}

	req := &aliyun.ListInstancesRequest{
		Region: region,
		Page:   page,
		Size:   size,
	}

	resp, err := a.ecsService.ListInstances(ctx, req)
	if err != nil {
		a.logger.Error("failed to list instances", zap.Error(err), zap.String("region", region))
		return nil, 0, fmt.Errorf("list instances failed: %w", err)
	}

	if resp == nil || len(resp.Instances) == 0 {
		return []*model.ResourceEcs{}, 0, nil
	}

	result := make([]*model.ResourceEcs, 0, len(resp.Instances))
	for _, instance := range resp.Instances {
		if instance == nil {
			continue
		}
		result = append(result, a.convertToResourceEcsFromListInstance(instance))
	}

	return result, resp.Total, nil
}

// GetInstanceDetail 获取ECS实例详情
func (a *AliyunProviderImpl) GetInstanceDetail(ctx context.Context, region string, instanceID string) (*model.ResourceEcs, error) {
	if region == "" || instanceID == "" {
		return nil, fmt.Errorf("region and instanceID cannot be empty")
	}

	instance, err := a.ecsService.GetInstanceDetail(ctx, region, instanceID)
	if err != nil {
		a.logger.Error("failed to get instance detail", zap.Error(err), zap.String("instanceID", instanceID))
		return nil, fmt.Errorf("get instance detail failed: %w", err)
	}

	if instance == nil {
		return nil, fmt.Errorf("instance not found")
	}

	return a.convertToResourceEcsFromInstanceDetail(instance), nil
}

// SyncResources 同步资源
func (a *AliyunProviderImpl) SyncResources(ctx context.Context, region string) error {
	if region == "" {
		return fmt.Errorf("region cannot be empty")
	}

	a.logger.Info("starting resource sync", zap.String("region", region))

	// TODO: 实现具体的资源同步逻辑
	// 可以包括同步ECS实例、VPC、安全组等资源

	a.logger.Info("resource sync completed", zap.String("region", region))
	return nil
}

// getFirstIP 获取第一个IP地址
func getFirstIP(ips []string) string {
	if len(ips) > 0 {
		return ips[0]
	}
	return ""
}

// CreateVPC 创建VPC
func (a *AliyunProviderImpl) CreateVPC(ctx context.Context, region string, config *model.CreateVpcResourceReq) error {
	if region == "" {
		return fmt.Errorf("region cannot be empty")
	}
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	req := &aliyun.CreateVpcRequest{
		Region:           region,
		VpcName:          config.VpcName,
		CidrBlock:        config.CidrBlock,
		Description:      config.Description,
		ZoneId:           config.ZoneId,
		VSwitchName:      config.VSwitchName,
		VSwitchCidrBlock: config.VSwitchCidrBlock,
	}

	_, err := a.vpcService.CreateVPC(ctx, req)
	if err != nil {
		a.logger.Error("failed to create VPC", zap.Error(err), zap.String("region", region))
		return fmt.Errorf("create VPC failed: %w", err)
	}

	return nil
}

// DeleteVPC 删除VPC
func (a *AliyunProviderImpl) DeleteVPC(ctx context.Context, region string, vpcID string) error {
	if region == "" || vpcID == "" {
		return fmt.Errorf("region and vpcID cannot be empty")
	}

	err := a.vpcService.DeleteVPC(ctx, region, vpcID)
	if err != nil {
		a.logger.Error("failed to delete VPC", zap.Error(err), zap.String("vpcID", vpcID))
		return fmt.Errorf("delete VPC failed: %w", err)
	}

	return nil
}

// GetVpcDetail 获取VPC详情
func (a *AliyunProviderImpl) GetVpcDetail(ctx context.Context, region string, vpcID string) (*model.ResourceVpc, error) {
	if region == "" || vpcID == "" {
		return nil, fmt.Errorf("region and vpcID cannot be empty")
	}

	vpcDetail, err := a.vpcService.GetVpcDetail(ctx, region, vpcID)
	if err != nil {
		a.logger.Error("failed to get VPC detail", zap.Error(err), zap.String("vpcID", vpcID))
		return nil, fmt.Errorf("get VPC detail failed: %w", err)
	}

	if vpcDetail == nil {
		return nil, fmt.Errorf("VPC not found")
	}

	return a.convertToResourceVpcFromDetail(vpcDetail, region), nil
}

// ListVPCs 获取VPC列表
func (a *AliyunProviderImpl) ListVPCs(ctx context.Context, region string, pageNumber int, pageSize int) ([]*model.ResourceVpc, int64, error) {
	if region == "" {
		return nil, 0, fmt.Errorf("region cannot be empty")
	}
	if pageNumber <= 0 || pageSize <= 0 {
		return nil, 0, fmt.Errorf("pageNumber and pageSize must be positive integers")
	}

	req := &aliyun.ListVpcsRequest{
		Region: region,
		Page:   pageNumber,
		Size:   pageSize,
	}

	resp, err := a.vpcService.ListVpcs(ctx, req)
	if err != nil {
		a.logger.Error("failed to list VPCs", zap.Error(err), zap.String("region", region))
		return nil, 0, fmt.Errorf("list VPCs failed: %w", err)
	}

	if resp == nil || len(resp.Vpcs) == 0 {
		return []*model.ResourceVpc{}, 0, nil
	}

	result := make([]*model.ResourceVpc, 0, len(resp.Vpcs))
	for _, vpcData := range resp.Vpcs {
		if vpcData == nil {
			continue
		}
		result = append(result, a.convertToResourceVpcFromListVpc(vpcData, region))
	}

	return result, resp.Total, nil
}

// ListSecurityGroups 获取安全组列表
func (a *AliyunProviderImpl) ListSecurityGroups(ctx context.Context, region string, pageNumber int, pageSize int) ([]*model.ResourceSecurityGroup, int64, error) {
	if region == "" {
		return nil, 0, fmt.Errorf("region cannot be empty")
	}
	if pageNumber <= 0 || pageSize <= 0 {
		return nil, 0, fmt.Errorf("pageNumber and pageSize must be positive integers")
	}

	req := &aliyun.ListSecurityGroupsRequest{
		Region:     region,
		PageNumber: pageNumber,
		PageSize:   pageSize,
	}

	resp, err := a.securityGroupService.ListSecurityGroups(ctx, req)
	if err != nil {
		a.logger.Error("failed to list security groups", zap.Error(err), zap.String("region", region))
		return nil, 0, fmt.Errorf("list security groups failed: %w", err)
	}

	if resp == nil || len(resp.SecurityGroups) == 0 {
		return []*model.ResourceSecurityGroup{}, 0, nil
	}

	result := make([]*model.ResourceSecurityGroup, 0, len(resp.SecurityGroups))
	for _, sg := range resp.SecurityGroups {
		if sg == nil {
			continue
		}
		result = append(result, a.convertToResourceSecurityGroupFromList(sg, region))
	}

	return result, resp.Total, nil
}

// CreateSecurityGroup 创建安全组
func (a *AliyunProviderImpl) CreateSecurityGroup(ctx context.Context, region string, config *model.CreateSecurityGroupReq) error {
	if region == "" {
		return fmt.Errorf("region cannot be empty")
	}
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	req := &aliyun.CreateSecurityGroupRequest{
		Region:            region,
		SecurityGroupName: config.SecurityGroupName,
		Description:       config.Description,
		VpcId:             config.VpcId,
		SecurityGroupType: config.SecurityGroupType,
		ResourceGroupId:   config.ResourceGroupId,
		Tags:              config.Tags,
	}

	_, err := a.securityGroupService.CreateSecurityGroup(ctx, req)
	if err != nil {
		a.logger.Error("failed to create security group", zap.Error(err), zap.String("region", region))
		return fmt.Errorf("create security group failed: %w", err)
	}

	return nil
}

// DeleteSecurityGroup 删除安全组
func (a *AliyunProviderImpl) DeleteSecurityGroup(ctx context.Context, region string, securityGroupID string) error {
	if region == "" || securityGroupID == "" {
		return fmt.Errorf("region and securityGroupID cannot be empty")
	}

	err := a.securityGroupService.DeleteSecurityGroup(ctx, region, securityGroupID)
	if err != nil {
		a.logger.Error("failed to delete security group", zap.Error(err), zap.String("securityGroupID", securityGroupID))
		return fmt.Errorf("delete security group failed: %w", err)
	}

	return nil
}

// GetSecurityGroupDetail 获取安全组详情
func (a *AliyunProviderImpl) GetSecurityGroupDetail(ctx context.Context, region string, securityGroupID string) (*model.ResourceSecurityGroup, error) {
	if region == "" || securityGroupID == "" {
		return nil, fmt.Errorf("region and securityGroupID cannot be empty")
	}

	sg, err := a.securityGroupService.GetSecurityGroupDetail(ctx, region, securityGroupID)
	if err != nil {
		a.logger.Error("failed to get security group detail", zap.Error(err), zap.String("securityGroupID", securityGroupID))
		return nil, fmt.Errorf("get security group detail failed: %w", err)
	}

	if sg == nil {
		return nil, fmt.Errorf("security group not found")
	}

	return a.convertToResourceSecurityGroupFromDetail(sg, region), nil
}

// CreateDisk 创建磁盘
func (a *AliyunProviderImpl) CreateDisk(ctx context.Context, region string, config *model.DiskCreationParams) error {
	if region == "" {
		return fmt.Errorf("region cannot be empty")
	}
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	req := &aliyun.CreateDiskRequest{
		Region:       region,
		ZoneId:       config.ZoneId,
		DiskName:     config.DiskName,
		DiskCategory: config.DiskCategory,
		Size:         config.Size,
		Description:  config.Description,
		Tags:         config.Tags,
	}

	_, err := a.diskService.CreateDisk(ctx, req)
	if err != nil {
		a.logger.Error("failed to create disk", zap.Error(err), zap.String("region", region))
		return fmt.Errorf("create disk failed: %w", err)
	}

	return nil
}

// AttachDisk 挂载磁盘
func (a *AliyunProviderImpl) AttachDisk(ctx context.Context, region string, zoneId string, diskCategory string, diskName string, diskSize int, description string, instanceID string) error {
	if region == "" || instanceID == "" {
		return fmt.Errorf("region and instanceID cannot be empty")
	}

	// 首先创建磁盘
	req := &aliyun.CreateDiskRequest{
		Region:       region,
		ZoneId:       zoneId,
		DiskName:     diskName,
		DiskCategory: diskCategory,
		Size:         diskSize,
		Description:  description,
	}

	diskResp, err := a.diskService.CreateDisk(ctx, req)
	if err != nil {
		a.logger.Error("failed to create disk for attach", zap.Error(err), zap.String("instanceID", instanceID))
		return fmt.Errorf("create disk for attach failed: %w", err)
	}

	// 然后挂载磁盘到实例
	if diskResp != nil && diskResp.DiskId != "" {
		err = a.diskService.AttachDisk(ctx, region, diskResp.DiskId, instanceID)
		if err != nil {
			a.logger.Error("failed to attach disk", zap.Error(err), zap.String("diskID", diskResp.DiskId), zap.String("instanceID", instanceID))
			return fmt.Errorf("attach disk failed: %w", err)
		}
	}

	return nil
}

// DetachDisk 卸载磁盘
func (a *AliyunProviderImpl) DetachDisk(ctx context.Context, region string, diskID string, instanceID string) error {
	if region == "" || diskID == "" || instanceID == "" {
		return fmt.Errorf("region, diskID and instanceID cannot be empty")
	}

	err := a.diskService.DetachDisk(ctx, region, diskID, instanceID)
	if err != nil {
		a.logger.Error("failed to detach disk", zap.Error(err), zap.String("diskID", diskID), zap.String("instanceID", instanceID))
		return fmt.Errorf("detach disk failed: %w", err)
	}

	return nil
}

// ListDisks 列出磁盘
func (a *AliyunProviderImpl) ListDisks(ctx context.Context, region string, page int, size int) (model.ListResp[*ecs.DescribeDisksResponseBodyDisksDisk], error) {
	if region == "" {
		return model.ListResp[*ecs.DescribeDisksResponseBodyDisksDisk]{}, fmt.Errorf("region cannot be empty")
	}
	if page <= 0 || size <= 0 {
		return model.ListResp[*ecs.DescribeDisksResponseBodyDisksDisk]{}, fmt.Errorf("page and size must be positive integers")
	}

	req := &aliyun.ListDisksRequest{
		Region: region,
		Page:   page,
		Size:   size,
	}

	resp, err := a.diskService.ListDisks(ctx, req)
	if err != nil {
		a.logger.Error("failed to list disks", zap.Error(err), zap.String("region", region))
		return model.ListResp[*ecs.DescribeDisksResponseBodyDisksDisk]{}, fmt.Errorf("list disks failed: %w", err)
	}

	return model.ListResp[*ecs.DescribeDisksResponseBodyDisksDisk]{
		Items: resp.Disks,
		Total: resp.Total,
	}, nil
}

// DeleteDisk 删除磁盘
func (a *AliyunProviderImpl) DeleteDisk(ctx context.Context, region string, diskID string) error {
	if region == "" || diskID == "" {
		return fmt.Errorf("region and diskID cannot be empty")
	}

	err := a.diskService.DeleteDisk(ctx, region, diskID)
	if err != nil {
		a.logger.Error("failed to delete disk", zap.Error(err), zap.String("diskID", diskID))
		return fmt.Errorf("delete disk failed: %w", err)
	}

	return nil
}

// GetZonesByVpc 获取VPC可用区
func (a *AliyunProviderImpl) GetZonesByVpc(ctx context.Context, region string, vpcId string) ([]*model.ZoneResp, error) {
	if region == "" || vpcId == "" {
		return nil, fmt.Errorf("region and vpcId cannot be empty")
	}

	zones, _, err := a.vpcService.GetZonesByVpc(ctx, region, vpcId)
	if err != nil {
		a.logger.Error("failed to get zones by VPC", zap.Error(err), zap.String("vpcId", vpcId))
		return nil, fmt.Errorf("get zones by VPC failed: %w", err)
	}

	if len(zones) == 0 {
		return []*model.ZoneResp{}, nil
	}

	result := make([]*model.ZoneResp, 0, len(zones))
	for _, zone := range zones {
		if zone == nil {
			continue
		}
		result = append(result, &model.ZoneResp{
			ZoneId:    tea.StringValue(zone.ZoneId),
			LocalName: tea.StringValue(zone.LocalName),
		})
	}

	return result, nil
}

// ListRegions 列出区域
func (a *AliyunProviderImpl) ListRegions(ctx context.Context) ([]*model.RegionResp, error) {
	regions, err := a.ecsService.ListRegions(ctx)
	if err != nil {
		a.logger.Error("failed to list regions", zap.Error(err))
		return nil, fmt.Errorf("list regions failed: %w", err)
	}

	if len(regions) == 0 {
		return []*model.RegionResp{}, nil
	}

	result := make([]*model.RegionResp, 0, len(regions))
	for _, region := range regions {
		if region == nil {
			continue
		}
		result = append(result, &model.RegionResp{
			RegionId:       tea.StringValue(region.RegionId),
			LocalName:      tea.StringValue(region.LocalName),
			RegionEndpoint: tea.StringValue(region.RegionEndpoint),
		})
	}

	return result, nil
}

// convertToResourceVpcFromListVpc 转换VPC列表项为内部模型
func (a *AliyunProviderImpl) convertToResourceVpcFromListVpc(vpcData *vpc.DescribeVpcsResponseBodyVpcsVpc, region string) *model.ResourceVpc {
	if vpcData == nil {
		return nil
	}

	var tags []string
	if vpcData.Tags != nil && vpcData.Tags.Tag != nil {
		tags = make([]string, 0, len(vpcData.Tags.Tag))
		for _, tag := range vpcData.Tags.Tag {
			if tag == nil || tag.Key == nil || tag.Value == nil {
				continue
			}
			tags = append(tags, fmt.Sprintf("%s=%s", tea.StringValue(tag.Key), tea.StringValue(tag.Value)))
		}
	}

	var vSwitchIds []string
	if vpcData.VSwitchIds != nil && vpcData.VSwitchIds.VSwitchId != nil {
		vSwitchIds = tea.StringSliceValue(vpcData.VSwitchIds.VSwitchId)
	}

	return &model.ResourceVpc{
		ResourceBase: model.ResourceBase{
			InstanceName: tea.StringValue(vpcData.VpcName),
			InstanceId:   tea.StringValue(vpcData.VpcId),
			Provider:     model.CloudProviderAliyun,
			RegionId:     region,
			VpcId:        tea.StringValue(vpcData.VpcId),
			Status:       tea.StringValue(vpcData.Status),
			CreationTime: tea.StringValue(vpcData.CreationTime),
			Description:  tea.StringValue(vpcData.Description),
			LastSyncTime: time.Now(),
			Tags:         model.StringList(tags),
		},
		VpcName:         tea.StringValue(vpcData.VpcName),
		CidrBlock:       tea.StringValue(vpcData.CidrBlock),
		Ipv6CidrBlock:   tea.StringValue(vpcData.Ipv6CidrBlock),
		VSwitchIds:      model.StringList(vSwitchIds),
		IsDefault:       tea.BoolValue(vpcData.IsDefault),
		ResourceGroupId: tea.StringValue(vpcData.ResourceGroupId),
	}
}

// convertToResourceVpcFromDetail 转换VPC详情为内部模型
func (a *AliyunProviderImpl) convertToResourceVpcFromDetail(vpcDetail *vpc.DescribeVpcAttributeResponseBody, region string) *model.ResourceVpc {
	if vpcDetail == nil {
		return nil
	}

	var tags []string
	if vpcDetail.Tags != nil && vpcDetail.Tags.Tag != nil {
		tags = make([]string, 0, len(vpcDetail.Tags.Tag))
		for _, tag := range vpcDetail.Tags.Tag {
			if tag == nil || tag.Key == nil || tag.Value == nil {
				continue
			}
			tags = append(tags, fmt.Sprintf("%s=%s", tea.StringValue(tag.Key), tea.StringValue(tag.Value)))
		}
	}

	var vSwitchIds []string
	if vpcDetail.VSwitchIds != nil && vpcDetail.VSwitchIds.VSwitchId != nil {
		vSwitchIds = tea.StringSliceValue(vpcDetail.VSwitchIds.VSwitchId)
	}

	return &model.ResourceVpc{
		ResourceBase: model.ResourceBase{
			InstanceName: tea.StringValue(vpcDetail.VpcName),
			InstanceId:   tea.StringValue(vpcDetail.VpcId),
			Provider:     model.CloudProviderAliyun,
			RegionId:     region,
			VpcId:        tea.StringValue(vpcDetail.VpcId),
			Status:       tea.StringValue(vpcDetail.Status),
			CreationTime: tea.StringValue(vpcDetail.CreationTime),
			Description:  tea.StringValue(vpcDetail.Description),
			LastSyncTime: time.Now(),
			Tags:         model.StringList(tags),
		},
		VpcName:         tea.StringValue(vpcDetail.VpcName),
		CidrBlock:       tea.StringValue(vpcDetail.CidrBlock),
		Ipv6CidrBlock:   tea.StringValue(vpcDetail.Ipv6CidrBlock),
		VSwitchIds:      model.StringList(vSwitchIds),
		IsDefault:       tea.BoolValue(vpcDetail.IsDefault),
		ResourceGroupId: tea.StringValue(vpcDetail.ResourceGroupId),
	}
}

// convertToResourceSecurityGroupFromList 转换安全组列表项为内部模型
func (a *AliyunProviderImpl) convertToResourceSecurityGroupFromList(sg *ecs.DescribeSecurityGroupsResponseBodySecurityGroupsSecurityGroup, region string) *model.ResourceSecurityGroup {
	if sg == nil {
		return nil
	}

	var tags []string
	if sg.Tags != nil && sg.Tags.Tag != nil {
		tags = make([]string, 0, len(sg.Tags.Tag))
		for _, tag := range sg.Tags.Tag {
			if tag == nil || tag.TagKey == nil || tag.TagValue == nil {
				continue
			}
			tags = append(tags, fmt.Sprintf("%s=%s", tea.StringValue(tag.TagKey), tea.StringValue(tag.TagValue)))
		}
	}

	return &model.ResourceSecurityGroup{
		ResourceBase: model.ResourceBase{
			InstanceName: tea.StringValue(sg.SecurityGroupName),
			InstanceId:   tea.StringValue(sg.SecurityGroupId),
			Provider:     model.CloudProviderAliyun,
			RegionId:     region,
			VpcId:        tea.StringValue(sg.VpcId),
			Description:  tea.StringValue(sg.Description),
			LastSyncTime: time.Now(),
			Tags:         model.StringList(tags),
		},
		SecurityGroupName: tea.StringValue(sg.SecurityGroupName),
	}
}

// convertToResourceSecurityGroupFromDetail 转换安全组详情为内部模型
func (a *AliyunProviderImpl) convertToResourceSecurityGroupFromDetail(sg *ecs.DescribeSecurityGroupAttributeResponseBody, region string) *model.ResourceSecurityGroup {
	return &model.ResourceSecurityGroup{
		ResourceBase: model.ResourceBase{
			InstanceName: tea.StringValue(sg.SecurityGroupName),
			InstanceId:   tea.StringValue(sg.SecurityGroupId),
			Provider:     model.CloudProviderAliyun,
			RegionId:     region,
			VpcId:        tea.StringValue(sg.VpcId),
			Description:  tea.StringValue(sg.Description),
			LastSyncTime: time.Now(),
		},
		SecurityGroupName: tea.StringValue(sg.SecurityGroupName),
	}
}

// convertToResourceEcsFromListInstance 转换ECS实例列表项为内部模型
func (a *AliyunProviderImpl) convertToResourceEcsFromListInstance(instance *ecs.DescribeInstancesResponseBodyInstancesInstance) *model.ResourceEcs {
	if instance == nil {
		return nil
	}

	var securityGroupIds []string
	if instance.SecurityGroupIds != nil && instance.SecurityGroupIds.SecurityGroupId != nil {
		securityGroupIds = tea.StringSliceValue(instance.SecurityGroupIds.SecurityGroupId)
	}

	var privateIPs []string
	if instance.VpcAttributes != nil && instance.VpcAttributes.PrivateIpAddress != nil && instance.VpcAttributes.PrivateIpAddress.IpAddress != nil {
		privateIPs = tea.StringSliceValue(instance.VpcAttributes.PrivateIpAddress.IpAddress)
	}

	var publicIPs []string
	if instance.PublicIpAddress != nil && instance.PublicIpAddress.IpAddress != nil {
		publicIPs = tea.StringSliceValue(instance.PublicIpAddress.IpAddress)
	}

	var vpcId string
	if instance.VpcAttributes != nil {
		vpcId = tea.StringValue(instance.VpcAttributes.VpcId)
	}

	// 计算内存，阿里云返回的是MB，转换为GB
	memory := int(tea.Int32Value(instance.Memory)) / 1024
	if memory == 0 && tea.Int32Value(instance.Memory) > 0 {
		memory = 1 // 如果小于1GB但大于0，设为1GB
	}

	var tags []string
	if instance.Tags != nil && instance.Tags.Tag != nil {
		tags = make([]string, 0, len(instance.Tags.Tag))
		for _, tag := range instance.Tags.Tag {
			if tag == nil || tag.TagKey == nil || tag.TagValue == nil {
				continue
			}
			tags = append(tags, fmt.Sprintf("%s=%s", tea.StringValue(tag.TagKey), tea.StringValue(tag.TagValue)))
		}
	}

	return &model.ResourceEcs{
		ResourceBase: model.ResourceBase{
			InstanceName:       tea.StringValue(instance.InstanceName),
			InstanceId:         tea.StringValue(instance.InstanceId),
			Provider:           model.CloudProviderAliyun,
			RegionId:           tea.StringValue(instance.RegionId),
			ZoneId:             tea.StringValue(instance.ZoneId),
			VpcId:              vpcId,
			Status:             tea.StringValue(instance.Status),
			CreationTime:       tea.StringValue(instance.CreationTime),
			InstanceChargeType: tea.StringValue(instance.InstanceChargeType),
			Description:        tea.StringValue(instance.Description),
			SecurityGroupIds:   model.StringList(securityGroupIds),
			PrivateIpAddress:   model.StringList(privateIPs),
			PublicIpAddress:    model.StringList(publicIPs),
			LastSyncTime:       time.Now(),
			Tags:               model.StringList(tags),
		},
		Cpu:          int(tea.Int32Value(instance.Cpu)),
		Memory:       memory,
		InstanceType: tea.StringValue(instance.InstanceType),
		ImageId:      tea.StringValue(instance.ImageId),
		HostName:     tea.StringValue(instance.HostName),
		IpAddr:       getFirstIP(privateIPs),
	}
}

// convertToResourceEcsFromInstanceDetail 转换ECS实例详情为内部模型
func (a *AliyunProviderImpl) convertToResourceEcsFromInstanceDetail(instance *ecs.DescribeInstanceAttributeResponseBody) *model.ResourceEcs {
	if instance == nil {
		return nil
	}

	var securityGroupIds []string
	if instance.SecurityGroupIds != nil && instance.SecurityGroupIds.SecurityGroupId != nil {
		securityGroupIds = tea.StringSliceValue(instance.SecurityGroupIds.SecurityGroupId)
	}

	var privateIPs []string
	if instance.VpcAttributes != nil && instance.VpcAttributes.PrivateIpAddress != nil && instance.VpcAttributes.PrivateIpAddress.IpAddress != nil {
		privateIPs = tea.StringSliceValue(instance.VpcAttributes.PrivateIpAddress.IpAddress)
	}

	var publicIPs []string
	if instance.PublicIpAddress != nil && instance.PublicIpAddress.IpAddress != nil {
		publicIPs = tea.StringSliceValue(instance.PublicIpAddress.IpAddress)
	}

	var vpcId string
	if instance.VpcAttributes != nil {
		vpcId = tea.StringValue(instance.VpcAttributes.VpcId)
	}

	// 计算内存，阿里云返回的是MB，转换为GB
	memory := int(tea.Int32Value(instance.Memory)) / 1024
	if memory == 0 && tea.Int32Value(instance.Memory) > 0 {
		memory = 1 // 如果小于1GB但大于0，设为1GB
	}

	return &model.ResourceEcs{
		ResourceBase: model.ResourceBase{
			InstanceName:       tea.StringValue(instance.InstanceName),
			InstanceId:         tea.StringValue(instance.InstanceId),
			Provider:           model.CloudProviderAliyun,
			RegionId:           tea.StringValue(instance.RegionId),
			ZoneId:             tea.StringValue(instance.ZoneId),
			VpcId:              vpcId,
			Status:             tea.StringValue(instance.Status),
			CreationTime:       tea.StringValue(instance.CreationTime),
			InstanceChargeType: tea.StringValue(instance.InstanceChargeType),
			Description:        tea.StringValue(instance.Description),
			SecurityGroupIds:   model.StringList(securityGroupIds),
			PrivateIpAddress:   model.StringList(privateIPs),
			PublicIpAddress:    model.StringList(publicIPs),
			LastSyncTime:       time.Now(),
		},
		Cpu:          int(tea.Int32Value(instance.Cpu)),
		Memory:       memory,
		InstanceType: tea.StringValue(instance.InstanceType),
		ImageId:      tea.StringValue(instance.ImageId),
		HostName:     tea.StringValue(instance.HostName),
		IpAddr:       getFirstIP(privateIPs),
	}
}
