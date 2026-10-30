package service

import (
	"context"
	"errors"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/google/uuid"
	"strings"
	"time"
	"unicode/utf8"
)

func (s *managedCloudService) ReviewManagedOperation(ctx context.Context, req *model.ReviewCloudManagedOperationReq) (*model.CloudManagedOperationView, error) {
	if req == nil || req.OperatorID <= 0 || req.OwnerID < 0 {
		return nil, errors.New("核对请求无效")
	}
	owner := req.OperatorID
	if req.OwnerID > 0 {
		if req.OwnerID != req.OperatorID && !req.TrustedSuper {
			return nil, errors.New("无权核对其他用户的操作")
		}
		owner = req.OwnerID
	}
	key, err := operationKey(req.OperationID, owner)
	if err != nil {
		return nil, err
	}
	reason := strings.TrimSpace(req.Reason)
	if !req.ResourcesVerified || utf8.RuneCountInString(reason) < 10 || utf8.RuneCountInString(reason) > 1000 {
		return nil, errors.New("必须确认资源已核对并填写10至1000字理由")
	}
	if err := validateManagedRequest(req.CloudAccountID, req.CloudAccountRegionID, req.Kind, req.ResourceIDs); err != nil {
		return nil, err
	}
	if len(req.ResourceIDs) == 0 {
		return nil, errors.New("必须明确核对的资源ID")
	}
	account, region, err := s.accountRegion(ctx, req.CloudAccountID, req.CloudAccountRegionID)
	if err != nil {
		return nil, err
	}
	if s.regions == nil || s.store == nil {
		return nil, errors.New("操作存储不可用")
	}
	lease, err := s.regions.AcquireRegionSyncLease(ctx, region.ID, time.Minute)
	if err != nil {
		return nil, err
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = s.regions.ReleaseRegionSyncLease(cleanup, lease)
	}()
	row := &model.CloudManagedOperation{Key: key, OperationID: req.OperationID, OperatorID: owner, ReviewedBy: req.OperatorID, AdmissionVersion: 1, CloudAccountID: req.CloudAccountID, CloudAccountRegionID: req.CloudAccountRegionID, Kind: req.Kind, ResourceIDs: append(model.StringList{}, req.ResourceIDs...), RequestHash: managedRequestHash(req.CloudAccountID, req.CloudAccountRegionID, req.Kind, req.ResourceIDs), State: "reviewed", ClaimToken: uuid.NewString()}
	if err := s.regions.ReviewManagedOperation(ctx, account, region, lease, row, reason); err != nil {
		return nil, err
	}
	return s.GetManagedOperation(ctx, req.OperationID, owner)
}
