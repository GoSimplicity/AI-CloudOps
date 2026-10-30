package dbschema

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/pkg/schema"
)

func TestKnownA8SafeV2PreservesReleasedInputs(t *testing.T) {
	old, err := KnownA8Upgrade()
	if err != nil {
		t.Fatal(err)
	}
	safe, err := KnownA8SafeUpgradeV2()
	if err != nil {
		t.Fatal(err)
	}
	if old.ImplementationChecksum != "44e466f5ff9dce7e1f304107d2e7d150810d61b9c39e957a8a67f1901d901e67" {
		t.Fatal("released implementation changed")
	}
	if safe.Source != old.Source || safe.SourceChecksum != old.SourceChecksum || safe.ImplementationChecksum == old.ImplementationChecksum {
		t.Fatal("v2 provenance did not bind unchanged source and new safety implementation")
	}
	// Filled with the complete self-embedded wrapper checksum before release.
	if safe.ImplementationChecksum != "94a07ba52e771ce4344e022556ff5d4eb21318cb574eefbda9054f0c833fab52" {
		t.Fatal("safe v2 immutable implementation changed")
	}
	safe.Source = "caller mutation"
	again, err := KnownA8SafeUpgradeV2()
	if err != nil || again.Source != KnownA8Source {
		t.Fatal("safe plan shares mutable caller state")
	}
}

type triggerGuardState struct {
	visible, triggers bool
	failure           string
	inventoryReads    int
}
type triggerGuardConnector struct{ state *triggerGuardState }

func (c triggerGuardConnector) Connect(context.Context) (driver.Conn, error) {
	return triggerGuardConnection{c.state}, nil
}
func (c triggerGuardConnector) Driver() driver.Driver { return triggerGuardDriver{} }

type triggerGuardDriver struct{}

func (triggerGuardDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New("unexpected open")
}

type triggerGuardConnection struct{ state *triggerGuardState }

func (triggerGuardConnection) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("unexpected prepare")
}
func (triggerGuardConnection) Close() error { return nil }
func (triggerGuardConnection) Begin() (driver.Tx, error) {
	return nil, errors.New("unexpected transaction")
}
func (c triggerGuardConnection) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	if c.state.failure != "" && strings.Contains(query, c.state.failure) {
		return nil, errors.New("private-driver-canary")
	}
	if query == "SELECT DATABASE(), CURRENT_USER()" {
		return &triggerGuardRows{values: []driver.Value{"knownupgrade", "migration@%"}}, nil
	}
	if strings.Contains(query, "SCHEMA_PRIVILEGES") {
		return &triggerGuardRows{values: []driver.Value{c.state.visible}}, nil
	}
	if strings.Contains(query, "information_schema.TRIGGERS") {
		c.state.inventoryReads++
		return &triggerGuardRows{values: []driver.Value{c.state.triggers}}, nil
	}
	return nil, errors.New("unexpected schema query")
}

type triggerGuardRows struct {
	values []driver.Value
	read   bool
}

func (r *triggerGuardRows) Columns() []string {
	names := make([]string, len(r.values))
	for i := range names {
		names[i] = "fixture"
	}
	return names
}
func (r *triggerGuardRows) Close() error { return nil }
func (r *triggerGuardRows) Next(values []driver.Value) error {
	if r.read {
		return io.EOF
	}
	r.read = true
	copy(values, r.values)
	return nil
}

func TestKnownA8SafeV2RequiresVisibilityBeforeTrustingZeroTriggers(t *testing.T) {
	for _, tc := range []struct {
		name  string
		state triggerGuardState
		want  error
		reads int
	}{
		{"no direct permission", triggerGuardState{}, ErrKnownA8TriggerVisibility, 0},
		{"unknown trigger", triggerGuardState{visible: true, triggers: true}, ErrKnownA8Triggers, 1},
		{"empty proven inventory", triggerGuardState{visible: true}, nil, 1},
		{"identity read fails", triggerGuardState{failure: "CURRENT_USER"}, ErrKnownA8TriggerRead, 0},
		{"privilege read fails", triggerGuardState{failure: "SCHEMA_PRIVILEGES"}, ErrKnownA8TriggerRead, 0},
		{"inventory read fails", triggerGuardState{visible: true, failure: "information_schema.TRIGGERS"}, ErrKnownA8TriggerRead, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := sql.OpenDB(triggerGuardConnector{&tc.state})
			defer db.Close()
			conn, err := db.Conn(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close()
			err = validateKnownA8NoTriggersV2(context.Background(), conn)
			if !errors.Is(err, tc.want) || tc.state.inventoryReads != tc.reads {
				t.Fatal("trigger visibility/inventory contract mismatch")
			}
			if err != nil && strings.Contains(err.Error(), "private-driver-canary") {
				t.Fatal("trigger guard exposed private database error")
			}
		})
	}
}

func TestKnownA8SafeV2GuardsBothSourceAndTargetCallbacks(t *testing.T) {
	plan, err := KnownA8SafeUpgradeV2()
	if err != nil {
		t.Fatal(err)
	}
	for _, callback := range []func(context.Context, *sql.Conn, schema.UpgradeEvidence) error{plan.ValidateSource, plan.ValidateTarget} {
		state := &triggerGuardState{visible: true, triggers: true}
		db := sql.OpenDB(triggerGuardConnector{state})
		conn, err := db.Conn(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if err := callback(context.Background(), conn, schema.UpgradeEvidence{}); !errors.Is(err, ErrKnownA8Triggers) {
			t.Fatal("callback bypassed trigger rejection")
		}
		conn.Close()
		db.Close()
	}
}
