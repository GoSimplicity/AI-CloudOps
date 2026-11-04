package model

import (
	"bytes"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func projectionModelFixture(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatal("projection model fixture unavailable")
	}
	conn, err := db.DB()
	if err != nil {
		t.Fatal("projection fixture pool unavailable")
	}
	conn.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = conn.Close() })
	if db.AutoMigrate(&MonitorConfigPoolControl{}, &MonitorConfigPoolOperation{}, &MonitorConfigArtifactEvent{}) != nil {
		t.Fatal("projection model migration failed")
	}
	return db
}
func TestProjectionControlCompositeIdentityAndRetainedPointers(t *testing.T) {
	db := projectionModelFixture(t)
	committed, attempt := uint64(1), uint64(2)
	row := &MonitorConfigPoolControl{Family: 1, PoolID: 7, ReservedGeneration: 2, CurrentAttemptOperationID: &attempt, LastCommittedOperationID: &committed, LastProjectedOperationID: &committed}
	if db.Create(row).Error != nil {
		t.Fatal("control creation failed")
	}
	if db.Create(&MonitorConfigPoolControl{Family: 1, PoolID: 7}).Error == nil {
		t.Fatal("control composite identity not unique")
	}
	if db.Create(&MonitorConfigPoolControl{Family: 2, PoolID: 7}).Error != nil {
		t.Fatal("independent publication family merged")
	}
	var saved MonitorConfigPoolControl
	if db.First(&saved, "family=? AND pool_id=?", 1, 7).Error != nil || saved.ReservedGeneration != 2 || saved.CurrentAttemptOperationID == nil || *saved.CurrentAttemptOperationID != 2 || saved.LastCommittedOperationID == nil || *saved.LastCommittedOperationID != 1 || saved.LastProjectedOperationID == nil || *saved.LastProjectedOperationID != 1 {
		t.Fatal("last-good receipt pointers overwritten by current attempt")
	}
	stmt := &gorm.Statement{DB: db}
	if stmt.Parse(&MonitorConfigPoolControl{}) != nil {
		t.Fatal("control schema parse failed")
	}
	for _, name := range []string{"Family", "PoolID"} {
		field := stmt.Schema.LookUpField(name)
		if field == nil || !field.PrimaryKey || field.AutoIncrement {
			t.Fatal("control primary key became auto increment")
		}
	}
}
func TestProjectionOperationStorageDistinguishesReservedAndEmptyHistory(t *testing.T) {
	db := projectionModelFixture(t)
	empty := "[]"
	reserved := &MonitorConfigPoolOperation{Family: 1, PoolID: 7, Generation: 1, Nonce: bytes.Repeat([]byte{1}, 16), Phase: ConfigPublicationReserved}
	ready := &MonitorConfigPoolOperation{Family: 1, PoolID: 7, Generation: 2, Nonce: bytes.Repeat([]byte{2}, 16), Phase: ConfigPublicationCommitted, SourceHash: bytes.Repeat([]byte{3}, 32), Manifest: &empty}
	if db.Create(reserved).Error != nil || db.Create(ready).Error != nil {
		t.Fatal("operation history creation failed")
	}
	var first, second MonitorConfigPoolOperation
	if db.First(&first, "operation_id=?", reserved.OperationID).Error != nil || db.First(&second, "operation_id=?", ready.OperationID).Error != nil || first.Manifest != nil || second.Manifest == nil || *second.Manifest != "[]" {
		t.Fatal("NULL and legitimate empty manifest became indistinguishable")
	}
	if db.Create(&MonitorConfigPoolOperation{Family: 1, PoolID: 7, Generation: 2, Nonce: bytes.Repeat([]byte{4}, 16), Phase: ConfigPublicationReserved}).Error == nil {
		t.Fatal("pool generation history was reused")
	}
	if db.Create(&MonitorConfigPoolOperation{Family: 1, PoolID: 7, Generation: 3, Nonce: bytes.Repeat([]byte{2}, 16), Phase: ConfigPublicationReserved}).Error == nil {
		t.Fatal("operation nonce was reused")
	}
	var count int64
	if db.Model(&MonitorConfigPoolOperation{}).Count(&count).Error != nil || count != 2 {
		t.Fatal("history insert replaced a prior receipt")
	}
}
func TestProjectionEventStoresByteKeysAndExplicitBindingPolicy(t *testing.T) {
	db := projectionModelFixture(t)
	for _, key := range []string{"2001:db8::1", "2001:DB8::1", "192.0.2.1 ", " 192.0.2.1", "pool:7:ips", strings.Repeat("🌍", 45)} {
		event := &MonitorConfigArtifactEvent{ConfigType: 4, InstanceKey: []byte(key), Action: ConfigProjectionInvalidate, ProjectionPolicy: ConfigProjectionNotApplicable, DeliveryState: ConfigProjectionPending}
		if db.Create(event).Error != nil {
			t.Fatal("ordinary byte-exact journal key storage failed")
		}
		var got MonitorConfigArtifactEvent
		if db.First(&got, "event_id=?", event.EventID).Error != nil || string(got.InstanceKey) != key || got.Binding != nil || got.BindingEventID != nil || got.ProjectionPolicy != ConfigProjectionNotApplicable {
			t.Fatal("journal invented projection authorization")
		}
	}
	binding := `{"version":1,"role":"artifact","config_type":4,"instance_ip":"192.0.2.1"}`
	generated := &MonitorConfigArtifactEvent{ConfigType: 4, InstanceKey: []byte("192.0.2.1"), Action: ConfigProjectionRender, ProjectionPolicy: ConfigProjectionGenerated, Binding: &binding, DeliveryState: ConfigProjectionPending}
	if db.Create(generated).Error != nil {
		t.Fatal("generated binding storage failed")
	}
	lineage := &MonitorConfigArtifactEvent{ConfigType: 4, InstanceKey: []byte("192.0.2.1"), Action: ConfigProjectionInvalidate, ProjectionPolicy: ConfigProjectionLineageOnly, BindingEventID: &generated.EventID, DeliveryState: ConfigProjectionPending}
	if db.Create(lineage).Error != nil || lineage.EventID <= generated.EventID {
		t.Fatal("namespace history did not append")
	}
	for _, model := range []interface{}{&MonitorConfigPoolControl{}, &MonitorConfigPoolOperation{}, &MonitorConfigArtifactEvent{}} {
		for _, column := range []string{"deleted_at", "config_content", "yaml_body", "credential_body"} {
			if db.Migrator().HasColumn(model, column) {
				t.Fatal("projection bookkeeping acquired body or soft-deletion field")
			}
		}
	}
}
