package service

import (
	"context"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/provider"
	"go.uber.org/zap"
)

type TreeSecurityGroupService interface {
	CreateSecurityGroup(ctx context.Context, req *model.CreateSecurityGroupReq) error
	DeleteSecurityGroup(ctx context.Context, req *model.DeleteSecurityGroupReq) error
	ListSecurityGroups(ctx context.Context, req *model.ListSecurityGroupsReq) (*model.ResourceSecurityGroupListResp, error)
	GetSecurityGroupDetail(ctx context.Context, req *model.GetSecurityGroupDetailReq) (*model.ResourceSecurityGroup, error)
}

type treeSecurityGroupService struct {
	providerFactory *provider.ProviderFactory
	logger          *zap.Logger
}

func NewTreeSecurityGroupService(providerFactory *provider.ProviderFactory, logger *zap.Logger) TreeSecurityGroupService {
	return &treeSecurityGroupService{
		providerFactory: providerFactory,
		logger:          logger,
	}
}

// CreateSecurityGroup 创建安全组
func (s *treeSecurityGroupService) CreateSecurityGroup(ctx context.Context, req *model.CreateSecurityGroupReq) error {
	cloudProvider, err := s.providerFactory.GetProvider(req.Provider)
	if err != nil {
		s.logger.Error("获取云提供商失败", zap.Error(err), zap.String("provider", string(req.Provider)))
		return err
	}

	err = cloudProvider.CreateSecurityGroup(ctx, req.Region, req)
	if err != nil {
		s.logger.Error("创建安全组失败", zap.Error(err), zap.Any("req", req))
		return err
	}

	s.logger.Info("创建安全组成功", zap.Any("req", req))
	return nil
}

// DeleteSecurityGroup 删除安全组
func (s *treeSecurityGroupService) DeleteSecurityGroup(ctx context.Context, req *model.DeleteSecurityGroupReq) error {
	cloudProvider, err := s.providerFactory.GetProvider(req.Provider)
	if err != nil {
		s.logger.Error("获取云提供商失败", zap.Error(err), zap.String("provider", string(req.Provider)))
		return err
	}

	err = cloudProvider.DeleteSecurityGroup(ctx, req.Region, req.SecurityGroupId)
	if err != nil {
		s.logger.Error("删除安全组失败", zap.Error(err), zap.Any("req", req))
		return err
	}

	s.logger.Info("删除安全组成功", zap.Any("req", req))
	return nil
}

// GetSecurityGroupDetail 获取安全组详情
func (s *treeSecurityGroupService) GetSecurityGroupDetail(ctx context.Context, req *model.GetSecurityGroupDetailReq) (*model.ResourceSecurityGroup, error) {
	cloudProvider, err := s.providerFactory.GetProvider(req.Provider)
	if err != nil {
		s.logger.Error("获取云提供商失败", zap.Error(err), zap.String("provider", string(req.Provider)))
		return nil, err
	}

	securityGroup, err := cloudProvider.GetSecurityGroupDetail(ctx, req.Region, req.SecurityGroupId)
	if err != nil {
		s.logger.Error("获取安全组详情失败", zap.Error(err), zap.Any("req", req))
		return nil, err
	}

	s.logger.Info("获取安全组详情成功", zap.Any("req", req), zap.String("securityGroupID", securityGroup.InstanceId))
	return securityGroup, nil
}

// ListSecurityGroups 获取安全组列表
func (s *treeSecurityGroupService) ListSecurityGroups(ctx context.Context, req *model.ListSecurityGroupsReq) (*model.ResourceSecurityGroupListResp, error) {
	cloudProvider, err := s.providerFactory.GetProvider(req.Provider)
	if err != nil {
		s.logger.Error("获取云提供商失败", zap.Error(err), zap.String("provider", string(req.Provider)))
		return nil, err
	}

	securityGroups, total, err := cloudProvider.ListSecurityGroups(ctx, req.Region, req.PageNumber, req.PageSize)
	if err != nil {
		s.logger.Error("获取安全组列表失败", zap.Error(err), zap.Any("req", req))
		return nil, err
	}

	s.logger.Info("获取安全组列表成功", zap.Any("req", req), zap.Int("count", len(securityGroups)), zap.Int64("total", total))
	return &model.ResourceSecurityGroupListResp{
		Total: total,
		Data:  securityGroups,
	}, nil
}
