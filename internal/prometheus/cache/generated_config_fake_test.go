package cache

import (
	"context"
	"reflect"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	configdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
)

// Existing stateful publication fixtures keep their fault injection and counters.
// This adapter only supplies the new ownership-aware DAO surface; native SQL
// behavior is exercised separately using the actual SQLite/MySQL DAO tests.
func fakeGeneratedChanges(changes []configdao.GeneratedConfigUpdate, find func(int) *model.MonitorConfig) ([]*model.MonitorConfig, error) {
	rows := make([]*model.MonitorConfig, 0, len(changes))
	for _, change := range changes {
		current := find(change.Expected.ID)
		if configdao.GeneratedConfigOwnerOf(current) != change.Expected {
			if current == nil || !reflect.DeepEqual(*current, change.Desired) {
				return nil, configdao.ErrGeneratedConfigConflict
			}
		}
		copy := change.Desired
		rows = append(rows, &copy)
	}
	return rows, nil
}
func fakeConfigByID(rows map[int]*model.MonitorConfig, id int) *model.MonitorConfig { return rows[id] }
func fakeConfigByInstanceID(rows map[string]*model.MonitorConfig, id int) *model.MonitorConfig {
	for _, row := range rows {
		if row.ID == id {
			return row
		}
	}
	return nil
}
func fakeGeneratedDelete(expected configdao.GeneratedConfigOwner, find func(int) *model.MonitorConfig, remove func(int) error) error {
	if configdao.GeneratedConfigOwnerOf(find(expected.ID)) != expected {
		return configdao.ErrGeneratedConfigConflict
	}
	return remove(expected.ID)
}
func (s *alertContractConfigs) UpdateGeneratedMonitorConfigs(ctx context.Context, changes []configdao.GeneratedConfigUpdate) error {
	rows, err := fakeGeneratedChanges(changes, func(id int) *model.MonitorConfig { return s.rows[id] })
	if err != nil {
		return err
	}
	return s.BatchUpdateMonitorConfigs(ctx, rows)
}
func (s *alertContractConfigs) DeleteGeneratedMonitorConfig(ctx context.Context, expected configdao.GeneratedConfigOwner) error {
	if s.rows[expected.ID] == nil && s.retiredOwners[expected.ID] == expected {
		for _, row := range s.rows {
			if row.ConfigType == expected.ConfigType && row.InstanceIP == expected.InstanceIP {
				return configdao.ErrGeneratedConfigConflict
			}
		}
		return nil
	}
	err := fakeGeneratedDelete(expected, func(id int) *model.MonitorConfig { return s.rows[id] }, func(id int) error { return s.DeleteMonitorConfig(ctx, id) })
	if err == nil {
		if s.retiredOwners == nil {
			s.retiredOwners = map[int]configdao.GeneratedConfigOwner{}
		}
		s.retiredOwners[expected.ID] = expected
	}
	return err
}
func (s *alertRulePubDB) UpdateGeneratedMonitorConfigs(ctx context.Context, changes []configdao.GeneratedConfigUpdate) error {
	rows, err := fakeGeneratedChanges(changes, func(id int) *model.MonitorConfig { return s.rows[id] })
	if err != nil {
		return err
	}
	return s.BatchUpdateMonitorConfigs(ctx, rows)
}
func (s *alertRulePubDB) DeleteGeneratedMonitorConfig(ctx context.Context, expected configdao.GeneratedConfigOwner) error {
	return fakeGeneratedDelete(expected, func(id int) *model.MonitorConfig { return s.rows[id] }, func(id int) error { return s.DeleteMonitorConfig(ctx, id) })
}
func (s *recordReconcileDB) UpdateGeneratedMonitorConfigs(ctx context.Context, changes []configdao.GeneratedConfigUpdate) error {
	rows, err := fakeGeneratedChanges(changes, func(id int) *model.MonitorConfig { return s.rows[id] })
	if err != nil {
		return err
	}
	return s.BatchUpdateMonitorConfigs(ctx, rows)
}
func (s *recordReconcileDB) DeleteGeneratedMonitorConfig(ctx context.Context, expected configdao.GeneratedConfigOwner) error {
	return fakeGeneratedDelete(expected, func(id int) *model.MonitorConfig { return s.rows[id] }, func(id int) error { return s.DeleteMonitorConfig(ctx, id) })
}
func (s *promReconcileDB) UpdateGeneratedMonitorConfigs(ctx context.Context, changes []configdao.GeneratedConfigUpdate) error {
	rows, err := fakeGeneratedChanges(changes, func(id int) *model.MonitorConfig { return s.rows[id] })
	if err != nil {
		return err
	}
	return s.BatchUpdateMonitorConfigs(ctx, rows)
}
func (s *promReconcileDB) DeleteGeneratedMonitorConfig(ctx context.Context, expected configdao.GeneratedConfigOwner) error {
	return fakeGeneratedDelete(expected, func(id int) *model.MonitorConfig { return s.rows[id] }, func(id int) error { return s.DeleteMonitorConfig(ctx, id) })
}
func (s *recordingPublicationConfigs) UpdateGeneratedMonitorConfigs(ctx context.Context, changes []configdao.GeneratedConfigUpdate) error {
	rows, err := fakeGeneratedChanges(changes, func(id int) *model.MonitorConfig { return fakeConfigByInstanceID(s.rows, id) })
	if err != nil {
		return err
	}
	return s.BatchUpdateMonitorConfigs(ctx, rows)
}
func (s *recordingPublicationConfigs) DeleteGeneratedMonitorConfig(ctx context.Context, expected configdao.GeneratedConfigOwner) error {
	return fakeGeneratedDelete(expected, func(id int) *model.MonitorConfig { return fakeConfigByInstanceID(s.rows, id) }, func(id int) error { return s.DeleteMonitorConfig(ctx, id) })
}
func (s alertCanceledWrite) UpdateGeneratedMonitorConfigs(ctx context.Context, changes []configdao.GeneratedConfigUpdate) error {
	rows, err := fakeGeneratedChanges(changes, func(id int) *model.MonitorConfig { return s.rows[id] })
	if err != nil {
		return err
	}
	return s.BatchUpdateMonitorConfigs(ctx, rows)
}

func (s *alertContractConfigs) GetGeneratedMonitorConfigByID(_ context.Context, id int) (*model.MonitorConfig, error) {
	if row := s.rows[id]; row != nil {
		copy := *row
		return &copy, nil
	}
	if owner, ok := s.retiredOwners[id]; ok {
		return &model.MonitorConfig{Model: model.Model{ID: owner.ID, DeletedAt: 1}, PoolID: owner.PoolID, Name: owner.Name, InstanceIP: owner.InstanceIP, ConfigType: owner.ConfigType}, nil
	}
	return nil, configdao.ErrGeneratedConfigConflict
}
