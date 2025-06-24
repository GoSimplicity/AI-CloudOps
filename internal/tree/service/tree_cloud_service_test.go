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
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// MockTreeCloudDAO is a mock implementation of TreeCloudDAO
type MockTreeCloudDAO struct {
	mock.Mock
}

func (m *MockTreeCloudDAO) CreateCloudAccount(ctx context.Context, account *model.CloudAccount) error {
	args := m.Called(ctx, account)
	return args.Error(0)
}

func (m *MockTreeCloudDAO) UpdateCloudAccount(ctx context.Context, id int, account *model.CloudAccount) error {
	args := m.Called(ctx, id, account)
	return args.Error(0)
}

func (m *MockTreeCloudDAO) DeleteCloudAccount(ctx context.Context, id int) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockTreeCloudDAO) GetCloudAccount(ctx context.Context, id int) (*model.CloudAccount, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.CloudAccount), args.Error(1)
}

func (m *MockTreeCloudDAO) ListCloudAccounts(ctx context.Context, req *model.ListCloudAccountsReq) (model.ListResp[model.CloudAccount], error) {
	args := m.Called(ctx, req)
	return args.Get(0).(model.ListResp[model.CloudAccount]), args.Error(1)
}

func (m *MockTreeCloudDAO) GetCloudAccountByProvider(ctx context.Context, provider model.CloudProvider) ([]*model.CloudAccount, error) {
	args := m.Called(ctx, provider)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.CloudAccount), args.Error(1)
}

func (m *MockTreeCloudDAO) GetEnabledCloudAccounts(ctx context.Context) ([]*model.CloudAccount, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.CloudAccount), args.Error(1)
}

func (m *MockTreeCloudDAO) CreateSyncStatus(ctx context.Context, status *model.CloudAccountSyncStatus) error {
	args := m.Called(ctx, status)
	return args.Error(0)
}

func (m *MockTreeCloudDAO) UpdateSyncStatus(ctx context.Context, id int, status *model.CloudAccountSyncStatus) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *MockTreeCloudDAO) GetSyncStatus(ctx context.Context, accountId int, resourceType, region string) (*model.CloudAccountSyncStatus, error) {
	args := m.Called(ctx, accountId, resourceType, region)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.CloudAccountSyncStatus), args.Error(1)
}

func (m *MockTreeCloudDAO) ListSyncStatus(ctx context.Context, accountId int) ([]*model.CloudAccountSyncStatus, error) {
	args := m.Called(ctx, accountId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.CloudAccountSyncStatus), args.Error(1)
}

func (m *MockTreeCloudDAO) DeleteSyncStatus(ctx context.Context, accountId int) error {
	args := m.Called(ctx, accountId)
	return args.Error(0)
}

func (m *MockTreeCloudDAO) CreateAuditLog(ctx context.Context, log *model.CloudAccountAuditLog) error {
	args := m.Called(ctx, log)
	return args.Error(0)
}

func (m *MockTreeCloudDAO) ListAuditLogs(ctx context.Context, accountId int, page, pageSize int) (model.ListResp[model.CloudAccountAuditLog], error) {
	args := m.Called(ctx, accountId, page, pageSize)
	return args.Get(0).(model.ListResp[model.CloudAccountAuditLog]), args.Error(1)
}

func (m *MockTreeCloudDAO) GetAuditLogsByOperation(ctx context.Context, operation string, page, pageSize int) (model.ListResp[model.CloudAccountAuditLog], error) {
	args := m.Called(ctx, operation, page, pageSize)
	return args.Get(0).(model.ListResp[model.CloudAccountAuditLog]), args.Error(1)
}

func (m *MockTreeCloudDAO) BatchGetCloudAccounts(ctx context.Context, ids []int) ([]*model.CloudAccount, error) {
	args := m.Called(ctx, ids)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.CloudAccount), args.Error(1)
}

func (m *MockTreeCloudDAO) BatchUpdateLastSyncTime(ctx context.Context, accountIds []int, syncTime time.Time) error {
	args := m.Called(ctx, accountIds, syncTime)
	return args.Error(0)
}

func (m *MockTreeCloudDAO) GetDecryptedSecretKey(ctx context.Context, accountId int) (string, error) {
	args := m.Called(ctx, accountId)
	return args.String(0), args.Error(1)
}

func (m *MockTreeCloudDAO) ReEncryptAccount(ctx context.Context, accountId int) error {
	args := m.Called(ctx, accountId)
	return args.Error(0)
}

func (m *MockTreeCloudDAO) CreateEcsResource(ctx context.Context, ecs *model.ResourceEcs) error {
	args := m.Called(ctx, ecs)
	return args.Error(0)
}

func (m *MockTreeCloudDAO) DeleteEcsResource(ctx context.Context, id int) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockTreeCloudDAO) UpdateEcsResource(ctx context.Context, ecs *model.ResourceEcs) error {
	args := m.Called(ctx, ecs)
	return args.Error(0)
}

func (m *MockTreeCloudDAO) GetEcsResourceById(ctx context.Context, id int) (*model.ResourceEcs, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ResourceEcs), args.Error(1)
}

func (m *MockTreeCloudDAO) ListEcsResources(ctx context.Context, req *model.ListEcsResourcesReq) ([]*model.ResourceEcs, int64, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*model.ResourceEcs), args.Get(1).(int64), args.Error(2)
}

// MockCryptoManager is a mock implementation of CryptoManager
type MockCryptoManager struct {
	mock.Mock
}

func (m *MockCryptoManager) EncryptSecretKey(secretKey string) (string, error) {
	args := m.Called(secretKey)
	return args.String(0), args.Error(1)
}

func (m *MockCryptoManager) DecryptSecretKey(encryptedSecretKey string) (string, error) {
	args := m.Called(encryptedSecretKey)
	return args.String(0), args.Error(1)
}

func (m *MockCryptoManager) EncryptBatch(secretKeys []string) ([]string, error) {
	args := m.Called(secretKeys)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockCryptoManager) DecryptBatch(encryptedSecretKeys []string) ([]string, error) {
	args := m.Called(encryptedSecretKeys)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockCryptoManager) RotateKey(newKey []byte) error {
	args := m.Called(newKey)
	return args.Error(0)
}

func (m *MockCryptoManager) GetKeyInfo() map[string]interface{} {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(map[string]interface{})
}

func (m *MockCryptoManager) ValidateEncryptedData(encryptedData string) error {
	args := m.Called(encryptedData)
	return args.Error(0)
}

func TestTreeCloudService_CreateCloudAccount(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	ctx := context.Background()
	validReq := &model.CreateCloudAccountReq{
		Name:        "test-account",
		Provider:    model.CloudProviderAliyun,
		AccountId:   "123456789",
		AccessKey:   "test-access-key",
		SecretKey:   "test-secret-key",
		Regions:     []string{"cn-hangzhou", "cn-shanghai"},
		IsEnabled:   true,
		Description: "Test account",
	}

	t.Run("Success", func(t *testing.T) {
		mockDAO := new(MockTreeCloudDAO)
		mockCrypto := new(MockCryptoManager)
		mockProviderFactory := &provider.ProviderFactory{}
		service := NewTreeCloudService(logger, mockDAO, mockCrypto, mockProviderFactory)

		// Setup mocks
		mockCrypto.On("EncryptSecretKey", "test-secret-key").Return("encrypted-secret", nil)
		mockDAO.On("CreateCloudAccount", ctx, mock.AnythingOfType("*model.CloudAccount")).Return(nil)
		mockDAO.On("CreateAuditLog", ctx, mock.AnythingOfType("*model.CloudAccountAuditLog")).Return(nil)

		// Execute
		err := service.CreateCloudAccount(ctx, validReq)

		// Assert
		assert.NoError(t, err)
		mockCrypto.AssertExpectations(t)
		mockDAO.AssertExpectations(t)
	})

	t.Run("ValidationFailure_NilRequest", func(t *testing.T) {
		mockDAO := new(MockTreeCloudDAO)
		mockCrypto := new(MockCryptoManager)
		mockProviderFactory := &provider.ProviderFactory{}
		service := NewTreeCloudService(logger, mockDAO, mockCrypto, mockProviderFactory)

		err := service.CreateCloudAccount(ctx, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "请求参数不能为空")
	})

	t.Run("ValidationFailure_EmptyName", func(t *testing.T) {
		mockDAO := new(MockTreeCloudDAO)
		mockCrypto := new(MockCryptoManager)
		mockProviderFactory := &provider.ProviderFactory{}
		service := NewTreeCloudService(logger, mockDAO, mockCrypto, mockProviderFactory)

		req := *validReq
		req.Name = ""
		err := service.CreateCloudAccount(ctx, &req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "账户名称不能为空")
	})

	t.Run("ValidationFailure_EmptyProvider", func(t *testing.T) {
		mockDAO := new(MockTreeCloudDAO)
		mockCrypto := new(MockCryptoManager)
		mockProviderFactory := &provider.ProviderFactory{}
		service := NewTreeCloudService(logger, mockDAO, mockCrypto, mockProviderFactory)

		req := *validReq
		req.Provider = ""
		err := service.CreateCloudAccount(ctx, &req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "云厂商不能为空")
	})

	t.Run("ValidationFailure_EmptyAccountId", func(t *testing.T) {
		mockDAO := new(MockTreeCloudDAO)
		mockCrypto := new(MockCryptoManager)
		mockProviderFactory := &provider.ProviderFactory{}
		service := NewTreeCloudService(logger, mockDAO, mockCrypto, mockProviderFactory)

		req := *validReq
		req.AccountId = ""
		err := service.CreateCloudAccount(ctx, &req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "账户ID不能为空")
	})

	t.Run("ValidationFailure_EmptyAccessKey", func(t *testing.T) {
		mockDAO := new(MockTreeCloudDAO)
		mockCrypto := new(MockCryptoManager)
		mockProviderFactory := &provider.ProviderFactory{}
		service := NewTreeCloudService(logger, mockDAO, mockCrypto, mockProviderFactory)

		req := *validReq
		req.AccessKey = ""
		err := service.CreateCloudAccount(ctx, &req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "AccessKey不能为空")
	})

	t.Run("ValidationFailure_EmptySecretKey", func(t *testing.T) {
		mockDAO := new(MockTreeCloudDAO)
		mockCrypto := new(MockCryptoManager)
		mockProviderFactory := &provider.ProviderFactory{}
		service := NewTreeCloudService(logger, mockDAO, mockCrypto, mockProviderFactory)

		req := *validReq
		req.SecretKey = ""
		err := service.CreateCloudAccount(ctx, &req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "SecretKey不能为空")
	})

	t.Run("EncryptionFailure", func(t *testing.T) {
		mockDAO := new(MockTreeCloudDAO)
		mockCrypto := new(MockCryptoManager)
		mockProviderFactory := &provider.ProviderFactory{}
		service := NewTreeCloudService(logger, mockDAO, mockCrypto, mockProviderFactory)

		mockCrypto.On("EncryptSecretKey", "test-secret-key").Return("", errors.New("encryption failed"))

		err := service.CreateCloudAccount(ctx, validReq)
		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), "加密SecretKey失败")
		}
		mockCrypto.AssertExpectations(t)
	})

	t.Run("DAOCreateFailure", func(t *testing.T) {
		mockDAO := new(MockTreeCloudDAO)
		mockCrypto := new(MockCryptoManager)
		mockProviderFactory := &provider.ProviderFactory{}
		service := NewTreeCloudService(logger, mockDAO, mockCrypto, mockProviderFactory)

		mockCrypto.On("EncryptSecretKey", "test-secret-key").Return("encrypted-secret", nil)
		mockDAO.On("CreateCloudAccount", ctx, mock.AnythingOfType("*model.CloudAccount")).Return(errors.New("database error"))

		err := service.CreateCloudAccount(ctx, validReq)
		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), "创建云账户失败")
		}
		mockCrypto.AssertExpectations(t)
		mockDAO.AssertExpectations(t)
	})

	t.Run("AuditLogFailure_NonBlocking", func(t *testing.T) {
		mockDAO := new(MockTreeCloudDAO)
		mockCrypto := new(MockCryptoManager)
		mockProviderFactory := &provider.ProviderFactory{}
		service := NewTreeCloudService(logger, mockDAO, mockCrypto, mockProviderFactory)

		mockCrypto.On("EncryptSecretKey", "test-secret-key").Return("encrypted-secret", nil)
		mockDAO.On("CreateCloudAccount", ctx, mock.AnythingOfType("*model.CloudAccount")).Return(nil)
		mockDAO.On("CreateAuditLog", ctx, mock.AnythingOfType("*model.CloudAccountAuditLog")).Return(errors.New("audit log failed"))

		// Should not fail even if audit log creation fails
		err := service.CreateCloudAccount(ctx, validReq)
		assert.NoError(t, err)
		mockCrypto.AssertExpectations(t)
		mockDAO.AssertExpectations(t)
	})
}

func TestTreeCloudService_UpdateCloudAccount(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockDAO := new(MockTreeCloudDAO)
	mockCrypto := new(MockCryptoManager)

	mockProviderFactory := &provider.ProviderFactory{}
	service := NewTreeCloudService(logger, mockDAO, mockCrypto, mockProviderFactory)

	ctx := context.Background()
	validReq := &model.UpdateCloudAccountReq{
		Name:        "updated-account",
		Provider:    model.CloudProviderHuawei,
		AccountId:   "987654321",
		AccessKey:   "updated-access-key",
		SecretKey:   "updated-secret-key",
		Regions:     []string{"cn-north-1"},
		IsEnabled:   false,
		Description: "Updated account",
	}

	existingAccount := &model.CloudAccount{
		Model:           model.Model{ID: 1},
		Name:            "old-name",
		Provider:        model.CloudProviderAliyun,
		AccountId:       "123456789",
		AccessKey:       "old-access-key",
		EncryptedSecret: "old-encrypted-secret",
		Regions:         []string{"cn-hangzhou"},
		IsEnabled:       true,
		Description:     "Old description",
	}

	t.Run("Success_WithNewSecretKey", func(t *testing.T) {
		mockDAO.On("GetCloudAccount", ctx, 1).Return(existingAccount, nil)
		mockCrypto.On("EncryptSecretKey", "updated-secret-key").Return("new-encrypted-secret", nil)
		mockDAO.On("UpdateCloudAccount", ctx, 1, mock.AnythingOfType("*model.CloudAccount")).Return(nil)
		mockDAO.On("CreateAuditLog", ctx, mock.AnythingOfType("*model.CloudAccountAuditLog")).Return(nil)

		err := service.UpdateCloudAccount(ctx, 1, validReq)
		assert.NoError(t, err)
		mockDAO.AssertExpectations(t)
		mockCrypto.AssertExpectations(t)
	})

	t.Run("Success_WithoutNewSecretKey", func(t *testing.T) {
		req := *validReq
		req.SecretKey = ""

		mockDAO.On("GetCloudAccount", ctx, 1).Return(existingAccount, nil)
		mockDAO.On("UpdateCloudAccount", ctx, 1, mock.AnythingOfType("*model.CloudAccount")).Return(nil)
		mockDAO.On("CreateAuditLog", ctx, mock.AnythingOfType("*model.CloudAccountAuditLog")).Return(nil)

		err := service.UpdateCloudAccount(ctx, 1, &req)
		assert.NoError(t, err)
		mockDAO.AssertExpectations(t)
	})

	t.Run("ValidationFailure_NilRequest", func(t *testing.T) {
		err := service.UpdateCloudAccount(ctx, 1, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "请求参数不能为空")
	})

	t.Run("ValidationFailure_EmptyName", func(t *testing.T) {
		req := *validReq
		req.Name = ""
		err := service.UpdateCloudAccount(ctx, 1, &req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "账户名称不能为空")
	})

	t.Run("ValidationFailure_EmptyProvider", func(t *testing.T) {
		req := *validReq
		req.Provider = ""
		err := service.UpdateCloudAccount(ctx, 1, &req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "云厂商不能为空")
	})

	t.Run("ValidationFailure_EmptyAccountId", func(t *testing.T) {
		req := *validReq
		req.AccountId = ""
		err := service.UpdateCloudAccount(ctx, 1, &req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "账户ID不能为空")
	})

	t.Run("ValidationFailure_EmptyAccessKey", func(t *testing.T) {
		req := *validReq
		req.AccessKey = ""
		err := service.UpdateCloudAccount(ctx, 1, &req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "AccessKey不能为空")
	})

	t.Run("GetAccountFailure", func(t *testing.T) {
		mockDAO := new(MockTreeCloudDAO)
		mockCrypto := new(MockCryptoManager)
		mockProviderFactory := &provider.ProviderFactory{}
		service := NewTreeCloudService(logger, mockDAO, mockCrypto, mockProviderFactory)

		mockDAO.On("GetCloudAccount", ctx, 1).Return(nil, errors.New("account not found"))

		err := service.UpdateCloudAccount(ctx, 1, validReq)
		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), "获取云账户失败")
		}
		mockDAO.AssertExpectations(t)
	})

	t.Run("EncryptionFailure", func(t *testing.T) {
		mockDAO := new(MockTreeCloudDAO)
		mockCrypto := new(MockCryptoManager)
		mockProviderFactory := &provider.ProviderFactory{}
		service := NewTreeCloudService(logger, mockDAO, mockCrypto, mockProviderFactory)

		mockDAO.On("GetCloudAccount", ctx, 1).Return(existingAccount, nil)
		mockCrypto.On("EncryptSecretKey", "updated-secret-key").Return("", errors.New("encryption failed"))

		err := service.UpdateCloudAccount(ctx, 1, validReq)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "加密SecretKey失败")
		mockDAO.AssertExpectations(t)
		mockCrypto.AssertExpectations(t)
	})

	t.Run("DAOUpdateFailure", func(t *testing.T) {
		mockDAO := new(MockTreeCloudDAO)
		mockCrypto := new(MockCryptoManager)
		mockProviderFactory := &provider.ProviderFactory{}
		service := NewTreeCloudService(logger, mockDAO, mockCrypto, mockProviderFactory)

		mockDAO.On("GetCloudAccount", ctx, 1).Return(existingAccount, nil)
		mockCrypto.On("EncryptSecretKey", "updated-secret-key").Return("new-encrypted-secret", nil)
		mockDAO.On("UpdateCloudAccount", ctx, 1, mock.AnythingOfType("*model.CloudAccount")).Return(errors.New("update failed"))

		err := service.UpdateCloudAccount(ctx, 1, validReq)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "更新云账户失败")
		mockDAO.AssertExpectations(t)
		mockCrypto.AssertExpectations(t)
	})
}

func TestTreeCloudService_GetCloudAccount(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockDAO := new(MockTreeCloudDAO)
	mockCrypto := new(MockCryptoManager)

	mockProviderFactory := &provider.ProviderFactory{}
	service := NewTreeCloudService(logger, mockDAO, mockCrypto, mockProviderFactory)

	ctx := context.Background()
	encryptedAccount := &model.CloudAccount{
		Model:           model.Model{ID: 1},
		Name:            "test-account",
		Provider:        model.CloudProviderAliyun,
		AccountId:       "123456789",
		AccessKey:       "test-access-key",
		EncryptedSecret: "encrypted-secret-key",
		Regions:         []string{"cn-hangzhou"},
		IsEnabled:       true,
		Description:     "Test account",
		LastSyncTime:    time.Now(),
	}

	t.Run("Success", func(t *testing.T) {
		mockDAO.On("GetCloudAccount", ctx, 1).Return(encryptedAccount, nil)
		mockCrypto.On("DecryptSecretKey", "encrypted-secret-key").Return("decrypted-secret-key", nil)

		result, err := service.GetCloudAccount(ctx, 1)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "decrypted-secret-key", result.EncryptedSecret) // Should contain decrypted secret
		assert.Equal(t, encryptedAccount.Name, result.Name)
		assert.Equal(t, encryptedAccount.Provider, result.Provider)
		mockDAO.AssertExpectations(t)
		mockCrypto.AssertExpectations(t)
	})

	t.Run("DAOGetFailure", func(t *testing.T) {
		mockDAO := new(MockTreeCloudDAO)
		mockCrypto := new(MockCryptoManager)
		mockProviderFactory := &provider.ProviderFactory{}
		service := NewTreeCloudService(logger, mockDAO, mockCrypto, mockProviderFactory)

		mockDAO.On("GetCloudAccount", ctx, 1).Return(nil, errors.New("account not found"))

		result, err := service.GetCloudAccount(ctx, 1)
		if assert.Error(t, err) {
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), "获取云账户失败")
		}
		mockDAO.AssertExpectations(t)
	})

	t.Run("DecryptionFailure", func(t *testing.T) {
		mockDAO := new(MockTreeCloudDAO)
		mockCrypto := new(MockCryptoManager)
		mockProviderFactory := &provider.ProviderFactory{}
		service := NewTreeCloudService(logger, mockDAO, mockCrypto, mockProviderFactory)

		mockDAO.On("GetCloudAccount", ctx, 1).Return(encryptedAccount, nil)
		mockCrypto.On("DecryptSecretKey", "encrypted-secret-key").Return("", errors.New("decryption failed"))

		result, err := service.GetCloudAccount(ctx, 1)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "解密SecretKey失败")
		mockDAO.AssertExpectations(t)
		mockCrypto.AssertExpectations(t)
	})
}

func TestTreeCloudService_ListCloudAccounts(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockDAO := new(MockTreeCloudDAO)
	mockCrypto := new(MockCryptoManager)

	mockProviderFactory := &provider.ProviderFactory{}
	service := NewTreeCloudService(logger, mockDAO, mockCrypto, mockProviderFactory)

	ctx := context.Background()
	req := &model.ListCloudAccountsReq{
		Page:     1,
		PageSize: 10,
		Name:     "test",
		Provider: model.CloudProviderAliyun,
		Enabled:  true,
	}

	accounts := []model.CloudAccount{
		{
			Model:           model.Model{ID: 1},
			Name:            "test-account-1",
			Provider:        model.CloudProviderAliyun,
			AccountId:       "123456789",
			AccessKey:       "access-key-1",
			EncryptedSecret: "encrypted-secret-1",
			Regions:         []string{"cn-hangzhou"},
			IsEnabled:       true,
		},
		{
			Model:           model.Model{ID: 2},
			Name:            "test-account-2",
			Provider:        model.CloudProviderHuawei,
			AccountId:       "987654321",
			AccessKey:       "access-key-2",
			EncryptedSecret: "encrypted-secret-2",
			Regions:         []string{"cn-north-1"},
			IsEnabled:       true,
		},
	}

	t.Run("Success", func(t *testing.T) {
		expectedResult := model.ListResp[model.CloudAccount]{
			Items: accounts,
			Total: 2,
		}
		mockDAO.On("ListCloudAccounts", ctx, req).Return(expectedResult, nil)

		result, err := service.ListCloudAccounts(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int64(2), result.Total)
		assert.Len(t, result.Items, 2)

		// Verify that EncryptedSecret fields are cleared for security
		for _, account := range result.Items {
			assert.Empty(t, account.EncryptedSecret)
		}
		mockDAO.AssertExpectations(t)
	})

	t.Run("DAOFailure", func(t *testing.T) {
		mockDAO := new(MockTreeCloudDAO)
		mockCrypto := new(MockCryptoManager)
		mockProviderFactory := &provider.ProviderFactory{}
		service := NewTreeCloudService(logger, mockDAO, mockCrypto, mockProviderFactory)

		mockDAO.On("ListCloudAccounts", ctx, mock.AnythingOfType("*model.ListCloudAccountsReq")).Return(model.ListResp[model.CloudAccount]{}, errors.New("dao error"))

		req := &model.ListCloudAccountsReq{
			Page:     1,
			PageSize: 10,
			Name:     "test",
			Provider: model.CloudProviderAliyun,
			Enabled:  true,
		}
		result, err := service.ListCloudAccounts(ctx, req)
		if assert.Error(t, err) {
			assert.Empty(t, result.Items)
			assert.Contains(t, err.Error(), "获取云账户列表失败")
		}
		mockDAO.AssertExpectations(t)
	})
}

func TestTreeCloudService_DeleteCloudAccount(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockDAO := new(MockTreeCloudDAO)
	mockCrypto := new(MockCryptoManager)

	mockProviderFactory := &provider.ProviderFactory{}
	service := NewTreeCloudService(logger, mockDAO, mockCrypto, mockProviderFactory)

	ctx := context.Background()
	account := &model.CloudAccount{
		Model:           model.Model{ID: 1},
		Name:            "test-account",
		Provider:        model.CloudProviderAliyun,
		AccountId:       "123456789",
		AccessKey:       "test-access-key",
		EncryptedSecret: "encrypted-secret",
		Regions:         []string{"cn-hangzhou"},
		IsEnabled:       true,
	}

	t.Run("Success", func(t *testing.T) {
		mockDAO.On("GetCloudAccount", ctx, 1).Return(account, nil)
		mockDAO.On("DeleteCloudAccount", ctx, 1).Return(nil)
		mockDAO.On("CreateAuditLog", ctx, mock.AnythingOfType("*model.CloudAccountAuditLog")).Return(nil)

		err := service.DeleteCloudAccount(ctx, 1)
		assert.NoError(t, err)
		mockDAO.AssertExpectations(t)
	})

	t.Run("GetAccountFailure", func(t *testing.T) {
		mockDAO := new(MockTreeCloudDAO)
		mockCrypto := new(MockCryptoManager)
		mockProviderFactory := &provider.ProviderFactory{}
		service := NewTreeCloudService(logger, mockDAO, mockCrypto, mockProviderFactory)

		mockDAO.On("GetCloudAccount", ctx, 1).Return(nil, errors.New("account not found"))

		err := service.DeleteCloudAccount(ctx, 1)
		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), "获取云账户失败")
		}
		mockDAO.AssertExpectations(t)
	})

	t.Run("DeleteFailure", func(t *testing.T) {
		mockDAO := new(MockTreeCloudDAO)
		mockCrypto := new(MockCryptoManager)
		mockProviderFactory := &provider.ProviderFactory{}
		service := NewTreeCloudService(logger, mockDAO, mockCrypto, mockProviderFactory)

		// 先返回一个有效的 account
		mockDAO.On("GetCloudAccount", ctx, 1).Return(&model.CloudAccount{
			Model: model.Model{ID: 1},
			Name:  "test-account",
		}, nil)
		// DeleteCloudAccount 返回 error
		mockDAO.On("DeleteCloudAccount", ctx, 1).Return(errors.New("delete failed"))

		err := service.DeleteCloudAccount(ctx, 1)
		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), "删除云账户失败")
		}
		mockDAO.AssertExpectations(t)
	})

	t.Run("AuditLogFailure_NonBlocking", func(t *testing.T) {
		mockDAO := new(MockTreeCloudDAO)
		mockCrypto := new(MockCryptoManager)
		mockProviderFactory := &provider.ProviderFactory{}
		service := NewTreeCloudService(logger, mockDAO, mockCrypto, mockProviderFactory)

		mockDAO.On("GetCloudAccount", ctx, 1).Return(account, nil)
		mockDAO.On("DeleteCloudAccount", ctx, 1).Return(nil)
		mockDAO.On("CreateAuditLog", ctx, mock.AnythingOfType("*model.CloudAccountAuditLog")).Return(errors.New("audit log failed"))

		// Should not fail even if audit log creation fails
		err := service.DeleteCloudAccount(ctx, 1)
		assert.NoError(t, err)
		mockDAO.AssertExpectations(t)
	})
}

func TestTreeCloudService_TestCloudAccount(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockDAO := new(MockTreeCloudDAO)
	mockCrypto := new(MockCryptoManager)

	mockProviderFactory := &provider.ProviderFactory{}
	service := NewTreeCloudService(logger, mockDAO, mockCrypto, mockProviderFactory)

	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		mockDAO.On("GetDecryptedSecretKey", ctx, 1).Return("decrypted-secret-key", nil)

		err := service.TestCloudAccount(ctx, 1)
		assert.NoError(t, err)
		mockDAO.AssertExpectations(t)
	})

	t.Run("GetSecretKeyFailure", func(t *testing.T) {
		mockDAO := new(MockTreeCloudDAO)
		mockCrypto := new(MockCryptoManager)
		mockProviderFactory := &provider.ProviderFactory{}
		service := NewTreeCloudService(logger, mockDAO, mockCrypto, mockProviderFactory)

		mockDAO.On("GetDecryptedSecretKey", ctx, 1).Return("", errors.New("secret key not found"))

		err := service.TestCloudAccount(ctx, 1)
		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), "获取SecretKey失败")
		}
		mockDAO.AssertExpectations(t)
	})
}

func TestTreeCloudService_SyncCloudResources(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockDAO := new(MockTreeCloudDAO)
	mockCrypto := new(MockCryptoManager)

	mockProviderFactory := &provider.ProviderFactory{}
	service := NewTreeCloudService(logger, mockDAO, mockCrypto, mockProviderFactory)

	ctx := context.Background()
	req := &model.SyncCloudReq{
		AccountIds:   []int{1, 2, 3},
		ResourceType: "ecs",
		Regions:      []string{"cn-hangzhou", "cn-shanghai"},
		Force:        true,
	}

	t.Run("Success", func(t *testing.T) {
		// This is currently a TODO implementation, so it should always succeed
		err := service.SyncCloudResources(ctx, req)
		assert.NoError(t, err)
	})
}

func TestTreeCloudService_GetDecryptedSecretKey(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		mockDAO := new(MockTreeCloudDAO)
		mockCrypto := new(MockCryptoManager)
		mockProviderFactory := &provider.ProviderFactory{}
		service := NewTreeCloudService(logger, mockDAO, mockCrypto, mockProviderFactory)

		mockDAO.On("GetDecryptedSecretKey", ctx, 1).Return("decrypted-secret-key", nil).Once()

		result, err := service.GetDecryptedSecretKey(ctx, 1)
		assert.NoError(t, err)
		assert.Equal(t, "decrypted-secret-key", result)
		mockDAO.AssertExpectations(t)
	})

	t.Run("Failure", func(t *testing.T) {
		mockDAO := new(MockTreeCloudDAO)
		mockCrypto := new(MockCryptoManager)
		mockProviderFactory := &provider.ProviderFactory{}
		service := NewTreeCloudService(logger, mockDAO, mockCrypto, mockProviderFactory)

		mockDAO.On("GetDecryptedSecretKey", ctx, 1).Return("", errors.New("failed to get secret key")).Once()

		result, err := service.GetDecryptedSecretKey(ctx, 1)
		assert.Error(t, err)
		assert.Empty(t, result)
		mockDAO.AssertExpectations(t)
	})
}

func TestTreeCloudService_ReEncryptAccount(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		mockDAO := new(MockTreeCloudDAO)
		mockCrypto := new(MockCryptoManager)
		mockProviderFactory := &provider.ProviderFactory{}
		service := NewTreeCloudService(logger, mockDAO, mockCrypto, mockProviderFactory)

		mockDAO.On("ReEncryptAccount", ctx, 1).Return(nil).Once()

		err := service.ReEncryptAccount(ctx, 1)
		assert.NoError(t, err)
		mockDAO.AssertExpectations(t)
	})

	t.Run("Failure", func(t *testing.T) {
		mockDAO := new(MockTreeCloudDAO)
		mockCrypto := new(MockCryptoManager)
		mockProviderFactory := &provider.ProviderFactory{}
		service := NewTreeCloudService(logger, mockDAO, mockCrypto, mockProviderFactory)

		mockDAO.On("ReEncryptAccount", ctx, 1).Return(errors.New("re-encryption failed")).Once()

		err := service.ReEncryptAccount(ctx, 1)
		assert.Error(t, err)
		mockDAO.AssertExpectations(t)
	})
}

func TestTreeCloudService_validateCreateRequest(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockDAO := new(MockTreeCloudDAO)
	mockCrypto := new(MockCryptoManager)

	mockProviderFactory := &provider.ProviderFactory{}
	service := NewTreeCloudService(logger, mockDAO, mockCrypto, mockProviderFactory)

	t.Run("ValidRequest", func(t *testing.T) {
		req := &model.CreateCloudAccountReq{
			Name:        "test-account",
			Provider:    model.CloudProviderAliyun,
			AccountId:   "123456789",
			AccessKey:   "test-access-key",
			SecretKey:   "test-secret-key",
			Regions:     []string{"cn-hangzhou"},
			IsEnabled:   true,
			Description: "Test account",
		}

		err := service.(*treeCloudService).validateCreateRequest(req)
		assert.NoError(t, err)
	})

	t.Run("NilRequest", func(t *testing.T) {
		err := service.(*treeCloudService).validateCreateRequest(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "请求参数不能为空")
	})

	t.Run("EmptyName", func(t *testing.T) {
		req := &model.CreateCloudAccountReq{
			Name:      "",
			Provider:  model.CloudProviderAliyun,
			AccountId: "123456789",
			AccessKey: "test-access-key",
			SecretKey: "test-secret-key",
		}

		err := service.(*treeCloudService).validateCreateRequest(req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "账户名称不能为空")
	})

	t.Run("EmptyProvider", func(t *testing.T) {
		req := &model.CreateCloudAccountReq{
			Name:      "test-account",
			Provider:  "",
			AccountId: "123456789",
			AccessKey: "test-access-key",
			SecretKey: "test-secret-key",
		}

		err := service.(*treeCloudService).validateCreateRequest(req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "云厂商不能为空")
	})

	t.Run("EmptyAccountId", func(t *testing.T) {
		req := &model.CreateCloudAccountReq{
			Name:      "test-account",
			Provider:  model.CloudProviderAliyun,
			AccountId: "",
			AccessKey: "test-access-key",
			SecretKey: "test-secret-key",
		}

		err := service.(*treeCloudService).validateCreateRequest(req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "账户ID不能为空")
	})

	t.Run("EmptyAccessKey", func(t *testing.T) {
		req := &model.CreateCloudAccountReq{
			Name:      "test-account",
			Provider:  model.CloudProviderAliyun,
			AccountId: "123456789",
			AccessKey: "",
			SecretKey: "test-secret-key",
		}

		err := service.(*treeCloudService).validateCreateRequest(req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "AccessKey不能为空")
	})

	t.Run("EmptySecretKey", func(t *testing.T) {
		req := &model.CreateCloudAccountReq{
			Name:      "test-account",
			Provider:  model.CloudProviderAliyun,
			AccountId: "123456789",
			AccessKey: "test-access-key",
			SecretKey: "",
		}

		err := service.(*treeCloudService).validateCreateRequest(req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "SecretKey不能为空")
	})
}

func TestTreeCloudService_validateUpdateRequest(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockDAO := new(MockTreeCloudDAO)
	mockCrypto := new(MockCryptoManager)

	mockProviderFactory := &provider.ProviderFactory{}
	service := NewTreeCloudService(logger, mockDAO, mockCrypto, mockProviderFactory)

	t.Run("ValidRequest", func(t *testing.T) {
		req := &model.UpdateCloudAccountReq{
			Name:        "updated-account",
			Provider:    model.CloudProviderHuawei,
			AccountId:   "987654321",
			AccessKey:   "updated-access-key",
			SecretKey:   "updated-secret-key",
			Regions:     []string{"cn-north-1"},
			IsEnabled:   false,
			Description: "Updated account",
		}

		err := service.(*treeCloudService).validateUpdateRequest(req)
		assert.NoError(t, err)
	})

	t.Run("NilRequest", func(t *testing.T) {
		err := service.(*treeCloudService).validateUpdateRequest(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "请求参数不能为空")
	})

	t.Run("EmptyName", func(t *testing.T) {
		req := &model.UpdateCloudAccountReq{
			Name:      "",
			Provider:  model.CloudProviderHuawei,
			AccountId: "987654321",
			AccessKey: "updated-access-key",
		}

		err := service.(*treeCloudService).validateUpdateRequest(req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "账户名称不能为空")
	})

	t.Run("EmptyProvider", func(t *testing.T) {
		req := &model.UpdateCloudAccountReq{
			Name:      "updated-account",
			Provider:  "",
			AccountId: "987654321",
			AccessKey: "updated-access-key",
		}

		err := service.(*treeCloudService).validateUpdateRequest(req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "云厂商不能为空")
	})

	t.Run("EmptyAccountId", func(t *testing.T) {
		req := &model.UpdateCloudAccountReq{
			Name:      "updated-account",
			Provider:  model.CloudProviderHuawei,
			AccountId: "",
			AccessKey: "updated-access-key",
		}

		err := service.(*treeCloudService).validateUpdateRequest(req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "账户ID不能为空")
	})

	t.Run("EmptyAccessKey", func(t *testing.T) {
		req := &model.UpdateCloudAccountReq{
			Name:      "updated-account",
			Provider:  model.CloudProviderHuawei,
			AccountId: "987654321",
			AccessKey: "",
		}

		err := service.(*treeCloudService).validateUpdateRequest(req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "AccessKey不能为空")
	})
}

func TestTreeCloudService_getUserInfoFromContext(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockDAO := new(MockTreeCloudDAO)
	mockCrypto := new(MockCryptoManager)

	mockProviderFactory := &provider.ProviderFactory{}
	service := NewTreeCloudService(logger, mockDAO, mockCrypto, mockProviderFactory)

	ctx := context.Background()

	t.Run("DefaultValues", func(t *testing.T) {
		userInfo := service.(*treeCloudService).getUserInfoFromContext(ctx)
		assert.NotNil(t, userInfo)
		assert.Equal(t, "system", userInfo.Username)
		assert.Equal(t, "unknown", userInfo.IP)
		assert.Equal(t, "system", userInfo.UserAgent)
	})
}
