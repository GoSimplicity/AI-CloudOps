package schema

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestAdoptionRejectsInvalidInputsWithoutDatabase(t *testing.T) {
	validator := func(context.Context, *sql.Conn) error { t.Fatal("invalid request called validator"); return nil }
	cases := []struct {
		name     string
		manifest []Migration
		validate func(context.Context, *sql.Conn) error
		note     string
		want     error
	}{
		{"no manifest", nil, validator, "reviewed", ErrManifest},
		{"multiple versions", append(validManifest(), SQLMigration(2, "second", "SELECT 1")), validator, "reviewed", ErrManifest},
		{"invalid manifest", []Migration{{Version: 1}}, validator, "reviewed", ErrManifest},
		{"nil validator", validManifest(), nil, "reviewed", ErrAdoption},
		{"empty note", validManifest(), validator, "", ErrAdoption},
		{"blank note", validManifest(), validator, " \n\t ", ErrAdoption},
		{"invalid UTF8", validManifest(), validator, "bad\xff", ErrAdoption},
		{"oversized note", validManifest(), validator, strings.Repeat("x", 1001), ErrAdoption},
		{"UTF8 byte bound", validManifest(), validator, strings.Repeat("中", 334), ErrAdoption},
		{"maximum note accepted", validManifest(), validator, strings.Repeat("x", 1000), ErrPreflight},
		{"UTF8 accepted", validManifest(), validator, strings.Repeat("中", 333), ErrPreflight},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := AdoptBaseline(context.Background(), nil, tc.manifest, tc.validate, tc.note); !errors.Is(err, tc.want) {
				t.Fatalf("got %v want %v", err, tc.want)
			}
		})
	}
}

func TestAdoptionLedgerDerivesCanonicalColumnsWithoutIfNotExists(t *testing.T) {
	ddl, err := adoptionLedgerDDL()
	if err != nil {
		t.Fatal(err)
	}
	body := strings.TrimSuffix(strings.TrimPrefix(ledgerDDL, "CREATE TABLE IF NOT EXISTS cloudops_schema_history ("), "\n) ENGINE=InnoDB")
	if !strings.HasPrefix(ddl, "CREATE TABLE cloudops_schema_history ("+body+",") || strings.Contains(ddl, "IF NOT EXISTS") {
		t.Fatal("adoption changed canonical columns or concealed an existing table")
	}
	for _, audit := range []string{"origin VARCHAR(16)", "DEFAULT 'migrated'", "adoption_note VARCHAR(1000)", "adopted_by VARCHAR(288)"} {
		if !strings.Contains(ddl, audit) {
			t.Fatal("audit column contract is missing")
		}
	}
}

func TestAdoptionStateMachineNeverAppliesMigrationOrLeaksEvidence(t *testing.T) {
	manifest := validManifest()
	good := Record{Version: 1, Name: manifest[0].Name, Checksum: manifest[0].Checksum, State: "applied"}
	bad := good
	bad.State = "failed"
	drift := good
	drift.Checksum = Checksum([]byte("changed"))
	cases := []struct {
		name           string
		ledger         bool
		records        []Record
		validatorError error
		failure        string
		want           error
		wantValidation int
		wantWrites     int
	}{
		{"new adoption", false, nil, nil, "", nil, 1, 2},
		{"existing current", true, []Record{good}, nil, "", nil, 0, 0},
		{"empty ledger", true, nil, nil, "", ErrPending, 0, 0},
		{"dirty ledger", true, []Record{bad}, nil, "", ErrDirty, 0, 0},
		{"drifted ledger", true, []Record{drift}, nil, "", ErrDrift, 0, 0},
		{"future ledger", true, []Record{good, good}, nil, "", ErrDrift, 0, 0},
		{"validator fails", false, nil, errors.New("private-validator-canary"), "", ErrPreflight, 1, 0},
		{"metadata unavailable", false, nil, nil, "metadata", ErrPreflight, 0, 0},
		{"history unavailable", true, []Record{good}, nil, "history", ErrPreflight, 0, 0},
		{"DDL outcome uncertain", false, nil, nil, "ddl", ErrUncertain, 1, 1},
		{"INSERT outcome uncertain", false, nil, nil, "insert", ErrUncertain, 1, 2},
		{"INSERT affected rows unavailable", false, nil, nil, "rows_error", ErrUncertain, 1, 2},
		{"INSERT unexpected rows", false, nil, nil, "rows_zero", ErrUncertain, 1, 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state := &adoptionRecorder{ledger: tc.ledger, records: tc.records, failure: tc.failure}
			db := sql.OpenDB(adoptionConnector{state})
			defer db.Close()
			calls := 0
			manifest := validManifest()
			manifest[0].Apply = func(context.Context, *sql.Conn) error { t.Fatal("adoption called migration Apply"); return nil }
			note := "reviewed ' ; private-note-canary \n保留原始说明"
			err := AdoptBaseline(context.Background(), db, manifest, func(ctx context.Context, conn *sql.Conn) error {
				calls++
				var id int
				if err := conn.QueryRowContext(ctx, "SELECT CONNECTION_ID()").Scan(&id); err != nil || id != 1 {
					t.Fatal("validator is not using the lock-owning physical connection")
				}
				return tc.validatorError
			}, note)
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v want %v", err, tc.want)
			}
			if calls != tc.wantValidation || len(state.writes) != tc.wantWrites {
				t.Fatalf("validator=%d writes=%d", calls, len(state.writes))
			}
			if err != nil && (strings.Contains(err.Error(), "private-note-canary") || strings.Contains(err.Error(), "private-validator-canary") || strings.Contains(err.Error(), "private-driver-canary")) {
				t.Fatal("private evidence leaked in error")
			}
			for _, statement := range state.writes {
				if strings.Contains(statement, note) {
					t.Fatal("adoption note was interpolated into SQL")
				}
			}
			if tc.want == nil && !tc.ledger && state.note != note {
				t.Fatal("audit note was not preserved exactly as a bound argument")
			}
		})
	}
}

func TestUncertainAdoptionDoesNotAutomaticallyFillEmptyLedger(t *testing.T) {
	for _, failure := range []string{"ddl", "insert"} {
		t.Run(failure, func(t *testing.T) {
			state := &adoptionRecorder{failure: failure}
			db := sql.OpenDB(adoptionConnector{state})
			defer db.Close()
			validatorCalls := 0
			validate := func(context.Context, *sql.Conn) error { validatorCalls++; return nil }
			if err := AdoptBaseline(context.Background(), db, validManifest(), validate, "approved"); !errors.Is(err, ErrUncertain) {
				t.Fatal("missing uncertain outcome")
			}
			writesBefore := len(state.writes)
			state.failure = ""
			if err := AdoptBaseline(context.Background(), db, validManifest(), validate, "retry"); !errors.Is(err, ErrPending) {
				t.Fatal("empty ledger was filled on retry")
			}
			if validatorCalls != 1 || len(state.writes) != writesBefore {
				t.Fatal("retry executed validator or write")
			}
		})
	}
}

func TestAdoptionPanicDiscardsConnectionAndSanitizesValue(t *testing.T) {
	state := &adoptionRecorder{}
	db := sql.OpenDB(adoptionConnector{state})
	defer db.Close()
	err := AdoptBaseline(context.Background(), db, validManifest(), func(context.Context, *sql.Conn) error { panic("private-panic-canary") }, "approved")
	if !errors.Is(err, ErrUncertain) || strings.Contains(err.Error(), "private-panic-canary") || state.closed != 1 {
		t.Fatal("callback panic was not sanitized and physical connection discarded")
	}
	if len(state.writes) != 0 {
		t.Fatal("panic before adoption modified history")
	}
}

type adoptionRecorder struct {
	ledger  bool
	records []Record
	failure string
	writes  []string
	note    string
	closed  int
}
type adoptionConnector struct{ state *adoptionRecorder }

func (c adoptionConnector) Connect(context.Context) (driver.Conn, error) {
	return adoptionConnection{c.state}, nil
}
func (adoptionConnector) Driver() driver.Driver { return adoptionDriver{} }

type adoptionDriver struct{}

func (adoptionDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New("unexpected direct Open")
}

type adoptionConnection struct{ state *adoptionRecorder }

func (adoptionConnection) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("unexpected prepare")
}
func (c adoptionConnection) Close() error { c.state.closed++; return nil }
func (adoptionConnection) Begin() (driver.Tx, error) {
	return nil, errors.New("unexpected adoption transaction")
}
func (c adoptionConnection) QueryContext(_ context.Context, statement string, _ []driver.NamedValue) (driver.Rows, error) {
	switch {
	case statement == "SELECT LOWER(DATABASE()), @@session.autocommit, @@global.read_only":
		return &adoptionRows{columns: []string{"database", "autocommit", "read_only"}, values: [][]driver.Value{{"adoptfixture", int64(1), int64(0)}}}, nil
	case strings.HasPrefix(statement, "SELECT GET_LOCK"), strings.HasPrefix(statement, "SELECT RELEASE_LOCK"), statement == "SELECT CONNECTION_ID()":
		return &adoptionRows{columns: []string{"value"}, values: [][]driver.Value{{int64(1)}}}, nil
	case strings.HasPrefix(statement, "SELECT EXISTS(SELECT 1 FROM information_schema.tables"):
		if c.state.failure == "metadata" {
			return nil, errors.New("private-driver-canary")
		}
		return &adoptionRows{columns: []string{"exists"}, values: [][]driver.Value{{c.state.ledger}}}, nil
	case strings.HasPrefix(statement, "SELECT version, name, checksum, state FROM cloudops_schema_history"):
		if c.state.failure == "history" {
			return nil, errors.New("private-driver-canary")
		}
		rows := &adoptionRows{columns: []string{"version", "name", "checksum", "state"}}
		for _, record := range c.state.records {
			rows.values = append(rows.values, []driver.Value{int64(record.Version), record.Name, record.Checksum, record.State})
		}
		return rows, nil
	default:
		return nil, errors.New("unexpected SELECT")
	}
}
func (c adoptionConnection) ExecContext(_ context.Context, statement string, args []driver.NamedValue) (driver.Result, error) {
	c.state.writes = append(c.state.writes, statement)
	if strings.HasPrefix(statement, "CREATE TABLE cloudops_schema_history") {
		c.state.ledger = true
		if c.state.failure == "ddl" {
			return nil, errors.New("private-driver-canary")
		}
		return driver.RowsAffected(0), nil
	}
	if strings.HasPrefix(statement, "INSERT INTO cloudops_schema_history") {
		if c.state.failure == "insert" {
			return nil, errors.New("private-driver-canary")
		}
		c.state.note = args[3].Value.(string)
		if c.state.failure == "rows_error" {
			return adoptionRowsError{}, nil
		}
		if c.state.failure == "rows_zero" {
			return driver.RowsAffected(0), nil
		}
		return driver.RowsAffected(1), nil
	}
	return nil, errors.New("unexpected write")
}

type adoptionRowsError struct{}

func (adoptionRowsError) LastInsertId() (int64, error) { return 0, nil }
func (adoptionRowsError) RowsAffected() (int64, error) { return 0, errors.New("private-driver-canary") }

type adoptionRows struct {
	columns []string
	values  [][]driver.Value
	index   int
}

func (r *adoptionRows) Columns() []string { return r.columns }
func (*adoptionRows) Close() error        { return nil }
func (r *adoptionRows) Next(values []driver.Value) error {
	if r.index >= len(r.values) {
		return io.EOF
	}
	copy(values, r.values[r.index])
	r.index++
	return nil
}
