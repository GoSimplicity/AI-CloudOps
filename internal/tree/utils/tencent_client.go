package utils

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
	regionapi "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/region/v20220627"
)

type tencentCVMAPI interface {
	DescribeInstancesWithContext(context.Context, *cvm.DescribeInstancesRequest) (*cvm.DescribeInstancesResponse, error)
}
type tencentRegionAPI interface {
	DescribeRegionsWithContext(context.Context, *regionapi.DescribeRegionsRequest) (*regionapi.DescribeRegionsResponse, error)
}
type TencentClient struct {
	client  tencentCVMAPI
	regions tencentRegionAPI
	region  string
}

func NewTencentClient(accessKey, secretKey, region string) (*TencentClient, error) {
	if strings.TrimSpace(accessKey) == "" || strings.TrimSpace(secretKey) == "" || strings.TrimSpace(region) == "" {
		return nil, fmt.Errorf("腾讯云访问密钥和地域不能为空")
	}
	credential := common.NewCredential(accessKey, secretKey)
	cp := profile.NewClientProfile()
	cp.HttpProfile.ReqTimeout = 30
	cp.HttpProfile.Scheme = "HTTPS"
	client, err := cvm.NewClient(credential, region, cp)
	if err != nil {
		return nil, err
	}
	rp := profile.NewClientProfile()
	rp.HttpProfile.ReqTimeout = 30
	rp.HttpProfile.Scheme = "HTTPS"
	regions, err := regionapi.NewClient(credential, region, rp)
	if err != nil {
		return nil, err
	}
	return &TencentClient{client: client, regions: regions, region: region}, nil
}
func (c *TencentClient) VerifyCredentials(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	request := cvm.NewDescribeInstancesRequest()
	request.Limit = common.Int64Ptr(1)
	response, err := c.client.DescribeInstancesWithContext(ctx, request)
	if err != nil {
		return fmt.Errorf("腾讯云CVM读取权限验证失败: %w", err)
	}
	if response == nil || response.Response == nil {
		return fmt.Errorf("腾讯云返回空响应")
	}
	return ctx.Err()
}
func (c *TencentClient) GetAvailableRegions(ctx context.Context) ([]model.AvailableRegion, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	request := regionapi.NewDescribeRegionsRequest()
	request.Product = common.StringPtr("cvm")
	response, err := c.regions.DescribeRegionsWithContext(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("腾讯云地域查询失败: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if response == nil || response.Response == nil {
		return nil, fmt.Errorf("腾讯云返回空地域响应")
	}
	regions := make([]model.AvailableRegion, 0, len(response.Response.RegionSet))
	for _, r := range response.Response.RegionSet {
		if r != nil {
			regions = append(regions, model.AvailableRegion{Region: deref(r.Region), RegionName: deref(r.RegionName), Available: deref(r.RegionState) == "AVAILABLE"})
		}
	}
	return regions, nil
}
func (c *TencentClient) ListInstances(ctx context.Context, ids []string) ([]*model.TreeCloudResource, error) {
	batches, err := instanceIDBatches(ids, 100)
	if err != nil {
		return nil, err
	}
	resources := make([]*model.TreeCloudResource, 0)
	seen := make(map[string]bool)
	for _, batch := range batches {
		var offset int64
		for {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			request := cvm.NewDescribeInstancesRequest()
			request.Limit = common.Int64Ptr(100)
			request.Offset = common.Int64Ptr(offset)
			for _, id := range batch {
				request.InstanceIds = append(request.InstanceIds, common.StringPtr(id))
			}
			response, err := c.client.DescribeInstancesWithContext(ctx, request)
			if err != nil {
				return nil, fmt.Errorf("腾讯云实例查询失败: %w", err)
			}
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			if response == nil || response.Response == nil || response.Response.TotalCount == nil {
				return nil, fmt.Errorf("腾讯云返回不完整实例响应")
			}
			page := response.Response
			total := *page.TotalCount
			if total < 0 {
				return nil, fmt.Errorf("腾讯云返回无效总数")
			}
			progress := 0
			for _, instance := range page.InstanceSet {
				if instance == nil || deref(instance.InstanceId) == "" {
					return nil, fmt.Errorf("腾讯云实例响应缺少ID")
				}
				id := deref(instance.InstanceId)
				if !seen[id] {
					seen[id] = true
					resources = append(resources, c.convertInstance(instance))
					progress++
				}
			}
			offset += int64(len(page.InstanceSet))
			if offset >= total {
				break
			}
			if progress == 0 {
				return nil, fmt.Errorf("腾讯云分页未取得进展，拒绝不完整同步")
			}
		}
	}
	return resources, nil
}
func deref[T any](value *T) (zero T) {
	if value != nil {
		return *value
	}
	return
}
func (c *TencentClient) convertInstance(instance *cvm.Instance) *model.TreeCloudResource {
	r := &model.TreeCloudResource{Name: deref(instance.InstanceName), InstanceID: deref(instance.InstanceId), ResourceType: model.ResourceTypeECS, InstanceType: deref(instance.InstanceType), Region: c.region, Cpu: int(deref(instance.CPU)), Memory: int(deref(instance.Memory)), ImageID: deref(instance.ImageId), OSName: deref(instance.OsName), Currency: model.CurrencyCNY, Port: 22, Status: model.CloudResourceUnknown}
	if r.Name == "" {
		r.Name = r.InstanceID
	}
	if instance.Placement != nil {
		r.ZoneID = deref(instance.Placement.Zone)
	}
	if instance.VirtualPrivateCloud != nil {
		r.VpcID = deref(instance.VirtualPrivateCloud.VpcId)
	}
	if len(instance.PublicIpAddresses) > 0 {
		r.PublicIP = deref(instance.PublicIpAddresses[0])
	}
	if len(instance.PrivateIpAddresses) > 0 {
		r.PrivateIP = deref(instance.PrivateIpAddresses[0])
	}
	if instance.SystemDisk != nil {
		r.Disk += int(deref(instance.SystemDisk.DiskSize))
	}
	for _, disk := range instance.DataDisks {
		if disk != nil {
			r.Disk += int(deref(disk.DiskSize))
		}
	}
	switch deref(instance.InstanceState) {
	case "RUNNING":
		r.Status = model.CloudResourceRunning
	case "STOPPED":
		r.Status = model.CloudResourceStopped
	case "PENDING", "STARTING", "REBOOTING":
		r.Status = model.CloudResourceStarting
	case "STOPPING", "TERMINATING":
		r.Status = model.CloudResourceStopping
	case "TERMINATED":
		r.Status = model.CloudResourceDeleted
	}
	switch deref(instance.InstanceChargeType) {
	case "PREPAID":
		r.ChargeType = model.ChargeTypePrePaid
	case "POSTPAID_BY_HOUR", "SPOTPAID":
		r.ChargeType = model.ChargeTypePostPaid
	}
	if expiry, err := time.Parse(time.RFC3339, deref(instance.ExpiredTime)); err == nil {
		r.ExpireTime = &expiry
	}
	osName := strings.ToLower(r.OSName)
	if strings.Contains(osName, "windows") {
		r.OSType = "windows"
	} else if strings.Contains(osName, "linux") || strings.Contains(osName, "ubuntu") || strings.Contains(osName, "centos") {
		r.OSType = "linux"
	}
	for _, tag := range instance.Tags {
		if tag != nil {
			r.Tags = append(r.Tags, model.KeyValue{Key: deref(tag.Key), Value: deref(tag.Value)})
		}
	}
	return r
}
