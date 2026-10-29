package utils

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

type storedProviderCredentials struct {
	Version     int                           `json:"version"`
	Credentials model.ProviderCredentialInput `json:"credentials"`
}

func ProviderCredentialsFromInput(input *model.ProviderCredentialInput) CloudCredentials {
	if input == nil {
		return CloudCredentials{}
	}
	return CloudCredentials{CredentialType: "long_term", AzureTenantID: input.TenantID, AzureClientID: input.ClientID, AzureClientSecret: input.ClientSecret, AzureSubscriptionID: input.SubscriptionID, GCPProjectID: input.ProjectID, GCPServiceAccountJSON: input.ServiceAccountJSON}
}

// EncryptProviderCredentials validates and encrypts a complete tuple. The TEXT
// column limit applies to ciphertext bytes, not to input characters.
func EncryptProviderCredentials(provider model.CloudProvider, input *model.ProviderCredentialInput) (string, error) {
	credentials := ProviderCredentialsFromInput(input)
	if input == nil || (provider != model.ProviderAzure && provider != model.ProviderGCP) {
		return "", errors.New("云账户凭据类型无效")
	}
	if err := ValidateCloudCredentials(provider, credentials, time.Now().UTC()); err != nil {
		return "", err
	}
	raw, err := json.Marshal(storedProviderCredentials{Version: 1, Credentials: *input})
	if err != nil || len(raw) > 40000 {
		return "", errors.New("云账户凭据格式无效")
	}
	encrypted, err := EncryptPassword(string(raw))
	if err != nil || len(encrypted) > 64000 {
		return "", errors.New("无法保存云账户凭据")
	}
	return encrypted, nil
}

// LoadCloudAccountCredentials 从同一账户快照解密完整凭据，失败前不构造云客户端。
func LoadCloudAccountCredentials(account *model.CloudAccount) (CloudCredentials, error) {
	if account == nil {
		return CloudCredentials{}, errors.New("云账户不存在")
	}
	if account.Provider == model.ProviderAzure || account.Provider == model.ProviderGCP {
		if account.ProviderCredentials == "" || len(account.ProviderCredentials) > 64000 {
			return CloudCredentials{}, errors.New("云账户缺少有效凭据，请重新配置")
		}
		plaintext, err := DecryptPassword(account.ProviderCredentials)
		if err != nil {
			return CloudCredentials{}, errors.New("无法读取云账户访问凭据")
		}
		var stored storedProviderCredentials
		if err = json.Unmarshal([]byte(plaintext), &stored); err != nil || stored.Version != 1 {
			return CloudCredentials{}, errors.New("云账户凭据格式无效")
		}
		credentials := ProviderCredentialsFromInput(&stored.Credentials)
		if err = ValidateCloudCredentials(account.Provider, credentials, time.Now().UTC()); err != nil {
			return CloudCredentials{}, err
		}
		return credentials, nil
	}
	access, err := DecryptPassword(account.AccessKey)
	if err != nil {
		return CloudCredentials{}, errors.New("无法读取云账户访问凭据")
	}
	secret, err := DecryptPassword(account.SecretKey)
	if err != nil {
		return CloudCredentials{}, errors.New("无法读取云账户访问凭据")
	}
	credentials := CloudCredentials{AccessKey: access, SecretKey: secret, CredentialType: account.CredentialType}
	if account.CredentialExpiresAt != nil {
		expires := account.CredentialExpiresAt.UTC()
		credentials.ExpiresAt = &expires
	}
	if account.SessionToken != "" {
		if len(account.SessionToken) > 16000 {
			return CloudCredentials{}, errors.New("云账户令牌密文无效")
		}
		credentials.SessionToken, err = DecryptPassword(account.SessionToken)
		if err != nil {
			return CloudCredentials{}, errors.New("无法读取云账户访问凭据")
		}
	}
	if err = ValidateCloudCredentials(account.Provider, credentials, time.Now().UTC()); err != nil {
		return CloudCredentials{}, err
	}
	return credentials, nil
}

// CredentialContext prevents use after the configured expiry, including retries.
func CredentialContext(ctx context.Context, credentials CloudCredentials) (context.Context, context.CancelFunc) {
	if credentials.ExpiresAt != nil {
		return context.WithDeadline(ctx, *credentials.ExpiresAt)
	}
	return context.WithCancel(ctx)
}
