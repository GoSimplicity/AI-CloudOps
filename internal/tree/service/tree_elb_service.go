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
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/dao"
	"go.uber.org/zap"
)

type TreeElbService interface {
	// 资源管理
	ListElbResources(ctx context.Context, req *model.ListElbResourcesReq) (model.ListResp[*model.ResourceElb], error)
	GetElbDetail(ctx context.Context, req *model.GetElbDetailReq) (*model.ResourceElb, error)
	CreateElbResource(ctx context.Context, req *model.CreateElbResourceReq) error
	UpdateElb(ctx context.Context, req *model.UpdateElbReq) error
	DeleteElb(ctx context.Context, req *model.DeleteElbReq) error
	StartElb(ctx context.Context, req *model.StartElbReq) error
	StopElb(ctx context.Context, req *model.StopElbReq) error
	RestartElb(ctx context.Context, req *model.RestartElbReq) error
	ResizeElb(ctx context.Context, req *model.ResizeElbReq) error
	BindServersToElb(ctx context.Context, req *model.BindServersToElbReq) error
	UnbindServersFromElb(ctx context.Context, req *model.UnbindServersFromElbReq) error
	ConfigureHealthCheck(ctx context.Context, req *model.ConfigureHealthCheckReq) error
}

type elbService struct {
	logger *zap.Logger
	dao    dao.TreeElbDAO
}

func NewTreeElbService(logger *zap.Logger, dao dao.TreeElbDAO) TreeElbService {
	return &elbService{
		logger: logger,
		dao:    dao,
	}
}

// BindServersToElb 绑定服务器到ELB
func (e *elbService) BindServersToElb(ctx context.Context, req *model.BindServersToElbReq) error {
	// 获取ELB实例信息
	resource, err := e.dao.GetElbResourceById(ctx, req.ElbID)
	if err != nil {
		e.logger.Error("获取ELB实例失败", zap.Error(err))
		return err
	}

	// 构建服务器列表
	servers := make([]string, 0, len(req.ServerIDs))
	for i, serverID := range req.ServerIDs {
		port := 80 // 默认端口
		if i < len(req.Ports) {
			port = req.Ports[i]
		}
		servers = append(servers, fmt.Sprintf("%d:%d:%d", serverID, port, req.Weight))
	}

	// 更新后端服务器列表
	existingServers := resource.BackendServers
	updatedServers := append(existingServers, servers...)
	resource.BackendServers = updatedServers

	// 更新ELB资源
	if err := e.dao.UpdateElbResource(ctx, resource); err != nil {
		e.logger.Error("更新ELB后端服务器失败", zap.Error(err))
		return err
	}

	return nil
}

// ConfigureHealthCheck 配置健康检查
func (e *elbService) ConfigureHealthCheck(ctx context.Context, req *model.ConfigureHealthCheckReq) error {
	// 获取ELB实例信息
	_, err := e.dao.GetElbResourceById(ctx, req.ID)
	if err != nil {
		e.logger.Error("获取ELB实例失败", zap.Error(err))
		return err
	}

	// 创建健康检查配置
	healthCheck := &model.ElbHealthCheck{
		Enabled:            req.HealthCheckEnabled,
		Type:               req.HealthCheckType,
		Port:               req.HealthCheckPort,
		Path:               req.HealthCheckPath,
		Interval:           req.HealthCheckInterval,
		Timeout:            req.HealthCheckTimeout,
		HealthyThreshold:   req.HealthyThreshold,
		UnhealthyThreshold: req.UnhealthyThreshold,
		HttpCode:           req.HealthCheckHttpCode,
		Domain:             req.HealthCheckDomain,
	}

	// 更新健康检查配置
	if err := e.dao.UpdateElbHealthCheck(ctx, healthCheck); err != nil {
		e.logger.Error("更新ELB健康检查配置失败", zap.Error(err))
		return err
	}

	e.logger.Info("ELB健康检查配置更新成功", zap.Int("elbId", req.ID))
	return nil
}

// CreateElbResource 创建ELB资源
func (e *elbService) CreateElbResource(ctx context.Context, req *model.CreateElbResourceReq) error {
	// 验证请求参数
	if err := validateCreateElbResourceReq(req); err != nil {
		e.logger.Error("创建ELB资源参数验证失败", zap.Error(err))
		return err
	}

	// 构建ELB资源对象
	resource := &model.ResourceElb{
		InstanceName:       req.InstanceName,
		Provider:           req.Provider,
		RegionId:           req.RegionId,
		ZoneId:             req.ZoneId,
		VpcId:              req.VpcId,
		LoadBalancerType:   req.LoadBalancerType,
		AddressType:        req.AddressType,
		BandwidthCapacity:  req.BandwidthCapacity,
		TreeNodeID:         req.TreeNodeID,
		Description:        req.Description,
		Tags:               req.Tags,
		SecurityGroupIds:   req.SecurityGroupIds,
		Env:                req.Env,
		InstanceChargeType: req.InstanceChargeType,
		CrossZoneEnabled:   req.CrossZoneEnabled,
		BandwidthPackageId: req.BandwidthPackageId,
		Status:             "Creating",
		CreationTime:       time.Now().Format(time.RFC3339),
		LastSyncTime:       time.Now(),
	}

	// 创建ELB资源
	if err := e.dao.CreateElbResource(ctx, resource); err != nil {
		e.logger.Error("创建ELB资源失败", zap.Error(err))
		return err
	}

	e.logger.Info("ELB资源创建成功", zap.String("instanceName", req.InstanceName))
	return nil
}

// DeleteElb 删除ELB
func (e *elbService) DeleteElb(ctx context.Context, req *model.DeleteElbReq) error {
	// 获取ELB实例信息
	resource, err := e.dao.GetElbResourceById(ctx, req.ID)
	if err != nil {
		e.logger.Error("获取ELB实例失败", zap.Error(err))
		return err
	}

	// 检查是否可以删除
	if !req.Force && resource.Status == "Running" {
		e.logger.Error("ELB正在运行中，不能删除", zap.Int("id", req.ID))
		return errors.New("ELB正在运行中，请先停止后再删除，或使用强制删除")
	}

	// 删除ELB资源
	if err := e.dao.DeleteElbResource(ctx, req.ID); err != nil {
		e.logger.Error("删除ELB资源失败", zap.Error(err))
		return err
	}

	e.logger.Info("ELB资源删除成功", zap.Int("id", req.ID))
	return nil
}

// GetElbDetail 获取ELB详情
func (e *elbService) GetElbDetail(ctx context.Context, req *model.GetElbDetailReq) (*model.ResourceElb, error) {
	resource, err := e.dao.GetElbResourceById(ctx, req.ID)
	if err != nil {
		e.logger.Error("获取ELB详情失败", zap.Error(err), zap.Int("id", req.ID))
		return nil, err
	}

	return resource, nil
}

// ListElbResources 获取ELB资源列表
func (e *elbService) ListElbResources(ctx context.Context, req *model.ListElbResourcesReq) (model.ListResp[*model.ResourceElb], error) {
	result, err := e.dao.ListElbResources(ctx, req)
	if err != nil {
		e.logger.Error("获取ELB资源列表失败", zap.Error(err))
		return model.ListResp[*model.ResourceElb]{}, err
	}

	return result, nil
}

// ResizeElb 调整ELB规格
func (e *elbService) ResizeElb(ctx context.Context, req *model.ResizeElbReq) error {
	// 获取ELB实例信息
	resource, err := e.dao.GetElbResourceById(ctx, req.ID)
	if err != nil {
		e.logger.Error("获取ELB实例失败", zap.Error(err))
		return err
	}

	// 更新带宽容量
	resource.BandwidthCapacity = req.BandwidthCapacity
	if req.LoadBalancerType != "" {
		resource.LoadBalancerType = req.LoadBalancerType
	}

	// 更新ELB资源
	if err := e.dao.UpdateElbResource(ctx, resource); err != nil {
		e.logger.Error("调整ELB规格失败", zap.Error(err))
		return err
	}

	e.logger.Info("ELB规格调整成功", zap.Int("id", req.ID))
	return nil
}

// RestartElb 重启ELB
func (e *elbService) RestartElb(ctx context.Context, req *model.RestartElbReq) error {
	// 获取ELB实例信息
	resource, err := e.dao.GetElbResourceById(ctx, req.ID)
	if err != nil {
		e.logger.Error("获取ELB实例失败", zap.Error(err))
		return err
	}

	// 更新状态为重启中
	resource.Status = "Restarting"
	if err := e.dao.UpdateElbResource(ctx, resource); err != nil {
		e.logger.Error("更新ELB状态失败", zap.Error(err))
		return err
	}

	e.logger.Info("ELB重启成功", zap.Int("id", req.ID))
	return nil
}

// StartElb 启动ELB
func (e *elbService) StartElb(ctx context.Context, req *model.StartElbReq) error {
	// 获取ELB实例信息
	resource, err := e.dao.GetElbResourceById(ctx, req.ID)
	if err != nil {
		e.logger.Error("获取ELB实例失败", zap.Error(err))
		return err
	}

	// 更新状态为启动中
	resource.Status = "Starting"
	if err := e.dao.UpdateElbResource(ctx, resource); err != nil {
		e.logger.Error("更新ELB状态失败", zap.Error(err))
		return err
	}

	e.logger.Info("ELB启动成功", zap.Int("id", req.ID))
	return nil
}

// StopElb 停止ELB
func (e *elbService) StopElb(ctx context.Context, req *model.StopElbReq) error {
	// 获取ELB实例信息
	resource, err := e.dao.GetElbResourceById(ctx, req.ID)
	if err != nil {
		e.logger.Error("获取ELB实例失败", zap.Error(err))
		return err
	}

	// 更新状态为停止中
	resource.Status = "Stopping"
	if err := e.dao.UpdateElbResource(ctx, resource); err != nil {
		e.logger.Error("更新ELB状态失败", zap.Error(err))
		return err
	}

	e.logger.Info("ELB停止成功", zap.Int("id", req.ID))
	return nil
}

// UnbindServersFromElb 从ELB解绑服务器
func (e *elbService) UnbindServersFromElb(ctx context.Context, req *model.UnbindServersFromElbReq) error {
	// 获取ELB实例信息
	resource, err := e.dao.GetElbResourceById(ctx, req.ElbID)
	if err != nil {
		e.logger.Error("获取ELB实例失败", zap.Error(err))
		return err
	}

	// 构建要删除的服务器列表
	serversToRemove := make(map[string]bool)
	for i, serverID := range req.ServerIDs {
		port := 80 // 默认端口
		if i < len(req.Ports) {
			port = req.Ports[i]
		}
		serversToRemove[fmt.Sprintf("%d:%d", serverID, port)] = true
	}

	// 过滤后端服务器列表
	filteredServers := make([]string, 0)
	for _, server := range resource.BackendServers {
		// 解析服务器信息 (格式: serverID:port:weight)
		parts := strings.Split(server, ":")
		if len(parts) >= 2 {
			key := parts[0] + ":" + parts[1]
			if !serversToRemove[key] {
				filteredServers = append(filteredServers, server)
			}
		}
	}

	// 更新后端服务器列表
	resource.BackendServers = filteredServers

	// 更新ELB资源
	if err := e.dao.UpdateElbResource(ctx, resource); err != nil {
		e.logger.Error("更新ELB后端服务器失败", zap.Error(err))
		return err
	}

	e.logger.Info("ELB服务器解绑成功", zap.Int("elbId", req.ElbID))
	return nil
}

// UpdateElb 更新ELB
func (e *elbService) UpdateElb(ctx context.Context, req *model.UpdateElbReq) error {
	// 获取ELB实例信息
	resource, err := e.dao.GetElbResourceById(ctx, req.ID)
	if err != nil {
		e.logger.Error("获取ELB实例失败", zap.Error(err))
		return err
	}

	// 更新字段
	if req.InstanceName != "" {
		resource.InstanceName = req.InstanceName
	}
	if req.Description != "" {
		resource.Description = req.Description
	}
	if req.Tags != nil {
		resource.Tags = req.Tags
	}
	if req.SecurityGroupIds != nil {
		resource.SecurityGroupIds = req.SecurityGroupIds
	}
	if req.BandwidthCapacity > 0 {
		resource.BandwidthCapacity = req.BandwidthCapacity
	}
	if req.CrossZoneEnabled != nil {
		resource.CrossZoneEnabled = *req.CrossZoneEnabled
	}
	if req.BandwidthPackageId != "" {
		resource.BandwidthPackageId = req.BandwidthPackageId
	}

	// 更新ELB资源
	if err := e.dao.UpdateElbResource(ctx, resource); err != nil {
		e.logger.Error("更新ELB资源失败", zap.Error(err))
		return err
	}

	e.logger.Info("ELB资源更新成功", zap.Int("id", req.ID))
	return nil
}

// validateCreateElbResourceReq 验证创建ELB资源请求参数
func validateCreateElbResourceReq(req *model.CreateElbResourceReq) error {
	if req.InstanceName == "" {
		return errors.New("实例名称不能为空")
	}

	if req.Provider == "" {
		return errors.New("云提供商不能为空")
	}

	if req.RegionId == "" {
		return errors.New("区域ID不能为空")
	}

	if req.VpcId == "" {
		return errors.New("VPC ID不能为空")
	}

	if req.LoadBalancerType == "" {
		return errors.New("负载均衡器类型不能为空")
	}

	if req.AddressType == "" {
		return errors.New("地址类型不能为空")
	}

	if req.TreeNodeID <= 0 {
		return errors.New("服务树节点ID无效")
	}

	if req.BandwidthCapacity <= 0 {
		return errors.New("带宽容量必须大于0")
	}

	return nil
}
