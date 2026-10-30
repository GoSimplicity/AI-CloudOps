package schema

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func upgradeFixturePlan() BaselineUpgrade {
	callback := func(context.Context, *sql.Conn, UpgradeEvidence) error { return nil }
	return BaselineUpgrade{Source: "a8f1700675bef0fe950bce5018893b9a2366f3b4", SourceChecksum: Checksum([]byte("source fixture")), ImplementationChecksum: Checksum([]byte("upgrade fixture")), ValidateSource: callback, Apply: callback, ValidateTarget: callback}
}
func upgradeFixtureEvidence() UpgradeEvidence {
	return UpgradeEvidence{Note: "reviewed private-note-canary ' ;\n已备份", BackupSHA256: Checksum([]byte("private-backup-canary")), NotificationZone: "UTC", HistoricalTimeAttested: true}
}

func TestUpgradeValidatesBeforeDatabase(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(*BaselineUpgrade, *UpgradeEvidence)
		want   error
	}{
		{"valid", func(*BaselineUpgrade, *UpgradeEvidence) {}, ErrPreflight},
		{"short source", func(p *BaselineUpgrade, _ *UpgradeEvidence) { p.Source = "a8f" }, ErrUpgrade},
		{"uppercase source", func(p *BaselineUpgrade, _ *UpgradeEvidence) { p.Source = strings.ToUpper(p.Source) }, ErrUpgrade},
		{"source hash", func(p *BaselineUpgrade, _ *UpgradeEvidence) { p.SourceChecksum = "invalid" }, ErrUpgrade},
		{"implementation hash", func(p *BaselineUpgrade, _ *UpgradeEvidence) { p.ImplementationChecksum = strings.Repeat("z", 64) }, ErrUpgrade},
		{"source callback", func(p *BaselineUpgrade, _ *UpgradeEvidence) { p.ValidateSource = nil }, ErrUpgrade},
		{"apply callback", func(p *BaselineUpgrade, _ *UpgradeEvidence) { p.Apply = nil }, ErrUpgrade},
		{"target callback", func(p *BaselineUpgrade, _ *UpgradeEvidence) { p.ValidateTarget = nil }, ErrUpgrade},
		{"blank note", func(_ *BaselineUpgrade, e *UpgradeEvidence) { e.Note = " \n\t" }, ErrUpgrade},
		{"invalid UTF8", func(_ *BaselineUpgrade, e *UpgradeEvidence) { e.Note = "\xff" }, ErrUpgrade},
		{"note byte bound", func(_ *BaselineUpgrade, e *UpgradeEvidence) { e.Note = strings.Repeat("中", 334) }, ErrUpgrade},
		{"note maximum", func(_ *BaselineUpgrade, e *UpgradeEvidence) { e.Note = strings.Repeat("x", 1000) }, ErrPreflight},
		{"backup", func(_ *BaselineUpgrade, e *UpgradeEvidence) { e.BackupSHA256 = "invalid" }, ErrUpgrade},
		{"missing zone", func(_ *BaselineUpgrade, e *UpgradeEvidence) { e.NotificationZone = "" }, ErrUpgrade},
		{"Local zone", func(_ *BaselineUpgrade, e *UpgradeEvidence) { e.NotificationZone = "Local" }, ErrUpgrade},
		{"zone whitespace", func(_ *BaselineUpgrade, e *UpgradeEvidence) { e.NotificationZone = " UTC " }, ErrUpgrade},
		{"invalid zone", func(_ *BaselineUpgrade, e *UpgradeEvidence) { e.NotificationZone = "../invalid" }, ErrUpgrade},
	} {
		t.Run(test.name, func(t *testing.T) {
			plan, evidence := upgradeFixturePlan(), upgradeFixtureEvidence()
			test.change(&plan, &evidence)
			if err := UpgradeBaseline(context.Background(), nil, validManifest(), plan, evidence); !errors.Is(err, test.want) {
				t.Fatalf("got %v want %v", err, test.want)
			}
		})
	}
	for _, manifest := range [][]Migration{nil, append(validManifest(), SQLMigration(2, "next", "SELECT 1")), {{Version: 1}}} {
		if !errors.Is(UpgradeBaseline(context.Background(), nil, manifest, upgradeFixturePlan(), upgradeFixtureEvidence()), ErrManifest) {
			t.Fatal("upgrade accepted a non-v1 manifest")
		}
	}
}

func TestUpgradeAuditDDLDerivesCanonicalBody(t *testing.T) {
	ddl, err := upgradeLedgerDDL()
	if err != nil {
		t.Fatal(err)
	}
	body := strings.TrimSuffix(strings.TrimPrefix(ledgerDDL, "CREATE TABLE IF NOT EXISTS cloudops_schema_history ("), "\n) ENGINE=InnoDB")
	if !strings.HasPrefix(ddl, "CREATE TABLE cloudops_schema_history ("+body+",") || strings.Contains(ddl, "IF NOT EXISTS") {
		t.Fatal("canonical columns changed or audit creation could hide an old table")
	}
	for _, column := range []string{"origin VARCHAR(16)", "source_revision CHAR(40)", "source_checksum CHAR(64)", "upgrade_checksum CHAR(64)", "upgrade_actor VARCHAR(288)", "upgrade_note VARCHAR(1000)", "backup_sha256 CHAR(64)", "notification_zone VARCHAR(100)", "historical_time_attested TINYINT(1)"} {
		if !strings.Contains(ddl, column) {
			t.Fatal("upgrade audit contract missing")
		}
	}
}

func TestUpgradeStateMachine(t *testing.T) {
	good := Record{Version: 1, Name: validManifest()[0].Name, Checksum: validManifest()[0].Checksum, State: "applied"}
	for _, test := range []struct {
		name              string
		ledger            bool
		records           []Record
		failure, callback string
		want              error
		writes            int
	}{
		{"new upgrade", false, nil, "", "", nil, 4},
		{"applied repeat", true, []Record{good}, "", "", nil, 0},
		{"empty ledger", true, nil, "", "", ErrPending, 0},
		{"started", true, []Record{{Version: 1, Name: good.Name, Checksum: good.Checksum, State: "started"}}, "", "", ErrDirty, 0},
		{"failed", true, []Record{{Version: 1, Name: good.Name, Checksum: good.Checksum, State: "failed"}}, "", "", ErrDirty, 0},
		{"drift", true, []Record{{Version: 1, Name: good.Name, Checksum: Checksum([]byte("other")), State: "applied"}}, "", "", ErrDrift, 0},
		{"future", true, []Record{good, good}, "", "", ErrDrift, 0},
		{"metadata unavailable", false, nil, "metadata", "", ErrPreflight, 0},
		{"history unavailable", true, nil, "history", "", ErrPreflight, 0},
		{"source fails", false, nil, "", "source-error", ErrPreflight, 0},
		{"source panics", false, nil, "", "source-panic", ErrPreflight, 0},
		{"create ack lost", false, nil, "ddl", "", ErrUncertain, 1},
		{"insert ack lost", false, nil, "insert", "", ErrUncertain, 2},
		{"insert zero", false, nil, "insert-zero", "", ErrUncertain, 2},
		{"insert rows error", false, nil, "insert-rows", "", ErrUncertain, 2},
		{"apply fails", false, nil, "", "apply-error", ErrDirty, 3},
		{"apply panics", false, nil, "", "apply-panic", ErrDirty, 3},
		{"target fails", false, nil, "", "target-error", ErrDirty, 4},
		{"target panics", false, nil, "", "target-panic", ErrDirty, 4},
		{"failed ack lost", false, nil, "failed", "apply-error", ErrUncertain, 3},
		{"failed zero", false, nil, "failed-zero", "apply-error", ErrUncertain, 3},
		{"failed rows error", false, nil, "failed-rows", "apply-error", ErrUncertain, 3},
		{"applied ack lost", false, nil, "applied", "", ErrUncertain, 4},
		{"applied zero", false, nil, "applied-zero", "", ErrUncertain, 4},
		{"applied rows error", false, nil, "applied-rows", "", ErrUncertain, 4},
		{"cancelled apply", false, nil, "", "apply-cancel", ErrDirty, 3},
		{"callback DML rollback on panic", false, nil, "", "apply-tx-panic", ErrDirty, 5},
	} {
		t.Run(test.name, func(t *testing.T) {
			state := &upgradeRecorder{adoptionRecorder: adoptionRecorder{ledger: test.ledger, records: test.records}, failure: test.failure}
			db := sql.OpenDB(upgradeConnector{state})
			defer db.Close()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			evidence := upgradeFixtureEvidence()
			plan := upgradeFixturePlan()
			manifest := validManifest()
			manifest[0].Apply = func(context.Context, *sql.Conn) error { t.Fatal("upgrade executed the fresh baseline"); return nil }
			phases := []string{}
			callback := func(phase string) func(context.Context, *sql.Conn, UpgradeEvidence) error {
				return func(ctx context.Context, conn *sql.Conn, received UpgradeEvidence) error {
					phases = append(phases, phase)
					if received != evidence {
						t.Fatal("upgrade evidence changed")
					}
					var id int
					if err := conn.QueryRowContext(ctx, "SELECT CONNECTION_ID()").Scan(&id); err != nil || id != 1 {
						t.Fatal("callback used another connection")
					}
					if phase == "target" && (len(state.records) != 1 || state.records[0].State != "started") {
						t.Fatal("target validator did not see started history")
					}
					if test.callback == phase+"-error" {
						return errors.New("private-callback-canary")
					}
					if test.callback == phase+"-panic" {
						panic("private-panic-canary")
					}
					if phase == "apply" {
						if test.callback == "apply-cancel" {
							cancel()
							return errors.New("private-cancel-canary")
						}
						if _, err := conn.ExecContext(ctx, "APPLY FIXTURE DDL"); err != nil {
							return err
						}
						if test.callback == "apply-tx-panic" {
							state.allowTransaction = true
							tx, err := conn.BeginTx(ctx, nil)
							if err != nil {
								return err
							}
							defer tx.Rollback()
							if _, err := tx.ExecContext(ctx, "CALLBACK DML"); err != nil {
								return err
							}
							panic("private-panic-canary")
						}
					}
					return nil
				}
			}
			plan.ValidateSource, plan.Apply, plan.ValidateTarget = callback("source"), callback("apply"), callback("target")
			err := UpgradeBaseline(ctx, db, manifest, plan, evidence)
			if !errors.Is(err, test.want) {
				t.Fatalf("got %v want %v", err, test.want)
			}
			if len(state.writes) != test.writes {
				t.Fatalf("writes=%d want=%d", len(state.writes), test.writes)
			}
			var expectedPhases []string
			if !test.ledger && test.failure != "metadata" {
				expectedPhases = append(expectedPhases, "source")
				if !strings.HasPrefix(test.callback, "source-") && test.failure != "ddl" && !strings.HasPrefix(test.failure, "insert") {
					expectedPhases = append(expectedPhases, "apply")
					if !strings.HasPrefix(test.callback, "apply-") {
						expectedPhases = append(expectedPhases, "target")
					}
				}
			}
			if !reflect.DeepEqual(phases, expectedPhases) && !(len(phases) == 0 && len(expectedPhases) == 0) {
				t.Fatalf("callback phase sequence %v want %v", phases, expectedPhases)
			}
			if err != nil && (strings.Contains(err.Error(), "private") || strings.Contains(err.Error(), evidence.BackupSHA256)) {
				t.Fatal("private callback/driver/evidence leaked")
			}
			for _, statement := range state.writes {
				if strings.Contains(statement, evidence.Note) || strings.Contains(statement, evidence.BackupSHA256) {
					t.Fatal("evidence interpolated into SQL")
				}
			}
			if state.insertArgs != nil {
				if state.insertArgs[6].Value != evidence.Note || state.insertArgs[7].Value != evidence.BackupSHA256 || state.insertArgs[8].Value != evidence.NotificationZone {
					t.Fatal("bound upgrade evidence was not preserved")
				}
			}
			if test.callback == "apply-tx-panic" {
				if state.begins != 1 || state.rollbacks != 1 || !state.businessDDL {
					t.Fatal("DML rollback contract or persistent DDL evidence was lost")
				}
			} else if state.begins != 0 {
				t.Fatal("engine wrapped MySQL DDL in a transaction")
			}
			if state.ledger && !(test.ledger && test.failure == "history") {
				before := len(state.writes)
				beforeCallbacks := len(phases)
				repeat := UpgradeBaseline(context.Background(), db, manifest, plan, evidence)
				if len(state.writes) != before || len(phases) != beforeCallbacks {
					t.Fatal("repeat retried upgrade writes or callbacks")
				}
				if len(state.records) == 0 && !errors.Is(repeat, ErrPending) {
					t.Fatal("empty audit ledger was retried")
				}
			}
		})
	}
}

type upgradeRecorder struct {
	adoptionRecorder
	failure                       string
	insertArgs                    []driver.NamedValue
	allowTransaction, businessDDL bool
	begins, rollbacks             int
}
type upgradeConnector struct{ state *upgradeRecorder }

func (c upgradeConnector) Connect(context.Context) (driver.Conn, error) {
	return upgradeConnection{c.state}, nil
}
func (upgradeConnector) Driver() driver.Driver { return adoptionDriver{} }

type upgradeConnection struct{ state *upgradeRecorder }

func (upgradeConnection) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("private-prepare-canary")
}
func (c upgradeConnection) Close() error { c.state.closed++; return nil }
func (c upgradeConnection) Begin() (driver.Tx, error) {
	c.state.begins++
	if !c.state.allowTransaction {
		return nil, errors.New("private-transaction-canary")
	}
	return upgradeTransaction{c.state}, nil
}
func (c upgradeConnection) QueryContext(ctx context.Context, statement string, args []driver.NamedValue) (driver.Rows, error) {
	c.state.adoptionRecorder.failure = c.state.failure
	return (adoptionConnection{&c.state.adoptionRecorder}).QueryContext(ctx, statement, args)
}
func (c upgradeConnection) ExecContext(_ context.Context, statement string, args []driver.NamedValue) (driver.Result, error) {
	state := c.state
	state.writes = append(state.writes, statement)
	phase := ""
	switch {
	case strings.HasPrefix(statement, "CREATE TABLE cloudops_schema_history"):
		phase = "ddl"
		state.ledger = true
	case strings.HasPrefix(statement, "INSERT INTO cloudops_schema_history"):
		phase = "insert"
		state.insertArgs = append([]driver.NamedValue(nil), args...)
		state.records = []Record{{Version: uint64(args[0].Value.(int64)), Name: args[1].Value.(string), Checksum: args[2].Value.(string), State: "started"}}
	case strings.Contains(statement, "SET state = 'failed'"):
		phase = "failed"
	case strings.Contains(statement, "SET state = 'applied'"):
		phase = "applied"
	case statement == "APPLY FIXTURE DDL":
		state.businessDDL = true
		return driver.RowsAffected(1), nil
	case statement == "CALLBACK DML":
		return driver.RowsAffected(1), nil
	default:
		return nil, errors.New("private-unexpected-write-canary")
	}
	if (phase == "failed" || phase == "applied") && state.failure != phase+"-zero" {
		state.records[0].State = phase
	}
	if state.failure == phase {
		return nil, errors.New("private-driver-canary")
	}
	if state.failure == phase+"-rows" {
		return adoptionRowsError{}, nil
	}
	if state.failure == phase+"-zero" {
		return driver.RowsAffected(0), nil
	}
	return driver.RowsAffected(1), nil
}

type upgradeTransaction struct{ state *upgradeRecorder }

func (upgradeTransaction) Commit() error      { return nil }
func (tx upgradeTransaction) Rollback() error { tx.state.rollbacks++; return nil }

func TestUpgradeRequiresExactlyOneAffectedRow(t *testing.T) {
	for _, test := range []struct {
		name   string
		result sql.Result
		valid  bool
	}{{"nil", nil, false}, {"zero", driver.RowsAffected(0), false}, {"one", driver.RowsAffected(1), true}, {"multiple", driver.RowsAffected(2), false}, {"unavailable", adoptionRowsError{}, false}} {
		t.Run(test.name, func(t *testing.T) {
			if upgradeOneRow(test.result) != test.valid {
				t.Fatal("uncertain row count accepted")
			}
		})
	}
}
