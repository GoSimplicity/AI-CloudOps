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
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/imdario/mergo"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type TreeLocalService interface {
	GetTreeLocalList(ctx context.Context, req *model.GetTreeLocalListReq) (model.ListResp[*model.TreeLocal], error)
	GetTreeLocalDetail(ctx context.Context, req *model.GetTreeLocalDetailReq) (*model.TreeLocal, error)
	GetTreeLocalForConnection(ctx context.Context, req *model.GetTreeLocalDetailReq) (*model.TreeLocal, error)
	CreateTreeLocal(ctx context.Context, req *model.CreateTreeLocalReq) error
	UpdateTreeLocal(ctx context.Context, req *model.UpdateTreeLocalReq) error
	DeleteTreeLocal(ctx context.Context, req *model.DeleteTreeLocalReq) error
	BindTreeLocal(ctx context.Context, req *model.BindLocalResourceReq) error
	UnBindLocalResource(ctx context.Context, req *model.UnBindLocalResourceReq) error
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

// GetTreeLocalForConnection 获取用于连接的本地主机详情(包含解密后的密码)
func (t *treeLocalService) GetTreeLocalForConnection(ctx context.Context, req *model.GetTreeLocalDetailReq) (*model.TreeLocal, error) {
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

	// 解密密码以供连接使用
	if local.AuthMode == model.AuthModePassword && local.Password != "" {
		plainPassword, err := t.decryptPassword(local.Password)
		if err != nil {
			t.logger.Error("密码解密失败", zap.Int("id", req.ID), zap.Error(err))
			return nil, fmt.Errorf("密码解密失败: %w", err)
		}
		local.Password = plainPassword
	}

	return local, nil
}
func (t *treeLocalService) CreateTreeLocal(ctx context.Context, req *model.CreateTreeLocalReq) error {
	// 检查IP地址是否已存在
	existing, err := t.dao.GetByIP(ctx, req.IpAddr)
	if err == nil && existing != nil {
		return fmt.Errorf("IP地址 %s 已存在", req.IpAddr)
	}

	// 创建本地主机对象
	local := &model.TreeLocal{
		Name:        req.Name,
		Status:      model.StatusStarting,
		Environment: req.Environment,
		Description: req.Description,
		Tags:        req.Tags,
		IpAddr:      req.IpAddr,
		Port:        req.Port,
		Username:    req.Username,
		Key:         req.Key,
		AuthMode:    req.AuthMode,
		OsType:      req.OsType,
		OSName:      req.OSName,
		ImageName:   req.ImageName,
	}

	// 设置默认值
	if local.Port == 0 {
		local.Port = 22
	}

	if local.Username == "" {
		local.Username = "root"
	}

	// 加密
	if local.AuthMode == model.AuthModePassword && req.Password != "" {
		encryptedPassword, err := t.encryptPassword(req.Password)
		if err != nil {
			t.logger.Error("密码加密失败", zap.Error(err))
			return fmt.Errorf("密码加密失败: %w", err)
		}
		local.Password = encryptedPassword
	}

	if err := t.dao.Create(ctx, local); err != nil {
		t.logger.Error("创建本地主机失败", zap.Error(err))
		return err
	}

	return nil
}

func (t *treeLocalService) UpdateTreeLocal(ctx context.Context, req *model.UpdateTreeLocalReq) error {
	if req.ID <= 0 {
		return errors.New("无效的主机ID")
	}

	// 检查是否存在
	host, err := t.dao.GetByID(ctx, req.ID)
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return errors.New("本地主机不存在")
	case err != nil:
		t.logger.Error("获取本地主机失败", zap.Int("id", req.ID), zap.Error(err))
		return err
	}

	// 检查 IP 冲突
	if req.IpAddr != "" && req.IpAddr != host.IpAddr {
		if h, _ := t.dao.GetByIP(ctx, req.IpAddr); h != nil && h.ID != req.ID {
			t.logger.Error("IP 已被占用", zap.String("ip", req.IpAddr), zap.Int("existing_id", h.ID))
			return fmt.Errorf("IP %s 已被其他主机使用", req.IpAddr)
		}
	}

	local := model.TreeLocal{
		Model: model.Model{
			ID: req.ID,
		},
		Name:        req.Name,
		Environment: req.Environment,
		Description: req.Description,
		Tags:        req.Tags,
		Status:      model.StatusStarting,
		IpAddr:      req.IpAddr,
		Port:        req.Port,
		OsType:      req.OsType,
		OSName:      req.OSName,
		ImageName:   req.ImageName,
		AuthMode:    req.AuthMode,
	}

	// 加密密码
	if req.AuthMode == model.AuthModePassword && req.Password != "" {
		pwd, err := t.encryptPassword(req.Password)
		if err != nil {
			t.logger.Error("密码加密失败", zap.Error(err))
			return fmt.Errorf("密码加密失败: %w", err)
		}
		local.Password = pwd
	}

	// 如果是密钥认证，直接使用提供的密钥
	if req.AuthMode == model.AuthModeKey && req.Key != "" {
		local.Key = req.Key
	}

	// 合并更新字段
	if err := mergo.Merge(host, &local, mergo.WithOverride); err != nil {
		return fmt.Errorf("合并字段失败: %w", err)
	}

	if err := t.dao.Update(ctx, host); err != nil {
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

func (t *treeLocalService) BindTreeLocal(ctx context.Context, req *model.BindLocalResourceReq) error {
	if req.ID <= 0 {
		return errors.New("无效的主机ID")
	}

	if err := t.dao.BindTreeNodes(ctx, req.ID, req.TreeNodeIDs); err != nil {
		t.logger.Error("绑定主机失败", zap.Int("id", req.ID), zap.Error(err))
		return err
	}

	return nil
}

func (t *treeLocalService) UnBindLocalResource(ctx context.Context, req *model.UnBindLocalResourceReq) error {
	if req.ID <= 0 {
		return errors.New("无效的主机ID")
	}

	if err := t.dao.UnBindTreeNodes(ctx, req.ID, req.TreeNodeIDs); err != nil {
		t.logger.Error("解绑主机失败", zap.Int("id", req.ID), zap.Error(err))
		return err
	}

	return nil
}

// encryptPassword 加密密码
func (t *treeLocalService) encryptPassword(password string) (string, error) {
	if password == "" {
		return "", nil
	}

	encryptionKey := viper.GetString("tree.password_encryption_key")
	if encryptionKey == "" {
		return "", errors.New("未配置密码加密密钥")
	}
	if len(encryptionKey) != 32 {
		return "", errors.New("密码加密密钥长度必须为32字节")
	}

	return utils.EncryptSecretKey(password, []byte(encryptionKey))
}

// decryptPassword 解密密码
func (t *treeLocalService) decryptPassword(encryptedPassword string) (string, error) {
	if encryptedPassword == "" {
		return "", nil
	}

	encryptionKey := viper.GetString("tree.password_encryption_key")
	if encryptionKey == "" {
		return "", errors.New("未配置密码加密密钥")
	}
	if len(encryptionKey) != 32 {
		return "", errors.New("密码加密密钥长度必须为32字节")
	}

	return utils.DecryptSecretKey(encryptedPassword, []byte(encryptionKey))
}
