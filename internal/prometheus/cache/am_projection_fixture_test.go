package cache

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	configdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type amProjectionFixture struct {
	configdao.MonitorConfigDAO
	base         *recordProjectionFixture
	journal      configdao.AMProjectionJournalDAO
	history      configdao.AMProjectionHistoryReader
	view         *alertContractConfigs
	currentRedis func() redis.Cmdable
}

func newAMProjectionFixture(t *testing.T, view *alertContractConfigs) *amProjectionFixture {
	t.Helper()
	f := &amProjectionFixture{view: view}
	base := newRecordProjectionFixture(t, nil, func([]*model.MonitorConfig) {
		if err := f.refresh(); err != nil {
			t.Fatal("AM SQL assertion view unavailable")
		}
	}, nil, func() { view.writes++ })
	f.base = base
	f.MonitorConfigDAO = base.MonitorConfigDAO
	view.MonitorConfigDAO = f
	f.history = configdao.NewAMProjectionHistoryReader(zap.NewNop(), base.db)
	f.journal = &amFixtureJournal{AMProjectionJournalDAO: configdao.NewAMProjectionJournalDAO(zap.NewNop(), base.db), fixture: f}
	after := func(mode string) func(*gorm.DB) {
		return func(tx *gorm.DB) {
			if tx.Statement.SkipHooks || tx.Statement.Table != "cl_monitor_configs" {
				return
			}
			kind := amFixtureWriteKind(tx)
			fault := view.failure
			if fault == "main DB" && kind == 2 || fault == "webhook DB" && kind == 5 || fault == "webhook DB delete" && kind == 5 && mode == "delete" {
				tx.AddError(errors.New("injected AM metadata failure"))
			}
		}
	}
	if base.db.Callback().Create().After("gorm:create").Register("am_fixture:create", after("create")) != nil || base.db.Callback().Update().After("gorm:update").Register("am_fixture:update", after("update")) != nil || base.db.Callback().Delete().After("gorm:delete").Register("am_fixture:delete", after("delete")) != nil {
		t.Fatal("AM SQL fault callbacks unavailable")
	}
	if base.db.Callback().Query().After("gorm:query").Register("am_fixture:read", func(tx *gorm.DB) {
		if !tx.Statement.SkipHooks && tx.Statement.Table == "cl_monitor_config_pool_controls" && f.currentRedis != nil {
			if client := amFixtureBaseRedis(f.currentRedis()); client != nil && client.failure == "hash read" {
				tx.AddError(errors.New("injected AM source/control read failure"))
			}
		}
		if tx.Statement.SkipHooks || tx.Statement.Table != "cl_monitor_configs" {
			return
		}
		if view.failure == "DB read" || view.failure == "webhook list" && amFixtureWriteKind(tx) == 5 {
			tx.AddError(errors.New("injected AM metadata read failure"))
		}
	}) != nil {
		t.Fatal("AM SQL read fault unavailable")
	}
	return f
}
func amFixtureWriteKind(tx *gorm.DB) int8 {
	if row, ok := tx.Statement.Dest.(*model.MonitorConfig); ok && row.ConfigType != 0 {
		return row.ConfigType
	}
	text := strings.ReplaceAll(tx.Statement.SQL.String(), "`", "")
	match := regexp.MustCompile(`(?i)config_type\s*=\s*\?`).FindStringIndex(text)
	if match != nil {
		at := strings.Count(text[:match[1]], "?") - 1
		if at >= 0 && at < len(tx.Statement.Vars) {
			value := reflect.ValueOf(tx.Statement.Vars[at])
			if value.IsValid() && value.Kind() >= reflect.Int && value.Kind() <= reflect.Int64 {
				return int8(value.Int())
			}
		}
	}
	return 0
}
func (f *amProjectionFixture) seed() error {
	f.base.seeding = true
	defer func() { f.base.seeding = false }()
	for _, input := range f.view.rows {
		if input == nil {
			continue
		}
		row := *input
		var actual model.MonitorConfig
		err := f.base.db.Unscoped().First(&actual, row.ID).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err == nil && reflect.DeepEqual(actual, row) {
			continue
		}
		if err := f.base.db.Session(&gorm.Session{SkipHooks: true}).Unscoped().Save(&row).Error; err != nil {
			return err
		}
	}
	for id, owner := range f.view.retiredOwners {
		var row model.MonitorConfig
		if err := f.base.db.Unscoped().First(&row, id).Error; err != nil {
			return err
		}
		if configdao.GeneratedConfigOwnerOf(&row) != owner {
			row.ID = owner.ID
			row.PoolID = owner.PoolID
			row.ConfigType = owner.ConfigType
			row.InstanceIP = owner.InstanceIP
			row.Name = owner.Name
			if err := f.base.db.Session(&gorm.Session{SkipHooks: true}).Unscoped().Save(&row).Error; err != nil {
				return err
			}
		}
	}
	return nil
}
func (f *amProjectionFixture) refresh() error {
	var rows []*model.MonitorConfig
	if err := f.base.db.Unscoped().Where("config_type IN ?", []int8{2, 5}).Order("id").Find(&rows).Error; err != nil {
		return err
	}
	for id, row := range f.view.rows {
		if row.ConfigType == 2 || row.ConfigType == 5 {
			delete(f.view.rows, id)
		}
	}
	if f.view.retiredOwners == nil {
		f.view.retiredOwners = map[int]configdao.GeneratedConfigOwner{}
	}
	for _, row := range rows {
		if row.ID > f.view.next {
			f.view.next = row.ID
		}
		if row.DeletedAt != 0 {
			f.view.retiredOwners[row.ID] = configdao.GeneratedConfigOwnerOf(row)
		} else {
			copy := *row
			f.view.rows[row.ID] = &copy
			delete(f.view.retiredOwners, row.ID)
		}
	}
	return nil
}
func (f *amProjectionFixture) GetGeneratedConfigPoolCandidates(ctx context.Context, req configdao.GeneratedConfigPoolQuery) ([]*model.MonitorConfig, int64, error) {
	if f.view.failure == "DB read" || f.view.failure == "webhook list" && req.ConfigType == 5 {
		return nil, 0, errors.New("injected metadata read failure")
	}
	if err := f.seed(); err != nil {
		return nil, 0, err
	}
	return f.MonitorConfigDAO.GetGeneratedConfigPoolCandidates(ctx, req)
}
func (f *amProjectionFixture) GetGeneratedConfigNamespacePage(ctx context.Context, req configdao.GeneratedConfigKeysQuery) ([]*model.MonitorConfig, int64, error) {
	if f.view.failure == "DB read" {
		return nil, 0, errors.New("injected metadata read failure")
	}
	return f.MonitorConfigDAO.GetGeneratedConfigNamespacePage(ctx, req)
}
func (f *amProjectionFixture) GetMonitorConfigList(ctx context.Context, req *model.GetMonitorConfigListReq) ([]*model.MonitorConfig, int64, error) {
	if err := f.seed(); err != nil {
		return nil, 0, err
	}
	return f.MonitorConfigDAO.GetMonitorConfigList(ctx, req)
}
func (f *amProjectionFixture) attach(c *alertManagerConfigCache) {
	f.currentRedis = func() redis.Cmdable { return c.redis }
	c.configDAO = f
	c.batchManager = NewBatchConfigManager(f, c.logger)
	c.journal = f.journal
	c.history = f.history
	c.projection = &amFixtureRunner{store: c.redis, current: func() redis.Cmdable { return c.redis }}
}

type amFixtureJournal struct {
	configdao.AMProjectionJournalDAO
	fixture *amProjectionFixture
}

func (j *amFixtureJournal) CommitAMProjectionPlan(ctx context.Context, req configdao.AMProjectionPlanRequest) (configdao.ProjectionReceipt, error) {
	// The legacy test hook now interleaves immediately before the actual typed
	// transaction, never after SQL commit or through a synthetic permission.
	if j.fixture.currentRedis != nil {
		if wrapped, ok := j.fixture.currentRedis().(*alertRetirementRedis); ok && wrapped.onPending != nil {
			hook := wrapped.onPending
			wrapped.onPending = nil
			hook()
			if err := j.fixture.seed(); err != nil {
				return configdao.ProjectionReceipt{}, err
			}
		}
	}
	return j.AMProjectionJournalDAO.CommitAMProjectionPlan(ctx, req)
}
func (j *amFixtureJournal) ReadAMSourceVersions(ctx context.Context) ([]configdao.AMSourceVersion, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := j.fixture.seed(); err != nil {
		return nil, err
	}
	return j.AMProjectionJournalDAO.ReadAMSourceVersions(ctx)
}

type amFixtureRunner struct {
	store   redis.Cmdable
	current func() redis.Cmdable
}

func (r *amFixtureRunner) Project(ctx context.Context, lease configdao.ProjectionLease) error {
	store := r.store
	if r.current != nil {
		store = r.current()
	}
	if lease == nil {
		return errors.New("AM fixture requires a real sealed lease")
	}
	if err := lease.RecheckCurrent(ctx); err != nil {
		return err
	}
	snapshot, err := lease.Snapshot()
	if err != nil {
		return err
	}
	if snapshot.Pool == nil || snapshot.Pool.AMControls == nil {
		return errors.New("AM fixture missing pair proof")
	}
	family, pool := snapshot.Pool.Reservation.Family, snapshot.Pool.Reservation.PoolID
	if family != 2 && family != 5 {
		return errors.New("AM fixture rejects another family")
	}
	if s := amFixtureBaseRedis(store); s != nil && s.failure == "pending write" {
		return errors.New("injected prepare failure")
	}
	for _, control := range snapshot.Pool.Controls {
		if control.Intents == nil {
			continue
		}
		if family != 2 {
			return errors.New("subset acquired intents")
		}
		for _, change := range control.Intents.Add {
			raw, err := json.Marshal(change.Owner)
			if err != nil {
				return err
			}
			if err := store.SAdd(ctx, alertRetirementIntentKey(pool), string(raw)).Err(); err != nil {
				return err
			}
		}
	}
	for _, material := range snapshot.Artifacts {
		if err := lease.RecheckCurrent(ctx); err != nil {
			return err
		}
		if family == 5 && material.Artifact.ConfigType != 5 {
			return errors.New("subset acquired main artifact")
		}
		key, err := projectionRawKey(material.Binding)
		if err != nil {
			return err
		}
		if material.Artifact.Action == model.ConfigProjectionRender {
			err = store.Set(ctx, key, string(material.Body), 0).Err()
		} else {
			err = store.Del(ctx, key).Err()
		}
		if err != nil {
			return err
		}
	}
	for _, control := range snapshot.Pool.Controls {
		if control.Members == nil {
			continue
		}
		if family != 2 {
			return errors.New("subset acquired main members")
		}
		key := buildRedisSetKeyAlertManagerMainPoolIPs(pool)
		old, err := store.SMembers(ctx, key).Result()
		if err != nil {
			return err
		}
		wanted := map[string]bool{}
		for _, ip := range control.Members.DesiredIPs {
			wanted[ip] = true
			if err := store.SAdd(ctx, key, ip).Err(); err != nil {
				return err
			}
		}
		for _, ip := range old {
			if !wanted[ip] {
				if err := store.SRem(ctx, key, ip).Err(); err != nil {
					return err
				}
			}
		}
	}
	for _, control := range snapshot.Pool.Controls {
		if control.Intents == nil {
			continue
		}
		for _, change := range control.Intents.Remove {
			raw, err := json.Marshal(change.Owner)
			if err != nil {
				return err
			}
			if err := store.SRem(ctx, alertRetirementIntentKey(pool), string(raw)).Err(); err != nil {
				return err
			}
		}
		remaining, err := store.SMembers(ctx, alertRetirementIntentKey(pool)).Result()
		if err != nil {
			return err
		}
		if control.Intents.RequireEmpty && len(remaining) != 0 {
			return errors.New("AM intents remain nonempty")
		}
	}
	if err := lease.RecheckCurrent(ctx); err != nil {
		return err
	}
	for _, control := range snapshot.Pool.Controls {
		if control.Ack == nil {
			continue
		}
		proof := configdao.ProjectionAckProof{Reservation: snapshot.Pool.Reservation, SourceHash: snapshot.Pool.SourceHash, PlanDigest: snapshot.Pool.PlanDigest, ArtifactVector: append([]configdao.ProjectionVectorEntry(nil), control.Ack.ArtifactVector...), RequiredControls: append([]configdao.ProjectionControlDigest(nil), control.Ack.RequiredControls...)}
		if family == 2 {
			if control.Kind != configdao.ProjectionACKFull || control.Ack.LegacySHA == nil {
				return errors.New("full AM lacks full ACK")
			}
			if err := store.Set(ctx, alertPublicationStateKey(pool), hex.EncodeToString(control.Ack.LegacySHA[:]), 0).Err(); err != nil {
				return err
			}
		} else if control.Kind != configdao.ProjectionACKWebhookSubset || control.Ack.LegacySHA != nil {
			return errors.New("subset acquired full ACK")
		}
		wire := proof
		wire.ArtifactVector = append([]configdao.ProjectionVectorEntry{}, proof.ArtifactVector...)
		wire.RequiredControls = append([]configdao.ProjectionControlDigest{}, proof.RequiredControls...)
		raw, err := json.Marshal(wire)
		if err != nil {
			return err
		}
		if err := store.Set(ctx, projectionPoolKey(family, pool, "ack"), string(raw), 0).Err(); err != nil {
			return err
		}
		marked, err := lease.MarkComplete(proof)
		if err != nil {
			return err
		}
		if !marked {
			return errors.New("AM fixture current mark failed")
		}
		return nil
	}
	return errors.New("AM fixture missing ACK control")
}
func amFixtureBaseRedis(client redis.Cmdable) *alertContractRedis {
	switch r := client.(type) {
	case *alertContractRedis:
		return r
	case *alertRetirementRedis:
		return r.alertContractRedis
	}
	return nil
}
func AMProjectionDependenciesForTest(db *gorm.DB, store redis.Cmdable) (configdao.AMProjectionJournalDAO, configdao.AMProjectionHistoryReader, AMProjectionRunner) {
	return configdao.NewAMProjectionJournalDAO(zap.NewNop(), db), configdao.NewAMProjectionHistoryReader(zap.NewNop(), db), &amFixtureRunner{store: store}
}

// The legacy SHA may remain as last-good data. Completion authority is the
// durable pair operation and its exact new ACK, never a pending string literal.
func amFixturePending(c *alertManagerConfigCache, family int8, pool int) bool {
	journal, ok := c.journal.(configdao.AMProjectionJournalDAO)
	if !ok {
		return false
	}
	versions, err := journal.ReadAMSourceVersions(context.Background())
	if err != nil {
		return false
	}
	var version configdao.AMSourceVersion
	found := false
	for _, item := range versions {
		if item.PoolID == pool {
			version = item
			found = true
		}
	}
	if !found {
		return false
	}
	counter := version.FullCounter
	if family == 5 {
		counter = version.SubsetCounter
	}
	receipt, err := journal.LoadLastCommittedProjection(context.Background(), family, pool)
	if err != nil {
		return false
	}
	if receipt == nil {
		return counter > 0
	}
	if counter > receipt.Reservation.Generation {
		return true
	}
	if receipt.Phase == model.ConfigPublicationProjected {
		return false
	}
	return true
}
