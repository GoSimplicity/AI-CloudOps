package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/google/uuid"
	"sort"
	"strconv"
	"time"
)

func operationKey(id string, owner int) (string, error) {
	parsed, err := uuid.Parse(id)
	if err != nil || parsed.String() != id || owner <= 0 {
		return "", errors.New("操作ID或身份无效")
	}
	sum := sha256.Sum256([]byte(strconv.Itoa(owner) + ":" + id))
	return hex.EncodeToString(sum[:]), nil
}
func operationView(row *model.CloudManagedOperation) (*model.CloudManagedOperationView, error) {
	view := &model.CloudManagedOperationView{OperationID: row.OperationID, CloudAccountID: row.CloudAccountID, CloudAccountRegionID: row.CloudAccountRegionID, Kind: row.Kind, State: row.State, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
	if row.State == "running" {
		if time.Since(row.CreatedAt) > 2*time.Minute {
			view.State = "unknown"
		}
		return view, nil
	}
	if row.State != "completed" || len(row.Result) == 0 || len(row.Result) > 8192 {
		return nil, errors.New("操作状态不可用")
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(row.Result, &fields) != nil {
		return nil, errors.New("操作结果不可用")
	}
	for _, name := range []string{"operation_id", "status", "count", "new_count", "update_count", "retry_safe"} {
		if value, ok := fields[name]; !ok || string(value) == "null" {
			return nil, errors.New("操作结果不完整")
		}
	}
	var result model.CloudManagedSyncResult
	if json.Unmarshal(row.Result, &result) != nil || result.OperationID != row.OperationID || result.Count < 0 || result.NewCount < 0 || result.UpdateCount < 0 || result.NewCount > result.Count || result.UpdateCount > result.Count || result.NewCount > result.Count-result.UpdateCount {
		return nil, errors.New("操作结果不可用")
	}
	switch result.Status {
	case "success", "failed", "cancelled", "unknown":
	default:
		return nil, errors.New("操作结果状态无效")
	}
	if result.Status == "unknown" && result.RetrySafe {
		return nil, errors.New("未知操作结果不能声明安全重试")
	}
	view.Result = &result
	return view, nil
}
func (s *managedCloudService) GetManagedOperation(ctx context.Context, id string, owner int) (*model.CloudManagedOperationView, error) {
	key, err := operationKey(id, owner)
	if err != nil {
		return nil, err
	}
	if s.store == nil {
		return nil, errors.New("操作存储不可用")
	}
	row, err := s.store.GetOperation(ctx, key, owner)
	if err != nil {
		return nil, errors.New("操作不存在或不可访问")
	}
	return operationView(row)
}
func (s *managedCloudService) SyncManaged(ctx context.Context, req *model.SyncCloudManagedResourcesReq) (*model.CloudManagedSyncResult, error) {
	// Released clients without IDs retain the existing endpoint contract. New
	// clients must supply and retain one UUID for each intentional synchronization.
	if req == nil || req.OperationID == "" {
		return s.syncManaged(ctx, req)
	}
	key, err := operationKey(req.OperationID, req.OperatorID)
	if err != nil {
		return nil, err
	}
	if err := validateManagedRequest(req.CloudAccountID, req.CloudAccountRegionID, req.Kind, req.ResourceIDs); err != nil {
		return nil, err
	}
	if len(req.ResourceIDs) == 0 || s.store == nil {
		return nil, errors.New("操作存储或资源ID不可用")
	}
	ids := append([]string{}, req.ResourceIDs...)
	sort.Strings(ids)
	body, _ := json.Marshal(struct {
		Account, Region int
		Kind            string
		IDs             []string
	}{req.CloudAccountID, req.CloudAccountRegionID, req.Kind, ids})
	hash := sha256.Sum256(body)
	row := &model.CloudManagedOperation{Key: key, OperationID: req.OperationID, OperatorID: req.OperatorID, CloudAccountID: req.CloudAccountID, CloudAccountRegionID: req.CloudAccountRegionID, Kind: req.Kind, RequestHash: hex.EncodeToString(hash[:]), State: "running", ClaimToken: uuid.NewString()}
	created, previous, err := s.store.BeginOperation(ctx, row)
	if err != nil {
		return nil, errors.New("操作未确认，禁止重复提交")
	}
	if !created {
		view, err := operationView(previous)
		if err != nil {
			return nil, err
		}
		if view.State != "completed" {
			return nil, errors.New("操作仍在处理或结果未知，禁止重复执行")
		}
		return view.Result, nil
	}
	result, runErr := s.syncManaged(ctx, req)
	if runErr != nil {
		result = &model.CloudManagedSyncResult{Status: "failed", ErrorCategory: "request_rejected", RetrySafe: true}
	}
	if result == nil {
		return nil, errors.New("操作结果未知，请查询操作状态")
	}
	result.OperationID = req.OperationID
	payload, err := json.Marshal(result)
	if err != nil {
		return nil, errors.New("操作结果未知，请查询操作状态")
	}
	auditCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer cancel()
	if err := s.store.CompleteOperation(auditCtx, key, payload); err != nil {
		return &model.CloudManagedSyncResult{Status: "unknown", OperationID: req.OperationID, HistoryID: result.HistoryID, ErrorCategory: "operation_audit_unavailable", RetrySafe: false}, nil
	}
	if runErr != nil {
		return nil, runErr
	}
	return result, nil
}
