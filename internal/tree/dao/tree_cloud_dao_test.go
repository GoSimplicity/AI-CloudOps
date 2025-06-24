package dao

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

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
	return args.Get(0).([]string), args.Error(1)
}
func (m *MockCryptoManager) DecryptBatch(encryptedSecretKeys []string) ([]string, error) {
	args := m.Called(encryptedSecretKeys)
	return args.Get(0).([]string), args.Error(1)
}
func (m *MockCryptoManager) RotateKey(newKey []byte) error {
	args := m.Called(newKey)
	return args.Error(0)
}
func (m *MockCryptoManager) GetKeyInfo() map[string]interface{} {
	args := m.Called()
	return args.Get(0).(map[string]interface{})
}
func (m *MockCryptoManager) ValidateEncryptedData(encryptedData string) error {
	args := m.Called(encryptedData)
	return args.Error(0)
}

var (
	mockCrypto *MockCryptoManager
	logger     *zap.Logger
)

func TestMain(m *testing.M) {
	logger, _ = zap.NewDevelopment()
	mockCrypto = new(MockCryptoManager)

	code := m.Run()
	os.Exit(code)
}

// 为每个测试创建独立的数据库连接
func setupTestDB(t *testing.T) (*gorm.DB, TreeCloudDAO) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// 自动迁移表结构
	db.AutoMigrate(&model.CloudAccount{}, &model.CloudAccountSyncStatus{}, &model.CloudAccountAuditLog{})

	treeCloudDao := NewTreeCloudDAO(logger, db, mockCrypto)
	return db, treeCloudDao
}

// 通用测试数据生成
func newTestCloudAccount(name string) *model.CloudAccount {
	return &model.CloudAccount{
		Name:            name,
		Provider:        model.CloudProviderAliyun,
		AccountId:       name + "-account-id",
		AccessKey:       name + "-access-key",
		EncryptedSecret: "encrypted-secret",
		Regions:         []string{"cn-hangzhou"},
		IsEnabled:       true,
		Description:     "test desc",
		LastSyncTime:    time.Now(),
	}
}

func TestTreeCloudDAO_CreateCloudAccount(t *testing.T) {
	ctx := context.Background()
	_, treeCloudDao := setupTestDB(t)

	// 正常创建
	acc := newTestCloudAccount("dao-create-1")
	err := treeCloudDao.CreateCloudAccount(ctx, acc)
	assert.NoError(t, err)

	// 名称唯一性
	acc2 := newTestCloudAccount("dao-create-1") // name相同
	acc2.AccountId = "dao-create-1-2-account-id"
	acc2.AccessKey = "dao-create-1-2-access-key"
	err = treeCloudDao.CreateCloudAccount(ctx, acc2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "账户名称已存在")

	// AccessKey唯一性
	acc3 := newTestCloudAccount("dao-create-3")
	acc3.AccountId = "dao-create-3-account-id"
	acc3.AccessKey = acc.AccessKey // 与acc1相同
	err = treeCloudDao.CreateCloudAccount(ctx, acc3)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "AccessKey已存在")
}

func TestTreeCloudDAO_GetCloudAccount(t *testing.T) {
	ctx := context.Background()
	_, treeCloudDao := setupTestDB(t)

	acc := newTestCloudAccount("dao-get-1")
	err := treeCloudDao.CreateCloudAccount(ctx, acc)
	assert.NoError(t, err)

	// 正常获取
	got, err := treeCloudDao.GetCloudAccount(ctx, int(acc.ID))
	assert.NoError(t, err)
	assert.Equal(t, acc.Name, got.Name)

	// 不存在
	_, err = treeCloudDao.GetCloudAccount(ctx, 99999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "云账户不存在")
}

func TestTreeCloudDAO_UpdateCloudAccount(t *testing.T) {
	ctx := context.Background()
	_, treeCloudDao := setupTestDB(t)

	acc := newTestCloudAccount("dao-update-1")
	err := treeCloudDao.CreateCloudAccount(ctx, acc)
	assert.NoError(t, err)

	// 正常更新
	accUpdate := *acc
	accUpdate.Name = "dao-update-1-new"
	err = treeCloudDao.UpdateCloudAccount(ctx, int(acc.ID), &accUpdate)
	assert.NoError(t, err)
	got, _ := treeCloudDao.GetCloudAccount(ctx, int(acc.ID))
	assert.Equal(t, "dao-update-1-new", got.Name)

	// 名称唯一性
	acc2 := newTestCloudAccount("dao-update-2")
	_ = treeCloudDao.CreateCloudAccount(ctx, acc2)
	accUpdate2 := *acc2
	accUpdate2.Name = "dao-update-1-new" // 已存在
	err = treeCloudDao.UpdateCloudAccount(ctx, int(acc2.ID), &accUpdate2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "账户名称已存在")

	// 不存在
	accUpdate3 := *acc
	accUpdate3.Name = "not-exist"
	err = treeCloudDao.UpdateCloudAccount(ctx, 99999, &accUpdate3)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "云账户不存在")
}

func TestTreeCloudDAO_DeleteCloudAccount(t *testing.T) {
	ctx := context.Background()
	_, treeCloudDao := setupTestDB(t)

	acc := newTestCloudAccount("dao-delete-1")
	err := treeCloudDao.CreateCloudAccount(ctx, acc)
	assert.NoError(t, err)

	// 正常删除
	err = treeCloudDao.DeleteCloudAccount(ctx, int(acc.ID))
	assert.NoError(t, err)
	_, err = treeCloudDao.GetCloudAccount(ctx, int(acc.ID))
	assert.Error(t, err)

	// 不存在
	err = treeCloudDao.DeleteCloudAccount(ctx, 99999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "云账户不存在")
}

func TestTreeCloudDAO_ListCloudAccounts(t *testing.T) {
	ctx := context.Background()
	_, treeCloudDao := setupTestDB(t)

	// 创建测试数据
	for i := 0; i < 5; i++ {
		acc := newTestCloudAccount("dao-list-" + string(rune(i+'A')))
		acc.AccountId = acc.Name + "-id"
		acc.AccessKey = acc.Name + "-ak"
		_ = treeCloudDao.CreateCloudAccount(ctx, acc)
	}

	req := &model.ListCloudAccountsReq{Page: 1, PageSize: 2, Name: "dao-list", Provider: model.CloudProviderAliyun, Enabled: true}
	resp, err := treeCloudDao.ListCloudAccounts(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, int64(5), resp.Total)
	assert.Len(t, resp.Items, 2)
}

func TestTreeCloudDAO_BatchGetCloudAccounts(t *testing.T) {
	ctx := context.Background()
	_, treeCloudDao := setupTestDB(t)

	acc1 := newTestCloudAccount("dao-batch-1")
	acc2 := newTestCloudAccount("dao-batch-2")
	_ = treeCloudDao.CreateCloudAccount(ctx, acc1)
	_ = treeCloudDao.CreateCloudAccount(ctx, acc2)
	ids := []int{int(acc1.ID), int(acc2.ID)}
	list, err := treeCloudDao.BatchGetCloudAccounts(ctx, ids)
	assert.NoError(t, err)
	assert.Len(t, list, 2)

	// 空ID
	list, err = treeCloudDao.BatchGetCloudAccounts(ctx, []int{})
	assert.NoError(t, err)
	assert.Len(t, list, 0)
}

func TestTreeCloudDAO_BatchUpdateLastSyncTime(t *testing.T) {
	ctx := context.Background()
	_, treeCloudDao := setupTestDB(t)

	acc1 := newTestCloudAccount("dao-sync-1")
	acc2 := newTestCloudAccount("dao-sync-2")
	_ = treeCloudDao.CreateCloudAccount(ctx, acc1)
	_ = treeCloudDao.CreateCloudAccount(ctx, acc2)
	ids := []int{int(acc1.ID), int(acc2.ID)}
	tm := time.Now().Add(1 * time.Hour)
	err := treeCloudDao.BatchUpdateLastSyncTime(ctx, ids, tm)
	assert.NoError(t, err)
	for _, id := range ids {
		acc, _ := treeCloudDao.GetCloudAccount(ctx, id)
		assert.WithinDuration(t, tm, acc.LastSyncTime, time.Second)
	}
	// 空ID
	err = treeCloudDao.BatchUpdateLastSyncTime(ctx, []int{}, tm)
	assert.NoError(t, err)
}

func TestTreeCloudDAO_CreateSyncStatus(t *testing.T) {
	ctx := context.Background()
	_, treeCloudDao := setupTestDB(t)

	acc := newTestCloudAccount("dao-sync-acc")
	_ = treeCloudDao.CreateCloudAccount(ctx, acc)
	status := &model.CloudAccountSyncStatus{
		AccountId:    int(acc.ID),
		ResourceType: "ecs",
		Region:       "cn-hangzhou",
		Status:       "pending",
		LastSyncTime: time.Now(),
		ErrorMessage: "",
		SyncCount:    0,
	}
	// 正常创建
	err := treeCloudDao.CreateSyncStatus(ctx, status)
	assert.NoError(t, err)
	// 重复创建应走更新逻辑
	status.Status = "success"
	err = treeCloudDao.CreateSyncStatus(ctx, status)
	assert.NoError(t, err)
}

func TestTreeCloudDAO_UpdateSyncStatus(t *testing.T) {
	ctx := context.Background()
	_, treeCloudDao := setupTestDB(t)

	acc := newTestCloudAccount("dao-sync-update-acc")
	_ = treeCloudDao.CreateCloudAccount(ctx, acc)
	status := &model.CloudAccountSyncStatus{
		AccountId:    int(acc.ID),
		ResourceType: "ecs",
		Region:       "cn-hangzhou",
		Status:       "pending",
		LastSyncTime: time.Now(),
	}
	_ = treeCloudDao.CreateSyncStatus(ctx, status)
	// 正常更新
	status.Status = "success"
	err := treeCloudDao.UpdateSyncStatus(ctx, int(status.ID), status)
	assert.NoError(t, err)
	// 不存在
	err = treeCloudDao.UpdateSyncStatus(ctx, 99999, status)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "同步状态记录不存在")
}

func TestTreeCloudDAO_GetSyncStatus(t *testing.T) {
	ctx := context.Background()
	_, treeCloudDao := setupTestDB(t)

	acc := newTestCloudAccount("dao-sync-get-acc")
	_ = treeCloudDao.CreateCloudAccount(ctx, acc)
	status := &model.CloudAccountSyncStatus{
		AccountId:    int(acc.ID),
		ResourceType: "ecs",
		Region:       "cn-hangzhou",
		Status:       "pending",
		LastSyncTime: time.Now(),
	}
	_ = treeCloudDao.CreateSyncStatus(ctx, status)
	// 正常获取
	got, err := treeCloudDao.GetSyncStatus(ctx, int(acc.ID), "ecs", "cn-hangzhou")
	assert.NoError(t, err)
	assert.NotNil(t, got)
	// 不存在
	got, err = treeCloudDao.GetSyncStatus(ctx, int(acc.ID), "ecs", "not-exist")
	assert.NoError(t, err)
	assert.Nil(t, got)
}

func TestTreeCloudDAO_ListSyncStatus(t *testing.T) {
	ctx := context.Background()
	_, treeCloudDao := setupTestDB(t)

	acc := newTestCloudAccount("dao-sync-list-acc")
	_ = treeCloudDao.CreateCloudAccount(ctx, acc)
	for i := 0; i < 3; i++ {
		status := &model.CloudAccountSyncStatus{
			AccountId:    int(acc.ID),
			ResourceType: "ecs",
			Region:       "region-" + string(rune(i+'A')),
			Status:       "pending",
			LastSyncTime: time.Now(),
		}
		_ = treeCloudDao.CreateSyncStatus(ctx, status)
	}
	list, err := treeCloudDao.ListSyncStatus(ctx, int(acc.ID))
	assert.NoError(t, err)
	assert.True(t, len(list) >= 3)
}

func TestTreeCloudDAO_DeleteSyncStatus(t *testing.T) {
	ctx := context.Background()
	_, treeCloudDao := setupTestDB(t)

	acc := newTestCloudAccount("dao-sync-del-acc")
	_ = treeCloudDao.CreateCloudAccount(ctx, acc)
	for i := 0; i < 2; i++ {
		status := &model.CloudAccountSyncStatus{
			AccountId:    int(acc.ID),
			ResourceType: "ecs",
			Region:       "region-" + string(rune(i+'A')),
			Status:       "pending",
			LastSyncTime: time.Now(),
		}
		_ = treeCloudDao.CreateSyncStatus(ctx, status)
	}
	err := treeCloudDao.DeleteSyncStatus(ctx, int(acc.ID))
	assert.NoError(t, err)
	list, _ := treeCloudDao.ListSyncStatus(ctx, int(acc.ID))
	assert.Len(t, list, 0)
}

func TestTreeCloudDAO_CreateAuditLog(t *testing.T) {
	ctx := context.Background()
	_, treeCloudDao := setupTestDB(t)

	acc := newTestCloudAccount("dao-audit-acc")
	_ = treeCloudDao.CreateCloudAccount(ctx, acc)
	log := &model.CloudAccountAuditLog{
		AccountId: int(acc.ID),
		Operation: "create",
		Operator:  "tester",
		Details:   "test create",
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	}
	err := treeCloudDao.CreateAuditLog(ctx, log)
	assert.NoError(t, err)
}

func TestTreeCloudDAO_ListAuditLogs(t *testing.T) {
	ctx := context.Background()
	_, treeCloudDao := setupTestDB(t)

	acc := newTestCloudAccount("dao-audit-list-acc")
	_ = treeCloudDao.CreateCloudAccount(ctx, acc)
	for i := 0; i < 3; i++ {
		log := &model.CloudAccountAuditLog{
			AccountId: int(acc.ID),
			Operation: "op",
			Operator:  "tester",
			Details:   "log-" + string(rune(i+'A')),
			IPAddress: "127.0.0.1",
			UserAgent: "test-agent",
		}
		_ = treeCloudDao.CreateAuditLog(ctx, log)
	}
	resp, err := treeCloudDao.ListAuditLogs(ctx, int(acc.ID), 1, 2)
	assert.NoError(t, err)
	assert.True(t, resp.Total >= 3)
	assert.Len(t, resp.Items, 2)
}

func TestTreeCloudDAO_GetAuditLogsByOperation(t *testing.T) {
	ctx := context.Background()
	_, treeCloudDao := setupTestDB(t)

	acc := newTestCloudAccount("dao-audit-op-acc")
	_ = treeCloudDao.CreateCloudAccount(ctx, acc)
	for i := 0; i < 2; i++ {
		log := &model.CloudAccountAuditLog{
			AccountId: int(acc.ID),
			Operation: "special-op",
			Operator:  "tester",
			Details:   "log-" + string(rune(i+'A')),
			IPAddress: "127.0.0.1",
			UserAgent: "test-agent",
		}
		_ = treeCloudDao.CreateAuditLog(ctx, log)
	}
	resp, err := treeCloudDao.GetAuditLogsByOperation(ctx, "special-op", 1, 10)
	assert.NoError(t, err)
	assert.True(t, resp.Total >= 2)
}

func TestTreeCloudDAO_GetDecryptedSecretKey(t *testing.T) {
	ctx := context.Background()
	_, treeCloudDao := setupTestDB(t)

	acc := newTestCloudAccount("dao-crypto-acc")
	_ = treeCloudDao.CreateCloudAccount(ctx, acc)
	mockCrypto.On("DecryptSecretKey", acc.EncryptedSecret).Return("plain-secret", nil).Once()
	// 正常解密
	plain, err := treeCloudDao.GetDecryptedSecretKey(ctx, int(acc.ID))
	assert.NoError(t, err)
	assert.Equal(t, "plain-secret", plain)

	// 账户不存在
	_, err = treeCloudDao.GetDecryptedSecretKey(ctx, 99999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "获取云账户失败")

	// 解密失败
	acc2 := newTestCloudAccount("dao-crypto-acc2")
	_ = treeCloudDao.CreateCloudAccount(ctx, acc2)
	mockCrypto.On("DecryptSecretKey", acc2.EncryptedSecret).Return("", assert.AnError).Once()
	_, err = treeCloudDao.GetDecryptedSecretKey(ctx, int(acc2.ID))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "解密SecretKey失败")
}

func TestTreeCloudDAO_ReEncryptAccount(t *testing.T) {
	ctx := context.Background()
	_, treeCloudDao := setupTestDB(t)

	acc := newTestCloudAccount("dao-reenc-acc")
	_ = treeCloudDao.CreateCloudAccount(ctx, acc)
	mockCrypto.On("DecryptSecretKey", acc.EncryptedSecret).Return("plain-secret", nil).Once()
	mockCrypto.On("EncryptSecretKey", "plain-secret").Return("new-encrypted", nil).Once()
	// 正常流程
	err := treeCloudDao.ReEncryptAccount(ctx, int(acc.ID))
	assert.NoError(t, err)

	// 解密失败
	acc2 := newTestCloudAccount("dao-reenc-acc2")
	_ = treeCloudDao.CreateCloudAccount(ctx, acc2)
	mockCrypto.On("DecryptSecretKey", acc2.EncryptedSecret).Return("", assert.AnError).Once()
	err = treeCloudDao.ReEncryptAccount(ctx, int(acc2.ID))
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "解密当前SecretKey失败")
	}

	// 加密失败
	acc3 := newTestCloudAccount("dao-reenc-acc3")
	_ = treeCloudDao.CreateCloudAccount(ctx, acc3)
	mockCrypto.On("DecryptSecretKey", acc3.EncryptedSecret).Return("plain-secret", nil).Once()
	mockCrypto.On("EncryptSecretKey", "plain-secret").Return("", assert.AnError).Once()
	err = treeCloudDao.ReEncryptAccount(ctx, int(acc3.ID))
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "重新加密SecretKey失败")
	}
}
