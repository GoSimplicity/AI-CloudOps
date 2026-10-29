package service

import (
	"errors"
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	treeUtils "github.com/GoSimplicity/AI-CloudOps/internal/tree/utils"
	"go.uber.org/zap"
)

func cloudClient(provider model.CloudProvider, credentials treeUtils.CloudCredentials, region *model.CloudAccountRegion, logger *zap.Logger, modern treeUtils.CloudCredentialClientFactory, legacy treeUtils.CloudClientFactory) (treeUtils.CloudComputeClient, error) {
	if err := treeUtils.ValidateCloudCredentials(provider, credentials, time.Now().UTC()); err != nil {
		return nil, err
	}
	if region == nil || strings.TrimSpace(region.Region) == "" {
		return nil, errors.New("云地域无效")
	}
	if region.ProjectID != nil {
		credentials.ProjectID = *region.ProjectID
	}
	if provider == model.ProviderHuawei && !treeUtils.ValidHuaweiProjectID(credentials.ProjectID) {
		return nil, errors.New("华为云地域缺少项目ID")
	}
	var client treeUtils.CloudComputeClient
	var err error
	if modern != nil {
		client, err = modern(provider, credentials, region.Region, logger)
	} else if legacy != nil {
		if credentials.CredentialType == "temporary" || provider == model.ProviderHuawei {
			return nil, errors.New("旧版云客户端不支持该凭据")
		}
		client, err = legacy(provider, credentials.AccessKey, credentials.SecretKey, region.Region, logger)
	} else {
		client, err = treeUtils.NewCloudComputeClientWithCredentials(provider, credentials, region.Region, logger)
	}
	if err != nil || client == nil {
		return nil, errors.New("云客户端初始化失败")
	}
	return client, nil
}

func defaultAccountRegion(regions []*model.CloudAccountRegion) (*model.CloudAccountRegion, error) {
	for _, region := range regions {
		if region != nil && region.IsDefault {
			return region, nil
		}
	}
	for _, region := range regions {
		if region != nil {
			return region, nil
		}
	}
	return nil, errors.New("云账户没有配置区域信息")
}
