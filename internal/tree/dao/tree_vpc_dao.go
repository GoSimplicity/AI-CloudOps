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

package dao

import (
	"context"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type TreeVpcDAO interface {
	// VPC资源管理
	ListVpcResources(ctx context.Context, req *model.ListVpcResourcesReq) (model.ListResp[*model.ResourceVpc], error)
	GetVpcResourceById(ctx context.Context, id int) (*model.ResourceVpc, error)
	CreateVpcResource(ctx context.Context, req *model.CreateVpcResourceReq) error
	UpdateVpcResource(ctx context.Context, req *model.UpdateVpcReq) error
	DeleteVpcResource(ctx context.Context, id int) error

	// 子网管理
	ListSubnets(ctx context.Context, req *model.ListSubnetsReq) (model.ListResp[*model.ResourceSubnet], error)
	GetSubnetById(ctx context.Context, id int) (*model.ResourceSubnet, error)
	CreateSubnet(ctx context.Context, req *model.CreateSubnetReq) error
	UpdateSubnet(ctx context.Context, req *model.UpdateSubnetReq) error
	DeleteSubnet(ctx context.Context, id int) error

	// VPC对等连接管理
	ListVpcPeerings(ctx context.Context, req *model.ListVpcPeeringsReq) (model.ListResp[*model.ResourceVpcPeering], error)
	GetVpcPeeringById(ctx context.Context, id int) (*model.ResourceVpcPeering, error)
	CreateVpcPeering(ctx context.Context, req *model.CreateVpcPeeringReq) error
	DeleteVpcPeering(ctx context.Context, id int) error

	// 同步操作
	SyncVPCResources(ctx context.Context, resources []*model.ResourceVpc, total int64) error
}

type treeVpcDAO struct {
	logger *zap.Logger
	db     *gorm.DB
}

func NewTreeVpcDAO(logger *zap.Logger, db *gorm.DB) TreeVpcDAO {
	return &treeVpcDAO{
		logger: logger,
		db:     db,
	}
}

// CreateSubnet 创建子网
func (t *treeVpcDAO) CreateSubnet(ctx context.Context, req *model.CreateSubnetReq) error {
	subnet := &model.ResourceSubnet{
		SubnetName:          req.SubnetName,
		VpcId:               req.VpcId,
		Provider:            req.Provider,
		RegionId:            req.Region,
		ZoneId:              req.ZoneId,
		CidrBlock:           req.CidrBlock,
		Description:         req.Description,
		MapPublicIpOnLaunch: req.MapPublicIpOnLaunch,
		TreeNodeID:          req.TreeNodeID,
		Status:              "Creating",
	}

	if err := t.db.WithContext(ctx).Create(subnet).Error; err != nil {
		t.logger.Error("创建子网失败", zap.Error(err))
		return err
	}

	return nil
}

// CreateVpcPeering 创建VPC对等连接
func (t *treeVpcDAO) CreateVpcPeering(ctx context.Context, req *model.CreateVpcPeeringReq) error {
	peering := &model.ResourceVpcPeering{
		PeeringName:   req.PeeringName,
		Provider:      req.Provider,
		RegionId:      req.Region,
		LocalVpcId:    req.LocalVpcId,
		PeerVpcId:     req.PeerVpcId,
		PeerRegionId:  req.PeerRegionId,
		PeerAccountId: req.PeerAccountId,
		Description:   req.Description,
		TreeNodeID:    req.TreeNodeID,
		Status:        "Creating",
	}

	if err := t.db.WithContext(ctx).Create(peering).Error; err != nil {
		t.logger.Error("创建VPC对等连接失败", zap.Error(err))
		return err
	}

	return nil
}

// CreateVpcResource 创建VPC资源
func (t *treeVpcDAO) CreateVpcResource(ctx context.Context, req *model.CreateVpcResourceReq) error {
	vpc := &model.ResourceVpc{
		InstanceName:       req.VpcName,
		VpcName:           req.VpcName,
		Provider:          req.Provider,
		RegionId:          req.Region,
		ZoneId:            req.ZoneId,
		CidrBlock:         req.CidrBlock,
		Description:       req.Description,
		TreeNodeID:        req.TreeNodeID,
		Env:               req.Env,
		Status:            "Creating",
		CreateByOrder:     true,
	}

	if err := t.db.WithContext(ctx).Create(vpc).Error; err != nil {
		t.logger.Error("创建VPC资源失败", zap.Error(err))
		return err
	}

	return nil
}

// DeleteSubnet 删除子网
func (t *treeVpcDAO) DeleteSubnet(ctx context.Context, id int) error {
	if err := t.db.WithContext(ctx).Where("id = ?", id).Delete(&model.ResourceSubnet{}).Error; err != nil {
		t.logger.Error("删除子网失败", zap.Error(err), zap.Int("id", id))
		return err
	}

	return nil
}

// DeleteVpcPeering 删除VPC对等连接
func (t *treeVpcDAO) DeleteVpcPeering(ctx context.Context, id int) error {
	if err := t.db.WithContext(ctx).Where("id = ?", id).Delete(&model.ResourceVpcPeering{}).Error; err != nil {
		t.logger.Error("删除VPC对等连接失败", zap.Error(err), zap.Int("id", id))
		return err
	}

	return nil
}

// DeleteVpcResource 删除VPC资源
func (t *treeVpcDAO) DeleteVpcResource(ctx context.Context, id int) error {
	if err := t.db.WithContext(ctx).Where("id = ?", id).Delete(&model.ResourceVpc{}).Error; err != nil {
		t.logger.Error("删除VPC资源失败", zap.Error(err), zap.Int("id", id))
		return err
	}

	return nil
}

// GetSubnetById 根据ID获取子网
func (t *treeVpcDAO) GetSubnetById(ctx context.Context, id int) (*model.ResourceSubnet, error) {
	var subnet model.ResourceSubnet

	if err := t.db.WithContext(ctx).Where("id = ?", id).First(&subnet).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, err
		}
		t.logger.Error("查询子网失败", zap.Error(err), zap.Int("id", id))
		return nil, err
	}

	return &subnet, nil
}

// GetVpcPeeringById 根据ID获取VPC对等连接
func (t *treeVpcDAO) GetVpcPeeringById(ctx context.Context, id int) (*model.ResourceVpcPeering, error) {
	var peering model.ResourceVpcPeering

	if err := t.db.WithContext(ctx).Where("id = ?", id).First(&peering).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, err
		}
		t.logger.Error("查询VPC对等连接失败", zap.Error(err), zap.Int("id", id))
		return nil, err
	}

	return &peering, nil
}

// GetVpcResourceById 根据ID获取VPC资源
func (t *treeVpcDAO) GetVpcResourceById(ctx context.Context, id int) (*model.ResourceVpc, error) {
	var resource model.ResourceVpc

	if err := t.db.WithContext(ctx).Where("id = ?", id).First(&resource).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, err
		}
		t.logger.Error("查询VPC资源失败", zap.Error(err), zap.Int("id", id))
		return nil, err
	}

	return &resource, nil
}

// ListSubnets 获取子网列表
func (t *treeVpcDAO) ListSubnets(ctx context.Context, req *model.ListSubnetsReq) (model.ListResp[*model.ResourceSubnet], error) {
	var subnets []*model.ResourceSubnet
	var total int64

	db := t.db.WithContext(ctx).Model(&model.ResourceSubnet{})

	// 构建查询条件
	if req.Provider != "" {
		db = db.Where("provider = ?", req.Provider)
	}
	if req.Region != "" {
		db = db.Where("region_id = ?", req.Region)
	}
	if req.VpcId != "" {
		db = db.Where("vpc_id = ?", req.VpcId)
	}
	if req.ZoneId != "" {
		db = db.Where("zone_id = ?", req.ZoneId)
	}
	if req.Status != "" {
		db = db.Where("status = ?", req.Status)
	}
	if req.SubnetName != "" {
		db = db.Where("subnet_name LIKE ?", "%"+req.SubnetName+"%")
	}
	if req.TreeNodeID > 0 {
		db = db.Where("tree_node_id = ?", req.TreeNodeID)
	}

	// 计算总数
	if err := db.Count(&total).Error; err != nil {
		t.logger.Error("统计子网数量失败", zap.Error(err))
		return model.ListResp[*model.ResourceSubnet]{}, err
	}

	// 处理分页
	if req.PageSize > 0 && req.PageNumber > 0 {
		offset := (req.PageNumber - 1) * req.PageSize
		db = db.Offset(offset).Limit(req.PageSize)
	}

	// 排序
	db = db.Order("created_at DESC")

	if err := db.Find(&subnets).Error; err != nil {
		t.logger.Error("查询子网列表失败", zap.Error(err))
		return model.ListResp[*model.ResourceSubnet]{}, err
	}

	return model.ListResp[*model.ResourceSubnet]{
		Items: subnets,
		Total: total,
	}, nil
}

// ListVpcPeerings 获取VPC对等连接列表
func (t *treeVpcDAO) ListVpcPeerings(ctx context.Context, req *model.ListVpcPeeringsReq) (model.ListResp[*model.ResourceVpcPeering], error) {
	var peerings []*model.ResourceVpcPeering
	var total int64

	db := t.db.WithContext(ctx).Model(&model.ResourceVpcPeering{})

	// 构建查询条件
	if req.Provider != "" {
		db = db.Where("provider = ?", req.Provider)
	}
	if req.Region != "" {
		db = db.Where("region_id = ?", req.Region)
	}
	if req.LocalVpcId != "" {
		db = db.Where("local_vpc_id = ?", req.LocalVpcId)
	}
	if req.PeerVpcId != "" {
		db = db.Where("peer_vpc_id = ?", req.PeerVpcId)
	}
	if req.Status != "" {
		db = db.Where("status = ?", req.Status)
	}
	if req.PeeringName != "" {
		db = db.Where("peering_name LIKE ?", "%"+req.PeeringName+"%")
	}
	if req.TreeNodeID > 0 {
		db = db.Where("tree_node_id = ?", req.TreeNodeID)
	}

	// 计算总数
	if err := db.Count(&total).Error; err != nil {
		t.logger.Error("统计VPC对等连接数量失败", zap.Error(err))
		return model.ListResp[*model.ResourceVpcPeering]{}, err
	}

	// 处理分页
	if req.PageSize > 0 && req.PageNumber > 0 {
		offset := (req.PageNumber - 1) * req.PageSize
		db = db.Offset(offset).Limit(req.PageSize)
	}

	// 排序
	db = db.Order("created_at DESC")

	if err := db.Find(&peerings).Error; err != nil {
		t.logger.Error("查询VPC对等连接列表失败", zap.Error(err))
		return model.ListResp[*model.ResourceVpcPeering]{}, err
	}

	return model.ListResp[*model.ResourceVpcPeering]{
		Items: peerings,
		Total: total,
	}, nil
}

// ListVpcResources 获取VPC资源列表
func (t *treeVpcDAO) ListVpcResources(ctx context.Context, req *model.ListVpcResourcesReq) (model.ListResp[*model.ResourceVpc], error) {
	var resources []*model.ResourceVpc
	var total int64

	db := t.db.WithContext(ctx).Model(&model.ResourceVpc{})

	// 构建查询条件
	if req.Provider != "" {
		db = db.Where("provider = ?", req.Provider)
	}
	if req.Region != "" {
		db = db.Where("region_id = ?", req.Region)
	}
	if req.Status != "" {
		db = db.Where("status = ?", req.Status)
	}
	if req.VpcName != "" {
		db = db.Where("vpc_name LIKE ?", "%"+req.VpcName+"%")
	}
	if req.TreeNodeID > 0 {
		db = db.Where("tree_node_id = ?", req.TreeNodeID)
	}
	if req.Env != "" {
		db = db.Where("environment = ?", req.Env)
	}

	// 计算总数
	if err := db.Count(&total).Error; err != nil {
		t.logger.Error("统计VPC资源数量失败", zap.Error(err))
		return model.ListResp[*model.ResourceVpc]{}, err
	}

	// 处理分页
	if req.PageSize > 0 && req.PageNumber > 0 {
		offset := (req.PageNumber - 1) * req.PageSize
		db = db.Offset(offset).Limit(req.PageSize)
	}

	// 排序
	db = db.Order("created_at DESC")

	if err := db.Find(&resources).Error; err != nil {
		t.logger.Error("查询VPC资源列表失败", zap.Error(err))
		return model.ListResp[*model.ResourceVpc]{}, err
	}

	return model.ListResp[*model.ResourceVpc]{
		Items: resources,
		Total: total,
	}, nil
}

// UpdateSubnet 更新子网
func (t *treeVpcDAO) UpdateSubnet(ctx context.Context, req *model.UpdateSubnetReq) error {
	updates := make(map[string]any)

	if req.SubnetName != "" {
		updates["subnet_name"] = req.SubnetName
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	updates["map_public_ip_on_launch"] = req.MapPublicIpOnLaunch
	if req.TreeNodeID > 0 {
		updates["tree_node_id"] = req.TreeNodeID
	}

	if err := t.db.WithContext(ctx).Model(&model.ResourceSubnet{}).Where("id = ?", req.ID).Updates(updates).Error; err != nil {
		t.logger.Error("更新子网失败", zap.Error(err), zap.Int("id", req.ID))
		return err
	}

	return nil
}

// UpdateVpcResource 更新VPC资源
func (t *treeVpcDAO) UpdateVpcResource(ctx context.Context, req *model.UpdateVpcReq) error {
	updates := make(map[string]any)

	if req.VpcName != "" {
		updates["vpc_name"] = req.VpcName
		updates["instance_name"] = req.VpcName
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.TreeNodeID > 0 {
		updates["tree_node_id"] = req.TreeNodeID
	}
	if req.Env != "" {
		updates["environment"] = req.Env
	}

	if len(updates) == 0 {
		return nil
	}

	if err := t.db.WithContext(ctx).Model(&model.ResourceVpc{}).Where("id = ?", req.ID).Updates(updates).Error; err != nil {
		t.logger.Error("更新VPC资源失败", zap.Error(err), zap.Int("id", req.ID))
		return err
	}

	return nil
}

// SyncVPCResources 同步VPC资源到数据库
func (t *treeVpcDAO) SyncVPCResources(ctx context.Context, resources []*model.ResourceVpc, total int64) error {
	if len(resources) == 0 {
		return nil
	}

	return t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, resource := range resources {
			var existingResource model.ResourceVpc
			err := tx.Where("vpc_id = ? AND provider = ? AND region_id = ?", 
				resource.VpcId, resource.Provider, resource.RegionId).First(&existingResource).Error
			
			if err == gorm.ErrRecordNotFound {
				if err := tx.Create(resource).Error; err != nil {
					t.logger.Error("创建VPC资源失败", zap.Error(err))
					return err
				}
			} else if err != nil {
				t.logger.Error("查询VPC资源失败", zap.Error(err))
				return err
			} else {
				resource.ID = existingResource.ID
				resource.CreatedAt = existingResource.CreatedAt
				if err := tx.Model(&existingResource).Updates(resource).Error; err != nil {
					t.logger.Error("更新VPC资源失败", zap.Error(err))
					return err
				}
			}
		}
		return nil
	})
}
