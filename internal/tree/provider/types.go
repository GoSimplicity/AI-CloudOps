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

package provider

import (
	"context"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

type Provider interface {
	SyncResources(ctx context.Context, region string) error

	// 资源管理
	ListInstances(ctx context.Context, region string, pageSize int, pageNumber int) ([]*model.ResourceEcs, int64, error)
	GetInstanceDetail(ctx context.Context, region string, instanceID string) (*model.ResourceEcs, error)
	CreateInstance(ctx context.Context, region string, config *model.CreateEcsResourceReq) error
	DeleteInstance(ctx context.Context, region string, instanceID string) error
	StartInstance(ctx context.Context, region string, instanceID string) error
	StopInstance(ctx context.Context, region string, instanceID string) error
	RestartInstance(ctx context.Context, region string, instanceID string) error

	// 网络管理
	ListVPCs(ctx context.Context, region string, pageNumber int, pageSize int) ([]*model.ResourceVpc, int64, error)
	CreateVPC(ctx context.Context, region string, config *model.CreateVpcResourceReq) error
	DeleteVPC(ctx context.Context, region string, vpcID string) error
	GetVpcDetail(ctx context.Context, region string, vpcID string) (*model.ResourceVpc, error)

	// 存储管理
	ListDisks(ctx context.Context, region string, pageSize int, pageNumber int) ([]*model.PageResp, error)
	CreateDisk(ctx context.Context, region string, config *model.DiskCreationParams) error
	DeleteDisk(ctx context.Context, region string, diskID string) error
	AttachDisk(ctx context.Context, region string, diskID string, instanceID string) error
	DetachDisk(ctx context.Context, region string, diskID string, instanceID string) error

	ListRegions(ctx context.Context) ([]*model.RegionResp, error)
	GetZonesByVpc(ctx context.Context, region string, vpcId string) ([]*model.ZoneResp, error)
	ListInstanceOptions(ctx context.Context, payType string, region string, zone string, instanceType string, systemDiskCategory string, dataDiskCategory string) ([]interface{}, error)
}
