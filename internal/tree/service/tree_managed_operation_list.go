package service

import (
	"context"
	"errors"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

func (s *managedCloudService) ListManagedOperations(ctx context.Context, req *model.ListCloudManagedOperationsReq) (model.ListResp[*model.CloudManagedOperationView], error) {
	empty := model.ListResp[*model.CloudManagedOperationView]{}
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
	return model.ListResp[*model.CloudManagedOperationView]{Items: views, Total: total}, nil
}
