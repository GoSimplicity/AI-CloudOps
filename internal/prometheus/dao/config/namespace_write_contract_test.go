package config

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const namespaceWriteCanary = "private-config-write-canary"

func newNamespaceWriteContractDB(t *testing.T) (MonitorConfigDAO, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Discard, CreateBatchSize: 100})
	if err != nil {
		t.Fatal("SQLite fixture unavailable")
	}
	conn, err := db.DB()
	if err != nil {
		t.Fatal("SQLite pool unavailable")
	}
	conn.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = conn.Close() })
	if db.AutoMigrate(&model.MonitorConfig{}, &model.MonitorConfigNamespaceGuard{}) != nil {
		t.Fatal("namespace fixture migration failed")
	}
	return NewMonitorConfigDAO(zap.NewNop(), db), db
}
func namespaceWriteRow(name, key string, kind int8, pool int, status int8) *model.MonitorConfig {
	return &model.MonitorConfig{Name: name, InstanceIP: key, ConfigType: kind, PoolID: pool, Status: status, ConfigContent: "original bytes", ConfigHash: "original hash", LastGeneratedTime: 17}
}
func namespaceWriteCounts(t *testing.T, db *gorm.DB) (int64, int64) {
	t.Helper()
	var configs, guards int64
	if db.Model(&model.MonitorConfig{}).Count(&configs).Error != nil || db.Model(&model.MonitorConfigNamespaceGuard{}).Count(&guards).Error != nil {
		t.Fatal("fixture count failed")
	}
	return configs, guards
}
func namespaceWriteState(t *testing.T, db *gorm.DB) []model.MonitorConfig {
	t.Helper()
	var rows []model.MonitorConfig
	if db.Order("id ASC").Find(&rows).Error != nil {
		t.Fatal("fixture read failed")
	}
	return rows
}
func requireNamespaceWriteRejected(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("occupied or invalid namespace write was accepted")
	}
	if strings.Contains(err.Error(), namespaceWriteCanary) {
		t.Fatal("write error disclosed private configuration input")
	}
}

type namespaceOrdinaryWrite struct {
	name   string
	call   func(MonitorConfigDAO, context.Context, *model.MonitorConfig) error
	update bool
}

func namespaceOrdinaryWrites() []namespaceOrdinaryWrite {
	return []namespaceOrdinaryWrite{
		{"create", func(d MonitorConfigDAO, c context.Context, r *model.MonitorConfig) error {
			return d.CreateMonitorConfig(c, r)
		}, false},
		{"batch create", func(d MonitorConfigDAO, c context.Context, r *model.MonitorConfig) error {
			return d.BatchCreateMonitorConfigs(c, []*model.MonitorConfig{r})
		}, false},
		{"update migration", func(d MonitorConfigDAO, c context.Context, r *model.MonitorConfig) error {
			return d.UpdateMonitorConfig(c, r)
		}, true},
		{"batch update migration", func(d MonitorConfigDAO, c context.Context, r *model.MonitorConfig) error {
			return d.BatchUpdateMonitorConfigs(c, []*model.MonitorConfig{r})
		}, true},
	}
}

func TestNamespaceOrdinaryWritesRejectEveryOccupiedTarget(t *testing.T) {
	for _, operation := range namespaceOrdinaryWrites() {
		for _, owner := range []struct {
			name   string
			pool   int
			status int8
		}{{"custom active", 7, 1}, {"custom inactive", 7, 2}, {"foreign active", 99, 1}, {"foreign inactive", 99, 2}} {
			t.Run(operation.name+"/"+owner.name, func(t *testing.T) {
				dao, db := newNamespaceWriteContractDB(t)
				ctx := context.Background()
				occupied := namespaceWriteRow("operator custom", "192.0.2.1", 4, owner.pool, owner.status)
				if dao.CreateMonitorConfig(ctx, occupied) != nil {
					t.Fatal("occupied fixture failed")
				}
				input := namespaceWriteRow(namespaceWriteCanary, "192.0.2.1", 4, 7, 1)
				if operation.update {
					source := namespaceWriteRow("source", "192.0.2.2", 4, 7, 1)
					if dao.CreateMonitorConfig(ctx, source) != nil {
						t.Fatal("source fixture failed")
					}
					input.ID = source.ID
				}
				beforeRows := namespaceWriteState(t, db)
				beforeInput := *input
				requireNamespaceWriteRejected(t, operation.call(dao, ctx, input))
				if !reflect.DeepEqual(beforeRows, namespaceWriteState(t, db)) || !reflect.DeepEqual(beforeInput, *input) {
					t.Fatal("rejected target write changed rows or caller input")
				}
			})
		}
	}
}

func TestNamespaceSameIDEditPreservesStructOmittedFields(t *testing.T) {
	for _, batch := range []bool{false, true} {
		t.Run(fmt.Sprint(batch), func(t *testing.T) {
			dao, db := newNamespaceWriteContractDB(t)
			ctx := context.Background()
			row := namespaceWriteRow("before", "192.0.2.1", 4, 7, 2)
			if dao.CreateMonitorConfig(ctx, row) != nil {
				t.Fatal("edit fixture failed")
			}
			before := *row
			patch := &model.MonitorConfig{Model: model.Model{ID: row.ID}, Name: "after", ConfigContent: "edited bytes"}
			var err error
			if batch {
				err = dao.BatchUpdateMonitorConfigs(ctx, []*model.MonitorConfig{patch})
			} else {
				err = dao.UpdateMonitorConfig(ctx, patch)
			}
			if err != nil {
				t.Fatal("same-ID same-key edit rejected")
			}
			rows := namespaceWriteState(t, db)
			if len(rows) != 1 {
				t.Fatal("edit created another row")
			}
			got := rows[0]
			if got.ID != before.ID || got.Name != "after" || got.ConfigContent != "edited bytes" || got.PoolID != before.PoolID || got.InstanceIP != before.InstanceIP || got.ConfigType != before.ConfigType || got.Status != before.Status || got.ConfigHash != before.ConfigHash || got.LastGeneratedTime != before.LastGeneratedTime || !got.CreatedAt.Equal(before.CreatedAt) {
				t.Fatal("same-key edit changed omitted fields")
			}
		})
	}
}

func TestNamespaceHistoricalDuplicatesAllowExplicitIDRepair(t *testing.T) {
	for _, repair := range []string{"delete one ID", "move one ID out", "edit one ID in place"} {
		t.Run(repair, func(t *testing.T) {
			dao, db := newNamespaceWriteContractDB(t)
			ctx := context.Background()
			// Direct SQL is solely a historical fixture: ordinary APIs must no longer
			// manufacture a second active row at an occupied exact namespace.
			first := namespaceWriteRow("historical A", "192.0.2.1", 4, 7, 2)
			second := namespaceWriteRow("historical B", "192.0.2.1", 4, 99, 2)
			if db.Create([]*model.MonitorConfig{first, second}).Error != nil {
				t.Fatal("historical duplicate seed failed")
			}
			secondStored, err := dao.GetMonitorConfigByID(ctx, second.ID)
			if err != nil {
				t.Fatal("historical stored fixture read failed")
			}
			secondBefore := *secondStored
			switch repair {
			case "delete one ID":
				if dao.DeleteMonitorConfig(ctx, first.ID) != nil {
					t.Fatal("ID delete could not repair duplicate")
				}
			case "move one ID out":
				if dao.UpdateMonitorConfig(ctx, &model.MonitorConfig{Model: model.Model{ID: first.ID}, InstanceIP: "192.0.2.2"}) != nil {
					t.Fatal("ID migration could not repair duplicate")
				}
			default:
				if dao.UpdateMonitorConfig(ctx, &model.MonitorConfig{Model: model.Model{ID: first.ID}, Name: "explicit ID edit"}) != nil {
					t.Fatal("ID-only edit rejected historical duplicate")
				}
			}
			kept, err := dao.GetMonitorConfigByID(ctx, second.ID)
			if err != nil || !reflect.DeepEqual(secondBefore, *kept) {
				t.Fatal("repair changed the other historical row")
			}
			if repair == "delete one ID" {
				if _, err := dao.GetMonitorConfigByID(ctx, first.ID); err == nil {
					t.Fatal("deleted row remains active")
				}
				var tombstone model.MonitorConfig
				if db.Unscoped().First(&tombstone, first.ID).Error != nil || tombstone.DeletedAt == 0 {
					t.Fatal("explicit repair hard-deleted audit history")
				}
			} else {
				got, err := dao.GetMonitorConfigByID(ctx, first.ID)
				if err != nil {
					t.Fatal("repair lost selected row")
				}
				if repair == "move one ID out" && got.InstanceIP != "192.0.2.2" {
					t.Fatal("selected row did not migrate")
				}
				if repair == "edit one ID in place" && (got.InstanceIP != "192.0.2.1" || got.Name != "explicit ID edit") {
					t.Fatal("same-key repair changed namespace")
				}
			}
		})
	}
}

func TestNamespaceBatchCreateLateSQLFailureRollsBackAndRestoresInputs(t *testing.T) {
	dao, db := newNamespaceWriteContractDB(t)
	if db.Exec("CREATE UNIQUE INDEX namespace_contract_unique_name ON cl_monitor_configs(name)").Error != nil {
		t.Fatal("late SQL constraint fixture failed")
	}
	input := make([]*model.MonitorConfig, 101)
	original := make([]model.MonitorConfig, 101)
	for i := range input {
		input[i] = namespaceWriteRow(fmt.Sprintf("batch-%03d", i), fmt.Sprintf("raw-key-%03d", i), 4, 7, 1)
	}
	input[0].Name = namespaceWriteCanary
	input[100].Name = namespaceWriteCanary
	for i, row := range input {
		original[i] = *row
	}
	successfulConfigStatements := 0
	if db.Callback().Create().After("gorm:create").Register("namespace-contract:observe-real-first-batch", func(tx *gorm.DB) {
		if tx.Statement.Table == "cl_monitor_configs" && tx.Error == nil && tx.RowsAffected > 0 {
			successfulConfigStatements++
		}
	}) != nil {
		t.Fatal("SQL observation registration failed")
	}
	err := dao.BatchCreateMonitorConfigs(context.Background(), input)
	requireNamespaceWriteRejected(t, err)
	if successfulConfigStatements == 0 {
		t.Fatal("fixture did not reach a successful first SQL batch before the unique-name failure")
	}
	configs, guards := namespaceWriteCounts(t, db)
	if configs != 0 || guards != 0 {
		t.Fatal("late SQL failure committed a partial config/namespace batch")
	}
	for i, row := range input {
		if !reflect.DeepEqual(original[i], *row) {
			t.Fatal("rolled-back batch leaked generated ID or timestamp into caller input")
		}
	}
}

func TestNamespaceInvalidKeysAndCancelledWritesHaveNoSideEffects(t *testing.T) {
	for _, operation := range namespaceOrdinaryWrites() {
		for _, invalid := range []struct {
			name, key string
			cancel    bool
		}{{"invalid UTF-8", string([]byte{255, 254}), false}, {"over 45 characters", strings.Repeat("🌍", 46), false}, {"cancelled", "192.0.2.2", true}} {
			t.Run(operation.name+"/"+invalid.name, func(t *testing.T) {
				dao, db := newNamespaceWriteContractDB(t)
				input := namespaceWriteRow(namespaceWriteCanary, invalid.key, 4, 7, 1)
				if operation.update {
					source := namespaceWriteRow("source", "192.0.2.1", 4, 7, 2)
					if dao.CreateMonitorConfig(context.Background(), source) != nil {
						t.Fatal("source fixture failed")
					}
					input.ID = source.ID
				}
				beforeRows := namespaceWriteState(t, db)
				beforeConfigs, beforeGuards := namespaceWriteCounts(t, db)
				beforeInput := *input
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				if invalid.cancel {
					cancel()
				}
				err := operation.call(dao, ctx, input)
				requireNamespaceWriteRejected(t, err)
				if invalid.cancel && !errors.Is(err, context.Canceled) {
					t.Fatal("cancelled write lost context identity")
				}
				afterConfigs, afterGuards := namespaceWriteCounts(t, db)
				if afterConfigs != beforeConfigs || afterGuards != beforeGuards || !reflect.DeepEqual(beforeRows, namespaceWriteState(t, db)) || !reflect.DeepEqual(beforeInput, *input) {
					t.Fatal("invalid/cancelled write changed config/guard state or input")
				}
			})
		}
	}
}

func TestNamespaceByteDistinctAndCrossKindWritesRemainLegal(t *testing.T) {
	dao, db := newNamespaceWriteContractDB(t)
	ctx := context.Background()
	for i, key := range []string{"2001:db8::1", "2001:DB8::1", "192.0.2.1", "192.0.2.1 ", " 192.0.2.1", strings.Repeat("🌍", 45)} {
		if dao.CreateMonitorConfig(ctx, namespaceWriteRow(fmt.Sprintf("raw-%d", i), key, 4, 7, 2)) != nil {
			t.Fatal("byte-distinct key or 45-character UTF-8 input rejected")
		}
	}
	if dao.CreateMonitorConfig(ctx, namespaceWriteRow("other kind", "192.0.2.1", 1, 7, 1)) != nil {
		t.Fatal("different config kind was treated as occupied")
	}
	configs, guards := namespaceWriteCounts(t, db)
	if configs != 7 || guards != 7 {
		t.Fatal("byte-distinct namespaces were aliased")
	}
}

func TestGeneratedWritesStillRejectHistoricalMixedOwnership(t *testing.T) {
	for _, operation := range []string{"update", "delete"} {
		t.Run(operation, func(t *testing.T) {
			dao, db := newNamespaceWriteContractDB(t)
			owned := namespaceWriteRow("prometheus_record_rule_7_192.0.2.1.yaml", "192.0.2.1", 4, 7, 1)
			custom := namespaceWriteRow("operator custom", "192.0.2.1", 4, 99, 2)
			if db.Create([]*model.MonitorConfig{owned, custom}).Error != nil {
				t.Fatal("historical ownership fixture failed")
			}
			before := namespaceWriteState(t, db)
			expected := GeneratedConfigOwnerOf(owned)
			var err error
			if operation == "update" {
				desired := *owned
				desired.ConfigContent = "generated change"
				err = dao.UpdateGeneratedMonitorConfigs(context.Background(), []GeneratedConfigUpdate{{Expected: expected, Desired: desired}})
			} else {
				err = dao.DeleteGeneratedMonitorConfig(context.Background(), expected)
			}
			if !errors.Is(err, ErrGeneratedConfigConflict) {
				t.Fatal("mixed ownership generated write did not classify conflict")
			}
			if !reflect.DeepEqual(before, namespaceWriteState(t, db)) {
				t.Fatal("generated conflict changed historical rows")
			}
		})
	}
}

func TestNamespaceBatchConflictPreservesEarlierInputsAndRows(t *testing.T) {
	for _, operation := range []string{"batch create", "batch update"} {
		t.Run(operation, func(t *testing.T) {
			dao, db := newNamespaceWriteContractDB(t)
			ctx := context.Background()
			occupied := namespaceWriteRow("occupied", "192.0.2.1", 4, 99, 2)
			if dao.CreateMonitorConfig(ctx, occupied) != nil {
				t.Fatal("occupied fixture failed")
			}
			input := []*model.MonitorConfig{namespaceWriteRow("first", "192.0.2.2", 4, 7, 1), namespaceWriteRow(namespaceWriteCanary, "192.0.2.1", 4, 7, 1)}
			if operation == "batch update" {
				for i, key := range []string{"192.0.2.2", "192.0.2.3"} {
					row := namespaceWriteRow("source", key, 4, 7, 2)
					if dao.CreateMonitorConfig(ctx, row) != nil {
						t.Fatal("batch source failed")
					}
					input[i].ID = row.ID
				}
				input[0].Name = "edited earlier row"
				input[0].ConfigContent = "edited earlier bytes"
			}
			original := []model.MonitorConfig{*input[0], *input[1]}
			rows := namespaceWriteState(t, db)
			configs, guards := namespaceWriteCounts(t, db)
			var err error
			if operation == "batch create" {
				err = dao.BatchCreateMonitorConfigs(ctx, input)
			} else {
				err = dao.BatchUpdateMonitorConfigs(ctx, input)
			}
			requireNamespaceWriteRejected(t, err)
			afterConfigs, afterGuards := namespaceWriteCounts(t, db)
			if configs != afterConfigs || guards != afterGuards || !reflect.DeepEqual(rows, namespaceWriteState(t, db)) {
				t.Fatal("occupied later target committed an earlier batch operation")
			}
			for i, row := range input {
				if !reflect.DeepEqual(original[i], *row) {
					t.Fatal("failed namespace batch mutated caller input")
				}
			}
		})
	}
}

func TestNamespaceCancelledDeletePreservesRow(t *testing.T) {
	dao, db := newNamespaceWriteContractDB(t)
	row := namespaceWriteRow("existing", "192.0.2.1", 4, 7, 2)
	if dao.CreateMonitorConfig(context.Background(), row) != nil {
		t.Fatal("delete fixture failed")
	}
	before := namespaceWriteState(t, db)
	configs, guards := namespaceWriteCounts(t, db)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := dao.DeleteMonitorConfig(ctx, row.ID)
	requireNamespaceWriteRejected(t, err)
	if !errors.Is(err, context.Canceled) {
		t.Fatal("cancelled delete lost context identity")
	}
	afterConfigs, afterGuards := namespaceWriteCounts(t, db)
	if configs != afterConfigs || guards != afterGuards || !reflect.DeepEqual(before, namespaceWriteState(t, db)) {
		t.Fatal("cancelled delete modified config or guard state")
	}
}
