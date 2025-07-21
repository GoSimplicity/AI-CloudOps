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
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/ssh"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type TreeLocalService interface {
	GetTreeLocalList(ctx context.Context, req *model.GetTreeLocalListReq) (model.ListResp[*model.TreeLocal], error)
	GetTreeLocalDetail(ctx context.Context, req *model.GetTreeLocalDetailReq) (*model.TreeLocal, error)
	CreateTreeLocal(ctx context.Context, req *model.CreateTreeLocalReq) error
	UpdateTreeLocal(ctx context.Context, req *model.UpdateTreeLocalReq) error
	DeleteTreeLocal(ctx context.Context, req *model.DeleteTreeLocalReq) error
	BatchDeleteTreeLocal(ctx context.Context, ids []int) error
	UpdateTreeLocalStatus(ctx context.Context, id int, status string) error
	GetTreeLocalByIP(ctx context.Context, ip string) (*model.TreeLocal, error)
}

type treeLocalService struct {
	logger *zap.Logger
	dao    dao.TreeLocalDAO
}

func NewTreeLocalService(logger *zap.Logger, dao dao.TreeLocalDAO) TreeLocalService {
	return &treeLocalService{
		logger: logger,
		dao:    dao,
	}
}

// GetTreeLocalList 获取本地主机列表
func (t *treeLocalService) GetTreeLocalList(ctx context.Context, req *model.GetTreeLocalListReq) (model.ListResp[*model.TreeLocal], error) {
	locals, total, err := t.dao.GetList(ctx, req)
	if err != nil {
		t.logger.Error("获取本地主机列表失败", zap.Error(err))
		return model.ListResp[*model.TreeLocal]{}, err
	}

	return model.ListResp[*model.TreeLocal]{
		Items: locals,
		Total: total,
	}, nil
}

// GetTreeLocalDetail 获取本地主机详情
func (t *treeLocalService) GetTreeLocalDetail(ctx context.Context, req *model.GetTreeLocalDetailReq) (*model.TreeLocal, error) {
	if req.ID <= 0 {
		return nil, errors.New("无效的主机ID")
	}

	local, err := t.dao.GetByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("本地主机不存在")
		}
		t.logger.Error("获取本地主机详情失败", zap.Int("id", req.ID), zap.Error(err))
		return nil, err
	}

	return local, nil
}

// CreateTreeLocal 创建本地主机
func (t *treeLocalService) CreateTreeLocal(ctx context.Context, req *model.CreateTreeLocalReq) error {
	// 数据验证
	if err := t.validateCreateReq(req); err != nil {
		return err
	}

	// 检查IP地址是否已存在
	existing, err := t.dao.GetByIP(ctx, req.IpAddr)
	if err == nil && existing != nil {
		return fmt.Errorf("IP地址 %s 已存在", req.IpAddr)
	}

	// 创建本地主机对象
	local := &model.TreeLocal{
		Name:        req.Name,
		Status:      "RUNNING",
		Environment: req.Environment,
		Description: req.Description,
		Tags:        req.Tags,
		Cpu:         0,
		Memory:      0,
		Disk:        0,
		IpAddr:      req.IpAddr,
		Port:        req.Port,
		HostName:    req.HostName,
		Username:    req.Username,
		Password:    req.Password,
		Key:         req.Key,
		AuthMode:    req.AuthMode,
		TreeNodeIDs: req.TreeNodeIDs,
	}

	// 设置默认值
	if local.Port == 0 {
		local.Port = 22
	}
	if local.AuthMode == "" {
		local.AuthMode = "password"
	}
	if local.Username == "" {
		local.Username = "root"
	}

	if err := t.dao.Create(ctx, local); err != nil {
		t.logger.Error("创建本地主机失败", zap.Error(err))
		return err
	}

	// 异步获取系统信息
	go t.collectSystemInfo(local.ID)

	return nil
}

// UpdateTreeLocal 更新本地主机
func (t *treeLocalService) UpdateTreeLocal(ctx context.Context, req *model.UpdateTreeLocalReq) error {
	if req.ID <= 0 {
		return errors.New("无效的主机ID")
	}

	// 检查主机是否存在
	existing, err := t.dao.GetByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("本地主机不存在")
		}
		t.logger.Error("获取本地主机失败", zap.Int("id", req.ID), zap.Error(err))
		return err
	}

	// 如果更新IP地址，检查新IP是否已存在
	if req.IpAddr != "" && req.IpAddr != existing.IpAddr {
		ipLocal, err := t.dao.GetByIP(ctx, req.IpAddr)
		if err == nil && ipLocal != nil && int(ipLocal.ID) != req.ID {
			return fmt.Errorf("IP地址 %s 已被其他主机使用", req.IpAddr)
		}
	}

	// 更新字段
	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.Environment != "" {
		existing.Environment = req.Environment
	}
	if req.Description != "" {
		existing.Description = req.Description
	}
	if req.Tags != nil {
		existing.Tags = req.Tags
	}
	if req.IpAddr != "" {
		existing.IpAddr = req.IpAddr
	}
	if req.Port != 0 {
		existing.Port = req.Port
	}
	if req.HostName != "" {
		existing.HostName = req.HostName
	}
	if req.Username != "" {
		existing.Username = req.Username
	}
	if req.Password != "" {
		existing.Password = req.Password
	}
	if req.Key != "" {
		existing.Key = req.Key
	}
	if req.AuthMode != "" {
		existing.AuthMode = req.AuthMode
	}
	if req.OsType != "" {
		existing.OsType = req.OsType
	}
	if req.OSName != "" {
		existing.OSName = req.OSName
	}
	if req.ImageName != "" {
		existing.ImageName = req.ImageName
	}
	if req.TreeNodeIDs != nil {
		existing.TreeNodeIDs = req.TreeNodeIDs
	}

	if err := t.dao.Update(ctx, existing); err != nil {
		t.logger.Error("更新本地主机失败", zap.Int("id", req.ID), zap.Error(err))
		return err
	}

	return nil
}

// DeleteTreeLocal 删除本地主机
func (t *treeLocalService) DeleteTreeLocal(ctx context.Context, req *model.DeleteTreeLocalReq) error {
	if req.ID <= 0 {
		return errors.New("无效的主机ID")
	}

	if err := t.dao.Delete(ctx, req.ID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("本地主机不存在")
		}
		t.logger.Error("删除本地主机失败", zap.Int("id", req.ID), zap.Error(err))
		return err
	}

	return nil
}

// BatchDeleteTreeLocal 批量删除本地主机
func (t *treeLocalService) BatchDeleteTreeLocal(ctx context.Context, ids []int) error {
	if len(ids) == 0 {
		return errors.New("主机ID列表不能为空")
	}

	if err := t.dao.BatchDelete(ctx, ids); err != nil {
		t.logger.Error("批量删除本地主机失败", zap.Ints("ids", ids), zap.Error(err))
		return err
	}

	return nil
}

// UpdateTreeLocalStatus 更新本地主机状态
func (t *treeLocalService) UpdateTreeLocalStatus(ctx context.Context, id int, status string) error {
	if id <= 0 {
		return errors.New("无效的主机ID")
	}

	validStatuses := map[string]bool{
		"RUNNING":    true,
		"STOPPED":    true,
		"STARTING":   true,
		"STOPPING":   true,
		"RESTARTING": true,
		"DELETING":   true,
		"ERROR":      true,
	}

	if !validStatuses[status] {
		return fmt.Errorf("无效的状态: %s", status)
	}

	if err := t.dao.UpdateStatus(ctx, id, status); err != nil {
		t.logger.Error("更新主机状态失败", zap.Int("id", id), zap.String("status", status), zap.Error(err))
		return err
	}

	return nil
}

// GetTreeLocalByIP 根据IP获取本地主机
func (t *treeLocalService) GetTreeLocalByIP(ctx context.Context, ip string) (*model.TreeLocal, error) {
	if ip == "" {
		return nil, errors.New("IP地址不能为空")
	}

	local, err := t.dao.GetByIP(ctx, ip)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("本地主机不存在")
		}
		t.logger.Error("根据IP获取本地主机失败", zap.String("ip", ip), zap.Error(err))
		return nil, err
	}

	return local, nil
}

// collectSystemInfo 收集系统信息
func (t *treeLocalService) collectSystemInfo(id int) {
	ctx := context.Background()

	// 获取主机信息
	local, err := t.dao.GetByID(ctx, id)
	if err != nil {
		t.logger.Error("获取主机信息失败", zap.Int("id", id), zap.Error(err))
		return
	}

	// 创建SSH客户端
	sshClient := ssh.NewSSH(t.logger)

	// 连接主机
	err = sshClient.Connect(local.IpAddr, local.Port, local.Username, local.Password, local.Key, local.AuthMode, 0)
	if err != nil {
		t.logger.Error("SSH连接失败", zap.String("ip", local.IpAddr), zap.Error(err))
		return
	}
	defer func() {
		if sshClient.Client != nil {
			sshClient.Client.Close()
		}
	}()

	// 根据操作系统类型获取系统信息
	var systemInfo *SystemInfo
	if local.OsType == "windows" {
		systemInfo, err = GetWindowsSystemInfo(ctx, sshClient)
	} else {
		systemInfo, err = GetSystemInfo(ctx, sshClient)
	}

	if err != nil {
		t.logger.Error("获取系统信息失败", zap.String("ip", local.IpAddr), zap.Error(err))
		return
	}

	// 更新主机信息
	local.Cpu = systemInfo.CPU
	local.Memory = systemInfo.Memory
	local.Disk = systemInfo.Disk
	local.OSName = systemInfo.OSName
	local.OsType = systemInfo.OSType

	if err := t.dao.Update(ctx, local); err != nil {
		t.logger.Error("更新主机信息失败", zap.Int("id", id), zap.Error(err))
		return
	}

	t.logger.Info("成功获取并更新系统信息",
		zap.Int("id", id),
		zap.Int("cpu", systemInfo.CPU),
		zap.Int("memory", systemInfo.Memory),
		zap.Int("disk", systemInfo.Disk))
}

// validateCreateReq 验证创建请求
func (t *treeLocalService) validateCreateReq(req *model.CreateTreeLocalReq) error {
	if req.Name == "" {
		return errors.New("主机名称不能为空")
	}
	if req.IpAddr == "" {
		return errors.New("IP地址不能为空")
	}
	if req.Port < 1 || req.Port > 65535 {
		return errors.New("端口号必须在1-65535之间")
	}
	return nil
}
