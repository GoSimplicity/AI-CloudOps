package schema

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"strings"
	"testing"
)

type modeRecorder struct {
	adoptionRecorder
	business  int64
	locked    bool
	applied   int
	onAcquire func()
}
type modeConnector struct{ state *modeRecorder }

func (c modeConnector) Connect(context.Context) (driver.Conn, error) {
	return modeConnection{c.state}, nil
}
func (modeConnector) Driver() driver.Driver { return adoptionDriver{} }

type modeConnection struct{ state *modeRecorder }

func (modeConnection) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("unexpected prepare")
}
func (modeConnection) Begin() (driver.Tx, error) {
	return nil, errors.New("DDL must not run in a transaction")
}
func (modeConnection) Close() error { return nil }
func (c modeConnection) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if strings.HasPrefix(query, "SELECT GET_LOCK") {
		c.state.locked = true
		if c.state.onAcquire != nil {
			c.state.onAcquire()
		}
	}
	if strings.HasPrefix(query, "SELECT RELEASE_LOCK") {
		c.state.locked = false
	}
	if strings.Contains(query, "information_schema.tables") || strings.Contains(query, "FROM cloudops_schema_history") {
		if !c.state.locked {
			return nil, errors.New("mode inspected before lock")
		}
	}
	if query == "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name <> 'cloudops_schema_history'" {
		return &adoptionRows{columns: []string{"count"}, values: [][]driver.Value{{c.state.business}}}, nil
	}
	return (adoptionConnection{&c.state.adoptionRecorder}).QueryContext(ctx, query, args)
}
func (c modeConnection) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if !c.state.locked {
		return nil, errors.New("write outside migration lock")
	}
	c.state.writes = append(c.state.writes, query)
	switch {
	case query == ledgerDDL:
		c.state.ledger = true
	case strings.HasPrefix(query, "INSERT INTO cloudops_schema_history"):
		c.state.records = append(c.state.records, Record{Version: uint64(args[0].Value.(int64)), Name: args[1].Value.(string), Checksum: args[2].Value.(string), State: "started"})
	case strings.HasPrefix(query, "UPDATE cloudops_schema_history SET state = 'applied'"):
		for i := range c.state.records {
			if c.state.records[i].Version == uint64(args[0].Value.(int64)) {
				c.state.records[i].State = "applied"
			}
		}
	case strings.HasPrefix(query, "APPLY "):
		c.state.applied++
	default:
		return nil, errors.New("unexpected mutation")
	}
	return driver.RowsAffected(1), nil
}
func TestConfirmedMigrationsRecheckModeBeforeAnyMutationUnderLock(t *testing.T) {
	manifest := []Migration{SQLMigration(1, "first", "APPLY first"), SQLMigration(2, "second", "APPLY second")}
	first := Record{Version: 1, Name: manifest[0].Name, Checksum: manifest[0].Checksum, State: "applied"}
	second := Record{Version: 2, Name: manifest[1].Name, Checksum: manifest[1].Checksum, State: "applied"}
	dirty := second
	dirty.State = "failed"
	drift := first
	drift.Checksum = Checksum([]byte("wrong"))
	for _, tc := range []struct {
		name            string
		pending, ledger bool
		records         []Record
		business        int64
		want            error
		applied         int
	}{
		{"fresh missing", false, false, nil, 0, nil, 2}, {"fresh empty", false, true, nil, 0, nil, 2},
		{"fresh has business", false, false, nil, 1, ErrMigrationMode, 0}, {"fresh prefix", false, true, []Record{first}, 1, ErrMigrationMode, 0},
		{"fresh current no DDL", false, true, []Record{first, second}, 1, nil, 0},
		{"pending missing", true, false, nil, 0, ErrMigrationMode, 0}, {"pending empty", true, true, nil, 0, ErrMigrationMode, 0},
		{"pending prefix", true, true, []Record{first}, 1, nil, 1}, {"pending current no DDL", true, true, []Record{first, second}, 1, nil, 0},
		{"fresh dirty", false, true, []Record{first, dirty}, 1, ErrDirty, 0}, {"pending dirty", true, true, []Record{first, dirty}, 1, ErrDirty, 0},
		{"fresh drift", false, true, []Record{drift}, 1, ErrDrift, 0}, {"pending drift", true, true, []Record{drift}, 1, ErrDrift, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state := &modeRecorder{adoptionRecorder: adoptionRecorder{ledger: tc.ledger, records: append([]Record(nil), tc.records...)}, business: tc.business}
			db := sql.OpenDB(modeConnector{state})
			defer db.Close()
			call := MigrateFresh
			if tc.pending {
				call = MigratePending
			}
			err := call(context.Background(), db, manifest)
			if !errors.Is(err, tc.want) || state.applied != tc.applied {
				t.Fatalf("classification=%v applied=%d", err, state.applied)
			}
			if tc.applied == 0 && len(state.writes) != 0 {
				t.Fatal("rejected or current mode issued DDL/ledger write")
			}
			if tc.ledger {
				for _, q := range state.writes {
					if q == ledgerDDL {
						t.Fatal("existing ledger received CREATE DDL")
					}
				}
			}
			if state.locked {
				t.Fatal("migration lock not released")
			}
		})
	}
}
func TestConfirmedModeDoesNotTrustPrelockObservation(t *testing.T) {
	manifest := []Migration{SQLMigration(1, "first", "APPLY first"), SQLMigration(2, "second", "APPLY second")}
	first := Record{Version: 1, Name: manifest[0].Name, Checksum: manifest[0].Checksum, State: "applied"}
	for _, mode := range []string{"fresh becomes v1", "pending becomes missing", "pending becomes empty"} {
		t.Run(mode, func(t *testing.T) {
			state := &modeRecorder{}
			if mode != "fresh becomes v1" {
				state.ledger = true
				state.records = []Record{first}
			}
			state.onAcquire = func() {
				if mode == "fresh becomes v1" {
					state.ledger = true
					state.records = []Record{first}
					state.business = 1
				} else {
					state.ledger = mode == "pending becomes empty"
					state.records = nil
				}
			}
			db := sql.OpenDB(modeConnector{state})
			defer db.Close()
			call := MigratePending
			if mode == "fresh becomes v1" {
				call = MigrateFresh
			}
			if err := call(context.Background(), db, manifest); !errors.Is(err, ErrMigrationMode) || len(state.writes) != 0 {
				t.Fatal("stale prelock mode authorized writes")
			}
		})
	}
}
