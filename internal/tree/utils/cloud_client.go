package utils

import (
	"context"
	"fmt"
	"strings"
	"time"

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
type CloudCredentialClientFactory func(model.CloudProvider, CloudCredentials, string, *zap.Logger) (CloudComputeClient, error)

// CloudCredentials 仅供内部传递；不可序列化或以格式化输出泄露。
type CloudCredentials struct {
	AccessKey             string     `json:"-"`
	SecretKey             string     `json:"-"`
	SessionToken          string     `json:"-"`
	ExpiresAt             *time.Time `json:"-"`
	CredentialType        string     `json:"-"`
	ProjectID             string     `json:"-"`
	AzureTenantID         string     `json:"-"`
	AzureClientID         string     `json:"-"`
	AzureClientSecret     string     `json:"-"`
	AzureSubscriptionID   string     `json:"-"`
	GCPProjectID          string     `json:"-"`
	GCPServiceAccountJSON string     `json:"-"`
}

func (CloudCredentials) String() string   { return "[REDACTED]" }
func (CloudCredentials) GoString() string { return "[REDACTED]" }

func ValidateCloudCredentials(provider model.CloudProvider, credentials CloudCredentials, now time.Time) error {
	if provider == model.ProviderAzure || provider == model.ProviderGCP {
		if credentials.CredentialType != "" && credentials.CredentialType != "long_term" || credentials.AccessKey != "" || credentials.SecretKey != "" || credentials.SessionToken != "" || credentials.ExpiresAt != nil {
			return fmt.Errorf("云账户凭据类型无效")
		}
		if provider == model.ProviderAzure {
			if credentials.GCPProjectID != "" || credentials.GCPServiceAccountJSON != "" {
				return fmt.Errorf("云账户凭据类型无效")
			}
			_, err := NewAzureClient(credentials.AzureTenantID, credentials.AzureClientID, credentials.AzureClientSecret, credentials.AzureSubscriptionID, "eastus")
			return err
		}
		if credentials.AzureTenantID != "" || credentials.AzureClientID != "" || credentials.AzureClientSecret != "" || credentials.AzureSubscriptionID != "" {
			return fmt.Errorf("云账户凭据类型无效")
		}
		_, err := NewGCPClient([]byte(credentials.GCPServiceAccountJSON), credentials.GCPProjectID, "us-central1-a")
		return err
	}
	if credentials.AzureTenantID != "" || credentials.AzureClientID != "" || credentials.AzureClientSecret != "" || credentials.AzureSubscriptionID != "" || credentials.GCPProjectID != "" || credentials.GCPServiceAccountJSON != "" {
		return fmt.Errorf("云账户凭据类型无效")
	}
	if strings.TrimSpace(credentials.AccessKey) == "" || strings.TrimSpace(credentials.SecretKey) == "" || len(credentials.AccessKey) > 500 || len(credentials.SecretKey) > 500 {
		return fmt.Errorf("云账户访问凭据无效")
	}
	switch credentials.CredentialType {
	case "", "long_term":
		if credentials.SessionToken != "" || credentials.ExpiresAt != nil || (provider == model.ProviderAWS && strings.HasPrefix(credentials.AccessKey, "ASIA")) {
			return fmt.Errorf("长期凭据不能包含临时令牌或到期时间")
		}
	case "temporary":
		if provider != model.ProviderAWS && provider != model.ProviderTencent {
			return fmt.Errorf("该云厂商暂不支持临时凭据")
		}
		if strings.TrimSpace(credentials.SessionToken) == "" || len(credentials.SessionToken) > 8192 || credentials.ExpiresAt == nil {
			return fmt.Errorf("临时凭据必须包含令牌和到期时间")
		}
		_, offset := credentials.ExpiresAt.Zone()
		if offset != 0 || !credentials.ExpiresAt.After(now) {
			return fmt.Errorf("临时凭据已过期或到期时间不是UTC")
		}
	default:
		return fmt.Errorf("未知的云凭据类型")
	}
	return nil
}

func NewCloudComputeClientWithCredentials(provider model.CloudProvider, credentials CloudCredentials, region string, logger *zap.Logger) (CloudComputeClient, error) {
	if err := ValidateCloudCredentials(provider, credentials, time.Now().UTC()); err != nil {
		return nil, err
	}
	switch provider {
	case model.ProviderAliyun:
		return NewAliyunClient(credentials.AccessKey, credentials.SecretKey, region, logger)
	case model.ProviderAWS:
		return NewAWSClientWithCredentials(credentials, region)
	case model.ProviderTencent:
		return NewTencentClientWithCredentials(credentials, region)
	case model.ProviderAzure:
		return NewAzureClient(credentials.AzureTenantID, credentials.AzureClientID, credentials.AzureClientSecret, credentials.AzureSubscriptionID, region)
	case model.ProviderGCP:
		return NewGCPClient([]byte(credentials.GCPServiceAccountJSON), credentials.GCPProjectID, region)
	case model.ProviderHuawei:
		if strings.TrimSpace(credentials.ProjectID) == "" {
			return nil, fmt.Errorf("华为云地域缺少项目ID")
		}
		return NewHuaweiClient(credentials.AccessKey, credentials.SecretKey, credentials.ProjectID, region)
	default:
		return nil, fmt.Errorf("云厂商 %d 暂未实现计算资源API", provider)
	}
}

func NewCloudComputeClient(provider model.CloudProvider, accessKey, secretKey, region string, logger *zap.Logger) (CloudComputeClient, error) {
	return NewCloudComputeClientWithCredentials(provider, CloudCredentials{AccessKey: accessKey, SecretKey: secretKey}, region, logger)
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

// ValidateCloudRegion keeps Azure locations and GCP zones explicit.
func ValidateCloudRegion(provider model.CloudProvider, region string) error {
	switch provider {
	case model.ProviderAzure:
		if len(region) > 50 || !azureRegionPattern.MatchString(region) {
			return fmt.Errorf("Azure 地域无效")
		}
	case model.ProviderGCP:
		if len(region) > 50 || !gcpZonePattern.MatchString(region) {
			return fmt.Errorf("GCP 可用区无效")
		}
	}
	return nil
}
