package service

import (
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/dao"
	"go.uber.org/zap"
)

func mustCloudAccountService(t testing.TB, encryptionKey string, logger *zap.Logger, store dao.CloudAccountDAO) CloudAccountService {
	t.Helper()
	result, err := NewCloudAccountService(&config.Config{Tree: config.TreeConfig{PasswordEncryptionKey: encryptionKey}}, logger, store)
	if err != nil {
		t.Fatal("configured cloud account service construction failed")
	}
	return result
}

func mustManagedCloudService(t testing.TB, encryptionKey string, logger *zap.Logger, accounts dao.CloudAccountDAO, regions dao.TreeCloudDAO, store dao.ManagedCloudDAO) ManagedCloudService {
	t.Helper()
	result, err := NewManagedCloudService(&config.Config{Tree: config.TreeConfig{PasswordEncryptionKey: encryptionKey}}, logger, accounts, regions, store)
	if err != nil {
		t.Fatal("configured managed cloud service construction failed")
	}
	return result
}
