package service

import (
	"context"
	"errors"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

func (s *managedCloudService) ListManagedOperations(ctx context.Context, req *model.ListCloudManagedOperationsReq) (model.CloudManagedOperationList, error) {
	empty := model.CloudManagedOperationList{ScopeBlocked: true}
	if req == nil || req.ViewerID <= 0 {
		return empty, errors.New("缺少可信用户身份")
	}
	if req.OwnerID < 0 || (req.AllOwners && req.OwnerID > 0) {
		return empty, errors.New("操作用户筛选无效")
	}
	owner := req.ViewerID
	if req.AllOwners {
		if !req.TrustedSuper {
			return empty, errors.New("无权读取其他用户操作")
		}
		owner = 0
	}
	if req.OwnerID > 0 {
		if req.OwnerID != req.ViewerID && !req.TrustedSuper {
			return empty, errors.New("无权读取其他用户操作")
		}
		owner = req.OwnerID
	}
	if err := validateManagedRequest(req.CloudAccountID, req.CloudAccountRegionID, req.Kind, nil); err != nil {
		return empty, err
	}
	if _, _, err := s.accountRegion(ctx, req.CloudAccountID, req.CloudAccountRegionID); err != nil {
		return empty, err
	}
	if err := normalizeManagedPage(&req.ListCloudManagedResourcesReq); err != nil {
		return empty, err
	}
	if s.store == nil {
		return empty, errors.New("操作存储不可用")
	}
	rows, total, err := s.store.ListOperations(ctx, req, owner)
	if err != nil {
		return empty, errors.New("操作列表不可用")
	}
	views := make([]*model.CloudManagedOperationView, 0, len(rows))
	for _, row := range rows {
		view, err := operationView(row)
		if err != nil {
			return empty, err
		}
		views = append(views, view)
	}
	operationsBlocked, err := s.store.HasUnresolvedOperations(ctx, req.CloudAccountID, req.CloudAccountRegionID, req.Kind, "")
	if err != nil {
		return empty, errors.New("无法确认范围操作状态")
	}
	historyBlocked, err := s.store.HasUnresolvedHistory(ctx, req.CloudAccountID, req.CloudAccountRegionID, req.Kind)
	if err != nil {
		return empty, errors.New("无法确认范围历史状态")
	}
	return model.CloudManagedOperationList{Items: views, Total: total, ScopeBlocked: operationsBlocked || historyBlocked}, nil
}
