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
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/dao"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"go.uber.org/zap"
)

type TreeRdsService interface {
	ListRdsResources(ctx context.Context, req *model.ListRdsResourcesReq) (model.ListResp[*model.ResourceRds], error)
	GetRdsDetail(ctx context.Context, req *model.GetRdsDetailReq) (*model.ResourceRds, error)
	CreateRdsResource(ctx context.Context, req *model.CreateRdsResourceReq) error
	DeleteRds(ctx context.Context, req *model.DeleteRdsReq) error
	StartRds(ctx context.Context, req *model.StartRdsReq) error
	StopRds(ctx context.Context, req *model.StopRdsReq) error
	RestartRds(ctx context.Context, req *model.RestartRdsReq) error
	UpdateRds(ctx context.Context, req *model.UpdateRdsReq) error
	ResizeRds(ctx context.Context, req *model.ResizeRdsReq) error
	BackupRds(ctx context.Context, req *model.BackupRdsReq) error
	RestoreRds(ctx context.Context, req *model.RestoreRdsReq) error
	ResetRdsPassword(ctx context.Context, req *model.ResetRdsPasswordReq) error
	RenewRds(ctx context.Context, req *model.RenewRdsReq) error
}

type treeRdsService struct {
	logger *zap.Logger
	dao    dao.TreeRdsDAO
}

func NewTreeRdsService(logger *zap.Logger, dao dao.TreeRdsDAO) TreeRdsService {
	return &treeRdsService{
		logger: logger,
		dao:    dao,
	}
}

// BackupRds RDS实例备份
func (t *treeRdsService) BackupRds(ctx context.Context, req *model.BackupRdsReq) error {
	// 获取RDS实例信息
	resource, err := t.dao.GetRdsResourceById(ctx, req.ID)
	if err != nil {
		t.logger.Error("获取RDS实例失败", zap.Error(err))
		return err
	}

	// 创建备份记录
	if err := t.dao.BackupRdsInstance(ctx, resource.InstanceId, req.BackupName); err != nil {
		t.logger.Error("创建RDS备份失败", zap.Error(err))
		return err
	}

	t.logger.Info("RDS实例备份成功", zap.Int("id", req.ID), zap.String("backupName", req.BackupName))
	return nil
}

// CreateRdsResource 创建RDS资源
func (t *treeRdsService) CreateRdsResource(ctx context.Context, req *model.CreateRdsResourceReq) error {
	// 验证请求参数
	if err := validateCreateRdsResourceReq(req); err != nil {
		t.logger.Error("创建RDS资源参数验证失败", zap.Error(err))
		return err
	}

	// 转换Tags格式
	tagsList := make([]string, 0, len(req.Tags))
	for k, v := range req.Tags {
		tagsList = append(tagsList, k+":"+v)
	}

	// 构建RDS资源对象
	resource := &model.ResourceRds{
		InstanceName:        req.InstanceName,
		Provider:            req.Provider,
		RegionId:            req.Region,
		ZoneId:              req.ZoneId,
		Engine:              req.Engine,
		EngineVersion:       req.EngineVersion,
		DBInstanceClass:     req.DBInstanceClass,
		VpcId:               req.VpcId,
		DBInstanceNetType:   req.DBInstanceNetType,
		InstanceChargeType:  req.InstanceChargeType,
		TreeNodeID:          req.TreeNodeId,
		Description:         req.Description,
		Tags:                tagsList,
		SecurityGroupIds:    req.SecurityGroupIds,
		AllocatedStorage:    req.AllocatedStorage,
		BackupRetentionDays: req.BackupRetentionDays,
		PreferredBackupTime: req.PreferredBackupTime,
		MaintenanceWindow:   req.MaintenanceWindow,
		Env:                 req.Environment,
		Status:              "Creating",
		CreationTime:        time.Now().Format(time.RFC3339),
		LastSyncTime:        time.Now(),
		Port:                3306, // 默认MySQL端口
		DBInstanceType:      "Primary",
		DBStatus:            "Creating",
	}

	// 创建RDS资源
	if err := t.dao.CreateRdsResource(ctx, resource); err != nil {
		t.logger.Error("创建RDS资源失败", zap.Error(err))
		return err
	}

	t.logger.Info("RDS资源创建成功", zap.String("instanceName", req.InstanceName))
	return nil
}

// DeleteRds 删除RDS实例
func (t *treeRdsService) DeleteRds(ctx context.Context, req *model.DeleteRdsReq) error {
	// 获取RDS实例信息
	resource, err := t.dao.GetRdsResourceById(ctx, req.ID)
	if err != nil {
		t.logger.Error("获取RDS实例失败", zap.Error(err))
		return err
	}

	// 检查是否可以删除
	if !req.ForceDelete && resource.Status == "Running" {
		t.logger.Error("RDS正在运行中，不能删除", zap.Int("id", req.ID))
		return errors.New("RDS正在运行中，请先停止后再删除，或使用强制删除")
	}

	// 删除RDS资源
	if err := t.dao.DeleteRdsResource(ctx, req.ID); err != nil {
		t.logger.Error("删除RDS资源失败", zap.Error(err))
		return err
	}

	t.logger.Info("RDS资源删除成功", zap.Int("id", req.ID))
	return nil
}

// GetRdsDetail 获取RDS详情
func (t *treeRdsService) GetRdsDetail(ctx context.Context, req *model.GetRdsDetailReq) (*model.ResourceRds, error) {
	resource, err := t.dao.GetRdsResourceById(ctx, req.ID)
	if err != nil {
		t.logger.Error("获取RDS详情失败", zap.Error(err), zap.Int("id", req.ID))
		return nil, err
	}

	return resource, nil
}

// ListRdsResources 获取RDS资源列表
func (t *treeRdsService) ListRdsResources(ctx context.Context, req *model.ListRdsResourcesReq) (model.ListResp[*model.ResourceRds], error) {
	result, err := t.dao.ListRdsResources(ctx, req)
	if err != nil {
		t.logger.Error("获取RDS资源列表失败", zap.Error(err))
		return model.ListResp[*model.ResourceRds]{}, err
	}

	return result, nil
}

// RenewRds 续费RDS实例
func (t *treeRdsService) RenewRds(ctx context.Context, req *model.RenewRdsReq) error {
	// 获取RDS实例信息
	resource, err := t.dao.GetRdsResourceById(ctx, req.ID)
	if err != nil {
		t.logger.Error("获取RDS实例失败", zap.Error(err))
		return err
	}

	// 续费逻辑（这里可以调用云厂商API或更新本地记录）
	// 更新续费信息到数据库
	if err := t.dao.RenewRdsInstance(ctx, resource.InstanceId, req.Period, req.PeriodUnit); err != nil {
		t.logger.Error("续费RDS实例失败", zap.Error(err))
		return err
	}

	t.logger.Info("RDS实例续费成功", zap.Int("id", req.ID), zap.Int("period", req.Period))
	return nil
}

// ResetRdsPassword 重置RDS实例密码
func (t *treeRdsService) ResetRdsPassword(ctx context.Context, req *model.ResetRdsPasswordReq) error {
	// 获取RDS实例信息
	resource, err := t.dao.GetRdsResourceById(ctx, req.ID)
	if err != nil {
		t.logger.Error("获取RDS实例失败", zap.Error(err))
		return err
	}

	// 加密新密码
	encryptedPassword := utils.Base64EncryptWithMagic(req.NewPassword)

	// 重置密码
	if err := t.dao.ResetRdsPassword(ctx, resource.InstanceId, req.Username, encryptedPassword); err != nil {
		t.logger.Error("重置RDS密码失败", zap.Error(err))
		return err
	}

	t.logger.Info("RDS密码重置成功", zap.Int("id", req.ID), zap.String("username", req.Username))
	return nil
}

// ResizeRds 调整RDS实例规格
func (t *treeRdsService) ResizeRds(ctx context.Context, req *model.ResizeRdsReq) error {
	// 获取RDS实例信息
	resource, err := t.dao.GetRdsResourceById(ctx, req.ID)
	if err != nil {
		t.logger.Error("获取RDS实例失败", zap.Error(err))
		return err
	}

	// 调整实例规格
	if err := t.dao.ResizeRdsInstance(ctx, resource.InstanceId, req.DBInstanceClass, req.AllocatedStorage); err != nil {
		t.logger.Error("调整RDS规格失败", zap.Error(err))
		return err
	}

	t.logger.Info("RDS规格调整成功", zap.Int("id", req.ID), zap.String("newClass", req.DBInstanceClass))
	return nil
}

// RestartRds 重启RDS实例
func (t *treeRdsService) RestartRds(ctx context.Context, req *model.RestartRdsReq) error {
	// 获取RDS实例信息
	resource, err := t.dao.GetRdsResourceById(ctx, req.ID)
	if err != nil {
		t.logger.Error("获取RDS实例失败", zap.Error(err))
		return err
	}

	// 重启RDS实例
	if err := t.dao.RestartRdsInstance(ctx, resource.InstanceId); err != nil {
		t.logger.Error("重启RDS实例失败", zap.Error(err))
		return err
	}

	t.logger.Info("RDS实例重启成功", zap.Int("id", req.ID))
	return nil
}

// RestoreRds 恢复RDS实例
func (t *treeRdsService) RestoreRds(ctx context.Context, req *model.RestoreRdsReq) error {
	// 获取RDS实例信息
	resource, err := t.dao.GetRdsResourceById(ctx, req.ID)
	if err != nil {
		t.logger.Error("获取RDS实例失败", zap.Error(err))
		return err
	}

	// 恢复RDS实例
	if err := t.dao.RestoreRdsInstance(ctx, resource.InstanceId, req.BackupId, req.RestoreTime); err != nil {
		t.logger.Error("恢复RDS实例失败", zap.Error(err))
		return err
	}

	t.logger.Info("RDS实例恢复成功", zap.Int("id", req.ID), zap.String("backupId", req.BackupId))
	return nil
}

// StartRds 启动RDS实例
func (t *treeRdsService) StartRds(ctx context.Context, req *model.StartRdsReq) error {
	// 获取RDS实例信息
	resource, err := t.dao.GetRdsResourceById(ctx, req.ID)
	if err != nil {
		t.logger.Error("获取RDS实例失败", zap.Error(err))
		return err
	}

	// 启动RDS实例
	if err := t.dao.StartRdsInstance(ctx, resource.InstanceId); err != nil {
		t.logger.Error("启动RDS实例失败", zap.Error(err))
		return err
	}

	t.logger.Info("RDS实例启动成功", zap.Int("id", req.ID))
	return nil
}

// StopRds 停止RDS实例
func (t *treeRdsService) StopRds(ctx context.Context, req *model.StopRdsReq) error {
	// 获取RDS实例信息
	resource, err := t.dao.GetRdsResourceById(ctx, req.ID)
	if err != nil {
		t.logger.Error("获取RDS实例失败", zap.Error(err))
		return err
	}

	// 停止RDS实例
	if err := t.dao.StopRdsInstance(ctx, resource.InstanceId); err != nil {
		t.logger.Error("停止RDS实例失败", zap.Error(err))
		return err
	}

	t.logger.Info("RDS实例停止成功", zap.Int("id", req.ID))
	return nil
}

// UpdateRds 更新RDS实例
func (t *treeRdsService) UpdateRds(ctx context.Context, req *model.UpdateRdsReq) error {
	// 获取RDS实例信息
	resource, err := t.dao.GetRdsResourceById(ctx, req.ID)
	if err != nil {
		t.logger.Error("获取RDS实例失败", zap.Error(err))
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
		// 转换Tags格式
		tagsList := make([]string, 0, len(req.Tags))
		for k, v := range req.Tags {
			tagsList = append(tagsList, k+":"+v)
		}
		resource.Tags = tagsList
	}
	if req.BackupRetentionDays > 0 {
		resource.BackupRetentionDays = req.BackupRetentionDays
	}
	if req.PreferredBackupTime != "" {
		resource.PreferredBackupTime = req.PreferredBackupTime
	}
	if req.MaintenanceWindow != "" {
		resource.MaintenanceWindow = req.MaintenanceWindow
	}
	if req.TreeNodeId > 0 {
		resource.TreeNodeID = req.TreeNodeId
	}

	// 更新RDS资源
	if err := t.dao.UpdateRdsResource(ctx, resource); err != nil {
		t.logger.Error("更新RDS资源失败", zap.Error(err))
		return err
	}

	t.logger.Info("RDS资源更新成功", zap.Int("id", req.ID))
	return nil
}

// validateCreateRdsResourceReq 验证创建RDS资源请求参数
func validateCreateRdsResourceReq(req *model.CreateRdsResourceReq) error {
	if req.InstanceName == "" {
		return errors.New("实例名称不能为空")
	}

	if req.Provider == "" {
		return errors.New("云提供商不能为空")
	}

	if req.Region == "" {
		return errors.New("区域不能为空")
	}

	if req.Engine == "" {
		return errors.New("数据库引擎不能为空")
	}

	if req.EngineVersion == "" {
		return errors.New("数据库版本不能为空")
	}

	if req.DBInstanceClass == "" {
		return errors.New("实例规格不能为空")
	}

	if req.VpcId == "" {
		return errors.New("VPC ID不能为空")
	}

	if req.TreeNodeId <= 0 {
		return errors.New("服务树节点ID无效")
	}

	if req.AllocatedStorage < 20 {
		return errors.New("分配存储空间不能小于20GB")
	}

	if req.BackupRetentionDays < 1 || req.BackupRetentionDays > 30 {
		return errors.New("备份保留天数必须在1-30天之间")
	}

	return nil
}
