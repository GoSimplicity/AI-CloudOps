package service

import (
	"context"
	"errors"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"strings"
	"time"
	"unicode/utf8"
)

func (s *managedCloudService) ResolveManagedHistory(ctx context.Context, req *model.ResolveCloudManagedHistoryReq) error {
	if req == nil {
		return errors.New("核对请求无效")
	}
	reason := strings.TrimSpace(req.Reason)
	if !req.ResourcesVerified || req.OperatorID <= 0 || req.HistoryID <= 0 || utf8.RuneCountInString(reason) < 10 || utf8.RuneCountInString(reason) > 1000 {
		return errors.New("必须确认资源已核对并填写10至1000字的理由")
	}
	if err := validateManagedRequest(req.CloudAccountID, req.CloudAccountRegionID, req.Kind, nil); err != nil {
		return err
	}
	account, region, err := s.accountRegion(ctx, req.CloudAccountID, req.CloudAccountRegionID)
	if err != nil {
		return err
	}
	if s.regions == nil {
		return errors.New("租约存储不可用")
	}
	lease, err := s.regions.AcquireRegionSyncLease(ctx, region.ID, time.Minute)
	if err != nil {
		return err
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = s.regions.ReleaseRegionSyncLease(cleanup, lease)
	}()
	copyReq := *req
	copyReq.Reason = reason
	// Acquiring a fresh fenced lease excludes active synchronization, including
	// old paused workers; this operation itself never calls a cloud API.
	return s.regions.ReviewManagedHistory(ctx, account, region, lease, &copyReq)
}
