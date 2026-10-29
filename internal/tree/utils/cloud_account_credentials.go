package utils

import (
	"context"
	"errors"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

// LoadCloudAccountCredentials 从同一账户快照解密完整凭据，失败前不构造云客户端。
func LoadCloudAccountCredentials(account *model.CloudAccount) (CloudCredentials, error) {
	if account == nil {
		return CloudCredentials{}, errors.New("云账户不存在")
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
