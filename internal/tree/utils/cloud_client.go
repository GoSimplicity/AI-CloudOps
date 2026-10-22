package utils

import (
	"context"
	"fmt"
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
)

// CloudComputeClient is a read-only provider boundary. A failed listing must
// return no partial snapshot, because full sync reconciles absent local assets.
type CloudComputeClient interface {
	VerifyCredentials(context.Context) error
	GetAvailableRegions(context.Context) ([]model.AvailableRegion, error)
	ListInstances(context.Context, []string) ([]*model.TreeCloudResource, error)
}
type CloudClientFactory func(model.CloudProvider, string, string, string, *zap.Logger) (CloudComputeClient, error)

func NewCloudComputeClient(provider model.CloudProvider, accessKey, secretKey, region string, logger *zap.Logger) (CloudComputeClient, error) {
	switch provider {
	case model.ProviderAliyun:
		return NewAliyunClient(accessKey, secretKey, region, logger)
	case model.ProviderAWS:
		return NewAWSClient(accessKey, secretKey, region)
	case model.ProviderTencent:
		return NewTencentClient(accessKey, secretKey, region)
	default:
		return nil, fmt.Errorf("云厂商 %d 暂未实现计算资源API", provider)
	}
}
func (c *AliyunClient) ListInstances(ctx context.Context, ids []string) ([]*model.TreeCloudResource, error) {
	return c.ListECSInstances(ctx, ids)
}

func instanceIDBatches(ids []string, size int) ([][]string, error) {
	if len(ids) == 0 {
		return [][]string{nil}, nil
	}
	unique := make([]string, 0, len(ids))
	seen := make(map[string]bool)
	for _, id := range ids {
		if strings.TrimSpace(id) == "" {
			return nil, fmt.Errorf("实例ID不能为空")
		}
		if !seen[id] {
			unique = append(unique, id)
			seen[id] = true
		}
	}
	batches := make([][]string, 0)
	for start := 0; start < len(unique); start += size {
		end := start + size
		if end > len(unique) {
			end = len(unique)
		}
		batches = append(batches, unique[start:end])
	}
	return batches, nil
}
