package aliyun

import (
	"context"

	ecs "github.com/alibabacloud-go/ecs-20140526/v2/client"
	"github.com/alibabacloud-go/tea/tea"
	"go.uber.org/zap"
)

type DiskService struct {
	sdk *SDK
}

func NewDiskService(sdk *SDK) *DiskService {
	return &DiskService{sdk: sdk}
}

type CreateDiskRequest struct {
	Region       string
	ZoneId       string
	DiskName     string
	DiskCategory string
	Size         int
	Description  string
	Tags         map[string]string
}

type CreateDiskResponseBody struct {
	DiskId string
}

// CreateDisk 创建磁盘
func (d *DiskService) CreateDisk(ctx context.Context, req *CreateDiskRequest) (*CreateDiskResponseBody, error) {
	client, err := d.sdk.CreateEcsClient(req.Region)
	if err != nil {
		d.sdk.logger.Error("创建ECS客户端失败", zap.Error(err))
		return nil, err
	}

	request := &ecs.CreateDiskRequest{
		RegionId:     tea.String(req.Region),
		ZoneId:       tea.String(req.ZoneId),
		DiskName:     tea.String(req.DiskName),
		DiskCategory: tea.String(req.DiskCategory),
		Size:         tea.Int32(int32(req.Size)),
		Description:  tea.String(req.Description),
	}

	// 设置标签
	if len(req.Tags) > 0 {
		tags := make([]*ecs.CreateDiskRequestTag, 0, len(req.Tags))
		for k, v := range req.Tags {
			tags = append(tags, &ecs.CreateDiskRequestTag{
				Key:   tea.String(k),
				Value: tea.String(v),
			})
		}
		request.Tag = tags
	}

	d.sdk.logger.Info("开始创建磁盘", zap.String("region", req.Region), zap.Any("request", req))
	response, err := client.CreateDisk(request)
	if err != nil {
		d.sdk.logger.Error("创建磁盘失败", zap.Error(err))
		return nil, err
	}

	diskId := tea.StringValue(response.Body.DiskId)
	d.sdk.logger.Info("创建磁盘成功", zap.String("diskID", diskId))

	return &CreateDiskResponseBody{
		DiskId: diskId,
	}, nil
}

// AttachDisk 挂载磁盘
func (d *DiskService) AttachDisk(ctx context.Context, region string, diskID string, instanceID string) error {
	client, err := d.sdk.CreateEcsClient(region)
	if err != nil {
		d.sdk.logger.Error("创建ECS客户端失败", zap.Error(err))
		return err
	}

	request := &ecs.AttachDiskRequest{
		DiskId:     tea.String(diskID),
		InstanceId: tea.String(instanceID),
	}

	d.sdk.logger.Info("开始挂载磁盘", zap.String("region", region), zap.String("diskID", diskID), zap.String("instanceID", instanceID))
	_, err = client.AttachDisk(request)
	if err != nil {
		d.sdk.logger.Error("挂载磁盘失败", zap.Error(err))
		return err
	}

	d.sdk.logger.Info("挂载磁盘成功", zap.String("diskID", diskID), zap.String("instanceID", instanceID))
	return nil
}

// DetachDisk 卸载磁盘
func (d *DiskService) DetachDisk(ctx context.Context, region string, diskID string, instanceID string) error {
	client, err := d.sdk.CreateEcsClient(region)
	if err != nil {
		d.sdk.logger.Error("创建ECS客户端失败", zap.Error(err))
		return err
	}

	request := &ecs.DetachDiskRequest{
		DiskId:     tea.String(diskID),
		InstanceId: tea.String(instanceID),
	}

	d.sdk.logger.Info("开始卸载磁盘", zap.String("region", region), zap.String("diskID", diskID), zap.String("instanceID", instanceID))
	_, err = client.DetachDisk(request)
	if err != nil {
		d.sdk.logger.Error("卸载磁盘失败", zap.Error(err))
		return err
	}

	d.sdk.logger.Info("卸载磁盘成功", zap.String("diskID", diskID), zap.String("instanceID", instanceID))
	return nil
}

// DeleteDisk 删除磁盘
func (d *DiskService) DeleteDisk(ctx context.Context, region string, diskID string) error {
	client, err := d.sdk.CreateEcsClient(region)
	if err != nil {
		d.sdk.logger.Error("创建ECS客户端失败", zap.Error(err))
		return err
	}

	request := &ecs.DeleteDiskRequest{
		DiskId: tea.String(diskID),
	}

	d.sdk.logger.Info("开始删除磁盘", zap.String("region", region), zap.String("diskID", diskID))
	_, err = client.DeleteDisk(request)
	if err != nil {
		d.sdk.logger.Error("删除磁盘失败", zap.Error(err))
		return err
	}

	d.sdk.logger.Info("删除磁盘成功", zap.String("diskID", diskID))
	return nil
}

// ListDisksRequest 查询磁盘列表请求参数
type ListDisksRequest struct {
	Region string
	Page   int
	Size   int
}

// ListDisksResponseBody 查询磁盘列表响应
type ListDisksResponseBody struct {
	Disks []*ecs.DescribeDisksResponseBodyDisksDisk
	Total int64
}

// ListDisks 查询磁盘列表
func (d *DiskService) ListDisks(ctx context.Context, req *ListDisksRequest) (*ListDisksResponseBody, error) {
	client, err := d.sdk.CreateEcsClient(req.Region)
	if err != nil {
		d.sdk.logger.Error("创建ECS客户端失败", zap.Error(err))
		return nil, err
	}

	request := &ecs.DescribeDisksRequest{
		RegionId:   tea.String(req.Region),
		PageSize:   tea.Int32(int32(req.Size)),
		PageNumber: tea.Int32(int32(req.Page)),
	}

	d.sdk.logger.Info("开始查询磁盘列表", zap.String("region", req.Region))
	response, err := client.DescribeDisks(request)
	if err != nil {
		d.sdk.logger.Error("查询磁盘列表失败", zap.Error(err))
		return nil, err
	}

	total := int64(tea.Int32Value(response.Body.TotalCount))
	d.sdk.logger.Info("查询磁盘列表成功", zap.Int64("total", total))

	return &ListDisksResponseBody{
		Disks: response.Body.Disks.Disk,
		Total: total,
	}, nil
}
