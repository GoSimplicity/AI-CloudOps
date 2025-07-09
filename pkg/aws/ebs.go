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

package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"go.uber.org/zap"
)

type EBSService struct {
	sdk *SDK
}

func NewEBSService(sdk *SDK) *EBSService {
	return &EBSService{sdk: sdk}
}

type CreateVolumeRequest struct {
	Region           string
	AvailabilityZone string
	VolumeName       string
	VolumeType       string // gp2, gp3, io1, io2, st1, sc1, standard
	Size             int32
	Description      string
	Encrypted        bool
	KmsKeyId         string
	Iops             int32
	Throughput       int32
	Tags             map[string]string
}

type CreateVolumeResponseBody struct {
	VolumeId string
}

// CreateVolume 创建EBS卷
func (e *EBSService) CreateVolume(ctx context.Context, req *CreateVolumeRequest) (*CreateVolumeResponseBody, error) {
	client, err := e.sdk.CreateEC2Client(ctx, req.Region)
	if err != nil {
		e.sdk.logger.Error("创建EC2客户端失败", zap.Error(err))
		return nil, err
	}

	// 根据卷类型设置参数
	var volumeType types.VolumeType
	switch req.VolumeType {
	case "gp2":
		volumeType = types.VolumeTypeGp2
	case "gp3":
		volumeType = types.VolumeTypeGp3
	case "io1":
		volumeType = types.VolumeTypeIo1
	case "io2":
		volumeType = types.VolumeTypeIo2
	case "st1":
		volumeType = types.VolumeTypeSt1
	case "sc1":
		volumeType = types.VolumeTypeSc1
	case "standard":
		volumeType = types.VolumeTypeStandard
	default:
		volumeType = types.VolumeTypeGp3 // 默认使用gp3
	}

	request := &ec2.CreateVolumeInput{
		AvailabilityZone: &req.AvailabilityZone,
		Size:             &req.Size,
		VolumeType:       volumeType,
		Encrypted:        &req.Encrypted,
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeVolume,
				Tags: []types.Tag{
					{
						Key:   stringPtr("Name"),
						Value: &req.VolumeName,
					},
					{
						Key:   stringPtr("Description"),
						Value: &req.Description,
					},
				},
			},
		},
	}

	// 设置KMS密钥ID（如果启用加密）
	if req.Encrypted && req.KmsKeyId != "" {
		request.KmsKeyId = &req.KmsKeyId
	}

	// 设置IOPS（仅适用于io1, io2, gp3类型）
	if req.Iops > 0 && (volumeType == types.VolumeTypeIo1 || volumeType == types.VolumeTypeIo2 || volumeType == types.VolumeTypeGp3) {
		request.Iops = &req.Iops
	}

	// 设置吞吐量（仅适用于gp3类型）
	if req.Throughput > 0 && volumeType == types.VolumeTypeGp3 {
		request.Throughput = &req.Throughput
	}

	// 添加自定义标签
	if req.Tags != nil {
		for key, value := range req.Tags {
			request.TagSpecifications[0].Tags = append(request.TagSpecifications[0].Tags, types.Tag{
				Key:   &key,
				Value: &value,
			})
		}
	}

	e.sdk.logger.Info("开始创建EBS卷", zap.String("region", req.Region), zap.String("name", req.VolumeName), zap.Int32("size", req.Size))
	response, err := client.CreateVolume(ctx, request)
	if err != nil {
		e.sdk.logger.Error("创建EBS卷失败", zap.Error(err))
		return nil, err
	}

	volumeId := *response.VolumeId
	e.sdk.logger.Info("EBS卷创建成功", zap.String("volumeId", volumeId))

	return &CreateVolumeResponseBody{
		VolumeId: volumeId,
	}, nil
}

// AttachVolume 挂载EBS卷
func (e *EBSService) AttachVolume(ctx context.Context, region string, volumeID string, instanceID string, device string) error {
	client, err := e.sdk.CreateEC2Client(ctx, region)
	if err != nil {
		e.sdk.logger.Error("创建EC2客户端失败", zap.Error(err))
		return err
	}

	request := &ec2.AttachVolumeInput{
		VolumeId:   &volumeID,
		InstanceId: &instanceID,
		Device:     &device,
	}

	e.sdk.logger.Info("开始挂载EBS卷", zap.String("region", region), zap.String("volumeId", volumeID), zap.String("instanceId", instanceID))
	_, err = client.AttachVolume(ctx, request)
	if err != nil {
		e.sdk.logger.Error("挂载EBS卷失败", zap.Error(err))
		return err
	}

	e.sdk.logger.Info("EBS卷挂载成功", zap.String("volumeId", volumeID), zap.String("instanceId", instanceID))
	return nil
}

// DetachVolume 卸载EBS卷
func (e *EBSService) DetachVolume(ctx context.Context, region string, volumeID string, instanceID string, force bool) error {
	client, err := e.sdk.CreateEC2Client(ctx, region)
	if err != nil {
		e.sdk.logger.Error("创建EC2客户端失败", zap.Error(err))
		return err
	}

	request := &ec2.DetachVolumeInput{
		VolumeId:   &volumeID,
		InstanceId: &instanceID,
		Force:      &force,
	}

	e.sdk.logger.Info("开始卸载EBS卷", zap.String("region", region), zap.String("volumeId", volumeID), zap.String("instanceId", instanceID))
	_, err = client.DetachVolume(ctx, request)
	if err != nil {
		e.sdk.logger.Error("卸载EBS卷失败", zap.Error(err))
		return err
	}

	e.sdk.logger.Info("EBS卷卸载成功", zap.String("volumeId", volumeID), zap.String("instanceId", instanceID))
	return nil
}

// DeleteVolume 删除EBS卷
func (e *EBSService) DeleteVolume(ctx context.Context, region string, volumeID string) error {
	client, err := e.sdk.CreateEC2Client(ctx, region)
	if err != nil {
		e.sdk.logger.Error("创建EC2客户端失败", zap.Error(err))
		return err
	}

	request := &ec2.DeleteVolumeInput{
		VolumeId: &volumeID,
	}

	e.sdk.logger.Info("开始删除EBS卷", zap.String("region", region), zap.String("volumeId", volumeID))
	_, err = client.DeleteVolume(ctx, request)
	if err != nil {
		e.sdk.logger.Error("删除EBS卷失败", zap.Error(err))
		return err
	}

	e.sdk.logger.Info("EBS卷删除成功", zap.String("volumeId", volumeID))
	return nil
}

// ListVolumesRequest 查询EBS卷列表请求参数
type ListVolumesRequest struct {
	Region string
	Page   int
	Size   int
}

// ListVolumesResponseBody 查询EBS卷列表响应
type ListVolumesResponseBody struct {
	Volumes []types.Volume
	Total   int64
}

// ListVolumes 查询EBS卷列表
func (e *EBSService) ListVolumes(ctx context.Context, req *ListVolumesRequest) (*ListVolumesResponseBody, int64, error) {
	client, err := e.sdk.CreateEC2Client(ctx, req.Region)
	if err != nil {
		e.sdk.logger.Error("创建EC2客户端失败", zap.Error(err))
		return nil, 0, err
	}

	request := &ec2.DescribeVolumesInput{}

	e.sdk.logger.Info("开始查询EBS卷列表", zap.String("region", req.Region))
	response, err := client.DescribeVolumes(ctx, request)
	if err != nil {
		e.sdk.logger.Error("查询EBS卷列表失败", zap.Error(err))
		return nil, 0, err
	}

	allVolumes := response.Volumes
	totalCount := int64(len(allVolumes))

	// 分页处理
	startIdx := (req.Page - 1) * req.Size
	endIdx := req.Page * req.Size
	if startIdx >= len(allVolumes) {
		return &ListVolumesResponseBody{
			Volumes: []types.Volume{},
		}, totalCount, nil
	}

	if endIdx > len(allVolumes) {
		endIdx = len(allVolumes)
	}

	e.sdk.logger.Info("查询EBS卷列表成功", zap.Int64("total", totalCount))

	return &ListVolumesResponseBody{
		Volumes: allVolumes[startIdx:endIdx],
		Total:   totalCount,
	}, totalCount, nil
}

// GetVolumeDetail 获取EBS卷详情
func (e *EBSService) GetVolumeDetail(ctx context.Context, region string, volumeID string) (*types.Volume, error) {
	client, err := e.sdk.CreateEC2Client(ctx, region)
	if err != nil {
		return nil, err
	}

	request := &ec2.DescribeVolumesInput{
		VolumeIds: []string{volumeID},
	}

	e.sdk.logger.Info("开始查询EBS卷详情", zap.String("volumeId", volumeID))
	response, err := client.DescribeVolumes(ctx, request)
	if err != nil {
		e.sdk.logger.Error("查询EBS卷详情失败", zap.Error(err))
		return nil, err
	}

	if len(response.Volumes) == 0 {
		return nil, fmt.Errorf("EBS卷 %s 不存在", volumeID)
	}

	e.sdk.logger.Info("查询EBS卷详情成功", zap.String("volumeId", volumeID))
	return &response.Volumes[0], nil
}

// ModifyVolume 修改EBS卷属性
func (e *EBSService) ModifyVolume(ctx context.Context, region string, volumeID string, size int32, volumeType string, iops int32, throughput int32) error {
	client, err := e.sdk.CreateEC2Client(ctx, region)
	if err != nil {
		e.sdk.logger.Error("创建EC2客户端失败", zap.Error(err))
		return err
	}

	request := &ec2.ModifyVolumeInput{
		VolumeId: &volumeID,
	}

	// 设置大小
	if size > 0 {
		request.Size = &size
	}

	// 设置卷类型
	if volumeType != "" {
		var vType types.VolumeType
		switch volumeType {
		case "gp2":
			vType = types.VolumeTypeGp2
		case "gp3":
			vType = types.VolumeTypeGp3
		case "io1":
			vType = types.VolumeTypeIo1
		case "io2":
			vType = types.VolumeTypeIo2
		case "st1":
			vType = types.VolumeTypeSt1
		case "sc1":
			vType = types.VolumeTypeSc1
		default:
			vType = types.VolumeTypeGp3
		}
		request.VolumeType = vType
	}

	// 设置IOPS
	if iops > 0 {
		request.Iops = &iops
	}

	// 设置吞吐量
	if throughput > 0 {
		request.Throughput = &throughput
	}

	e.sdk.logger.Info("开始修改EBS卷", zap.String("region", region), zap.String("volumeId", volumeID))
	_, err = client.ModifyVolume(ctx, request)
	if err != nil {
		e.sdk.logger.Error("修改EBS卷失败", zap.Error(err))
		return err
	}

	e.sdk.logger.Info("EBS卷修改成功", zap.String("volumeId", volumeID))
	return nil
}

// CreateSnapshot 创建EBS快照
func (e *EBSService) CreateSnapshot(ctx context.Context, region string, volumeID string, description string, tags map[string]string) (string, error) {
	client, err := e.sdk.CreateEC2Client(ctx, region)
	if err != nil {
		e.sdk.logger.Error("创建EC2客户端失败", zap.Error(err))
		return "", err
	}

	request := &ec2.CreateSnapshotInput{
		VolumeId:    &volumeID,
		Description: &description,
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeSnapshot,
				Tags: []types.Tag{
					{
						Key:   stringPtr("Description"),
						Value: &description,
					},
				},
			},
		},
	}

	// 添加自定义标签
	if tags != nil {
		for key, value := range tags {
			request.TagSpecifications[0].Tags = append(request.TagSpecifications[0].Tags, types.Tag{
				Key:   &key,
				Value: &value,
			})
		}
	}

	e.sdk.logger.Info("开始创建EBS快照", zap.String("region", region), zap.String("volumeId", volumeID))
	response, err := client.CreateSnapshot(ctx, request)
	if err != nil {
		e.sdk.logger.Error("创建EBS快照失败", zap.Error(err))
		return "", err
	}

	snapshotId := *response.SnapshotId
	e.sdk.logger.Info("EBS快照创建成功", zap.String("snapshotId", snapshotId))

	return snapshotId, nil
}

// DeleteSnapshot 删除EBS快照
func (e *EBSService) DeleteSnapshot(ctx context.Context, region string, snapshotID string) error {
	client, err := e.sdk.CreateEC2Client(ctx, region)
	if err != nil {
		e.sdk.logger.Error("创建EC2客户端失败", zap.Error(err))
		return err
	}

	request := &ec2.DeleteSnapshotInput{
		SnapshotId: &snapshotID,
	}

	e.sdk.logger.Info("开始删除EBS快照", zap.String("region", region), zap.String("snapshotId", snapshotID))
	_, err = client.DeleteSnapshot(ctx, request)
	if err != nil {
		e.sdk.logger.Error("删除EBS快照失败", zap.Error(err))
		return err
	}

	e.sdk.logger.Info("EBS快照删除成功", zap.String("snapshotId", snapshotID))
	return nil
}

// waitForVolumeAvailable 等待EBS卷可用
func (e *EBSService) waitForVolumeAvailable(ctx context.Context, client *ec2.Client, volumeId string) error {
	request := &ec2.DescribeVolumesInput{
		VolumeIds: []string{volumeId},
	}

	for i := 0; i < 30; i++ {
		response, err := client.DescribeVolumes(ctx, request)
		if err != nil {
			return err
		}

		if len(response.Volumes) > 0 && response.Volumes[0].State == types.VolumeStateAvailable {
			return nil
		}

		time.Sleep(5 * time.Second)
	}

	return fmt.Errorf("等待EBS卷可用超时")
}
