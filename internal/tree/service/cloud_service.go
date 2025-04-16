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

package service

import (
	"context"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/dao"
	"go.uber.org/zap"
)

type CloudService interface {
	ListCloudProviders(ctx context.Context) ([]model.CloudProviderResp, error)
	ListRegions(ctx context.Context, provider model.CloudProvider) ([]model.RegionResp, error)
	ListZones(ctx context.Context, provider model.CloudProvider, region string) ([]model.ZoneResp, error)
	ListInstanceTypes(ctx context.Context, provider model.CloudProvider, region string) ([]model.InstanceTypeResp, error)
	ListImages(ctx context.Context, provider model.CloudProvider, region string) ([]model.ImageResp, error)
	ListVpcs(ctx context.Context, provider model.CloudProvider, region string) ([]model.VpcResp, error)
	ListSecurityGroups(ctx context.Context, provider model.CloudProvider, region string) ([]model.SecurityGroupResp, error)
}

type cloudService struct {
	logger *zap.Logger
	dao    dao.CloudDAO
}

func NewCloudService(logger *zap.Logger, dao dao.CloudDAO) CloudService {
	return &cloudService{
		logger: logger,
		dao:    dao,
	}
}

// ListCloudProviders 获取云厂商列表
func (c *cloudService) ListCloudProviders(ctx context.Context) ([]model.CloudProviderResp, error) {
	panic("unimplemented")
}

// ListImages 获取镜像列表
func (c *cloudService) ListImages(ctx context.Context, provider model.CloudProvider, region string) ([]model.ImageResp, error) {
	panic("unimplemented")
}

// ListInstanceTypes 获取实例类型列表
func (c *cloudService) ListInstanceTypes(ctx context.Context, provider model.CloudProvider, region string) ([]model.InstanceTypeResp, error) {
	panic("unimplemented")
}

// ListRegions 获取区域列表
func (c *cloudService) ListRegions(ctx context.Context, provider model.CloudProvider) ([]model.RegionResp, error) {
	panic("unimplemented")
}

// ListSecurityGroups 获取安全组列表
func (c *cloudService) ListSecurityGroups(ctx context.Context, provider model.CloudProvider, region string) ([]model.SecurityGroupResp, error) {
	panic("unimplemented")
}

// ListVpcs 获取VPC列表
func (c *cloudService) ListVpcs(ctx context.Context, provider model.CloudProvider, region string) ([]model.VpcResp, error) {
	panic("unimplemented")
}

// ListZones 获取可用区列表
func (c *cloudService) ListZones(ctx context.Context, provider model.CloudProvider, region string) ([]model.ZoneResp, error) {
	panic("unimplemented")
}
