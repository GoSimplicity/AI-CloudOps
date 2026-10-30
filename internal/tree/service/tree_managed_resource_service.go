package service

import (
	"context"
	"errors"
	"regexp"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/dao"
	treeUtils "github.com/GoSimplicity/AI-CloudOps/internal/tree/utils"
	"go.uber.org/zap"
)

type managedReader interface {
	ListRDS(context.Context, []string) ([]model.AliyunRDSInstance, error)
	ListSLB(context.Context, []string) ([]model.AliyunLoadBalancer, error)
	ListCache(context.Context, []string) ([]model.AliyunCacheInstance, error)
}
type ManagedCloudService interface {
	ResolveManagedHistory(context.Context, *model.ResolveCloudManagedHistoryReq) error
	ListManaged(context.Context, *model.ListCloudManagedResourcesReq) (model.ListResp[*model.CloudManagedResource], error)
	GetManaged(context.Context, *model.ListCloudManagedResourcesReq, int) (*model.CloudManagedResource, error)
	ListManagedHistory(context.Context, *model.ListCloudManagedResourcesReq) (model.ListResp[*model.CloudManagedSyncHistory], error)
	SyncManaged(context.Context, *model.SyncCloudManagedResourcesReq) (*model.CloudManagedSyncResult, error)
}

func (s *managedCloudService) GetManaged(ctx context.Context, req *model.ListCloudManagedResourcesReq, id int) (*model.CloudManagedResource, error) {
	if req == nil || id <= 0 {
		return nil, errors.New("请求无效")
	}
	if err := validateManagedRequest(req.CloudAccountID, req.CloudAccountRegionID, req.Kind, nil); err != nil {
		return nil, err
	}
	if _, _, err := s.accountRegion(ctx, req.CloudAccountID, req.CloudAccountRegionID); err != nil {
		return nil, err
	}
	if s.store == nil {
		return nil, errors.New("资源存储不可用")
	}
	row, err := s.store.Get(ctx, req, id)
	if err != nil {
		return nil, errors.New("托管资源不存在或不可用")
	}
	return row, nil
}

type managedCloudService struct {
	logger   *zap.Logger
	accounts dao.CloudAccountDAO
	regions  dao.TreeCloudDAO
	store    dao.ManagedCloudDAO
	factory  func(treeUtils.CloudCredentials, string) (managedReader, error)
}

func NewManagedCloudService(logger *zap.Logger, accounts dao.CloudAccountDAO, regions dao.TreeCloudDAO, store dao.ManagedCloudDAO) ManagedCloudService {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &managedCloudService{logger: logger, accounts: accounts, regions: regions, store: store, factory: func(c treeUtils.CloudCredentials, r string) (managedReader, error) {
		return treeUtils.NewAliyunManagedResourceClient(c, r)
	}}
}

var managedIDPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-]{1,99}$`)

func validateManagedRequest(accountID, regionID int, kind string, ids []string) error {
	if accountID <= 0 || regionID <= 0 || (kind != "rds" && kind != "clb" && kind != "cache") {
		return errors.New("账户、地域或资源类型无效")
	}
	if ids != nil {
		if len(ids) < 1 || len(ids) > 100 {
			return errors.New("必须指定1至100个资源ID")
		}
		seen := map[string]bool{}
		for _, id := range ids {
			if !managedIDPattern.MatchString(id) || seen[id] {
				return errors.New("资源ID无效或重复")
			}
			seen[id] = true
		}
	}
	return nil
}
func (s *managedCloudService) accountRegion(ctx context.Context, accountID, regionID int) (*model.CloudAccount, *model.CloudAccountRegion, error) {
	if s.accounts == nil {
		return nil, nil, errors.New("云账户服务不可用")
	}
	account, err := s.accounts.GetByID(ctx, accountID)
	if err != nil || account == nil {
		return nil, nil, errors.New("云账户不存在或不可用")
	}
	if account.Provider != model.ProviderAliyun {
		return nil, nil, errors.New("当前仅支持阿里云指定资源")
	}
	for _, region := range account.Regions {
		if region != nil && region.ID == regionID && region.CloudAccountID == accountID {
			return account, region, nil
		}
	}
	return nil, nil, errors.New("地域不存在或不属于该账户")
}
func normalizeManagedPage(req *model.ListCloudManagedResourcesReq) error {
	if req.Page > 1000000 {
		return errors.New("页码超出范围")
	}
	if req.Page < 1 {
		req.Page = 1
	}
	if req.Size < 1 {
		req.Size = 20
	}
	if req.Size > 100 {
		req.Size = 100
	}
	return nil
}
func (s *managedCloudService) ListManaged(ctx context.Context, req *model.ListCloudManagedResourcesReq) (model.ListResp[*model.CloudManagedResource], error) {
	if req == nil {
		return model.ListResp[*model.CloudManagedResource]{}, errors.New("请求无效")
	}
	if err := validateManagedRequest(req.CloudAccountID, req.CloudAccountRegionID, req.Kind, nil); err != nil {
		return model.ListResp[*model.CloudManagedResource]{}, err
	}
	if _, _, err := s.accountRegion(ctx, req.CloudAccountID, req.CloudAccountRegionID); err != nil {
		return model.ListResp[*model.CloudManagedResource]{}, err
	}
	if s.store == nil {
		return model.ListResp[*model.CloudManagedResource]{}, errors.New("资源存储不可用")
	}
	if err := normalizeManagedPage(req); err != nil {
		return model.ListResp[*model.CloudManagedResource]{}, err
	}
	rows, total, err := s.store.List(ctx, req)
	if err != nil {
		return model.ListResp[*model.CloudManagedResource]{}, errors.New("托管资源读取失败")
	}
	return model.ListResp[*model.CloudManagedResource]{Items: rows, Total: total}, nil
}
func (s *managedCloudService) ListManagedHistory(ctx context.Context, req *model.ListCloudManagedResourcesReq) (model.ListResp[*model.CloudManagedSyncHistory], error) {
	if req == nil {
		return model.ListResp[*model.CloudManagedSyncHistory]{}, errors.New("请求无效")
	}
	if err := validateManagedRequest(req.CloudAccountID, req.CloudAccountRegionID, req.Kind, nil); err != nil {
		return model.ListResp[*model.CloudManagedSyncHistory]{}, err
	}
	if _, _, err := s.accountRegion(ctx, req.CloudAccountID, req.CloudAccountRegionID); err != nil {
		return model.ListResp[*model.CloudManagedSyncHistory]{}, err
	}
	if s.store == nil {
		return model.ListResp[*model.CloudManagedSyncHistory]{}, errors.New("资源存储不可用")
	}
	if err := normalizeManagedPage(req); err != nil {
		return model.ListResp[*model.CloudManagedSyncHistory]{}, err
	}
	rows, total, err := s.store.ListHistory(ctx, req)
	if err != nil {
		return model.ListResp[*model.CloudManagedSyncHistory]{}, errors.New("同步历史读取失败")
	}
	return model.ListResp[*model.CloudManagedSyncHistory]{Items: rows, Total: total}, nil
}
func (s *managedCloudService) SyncManaged(ctx context.Context, req *model.SyncCloudManagedResourcesReq) (result *model.CloudManagedSyncResult, resultErr error) {
	if req == nil {
		return nil, errors.New("请求无效")
	}
	if err := validateManagedRequest(req.CloudAccountID, req.CloudAccountRegionID, req.Kind, req.ResourceIDs); err != nil {
		return nil, err
	}
	if len(req.ResourceIDs) == 0 || req.OperatorID <= 0 {
		return nil, errors.New("资源ID或操作人无效")
	}
	account, region, err := s.accountRegion(ctx, req.CloudAccountID, req.CloudAccountRegionID)
	if err != nil {
		return nil, err
	}
	if account.Status != model.CloudAccountEnabled || region.Status != model.CloudAccountRegionEnabled {
		return nil, errors.New("账户或地域已禁用")
	}
	credentials, err := treeUtils.LoadCloudAccountCredentials(account)
	if err != nil {
		return nil, errors.New("账户凭据不可用")
	}
	if s.regions == nil || s.store == nil || s.factory == nil {
		return nil, errors.New("同步服务不可用")
	}
	result = &model.CloudManagedSyncResult{Status: "failed", RetrySafe: true}
	operationCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	lockCtx, stop := context.WithTimeout(operationCtx, 5*time.Second)
	lease, err := s.regions.AcquireRegionSyncLease(lockCtx, region.ID, 5*time.Minute)
	stop()
	if err != nil {
		auditCtx, stopAudit := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
		defer stopAudit()
		status, category := "failed", "lease_failed"
		if errors.Is(err, dao.ErrRegionSyncBusy) {
			category = "busy"
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			status, category = "cancelled", "cancelled"
		}
		now := time.Now().UTC()
		history := &model.CloudManagedSyncHistory{CloudAccountID: account.ID, CloudAccountRegionID: region.ID, Kind: req.Kind, ResourceIDs: append(model.StringList{}, req.ResourceIDs...), Status: status, ErrorCategory: category, OperatorID: req.OperatorID, StartedAt: now, FinishedAt: &now}
		if s.store.CreateHistory(auditCtx, history) != nil {
			result.ErrorCategory = "audit_unavailable"
			result.RetrySafe = false
			return result, nil
		}
		result.HistoryID = history.ID
		result.Status = status
		result.ErrorCategory = category
		return result, nil
	}
	release := true
	defer func() {
		if release {
			releaseCtx, stopRelease := context.WithTimeout(context.Background(), 3*time.Second)
			defer stopRelease()
			if s.regions.ReleaseRegionSyncLease(releaseCtx, lease) != nil {
				s.logger.Warn("托管资源租约释放失败")
			}
		}
	}()
	blocked, checkErr := s.store.HasUnresolvedHistory(operationCtx, account.ID, region.ID, req.Kind)
	if checkErr != nil {
		return nil, errors.New("无法核对同步历史，禁止重试")
	}
	if blocked {
		return nil, errors.New("存在未核对的同步结果，请先人工核对并记录处置")
	}
	history := &model.CloudManagedSyncHistory{CloudAccountID: account.ID, CloudAccountRegionID: region.ID, Kind: req.Kind, ResourceIDs: append(model.StringList{}, req.ResourceIDs...), OperationID: lease.Token, LeaseExpiresMS: lease.ExpiresEpochMS, Status: "running", OperatorID: req.OperatorID, StartedAt: time.Now().UTC()}
	if s.store.CreateHistory(operationCtx, history) != nil {
		result.ErrorCategory = "audit_unavailable"
		result.RetrySafe = false
		return result, nil
	}
	result.HistoryID = history.ID
	defer func() {
		if resultErr == nil {
			return
		}
		status, category := "failed", "sync_failed"
		if errors.Is(resultErr, dao.ErrRegionSyncCommitUnknown) {
			status, category = "unknown", "commit_unknown"
			release = false
			result.RetrySafe = false
		} else if errors.Is(resultErr, context.Canceled) || errors.Is(resultErr, context.DeadlineExceeded) {
			status, category = "cancelled", "cancelled"
		}
		result.Status = status
		result.ErrorCategory = category
		auditCtx, stopAudit := context.WithTimeout(context.Background(), 3*time.Second)
		defer stopAudit()
		if s.store.FinishHistory(auditCtx, history.ID, lease.Token, status, category) != nil {
			result.RetrySafe = false
			if status == "unknown" {
				result.ErrorCategory = "commit_unknown_audit_unavailable"
			} else {
				result.ErrorCategory = "audit_unavailable"
			}
		}
		resultErr = nil
	}()
	reader, err := s.factory(credentials, region.Region)
	if err != nil {
		return result, err
	}
	items := make([]*model.CloudManagedResource, 0, len(req.ResourceIDs))
	switch req.Kind {
	case "rds":
		values, e := reader.ListRDS(operationCtx, req.ResourceIDs)
		if e != nil {
			if err := operationCtx.Err(); err != nil {
				return result, err
			}
			return result, e
		}
		for i := range values {
			v := values[i]
			items = append(items, &model.CloudManagedResource{CloudAccountID: account.ID, CloudAccountRegionID: region.ID, Kind: "rds", ProviderID: v.InstanceID, RDS: &v})
		}
	case "clb":
		values, e := reader.ListSLB(operationCtx, req.ResourceIDs)
		if e != nil {
			if err := operationCtx.Err(); err != nil {
				return result, err
			}
			return result, e
		}
		for i := range values {
			v := values[i]
			items = append(items, &model.CloudManagedResource{CloudAccountID: account.ID, CloudAccountRegionID: region.ID, Kind: "clb", ProviderID: v.InstanceID, CLB: &v})
		}
	case "cache":
		values, e := reader.ListCache(operationCtx, req.ResourceIDs)
		if e != nil {
			if err := operationCtx.Err(); err != nil {
				return result, err
			}
			return result, e
		}
		for i := range values {
			v := values[i]
			items = append(items, &model.CloudManagedResource{CloudAccountID: account.ID, CloudAccountRegionID: region.ID, Kind: "cache", ProviderID: v.InstanceID, Cache: &v})
		}
	}
	if len(items) != len(req.ResourceIDs) {
		return result, errors.New("云端结果不完整")
	}
	seen := map[string]bool{}
	expected := map[string]bool{}
	for _, id := range req.ResourceIDs {
		expected[id] = true
	}
	for _, item := range items {
		validRegion := (item.RDS != nil && item.RDS.Region == region.Region) || (item.CLB != nil && item.CLB.Region == region.Region) || (item.Cache != nil && item.Cache.Region == region.Region)
		if !expected[item.ProviderID] || seen[item.ProviderID] || !validRegion {
			return result, errors.New("云端结果不匹配")
		}
		seen[item.ProviderID] = true
	}
	if err := operationCtx.Err(); err != nil {
		return result, err
	}
	syncedAt := time.Now().UTC()
	for _, item := range items {
		item.LastSyncTime = syncedAt
	}
	computed := model.CloudManagedSyncResult{Status: "success", Count: len(items), RetrySafe: true, HistoryID: history.ID}
	err = s.regions.ApplyRegionSync(operationCtx, account, region, lease, func(tx dao.TreeCloudDAO) error {
		managed := tx.ManagedResourceStore()
		if managed == nil {
			return errors.New("资源存储不可用")
		}
		for _, item := range items {
			if err := operationCtx.Err(); err != nil {
				return err
			}
			created, err := managed.Upsert(operationCtx, item)
			if err != nil {
				return err
			}
			if created {
				computed.NewCount++
			} else {
				computed.UpdateCount++
			}
		}
		return managed.CompleteHistory(operationCtx, history.ID, lease.Token, computed)
	})
	if err != nil {
		return result, err
	}
	return &computed, nil
}
