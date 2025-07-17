package provider

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	ecsmodel "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/ecs/v2/model"
	evsmode "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/evs/v2/model"
	vpcv3model "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/vpc/v3/model"
)

func (h *HuaweiProviderImpl) convertToResourceEcsFromListInstance(instance ecsmodel.ServerDetail) *model.ResourceEcs {
	lastSyncTime := time.Now()
	var securityGroupIds []string
	if instance.SecurityGroups != nil {
		for _, sg := range instance.SecurityGroups {
			securityGroupIds = append(securityGroupIds, sg.Id)
		}
	}
	var privateIPs []string
	var publicIPs []string
	if instance.Addresses != nil {
		for networkName, addresses := range instance.Addresses {
			for _, addr := range addresses {
				if addr.OSEXTIPStype != nil {
					switch *addr.OSEXTIPStype {
					case ecsmodel.GetServerAddressOSEXTIPStypeEnum().FIXED:
						privateIPs = append(privateIPs, addr.Addr)
					case ecsmodel.GetServerAddressOSEXTIPStypeEnum().FLOATING:
						publicIPs = append(publicIPs, addr.Addr)
					}
				} else {
					if networkName == "external" || networkName == "public" {
						publicIPs = append(publicIPs, addr.Addr)
					} else {
						privateIPs = append(privateIPs, addr.Addr)
					}
				}
			}
		}
	}
	var tags []string
	if instance.Tags != nil {
		for _, tag := range *instance.Tags {
			tags = append(tags, tag)
		}
	}
	var vpcId string
	if instance.Metadata != nil {
		if vpc, ok := instance.Metadata["vpc_id"]; ok {
			vpcId = vpc
		}
	}
	regionId := ""
	if instance.OSEXTAZavailabilityZone != "" {
		if len(instance.OSEXTAZavailabilityZone) >= 2 {
			parts := strings.Split(instance.OSEXTAZavailabilityZone, "-")
			if len(parts) >= 3 {
				regionId = strings.Join(parts[:len(parts)-1], "-")
			}
		}
	}
	var cpu int
	var memory int
	if instance.Flavor != nil {
		if vcpus, err := strconv.Atoi(instance.Flavor.Vcpus); err == nil {
			cpu = vcpus
		}
		if ram, err := strconv.Atoi(instance.Flavor.Ram); err == nil {
			memory = ram / 1024
			if memory == 0 && ram > 0 {
				memory = 1
			}
		}
	}
	var imageId string
	if instance.Image != nil {
		imageId = instance.Image.Id
	}
	instanceType := ""
	if instance.Flavor != nil {
		instanceType = instance.Flavor.Name
	}
	hostName := instance.OSEXTSRVATTRhostname
	description := ""
	if instance.Description != nil {
		description = *instance.Description
	}
	instanceChargeType := "PostPaid"
	if instance.Metadata != nil {
		if chargingMode, ok := instance.Metadata["charging_mode"]; ok {
			switch chargingMode {
			case "0":
				instanceChargeType = "PostPaid"
			case "1":
				instanceChargeType = "PrePaid"
			case "2":
				instanceChargeType = "Spot"
			}
		}
	}
	return &model.ResourceEcs{
		InstanceName: instance.Name,
		InstanceId:   instance.Id,
		Provider: model.
			CloudProviderHuawei,
		RegionId:           regionId,
		ZoneId:             instance.OSEXTAZavailabilityZone,
		VpcId:              vpcId,
		Status:             instance.Status,
		CreationTime:       instance.Created,
		InstanceChargeType: instanceChargeType,
		Description:        description,
		SecurityGroupIds:   model.StringList(securityGroupIds),
		PrivateIpAddress:   model.StringList(privateIPs),
		PublicIpAddress:    model.StringList(publicIPs),
		LastSyncTime:       &lastSyncTime,
		Tags:               model.StringList(tags),
		Cpu:                cpu,
		Memory:             memory,
		InstanceType:       instanceType,
		ImageId:            imageId,
		HostName:           hostName,
		IpAddr: func(ips []string) string {
			if len(ips) > 0 {
				return ips[0]
			}
			return ""
		}(privateIPs),
	}
}

func (h *HuaweiProviderImpl) convertToResourceEcsFromInstanceDetail(instance *ecsmodel.ServerDetail) *model.ResourceEcs {
	if instance == nil {
		return nil
	}
	return h.convertToResourceEcsFromListInstance(*instance)
}

// v3 版本 VPC
func (h *HuaweiProviderImpl) convertToResourceVpcFromListVpc(vpcData vpcv3model.Vpc, region string) *model.ResourceVpc {
	var tags []string
	for _, tag := range vpcData.Tags {
		tags = append(tags, tag.Value)
	}
	creationTime := ""
	if vpcData.CreatedAt != nil {
		creationTime = vpcData.CreatedAt.String()
	}
	isDefault := false
	if vpcData.Name == "default" || vpcData.Name == "Default" {
		isDefault = true
	}
	ipv6CidrBlock := ""
	status := vpcData.Status
	return &model.ResourceVpc{
		InstanceName:    vpcData.Name,
		InstanceId:      vpcData.Id,
		Provider:        model.CloudProviderHuawei,
		RegionId:        region,
		VpcId:           vpcData.Id,
		Status:          status,
		CreationTime:    creationTime,
		Description:     vpcData.Description,
		LastSyncTime:    time.Now(),
		Tags:            model.StringList(tags),
		VpcName:         vpcData.Name,
		CidrBlock:       vpcData.Cidr,
		Ipv6CidrBlock:   ipv6CidrBlock,
		VSwitchIds:      model.StringList([]string{}),
		IsDefault:       isDefault,
		ResourceGroupId: vpcData.EnterpriseProjectId,
	}
}

func (h *HuaweiProviderImpl) convertToResourceVpcFromDetail(vpcDetail *vpcv3model.Vpc, region string) *model.ResourceVpc {
	if vpcDetail == nil {
		return nil
	}
	return h.convertToResourceVpcFromListVpc(*vpcDetail, region)
}

// 用于 v3 版本 SecurityGroup
func (h *HuaweiProviderImpl) convertToResourceSecurityGroupFromList(sg vpcv3model.SecurityGroup, region string) *model.ResourceSecurityGroup {
	var tags []string
	for _, tag := range sg.Tags {
		tags = append(tags, tag.Value)
	}
	creationTime := ""
	if sg.CreatedAt != nil {
		creationTime = sg.CreatedAt.String()
	}
	// v3 SDK 结构体实际无 VpcId/Status 字段，置空
	return &model.ResourceSecurityGroup{
		InstanceName:      sg.Name,
		InstanceId:        sg.Id,
		Provider:          model.CloudProviderHuawei,
		RegionId:          region,
		VpcId:             "",
		Status:            "",
		CreationTime:      creationTime,
		Description:       sg.Description,
		LastSyncTime:      time.Now(),
		Tags:              model.StringList(tags),
		SecurityGroupName: sg.Name,
		ResourceGroupId:   sg.EnterpriseProjectId,
	}
}

func (h *HuaweiProviderImpl) convertToResourceSecurityGroupFromDetail(sg *vpcv3model.SecurityGroupInfo, region string) *model.ResourceSecurityGroup {
	if sg == nil {
		return nil
	}
	var tags []string
	for _, tag := range sg.Tags {
		tags = append(tags, tag.Value)
	}
	creationTime := ""
	if sg.CreatedAt != nil {
		creationTime = sg.CreatedAt.String()
	}
	return &model.ResourceSecurityGroup{
		InstanceName:      sg.Name,
		InstanceId:        sg.Id,
		Provider:          model.CloudProviderHuawei,
		RegionId:          region,
		VpcId:             "",
		Status:            "",
		CreationTime:      creationTime,
		Description:       sg.Description,
		LastSyncTime:      time.Now(),
		Tags:              model.StringList(tags),
		SecurityGroupName: sg.Name,
		ResourceGroupId:   sg.EnterpriseProjectId,
	}
}

func (h *HuaweiProviderImpl) convertToResourceDiskFromList(disk evsmode.VolumeDetail, region string) *model.ResourceDisk {
	if disk.Id == "" {
		return nil
	}

	// 提取标签信息
	var tags []string
	if disk.Tags != nil {
		for key, value := range disk.Tags {
			tags = append(tags, fmt.Sprintf("%s=%s", key, value))
		}
	}

	// 提取挂载的实例ID
	var instanceID string
	if disk.Attachments != nil {
		// 获取第一个挂载的实例ID
		instanceID = disk.Attachments[0].ServerId
	}

	return &model.ResourceDisk{
		InstanceName: disk.Name,
		InstanceID:   instanceID,
		Provider:     model.CloudProviderHuawei,
		RegionId:     region,
		ZoneId:       disk.AvailabilityZone,
		Status:       disk.Status,
		CreationTime: disk.CreatedAt,
		Description:  disk.Description,
		LastSyncTime: time.Now(),
		Tags:         model.StringList(tags),
		DiskID:       disk.Id,
		DiskName:     disk.Name,
		Size:         int(disk.Size),
		Category:     disk.VolumeType,
	}
}

func (h *HuaweiProviderImpl) convertToResourceDiskFromDetail(disk *evsmode.VolumeDetail, region string) *model.ResourceDisk {
	if disk == nil {
		return nil
	}

	// 直接复用列表转换函数的逻辑
	return h.convertToResourceDiskFromList(*disk, region)
}
