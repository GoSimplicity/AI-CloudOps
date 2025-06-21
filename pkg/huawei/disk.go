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

package huawei

import (
	"context"
	"fmt"

	ecsmodel "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/ecs/v2/model"
	evsmodel "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/evs/v2/model"
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
	client, err := d.sdk.CreateEvsClient(req.Region, d.sdk.accessKey)
	if err != nil {
		d.sdk.logger.Error("创建EVS客户端失败", zap.Error(err))
		return nil, err
	}

	// 根据磁盘类型获取对应的枚举值
	var volumeType evsmodel.CreateVolumeOptionVolumeType
	volumeTypeEnum := evsmodel.GetCreateVolumeOptionVolumeTypeEnum()
	switch req.DiskCategory {
	case "SSD":
		volumeType = volumeTypeEnum.SSD
	case "GPSSD":
		volumeType = volumeTypeEnum.GPSSD
	case "SAS":
		volumeType = volumeTypeEnum.SAS
	case "SATA":
		volumeType = volumeTypeEnum.SATA
	case "ESSD":
		volumeType = volumeTypeEnum.ESSD
	case "GPSSD2":
		volumeType = volumeTypeEnum.GPSSD2
	case "ESSD2":
		volumeType = volumeTypeEnum.ESSD2
	default:
		volumeType = volumeTypeEnum.SSD // 默认使用SSD
	}

	request := &evsmodel.CreateVolumeRequest{
		Body: &evsmodel.CreateVolumeRequestBody{
			Volume: &evsmodel.CreateVolumeOption{
				Name:             &req.DiskName,
				Size:             int32(req.Size),
				VolumeType:       volumeType,
				Description:      &req.Description,
				AvailabilityZone: req.ZoneId,
			},
		},
	}

	d.sdk.logger.Info("开始创建磁盘", zap.String("region", req.Region), zap.Any("request", req))
	response, err := client.CreateVolume(request)
	if err != nil {
		d.sdk.logger.Error("创建磁盘失败", zap.Error(err))
		return nil, err
	}

	diskId := ""
	if response.VolumeIds != nil && len(*response.VolumeIds) > 0 {
		diskId = (*response.VolumeIds)[0]
	}
	if diskId == "" {
		d.sdk.logger.Error("未获取到磁盘ID")
		return nil, fmt.Errorf("未获取到磁盘ID")
	}

	d.sdk.logger.Info("创建磁盘成功", zap.String("diskID", diskId))

	return &CreateDiskResponseBody{
		DiskId: diskId,
	}, nil
}

// AttachDisk 挂载磁盘
func (d *DiskService) AttachDisk(ctx context.Context, region string, diskID string, instanceID string) error {
	client, err := d.sdk.CreateEcsClient(region, d.sdk.accessKey)
	if err != nil {
		d.sdk.logger.Error("创建ECS客户端失败", zap.Error(err))
		return err
	}

	request := &ecsmodel.AttachServerVolumeRequest{
		ServerId: instanceID,
		Body: &ecsmodel.AttachServerVolumeRequestBody{
			VolumeAttachment: &ecsmodel.AttachServerVolumeOption{
				VolumeId: diskID,
			},
		},
	}

	d.sdk.logger.Info("开始挂载磁盘", zap.String("region", region), zap.String("diskID", diskID), zap.String("instanceID", instanceID))
	_, err = client.AttachServerVolume(request)
	if err != nil {
		d.sdk.logger.Error("挂载磁盘失败", zap.Error(err))
		return err
	}

	d.sdk.logger.Info("挂载磁盘成功", zap.String("diskID", diskID), zap.String("instanceID", instanceID))
	return nil
}

// DetachDisk 卸载磁盘
func (d *DiskService) DetachDisk(ctx context.Context, region string, diskID string, instanceID string) error {
	client, err := d.sdk.CreateEcsClient(region, d.sdk.accessKey)
	if err != nil {
		d.sdk.logger.Error("创建ECS客户端失败", zap.Error(err))
		return err
	}

	request := &ecsmodel.DetachServerVolumeRequest{
		ServerId: instanceID,
		VolumeId: diskID,
	}

	d.sdk.logger.Info("开始卸载磁盘", zap.String("region", region), zap.String("diskID", diskID), zap.String("instanceID", instanceID))
	_, err = client.DetachServerVolume(request)
	if err != nil {
		d.sdk.logger.Error("卸载磁盘失败", zap.Error(err))
		return err
	}

	d.sdk.logger.Info("卸载磁盘成功", zap.String("diskID", diskID), zap.String("instanceID", instanceID))
	return nil
}

// DeleteDisk 删除磁盘
func (d *DiskService) DeleteDisk(ctx context.Context, region string, diskID string) error {
	client, err := d.sdk.CreateEvsClient(region, d.sdk.accessKey)
	if err != nil {
		d.sdk.logger.Error("创建EVS客户端失败", zap.Error(err))
		return err
	}

	request := &evsmodel.DeleteVolumeRequest{
		VolumeId: diskID,
	}

	d.sdk.logger.Info("开始删除磁盘", zap.String("region", region), zap.String("diskID", diskID))
	_, err = client.DeleteVolume(request)
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
	Disks []evsmodel.VolumeDetail
	Total int32
}

// ListDisks 查询磁盘列表
func (d *DiskService) ListDisks(ctx context.Context, req *ListDisksRequest) (*ListDisksResponseBody, error) {
	client, err := d.sdk.CreateEvsClient(req.Region, d.sdk.accessKey)
	if err != nil {
		d.sdk.logger.Error("创建EVS客户端失败", zap.Error(err))
		return nil, err
	}

	limit := int32(req.Size)
	request := &evsmodel.ListVolumesRequest{
		Limit: &limit,
	}

	d.sdk.logger.Info("开始查询磁盘列表", zap.String("region", req.Region))
	response, err := client.ListVolumes(request)
	if err != nil {
		d.sdk.logger.Error("查询磁盘列表失败", zap.Error(err))
		return nil, err
	}

	var total int32
	var disks []evsmodel.VolumeDetail
	if response.Volumes != nil {
		disks = *response.Volumes
		total = int32(len(disks))
	}
	if response.Count != nil {
		total = *response.Count
	}

	d.sdk.logger.Info("查询磁盘列表成功", zap.Int32("total", total))

	return &ListDisksResponseBody{
		Disks: disks,
		Total: total,
	}, nil
}

// GetDisk 获取磁盘详情
func (d *DiskService) GetDisk(ctx context.Context, region string, diskID string) (*evsmodel.VolumeDetail, error) {
	client, err := d.sdk.CreateEvsClient(region, d.sdk.accessKey)
	if err != nil {
		d.sdk.logger.Error("创建EVS客户端失败", zap.Error(err))
		return nil, err
	}

	request := &evsmodel.ShowVolumeRequest{
		VolumeId: diskID,
	}

	d.sdk.logger.Info("开始获取磁盘详情", zap.String("region", region), zap.String("diskID", diskID))
	response, err := client.ShowVolume(request)
	if err != nil {
		d.sdk.logger.Error("获取磁盘详情失败", zap.Error(err))
		return nil, err
	}

	return response.Volume, nil
}
