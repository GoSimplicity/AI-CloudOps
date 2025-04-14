package service

import (
	"context"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/provider"
	"go.uber.org/zap"
)

type VpcService interface {
	GetVpcResourceById(ctx context.Context, id int) (*model.ResourceVpc, error)
	CreateVpcResource(ctx context.Context, req *model.VpcCreationParams) error
	DeleteVpcResource(ctx context.Context, id int) error
}

type vpcService struct {
	logger *zap.Logger
	dao    dao.VpcDAO
	AliyunProvider provider.AliyunProvider
	TencentProvider provider.TencentProvider
	HuaweiProvider provider.HuaweiProvider
	AWSProvider provider.AwsProvider
	AzureProvider provider.AzureProvider
	GCPProvider provider.GcpProvider
}

func NewVpcService(logger *zap.Logger, dao dao.VpcDAO, AliyunProvider provider.AliyunProvider, TencentProvider provider.TencentProvider, HuaweiProvider provider.HuaweiProvider, AWSProvider provider.AwsProvider, AzureProvider provider.AzureProvider, GCPProvider provider.GcpProvider) VpcService {
	return &vpcService{
		logger: logger,
		dao:    dao,
		AliyunProvider: AliyunProvider,
		TencentProvider: TencentProvider,
		HuaweiProvider: HuaweiProvider,
		AWSProvider: AWSProvider,
		AzureProvider: AzureProvider,
		GCPProvider: GCPProvider,
	}
}

// CreateVpcResource 创建VPC资源
func (v *vpcService) CreateVpcResource(ctx context.Context, req *model.VpcCreationParams) error {
	if req.Provider == model.CloudProviderLocal {
		err := v.dao.CreateVpcResource(ctx, req)
		if err != nil {
			v.logger.Error("[CreateVpcResource] 创建VPC资源失败", zap.Error(err))
			return err
		}
		return nil
	}

	var err error
	switch req.Provider {
	case model.CloudProviderAliyun:
		err = v.AliyunProvider.CreateVPC(ctx, req.Region, req)
	case model.CloudProviderTencent:
		err = v.TencentProvider.CreateVPC(ctx, req.Region, req)
	case model.CloudProviderHuawei:
		err = v.HuaweiProvider.CreateVPC(ctx, req.Region, req)
	case model.CloudProviderAWS:
		err = v.AWSProvider.CreateVPC(ctx, req.Region, req)
	case model.CloudProviderAzure:
		err = v.AzureProvider.CreateVPC(ctx, req.Region, req)
	case model.CloudProviderGCP:
		err = v.GCPProvider.CreateVPC(ctx, req.Region, req)
	default:
		v.logger.Error("[CreateVpcResource] 不支持的云厂商", zap.String("provider", string(req.Provider)))
		return err
	}

	if err != nil {
		v.logger.Error("[CreateVpcResource] 创建VPC资源失败", zap.Error(err))
		return err
	}

	return nil
}

// DeleteVpcResource 删除VPC资源
func (v *vpcService) DeleteVpcResource(ctx context.Context, id int) error {
	// 先获取VPC资源信息
	vpc, err := v.dao.GetVpcResourceById(ctx, id)
	if err != nil {
		v.logger.Error("[DeleteVpcResource] 获取VPC资源信息失败", zap.Error(err))
		return err
	}

	if vpc.Provider == model.CloudProviderLocal {
		err := v.dao.DeleteVpcResource(ctx, id)
		if err != nil {
			v.logger.Error("[DeleteVpcResource] 删除VPC资源失败", zap.Error(err))
			return err
		}
		return nil
	}

	// 根据不同云厂商调用不同的删除接口
	switch vpc.Provider {
	case model.CloudProviderAliyun:
		err = v.AliyunProvider.DeleteVPC(ctx, vpc.Region, vpc.VpcId)
	case model.CloudProviderTencent:
		err = v.TencentProvider.DeleteVPC(ctx, vpc.Region, vpc.VpcId)
	case model.CloudProviderHuawei:
		err = v.HuaweiProvider.DeleteVPC(ctx, vpc.Region, vpc.VpcId)
	case model.CloudProviderAWS:
		err = v.AWSProvider.DeleteVPC(ctx, vpc.Region, vpc.VpcId)
	case model.CloudProviderAzure:
		err = v.AzureProvider.DeleteVPC(ctx, vpc.Region, vpc.VpcId)
	case model.CloudProviderGCP:
		err = v.GCPProvider.DeleteVPC(ctx, vpc.Region, vpc.VpcId)
	default:
		v.logger.Error("[DeleteVpcResource] 不支持的云厂商", zap.String("provider", string(vpc.Provider)))
		return err
	}

	if err != nil {
		v.logger.Error("[DeleteVpcResource] 删除VPC资源失败", zap.Error(err))
		return err
	}

	// 删除数据库中的记录
	err = v.dao.DeleteVpcResource(ctx, id)
	if err != nil {
		v.logger.Error("[DeleteVpcResource] 删除数据库VPC记录失败", zap.Error(err))
		return err
	}

	return nil
}

// GetVpcResourceById 根据ID获取VPC资源
func (v *vpcService) GetVpcResourceById(ctx context.Context, id int) (*model.ResourceVpc, error) {
	vpc, err := v.dao.GetVpcResourceById(ctx, id)
	if err != nil {
		v.logger.Error("[GetVpcResourceById] 获取VPC资源失败", zap.Error(err))
		return nil, err
	}
	return vpc, nil
}
