package cache

import (
	"context"
	"errors"
	"sort"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	configdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
)

// Stateful fixtures implement the new private API directly, with the same scope
// as SQL. No production reader falls back to the old full-type list operation.
func targetedFakePage(all []*model.MonitorConfig, poolID *int, kind int8, keys []string, namespace bool, page, size int) ([]*model.MonitorConfig, int64, error) {
	if page <= 0 || size <= 0 {
		return nil, 0, configdao.ErrInvalidGeneratedConfigQuery
	}
	wanted := map[string]bool{}
	for _, key := range keys {
		wanted[key] = true
	}
	var rows []*model.MonitorConfig
	for _, row := range all {
		if row == nil || row.DeletedAt != 0 || row.ConfigType != kind || poolID != nil && row.PoolID != *poolID || namespace && !wanted[row.InstanceIP] {
			continue
		}
		copy := *row
		rows = append(rows, &copy)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
	total := len(rows)
	start := (page - 1) * size
	if start >= total {
		return nil, int64(total), nil
	}
	end := start + size
	if end > total {
		end = total
	}
	return rows[start:end], int64(total), nil
}
func targetedFakeMapRows(rows map[int]*model.MonitorConfig) []*model.MonitorConfig {
	var out []*model.MonitorConfig
	for _, row := range rows {
		out = append(out, row)
	}
	return out
}
func (s *recordReconcileDB) GetGeneratedConfigPoolCandidates(ctx context.Context, req configdao.GeneratedConfigPoolQuery) ([]*model.MonitorConfig, int64, error) {
	if s.failure == "db read" {
		return nil, 0, errors.New("fixture metadata failure")
	}
	if s.projection != nil {
		return s.projection.GetGeneratedConfigPoolCandidates(ctx, req)
	}
	return targetedFakePage(targetedFakeMapRows(s.rows), &req.PoolID, req.ConfigType, nil, false, req.Page, req.Size)
}
func (s *recordReconcileDB) GetGeneratedConfigNamespacePage(ctx context.Context, req configdao.GeneratedConfigKeysQuery) ([]*model.MonitorConfig, int64, error) {
	if s.failure == "db read" {
		return nil, 0, errors.New("fixture metadata failure")
	}
	if s.projection != nil {
		return s.projection.GetGeneratedConfigNamespacePage(ctx, req)
	}
	return targetedFakePage(targetedFakeMapRows(s.rows), nil, req.ConfigType, req.InstanceKeys, true, req.Page, req.Size)
}
func (s *promReconcileDB) GetGeneratedConfigPoolCandidates(_ context.Context, req configdao.GeneratedConfigPoolQuery) ([]*model.MonitorConfig, int64, error) {
	if s.failure == "db read" {
		return nil, 0, errors.New("fixture metadata failure")
	}
	return targetedFakePage(targetedFakeMapRows(s.rows), &req.PoolID, req.ConfigType, nil, false, req.Page, req.Size)
}
func (s *promReconcileDB) GetGeneratedConfigNamespacePage(_ context.Context, req configdao.GeneratedConfigKeysQuery) ([]*model.MonitorConfig, int64, error) {
	if s.failure == "db read" {
		return nil, 0, errors.New("fixture metadata failure")
	}
	return targetedFakePage(targetedFakeMapRows(s.rows), nil, req.ConfigType, req.InstanceKeys, true, req.Page, req.Size)
}
func (s *recordingPublicationConfigs) targetedRows() []*model.MonitorConfig {
	var rows []*model.MonitorConfig
	for _, row := range s.rows {
		rows = append(rows, row)
	}
	return rows
}
func (s *recordingPublicationConfigs) GetGeneratedConfigPoolCandidates(_ context.Context, req configdao.GeneratedConfigPoolQuery) ([]*model.MonitorConfig, int64, error) {
	return targetedFakePage(s.targetedRows(), &req.PoolID, req.ConfigType, nil, false, req.Page, req.Size)
}
func (s *recordingPublicationConfigs) GetGeneratedConfigNamespacePage(_ context.Context, req configdao.GeneratedConfigKeysQuery) ([]*model.MonitorConfig, int64, error) {
	return targetedFakePage(s.targetedRows(), nil, req.ConfigType, req.InstanceKeys, true, req.Page, req.Size)
}

// Existing contradiction fixtures keep their exact malformed page/total/failure
// behavior on the newly used API. Their first-page rows are now pool candidates.
func (d recordIncompleteMetadata) GetGeneratedConfigPoolCandidates(ctx context.Context, req configdao.GeneratedConfigPoolQuery) ([]*model.MonitorConfig, int64, error) {
	return d.GetMonitorConfigList(ctx, &model.GetMonitorConfigListReq{ListReq: model.ListReq{Page: req.Page, Size: req.Size}, ConfigType: &req.ConfigType})
}
func (d promIncompleteMetadata) GetGeneratedConfigPoolCandidates(ctx context.Context, req configdao.GeneratedConfigPoolQuery) ([]*model.MonitorConfig, int64, error) {
	return d.GetMonitorConfigList(ctx, &model.GetMonitorConfigListReq{ListReq: model.ListReq{Page: req.Page, Size: req.Size}, ConfigType: &req.ConfigType})
}

// The old reader tests retain their original GetList injection. Only the two
// publisher policy cases need new API behavior, which filters query scope in SQL.
func (d *configSnapshotPages) GetGeneratedConfigPoolCandidates(ctx context.Context, req configdao.GeneratedConfigPoolQuery) ([]*model.MonitorConfig, int64, error) {
	d.calls++
	rows, _, err := d.read(ctx, &model.GetMonitorConfigListReq{ListReq: model.ListReq{Page: req.Page, Size: req.Size}, ConfigType: &req.ConfigType})
	if err != nil {
		return nil, 0, err
	}
	return targetedFakePage(rows, &req.PoolID, req.ConfigType, nil, false, req.Page, req.Size)
}
func (d *configSnapshotPages) GetGeneratedConfigNamespacePage(ctx context.Context, req configdao.GeneratedConfigKeysQuery) ([]*model.MonitorConfig, int64, error) {
	d.calls++
	rows, _, err := d.read(ctx, &model.GetMonitorConfigListReq{ListReq: model.ListReq{Page: req.Page, Size: req.Size}, ConfigType: &req.ConfigType})
	if err != nil {
		return nil, 0, err
	}
	return targetedFakePage(rows, nil, req.ConfigType, req.InstanceKeys, true, req.Page, req.Size)
}

// Support-lifecycle fixtures intentionally share this stateful metadata store
// across domains; implement the private query contract for those consumers too.
func (s *alertRulePubDB) GetGeneratedConfigPoolCandidates(ctx context.Context, req configdao.GeneratedConfigPoolQuery) ([]*model.MonitorConfig, int64, error) {
	if s.projection != nil {
		return s.projection.base.GetGeneratedConfigPoolCandidates(ctx, req)
	}
	if s.failure == "db read" {
		return nil, 0, errors.New("fixture metadata failure")
	}
	return targetedFakePage(targetedFakeMapRows(s.rows), &req.PoolID, req.ConfigType, nil, false, req.Page, req.Size)
}
func (s *alertRulePubDB) GetGeneratedConfigNamespacePage(ctx context.Context, req configdao.GeneratedConfigKeysQuery) ([]*model.MonitorConfig, int64, error) {
	if s.projection != nil {
		return s.projection.base.GetGeneratedConfigNamespacePage(ctx, req)
	}
	if s.failure == "db read" {
		return nil, 0, errors.New("fixture metadata failure")
	}
	return targetedFakePage(targetedFakeMapRows(s.rows), nil, req.ConfigType, req.InstanceKeys, true, req.Page, req.Size)
}

func (d alertRuleIncompleteMetadata) GetGeneratedConfigPoolCandidates(ctx context.Context, req configdao.GeneratedConfigPoolQuery) ([]*model.MonitorConfig, int64, error) {
	return d.GetMonitorConfigList(ctx, &model.GetMonitorConfigListReq{ListReq: model.ListReq{Page: req.Page, Size: req.Size}, ConfigType: &req.ConfigType})
}
