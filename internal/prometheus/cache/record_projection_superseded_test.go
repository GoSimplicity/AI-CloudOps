package cache

import (
	"context"
	"fmt"
	"gorm.io/gorm"
	"reflect"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
)

type recordReuseFixture struct {
	sql     *recordProjectionFixture
	store   *supportRedis
	pools   map[int]*model.MonitorScrapePool
	sources map[int]*recordReconcileSource
}

func newRecordReuseFixture(t *testing.T) *recordReuseFixture {
	t.Helper()
	f := &recordReuseFixture{sql: newRecordProjectionFixture(t, nil, nil, nil, nil), store: &supportRedis{values: map[string]string{}, sets: map[string]map[string]bool{}}, pools: map[int]*model.MonitorScrapePool{}, sources: map[int]*recordReconcileSource{}}
	for _, id := range []int{7, 99} {
		f.pools[id] = &model.MonitorScrapePool{Model: model.Model{ID: id}, Name: fmt.Sprintf("reuse-%d", id), SupportRecord: 1, PrometheusInstances: model.StringList{"127.0.0.27"}}
		f.sources[id] = &recordReconcileSource{rows: []*model.MonitorRecordRule{{Model: model.Model{ID: id}, PoolID: id, Name: fmt.Sprintf("fixture_%d:up", id), Expr: "up", Enable: 1}}}
	}
	return f
}
func (f *recordReuseFixture) cache(id int) *recordRuleConfigCache {
	c := &recordRuleConfigCache{logger: zap.NewNop(), redis: f.store, scrapePoolDAO: &retirementInventory{pool: f.pools[id]}, alertRecordDAO: f.sources[id]}
	f.sql.attach(c)
	return c
}

type recordReuseState struct {
	Row           model.MonitorConfig
	Raw, SHA, Ack string
	Members       map[string]bool
}

func (f *recordReuseFixture) replacement(t *testing.T) recordReuseState {
	t.Helper()
	rows, err := f.sql.MonitorConfigDAO.GetMonitorConfigsByInstances(context.Background(), []string{"127.0.0.27"}, 4)
	if err != nil || len(rows) != 1 || rows[0].PoolID != 99 {
		t.Fatal("replacement SQL owner missing")
	}
	state := recordReuseState{Row: *rows[0], Raw: f.store.values[buildRedisKeyRecordRule("127.0.0.27")], SHA: f.store.values[recordPublicationKey(99)], Ack: f.store.values[projectionPoolKey(4, 99, "ack")], Members: map[string]bool{}}
	for ip, v := range f.store.sets[buildRedisSetKeyRecordRulePoolIPs(99)] {
		state.Members[ip] = v
	}
	return state
}
func (f *recordReuseFixture) assertReplacement(t *testing.T, want recordReuseState) {
	t.Helper()
	if got := f.replacement(t); !reflect.DeepEqual(got, want) {
		t.Fatal("retired pool changed replacement SQL/raw/ACK/membership")
	}
}
func (f *recordReuseFixture) editReplacementBody(t *testing.T) {
	t.Helper()
	state := f.replacement(t)
	row := state.Row
	row.ConfigContent += "# legitimate ordinary B edit\n"
	row.ConfigHash = calculateConfigHash(row.ConfigContent)
	if err := f.sql.MonitorConfigDAO.UpdateMonitorConfig(context.Background(), &row); err != nil {
		t.Fatal("ordinary replacement body edit failed", err)
	}
}
func TestRecordProjectionCompletedRetirementDoesNotReclaimReusedIP(t *testing.T) {
	for _, mode := range []string{"off", "delete"} {
		t.Run(mode, func(t *testing.T) {
			f := newRecordReuseFixture(t)
			ctx := context.Background()
			a := f.cache(7)
			if err := a.GenerateMainConfig(ctx); err != nil {
				t.Fatal(err)
			}
			f.pools[7].SupportRecord = 2
			retire := func(c *recordRuleConfigCache) error {
				if mode == "delete" {
					return c.RetireDeletedScrapePool(ctx, 7)
				}
				return c.GenerateMainConfig(ctx)
			}
			if mode == "delete" {
				f.pools[7].DeletedAt = 123
			}
			if err := retire(a); err != nil {
				t.Fatal("initial A retirement failed", err)
			}
			f.sources[7].failure = true
			if err := f.cache(99).GenerateMainConfig(ctx); err != nil {
				t.Fatal("B cannot reuse fully retired namespace", err)
			}
			want := f.replacement(t)
			for i := 0; i < 2; i++ {
				if err := retire(f.cache(7)); err != nil {
					t.Fatal("completed A retirement became pending after B reuse", err)
				}
				f.assertReplacement(t, want)
			}
			f.editReplacementBody(t)
			want = f.replacement(t)
			for i := 0; i < 2; i++ {
				if err := retire(f.cache(7)); err != nil {
					t.Fatal("B body update invalidated A's completed retirement", err)
				}
				f.assertReplacement(t, want)
			}
		})
	}
}

func TestRecordProjectionPendingRetirementRequiresProjectedReplacement(t *testing.T) {
	for _, mode := range []string{"ordinary custom", "ordinary N/A", "generated unprojected", "generated projected"} {
		t.Run(mode, func(t *testing.T) {
			f := newRecordReuseFixture(t)
			ctx := context.Background()
			a := f.cache(7)
			if err := a.GenerateMainConfig(ctx); err != nil {
				t.Fatal(err)
			}
			f.pools[7].SupportRecord = 2
			f.store.failure = "del"
			if err := a.GenerateMainConfig(ctx); err == nil {
				t.Fatal("retirement deletion failure was acknowledged")
			}
			f.store.failure = ""
			pending, err := f.sql.journal.LoadLastCommittedProjection(ctx, 4, 7)
			if err != nil || pending == nil || pending.Phase != model.ConfigPublicationCommitted || len(pending.Artifacts) != 1 || pending.Artifacts[0].Action != model.ConfigProjectionRetire {
				t.Fatal("fixture lacks a real committed/unprojected A retirement")
			}
			oldAck := f.store.values[projectionPoolKey(4, 7, "ack")]
			if mode == "ordinary custom" || mode == "ordinary N/A" {
				row := &model.MonitorConfig{PoolID: 99, ConfigType: 4, InstanceIP: "127.0.0.27", Name: "ordinary custom owner", ConfigContent: "custom synthetic body", Status: 2}
				row.ConfigHash = calculateConfigHash(row.ConfigContent)
				if mode == "ordinary N/A" {
					// A legacy SQL-only custom owner has no generated grant. A real
					// ordinary edit must emit N/A instead of inheriting A's old grant.
					if f.sql.db.Session(&gorm.Session{SkipHooks: true}).Create(row).Error != nil {
						t.Fatal("legacy ordinary owner fixture failed")
					}
					row.ConfigContent += " # ordinary edit"
					row.ConfigHash = calculateConfigHash(row.ConfigContent)
					if err := f.sql.MonitorConfigDAO.UpdateMonitorConfig(ctx, row); err != nil {
						t.Fatal(err)
					}
					var latest model.MonitorConfigArtifactEvent
					if f.sql.db.Order("event_id DESC").First(&latest).Error != nil || latest.ProjectionPolicy != model.ConfigProjectionNotApplicable || latest.OperationID != nil {
						t.Fatal("ordinary fixture unexpectedly has generated authority")
					}
				} else if err := f.sql.MonitorConfigDAO.CreateMonitorConfig(ctx, row); err != nil {
					t.Fatal(err)
				}
				f.store.values[buildRedisKeyRecordRule(row.InstanceIP)] = row.ConfigContent
			} else {
				if mode == "generated unprojected" {
					f.store.failure = "pending"
				}
				err := f.cache(99).GenerateMainConfig(ctx)
				if mode == "generated projected" && err != nil {
					t.Fatal("fully projected B fixture failed", err)
				}
				if mode == "generated unprojected" && err == nil {
					t.Fatal("unprojected B fixture accidentally completed")
				}
				b, loadErr := f.sql.journal.LoadLastCommittedProjection(ctx, 4, 99)
				if loadErr != nil || b == nil || len(b.Artifacts) != 1 {
					t.Fatal("B fixture lacks a real committed generation")
				}
				if mode == "generated unprojected" && b.Phase != model.ConfigPublicationCommitted {
					t.Fatal("unprojected B fixture already has completion proof")
				}
				if mode == "generated projected" && b.Phase != model.ConfigPublicationProjected {
					t.Fatal("projected B fixture lacks completion proof")
				}

				f.store.failure = ""
				if mode == "generated unprojected" {
					state := f.replacement(t)
					f.store.values[buildRedisKeyRecordRule(state.Row.InstanceIP)] = state.Row.ConfigContent
				}
			}
			want := f.replacement(t)
			err = f.cache(7).GenerateMainConfig(ctx)
			if mode != "generated projected" {
				if err == nil {
					t.Fatal("A treated an ordinary/unprojected replacement as a completed supersession")
				}
				f.assertReplacement(t, want)
				current, loadErr := f.sql.journal.LoadLastCommittedProjection(ctx, 4, 7)
				if loadErr != nil || current == nil || current.Reservation != pending.Reservation || current.Phase != model.ConfigPublicationCommitted || f.store.values[projectionPoolKey(4, 7, "ack")] != oldAck {
					t.Fatal("unsupported replacement completed or replaced pending A authority")
				}
				var oldEvent model.MonitorConfigArtifactEvent
				if f.sql.db.First(&oldEvent, "event_id=?", pending.Artifacts[0].EventID).Error != nil || oldEvent.DeliveryState != model.ConfigProjectionPending {
					t.Fatal("unsupported replacement marked old retirement resolved")
				}
				return
			}
			if err != nil {
				t.Fatal("projected B could not close pending A with empty recovery", err)
			}
			f.assertReplacement(t, want)
			resolved, err := f.sql.journal.LoadLastProjectedProjection(ctx, 4, 7)
			if err != nil || resolved == nil || len(resolved.Artifacts) != 0 || resolved.SourceHash != pending.SourceHash || resolved.Reservation.OperationID == pending.Reservation.OperationID {
				t.Fatal("A recovery did not persist a distinct empty projected plan")
			}
			var oldEvent model.MonitorConfigArtifactEvent
			if f.sql.db.First(&oldEvent, "event_id=?", pending.Artifacts[0].EventID).Error != nil || oldEvent.DeliveryState != model.ConfigProjectionSuperseded {
				t.Fatal("A recovery did not durably supersede its old retirement event")
			}
			for i := 0; i < 2; i++ {
				if err := f.cache(7).GenerateMainConfig(ctx); err != nil {
					t.Fatal("empty A recovery was not stable after rebuild", err)
				}
				f.assertReplacement(t, want)
			}
			f.editReplacementBody(t)
			want = f.replacement(t)
			for i := 0; i < 2; i++ {
				if err := f.cache(7).GenerateMainConfig(ctx); err != nil {
					t.Fatal("B ordinary edit erased A's durable supersession", err)
				}
				f.assertReplacement(t, want)
			}
		})
	}
}

func TestRecordProjectionSupersededRecoveryRetainsProofBeforeACK(t *testing.T) {
	f := newRecordReuseFixture(t)
	ctx := context.Background()
	if err := f.cache(7).GenerateMainConfig(ctx); err != nil {
		t.Fatal(err)
	}
	f.pools[7].SupportRecord = 2
	f.store.failure = "del"
	if err := f.cache(7).GenerateMainConfig(ctx); err == nil {
		t.Fatal("A retirement unexpectedly completed")
	}
	f.store.failure = ""
	original, err := f.sql.journal.LoadLastCommittedProjection(ctx, 4, 7)
	if err != nil || original == nil || original.Phase != model.ConfigPublicationCommitted {
		t.Fatal("missing pending A retirement")
	}
	if err := f.cache(99).GenerateMainConfig(ctx); err != nil {
		t.Fatal("B projection failed", err)
	}
	want := f.replacement(t)
	f.store.failure = "pending"
	if err := f.cache(7).GenerateMainConfig(ctx); err == nil {
		t.Fatal("A recovery prepare failure reported success")
	}
	f.store.failure = ""
	f.assertReplacement(t, want)
	recovery, err := f.sql.journal.LoadLastCommittedProjection(ctx, 4, 7)
	if err != nil || recovery == nil || recovery.Phase != model.ConfigPublicationCommitted || len(recovery.Artifacts) != 0 || recovery.Reservation == original.Reservation {
		t.Fatal("failed recovery lost its durable empty committed plan")
	}
	projected, err := f.sql.journal.LoadLastProjectedProjection(ctx, 4, 7)
	if err != nil || projected == nil || projected.Reservation == recovery.Reservation {
		t.Fatal("unacknowledged recovery was reported projected")
	}
	// B's subsequent ordinary edit changes its current body and latest event. It
	// cannot invalidate the immutable projected B receipt accepted by A's commit.
	f.editReplacementBody(t)
	want = f.replacement(t)
	if err := f.cache(7).GenerateMainConfig(ctx); err != nil {
		t.Fatal("committed A recovery could not resume after B ordinary edit", err)
	}
	f.assertReplacement(t, want)
	completed, err := f.sql.journal.LoadLastProjectedProjection(ctx, 4, 7)
	if err != nil || completed == nil || completed.Reservation != recovery.Reservation {
		t.Fatal("retry replaced rather than completed the durable recovery")
	}
	for i := 0; i < 2; i++ {
		if err := f.cache(7).GenerateMainConfig(ctx); err != nil {
			t.Fatal("resolved retirement was forgotten after rebuild", err)
		}
		f.assertReplacement(t, want)
	}
}

func TestRecordProjectionAcceptedRenderCanCloseOnlyAfterTargetRemoved(t *testing.T) {
	for _, mode := range []string{"off", "remove target", "still desired"} {
		t.Run(mode, func(t *testing.T) {
			f := newRecordReuseFixture(t)
			ctx := context.Background()
			// The real journal commits A's render; the fake transport writes its raw
			// body and membership, then fails before ACK/SQL projection completion.
			f.store.failure = "ack"
			if err := f.cache(7).GenerateMainConfig(ctx); err == nil {
				t.Fatal("accepted A render unexpectedly acknowledged")
			}
			f.store.failure = ""
			accepted, err := f.sql.journal.LoadLastCommittedProjection(ctx, 4, 7)
			if err != nil || accepted == nil || accepted.Phase != model.ConfigPublicationCommitted || len(accepted.Artifacts) != 1 || accepted.Artifacts[0].Action != model.ConfigProjectionRender {
				t.Fatal("fixture lacks a real accepted/unprojected render")
			}
			const ip = "127.0.0.27"
			rows, err := f.sql.MonitorConfigDAO.GetMonitorConfigsByInstances(ctx, []string{ip}, 4)
			if err != nil || len(rows) != 1 || rows[0].PoolID != 7 || f.store.values[buildRedisKeyRecordRule(ip)] != rows[0].ConfigContent {
				t.Fatal("A's accepted render bytes were not materialized")
			}
			if f.store.values[projectionPoolKey(4, 7, "ack")] != "" {
				t.Fatal("failed A render has a completion receipt")
			}
			if err := f.sql.MonitorConfigDAO.DeleteMonitorConfig(ctx, rows[0].ID); err != nil {
				t.Fatal("ordinary deletion of A failed", err)
			}
			if err := f.cache(99).GenerateMainConfig(ctx); err != nil {
				t.Fatal("B could not project the vacated namespace", err)
			}
			b, err := f.sql.journal.LoadLastProjectedProjection(ctx, 4, 99)
			if err != nil || b == nil || b.Phase != model.ConfigPublicationProjected {
				t.Fatal("B fixture lacks actual projection completion")
			}
			want := f.replacement(t)
			oldAck := f.store.values[projectionPoolKey(4, 7, "ack")]
			switch mode {
			case "off":
				f.pools[7].SupportRecord = 2
			case "remove target":
				f.pools[7].PrometheusInstances = nil
			}
			err = f.cache(7).GenerateMainConfig(ctx)
			if mode == "still desired" {
				if err == nil {
					t.Fatal("active A silently surrendered its desired foreign namespace")
				}
				f.assertReplacement(t, want)
				current, loadErr := f.sql.journal.LoadLastCommittedProjection(ctx, 4, 7)
				if loadErr != nil || current == nil || current.Reservation != accepted.Reservation || current.Phase != model.ConfigPublicationCommitted || f.store.values[projectionPoolKey(4, 7, "ack")] != oldAck {
					t.Fatal("foreign desired conflict was resolved or acknowledged")
				}
				var event model.MonitorConfigArtifactEvent
				if f.sql.db.First(&event, "event_id=?", accepted.Artifacts[0].EventID).Error != nil || event.DeliveryState != model.ConfigProjectionPending {
					t.Fatal("active desired render was incorrectly marked superseded")
				}
				return
			}
			if err != nil {
				t.Fatal("removed accepted render could not close with a durable empty recovery", err)
			}
			f.assertReplacement(t, want)
			recovered, err := f.sql.journal.LoadLastProjectedProjection(ctx, 4, 7)
			if err != nil || recovered == nil || len(recovered.Artifacts) != 0 || recovered.Reservation.OperationID == accepted.Reservation.OperationID {
				t.Fatal("render recovery did not create an empty projected plan")
			}
			var operation model.MonitorConfigPoolOperation
			if f.sql.db.First(&operation, "operation_id=?", recovered.Reservation.OperationID).Error != nil || operation.RecoveryOfOperationID == nil || *operation.RecoveryOfOperationID != accepted.Reservation.OperationID {
				t.Fatal("empty recovery lost its accepted-render parent")
			}
			var oldEvent model.MonitorConfigArtifactEvent
			if f.sql.db.First(&oldEvent, "event_id=?", accepted.Artifacts[0].EventID).Error != nil || oldEvent.DeliveryState != model.ConfigProjectionSuperseded {
				t.Fatal("accepted render was not durably superseded")
			}
			for i := 0; i < 2; i++ {
				if err := f.cache(7).GenerateMainConfig(ctx); err != nil {
					t.Fatal("render recovery was forgotten after rebuild", err)
				}
				f.assertReplacement(t, want)
			}
			f.editReplacementBody(t)
			want = f.replacement(t)
			for i := 0; i < 2; i++ {
				if err := f.cache(7).GenerateMainConfig(ctx); err != nil {
					t.Fatal("B ordinary edit erased accepted-render recovery", err)
				}
				f.assertReplacement(t, want)
			}
		})
	}
}

func TestRecordProjectionLegacyOwnerRequiresCompletedReplacement(t *testing.T) {
	for _, mode := range []string{"projected replacement", "still desired", "no legacy owner", "ordinary N/A replacement", "unprojected replacement"} {
		t.Run(mode, func(t *testing.T) {
			f := newRecordReuseFixture(t)
			ctx := context.Background()
			const ip = "127.0.0.27"
			var legacy model.MonitorConfig
			if mode != "no legacy owner" {
				legacy = model.MonitorConfig{PoolID: 7, ConfigType: 4, InstanceIP: ip, Name: fmt.Sprintf(ConfigNameRecordRule, 7, ip), ConfigContent: "groups: [] # legacy A\n", Status: 1}
				legacy.ConfigHash = calculateConfigHash(legacy.ConfigContent)
				// Actual persisted pre-journal SQL: neither operation nor artifact event is
				// invented to make this owner look like a modern issued publication.
				if f.sql.db.Session(&gorm.Session{SkipHooks: true}).Create(&legacy).Error != nil || f.sql.db.Session(&gorm.Session{SkipHooks: true}).Delete(&legacy).Error != nil || legacy.DeletedAt == 0 {
					t.Fatal("real legacy create/delete fixture failed")
				}
			}
			for _, value := range []any{&model.MonitorConfigPoolOperation{}, &model.MonitorConfigArtifactEvent{}} {
				var count int64
				if f.sql.db.Model(value).Count(&count).Error != nil || count != 0 {
					t.Fatal("legacy fixture unexpectedly contains journal permission")
				}
			}
			f.store.values[buildRedisKeyRecordRule(ip)] = "groups: [] # legacy A\n"
			f.store.sets[buildRedisSetKeyRecordRulePoolIPs(7)] = map[string]bool{ip: true}
			if mode == "ordinary N/A replacement" {
				row := &model.MonitorConfig{PoolID: 99, ConfigType: 4, InstanceIP: ip, Name: "ordinary custom B", ConfigContent: "groups: [] # ordinary B\n", Status: 2}
				row.ConfigHash = calculateConfigHash(row.ConfigContent)
				if err := f.sql.MonitorConfigDAO.CreateMonitorConfig(ctx, row); err != nil {
					t.Fatal(err)
				}
				var event model.MonitorConfigArtifactEvent
				if f.sql.db.Order("event_id DESC").First(&event).Error != nil || event.ProjectionPolicy != model.ConfigProjectionNotApplicable || event.OperationID != nil {
					t.Fatal("ordinary replacement unexpectedly has issued authority")
				}
				f.store.values[buildRedisKeyRecordRule(ip)] = row.ConfigContent
			} else {
				if mode == "unprojected replacement" {
					f.store.failure = "ack"
				}
				err := f.cache(99).GenerateMainConfig(ctx)
				if mode == "unprojected replacement" {
					if err == nil {
						t.Fatal("unprojected replacement accidentally completed")
					}
				} else if err != nil {
					t.Fatal("projected replacement failed", err)
				}
				f.store.failure = ""
			}
			want := f.replacement(t)
			if mode != "still desired" {
				f.pools[7].SupportRecord = 2
			}
			err := f.cache(7).GenerateMainConfig(ctx)
			if mode != "projected replacement" {
				if err == nil {
					t.Fatal("unproven legacy owner/replacement or desired conflict was silently resolved")
				}
				f.assertReplacement(t, want)
				a, loadErr := f.sql.journal.LoadLastCommittedProjection(ctx, 4, 7)
				if loadErr != nil || a != nil || f.store.values[projectionPoolKey(4, 7, "ack")] != "" {
					t.Fatal("unproven legacy state acquired a completion receipt")
				}
				if !f.store.sets[buildRedisSetKeyRecordRulePoolIPs(7)][ip] {
					t.Fatal("unproven legacy member was discarded")
				}
				return
			}
			if err != nil {
				t.Fatal("proven legacy owner could not sign an empty replacement recovery", err)
			}
			f.assertReplacement(t, want)
			a, err := f.sql.journal.LoadLastProjectedProjection(ctx, 4, 7)
			if err != nil || a == nil || len(a.Artifacts) != 0 {
				t.Fatal("legacy adoption invented an artifact instead of control-only recovery")
			}
			var events int64
			if f.sql.db.Model(&model.MonitorConfigArtifactEvent{}).Where("operation_id IN (?)", f.sql.db.Model(&model.MonitorConfigPoolOperation{}).Select("operation_id").Where("family=4 AND pool_id=7")).Count(&events).Error != nil || events != 0 {
				t.Fatal("legacy adoption invented an issued A artifact event")
			}
			for i := 0; i < 2; i++ {
				if err := f.cache(7).GenerateMainConfig(ctx); err != nil {
					t.Fatal("legacy control-only recovery was forgotten after rebuild", err)
				}
				f.assertReplacement(t, want)
			}
			f.editReplacementBody(t)
			want = f.replacement(t)
			for i := 0; i < 2; i++ {
				if err := f.cache(7).GenerateMainConfig(ctx); err != nil {
					t.Fatal("B ordinary edit erased legacy owner resolution", err)
				}
				f.assertReplacement(t, want)
			}
			var audit model.MonitorConfig
			if f.sql.db.Unscoped().First(&audit, legacy.ID).Error != nil || audit.DeletedAt != legacy.DeletedAt || audit.Name != legacy.Name {
				t.Fatal("legacy adoption changed its historical SQL owner")
			}
		})
	}
}

func TestRecordProjectionCurrentOwnerCanRetireWithDeletedForeignHistory(t *testing.T) {
	for _, kind := range []string{"canonical foreign", "custom foreign"} {
		t.Run(kind, func(t *testing.T) {
			f := newRecordReuseFixture(t)
			ctx := context.Background()
			const ip = "127.0.0.27"
			legacyPool := 7
			name := fmt.Sprintf(ConfigNameRecordRule, 7, ip)
			if kind == "custom foreign" {
				legacyPool = 55
				name = "operator-custom-history"
			}
			historical := &model.MonitorConfig{PoolID: legacyPool, ConfigType: 4, InstanceIP: ip, Name: name, ConfigContent: "groups: [] # historical foreign\n", Status: 2}
			historical.ConfigHash = calculateConfigHash(historical.ConfigContent)
			if f.sql.db.Session(&gorm.Session{SkipHooks: true}).Create(historical).Error != nil || f.sql.db.Session(&gorm.Session{SkipHooks: true}).Delete(historical).Error != nil {
				t.Fatal("historical foreign create/delete failed")
			}
			if err := f.cache(99).GenerateMainConfig(ctx); err != nil {
				t.Fatal("current B owner could not publish over retired historical namespace", err)
			}
			if kind == "canonical foreign" {
				// First finish A's proven legacy control-only resolution while B is current.
				f.store.sets[buildRedisSetKeyRecordRulePoolIPs(7)] = map[string]bool{ip: true}
				f.pools[7].SupportRecord = 2
				if err := f.cache(7).GenerateMainConfig(ctx); err != nil {
					t.Fatal("legacy A resolution failed before B retirement", err)
				}
			}
			f.pools[99].SupportRecord = 2
			if err := f.cache(99).GenerateMainConfig(ctx); err != nil {
				t.Fatal("live canonical B could not retire because of deleted foreign history", err)
			}
			if f.store.values[buildRedisKeyRecordRule(ip)] != "" || len(f.store.sets[buildRedisSetKeyRecordRulePoolIPs(99)]) != 0 {
				t.Fatal("B retirement retained its own raw or membership")
			}
			b, err := f.sql.journal.LoadLastProjectedProjection(ctx, 4, 99)
			if err != nil || b == nil || len(b.Artifacts) != 1 || b.Artifacts[0].Action != model.ConfigProjectionRetire {
				t.Fatal("current-owner retirement lacks a projected retire receipt")
			}
			var audit model.MonitorConfig
			if f.sql.db.Unscoped().First(&audit, historical.ID).Error != nil || audit.DeletedAt != historical.DeletedAt || audit.Name != historical.Name || audit.PoolID != historical.PoolID {
				t.Fatal("current B retirement modified historical foreign audit data")
			}
			if kind == "canonical foreign" {
				for i := 0; i < 2; i++ {
					if err := f.cache(7).GenerateMainConfig(ctx); err != nil {
						t.Fatal("resolved A was forgotten after B retired", err)
					}
				}
				if f.store.values[buildRedisKeyRecordRule(ip)] != "" || len(f.store.sets[buildRedisSetKeyRecordRulePoolIPs(99)]) != 0 {
					t.Fatal("A retry resurrected B's retired data")
				}
			}
		})
	}
}

func TestRecordProjectionIssuedRetirementRetriesWithDeletedForeignHistory(t *testing.T) {
	f := newRecordReuseFixture(t)
	ctx := context.Background()
	const ip = "127.0.0.27"
	history := &model.MonitorConfig{PoolID: 55, ConfigType: 4, InstanceIP: ip, Name: "deleted custom history", ConfigContent: "groups: [] # prior owner\n", Status: 2}
	if f.sql.db.Session(&gorm.Session{SkipHooks: true}).Create(history).Error != nil || f.sql.db.Session(&gorm.Session{SkipHooks: true}).Delete(history).Error != nil {
		t.Fatal("deleted foreign fixture failed")
	}
	if err := f.cache(99).GenerateMainConfig(ctx); err != nil {
		t.Fatal(err)
	}
	f.pools[99].SupportRecord = 2
	f.store.failure = "del"
	if err := f.cache(99).GenerateMainConfig(ctx); err == nil {
		t.Fatal("failed retirement delete was acknowledged")
	}
	accepted, err := f.sql.journal.LoadLastCommittedProjection(ctx, 4, 99)
	if err != nil || accepted == nil || accepted.Phase != model.ConfigPublicationCommitted || len(accepted.Artifacts) != 1 || accepted.Artifacts[0].Action != model.ConfigProjectionRetire {
		t.Fatal("failed deletion lacks its already-authorized retirement receipt")
	}
	if f.store.values[buildRedisKeyRecordRule(ip)] == "" {
		t.Fatal("failed deletion lost its raw retry fixture")
	}
	f.store.failure = ""
	if err := f.cache(99).GenerateMainConfig(ctx); err != nil {
		t.Fatal("rebuild could not replay its own issued retirement alongside deleted foreign observations", err)
	}
	completed, err := f.sql.journal.LoadLastProjectedProjection(ctx, 4, 99)
	if err != nil || completed == nil || completed.Reservation != accepted.Reservation {
		t.Fatal("retirement retry did not complete the original issued operation")
	}
	if f.store.values[buildRedisKeyRecordRule(ip)] != "" || len(f.store.sets[buildRedisSetKeyRecordRulePoolIPs(99)]) != 0 {
		t.Fatal("authorized retry retained its raw or members")
	}
	var audit model.MonitorConfig
	if f.sql.db.Unscoped().First(&audit, history.ID).Error != nil || audit.DeletedAt != history.DeletedAt || audit.Name != history.Name {
		t.Fatal("authorized replay modified foreign historical data")
	}
}

func TestRecordProjectionRetirementReplayDoesNotHideNewDBOnlyStale(t *testing.T) {
	f := newRecordReuseFixture(t)
	ctx := context.Background()
	if err := f.cache(7).GenerateMainConfig(ctx); err != nil {
		t.Fatal(err)
	}
	f.pools[7].SupportRecord = 2
	f.store.failure = "del"
	if err := f.cache(7).GenerateMainConfig(ctx); err == nil {
		t.Fatal("pending retirement fixture unexpectedly completed")
	}
	f.store.failure = ""
	pending, err := f.sql.journal.LoadLastCommittedProjection(ctx, 4, 7)
	if err != nil || pending == nil || len(pending.Artifacts) != 1 || pending.Phase != model.ConfigPublicationCommitted {
		t.Fatal("missing issued pending retirement")
	}
	const orphanIP = "127.0.0.28"
	orphan := &model.MonitorConfig{PoolID: 7, ConfigType: 4, InstanceIP: orphanIP, Name: fmt.Sprintf(ConfigNameRecordRule, 7, orphanIP), ConfigContent: "groups: [] # new DB-only owner\n", Status: 2}
	orphan.ConfigHash = calculateConfigHash(orphan.ConfigContent)
	if err := f.sql.MonitorConfigDAO.CreateMonitorConfig(ctx, orphan); err != nil {
		t.Fatal("real DB-only owner creation failed", err)
	}
	f.store.values[buildRedisKeyRecordRule(orphanIP)] = orphan.ConfigContent
	if f.store.sets[buildRedisSetKeyRecordRulePoolIPs(7)][orphanIP] {
		t.Fatal("orphan fixture unexpectedly indexed")
	}
	if err := f.cache(7).GenerateMainConfig(ctx); err != nil {
		t.Fatal("fresh stale inventory could not complete", err)
	}
	completed, err := f.sql.journal.LoadLastProjectedProjection(ctx, 4, 7)
	if err != nil || completed == nil || completed.Reservation == pending.Reservation || completed.SourceHash != pending.SourceHash || len(completed.Artifacts) != 2 {
		t.Fatal("same-hash old receipt hid a newly discovered SQL-owned stale artifact")
	}
	if f.store.values[buildRedisKeyRecordRule(orphanIP)] != "" || len(f.store.sets[buildRedisSetKeyRecordRulePoolIPs(7)]) != 0 {
		t.Fatal("new DB-only stale state survived completion")
	}
	var saved model.MonitorConfig
	if f.sql.db.Unscoped().First(&saved, orphan.ID).Error != nil || saved.DeletedAt == 0 {
		t.Fatal("new DB-only owner was not retired through SQL")
	}
}
