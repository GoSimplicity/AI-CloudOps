package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/dbschema"
	"github.com/GoSimplicity/AI-CloudOps/pkg/schema"
)

const privateCanary = "credential-error-canary"

type fakeDatabase struct {
	actions        []string
	verifyErrors   []error
	preflightError error
	migrationError error
	clockError     error
	deadline       bool
}

func (db *fakeDatabase) Verify(ctx context.Context, _ []schema.Migration) error {
	db.actions = append(db.actions, "verify")
	_, db.deadline = ctx.Deadline()
	if len(db.verifyErrors) == 0 {
		return nil
	}
	err := db.verifyErrors[0]
	db.verifyErrors = db.verifyErrors[1:]
	return err
}
func (db *fakeDatabase) PreflightFresh(context.Context) error {
	db.actions = append(db.actions, "preflight")
	return db.preflightError
}
func (db *fakeDatabase) Migrate(context.Context, []schema.Migration) error {
	db.actions = append(db.actions, "migrate")
	return db.migrationError
}
func (db *fakeDatabase) VerifyClock(context.Context, string) error {
	db.actions = append(db.actions, "verify-clock")
	return db.clockError
}
func (db *fakeDatabase) InitializeClock(context.Context, string) error {
	db.actions = append(db.actions, "init-clock")
	return db.clockError
}
func (db *fakeDatabase) Close() error { db.actions = append(db.actions, "close"); return nil }

func TestDefaultPlanIsOffline(t *testing.T) {
	var out, diagnostic bytes.Buffer
	for _, args := range [][]string{nil, {"plan"}, {"--help"}} {
		status := run(context.Background(), args, &out, &diagnostic, dependencies{
			manifest: dbschema.Manifest,
			getenv:   func(string) string { t.Fatal("offline command read credentials"); return "" },
			open: func(string, time.Duration) (schemaDatabase, error) {
				t.Fatal("offline command opened database")
				return nil, nil
			},
		})
		if status != 0 {
			t.Fatalf("offline command failed: %d", status)
		}
	}
	if !strings.Contains(out.String(), "database state has not been inspected") || !strings.Contains(out.String(), "version=1") {
		t.Fatal("plan must distinguish manifest listing from database verification")
	}
}

func TestInvalidArgumentsNeverOpenDatabaseOrEchoValues(t *testing.T) {
	cases := [][]string{
		{"migrate"},
		{"verify", "--confirm-new-database"},
		{"plan", "--dsn-file", privateCanary},
		{"verify", "--timeout", privateCanary},
		{"verify", "--timeout", "0s"},
		{"verify", "--timeout", "31m"},
		{"verify", privateCanary},
		{privateCanary},
		{"migrate", "--confirm-new-database", "--unknown", privateCanary},
	}
	for i, args := range cases {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			var out, diagnostic bytes.Buffer
			status := run(context.Background(), args, &out, &diagnostic, dependencies{
				manifest: func() ([]schema.Migration, error) { t.Fatal("invalid command loaded manifest"); return nil, nil },
				getenv:   func(string) string { t.Fatal("invalid command read credentials"); return "" },
				open: func(string, time.Duration) (schemaDatabase, error) {
					t.Fatal("invalid command opened database")
					return nil, nil
				},
			})
			if status != 2 {
				t.Fatalf("got status %d", status)
			}
			if strings.Contains(out.String()+diagnostic.String(), privateCanary) {
				t.Fatal("argument value leaked")
			}
		})
	}
}

func TestDatabaseCommandLifecycle(t *testing.T) {
	cases := []struct {
		name                 string
		args                 []string
		verify               []error
		preflight, migration error
		wantStatus           int
		wantActions          []string
	}{
		{"verify current", []string{"verify"}, nil, nil, nil, 0, []string{"verify", "close"}},
		{"verify pending never migrates", []string{"verify"}, []error{schema.ErrPending}, nil, nil, 1, []string{"verify", "close"}},
		{"verify unavailable never preflights", []string{"verify"}, []error{schema.ErrPreflight}, nil, nil, 1, []string{"verify", "close"}},
		{"new database", []string{"migrate", "--confirm-new-database"}, []error{schema.ErrPreflight, nil}, nil, nil, 0, []string{"verify", "preflight", "migrate", "verify", "close"}},
		{"empty existing history", []string{"migrate", "--confirm-new-database"}, []error{schema.ErrPending, nil}, nil, nil, 0, []string{"verify", "preflight", "migrate", "verify", "close"}},
		{"already current no writes", []string{"migrate", "--confirm-new-database"}, nil, nil, nil, 0, []string{"verify", "close"}},
		{"legacy blocked before write", []string{"migrate", "--confirm-new-database"}, []error{schema.ErrPreflight}, dbschema.ErrLegacy, nil, 1, []string{"verify", "preflight", "close"}},
		{"dirty blocked", []string{"migrate", "--confirm-new-database"}, []error{schema.ErrDirty}, nil, nil, 1, []string{"verify", "close"}},
		{"drift blocked", []string{"migrate", "--confirm-new-database"}, []error{schema.ErrDrift}, nil, nil, 1, []string{"verify", "close"}},
		{"unknown verify blocked", []string{"migrate", "--confirm-new-database"}, []error{errors.New(privateCanary)}, nil, nil, 1, []string{"verify", "close"}},
		{"failed preflight", []string{"migrate", "--confirm-new-database"}, []error{schema.ErrPreflight}, errors.New(privateCanary), nil, 1, []string{"verify", "preflight", "close"}},
		{"partial write not retried", []string{"migrate", "--confirm-new-database"}, []error{schema.ErrPreflight}, nil, errors.Join(schema.ErrUncertain, errors.New(privateCanary)), 1, []string{"verify", "preflight", "migrate", "close"}},
		{"post write verify fails", []string{"migrate", "--confirm-new-database"}, []error{schema.ErrPreflight, schema.ErrDirty}, nil, nil, 1, []string{"verify", "preflight", "migrate", "verify", "close"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := &fakeDatabase{verifyErrors: tc.verify, preflightError: tc.preflight, migrationError: tc.migration}
			var out, diagnostic bytes.Buffer
			status := run(context.Background(), tc.args, &out, &diagnostic, dependencies{
				manifest: dbschema.Manifest,
				getenv: func(key string) string {
					if key != credentialEnvironment {
						t.Fatal("read runtime environment")
					}
					return privateCanary
				},
				open: func(dsn string, timeout time.Duration) (schemaDatabase, error) {
					if dsn != privateCanary || timeout != 2*time.Minute {
						t.Fatal("incorrect configuration")
					}
					return db, nil
				},
			})
			if status != tc.wantStatus || !reflect.DeepEqual(db.actions, tc.wantActions) {
				t.Fatalf("status %d actions %v", status, db.actions)
			}
			if !db.deadline {
				t.Fatal("database operation has no deadline")
			}
			if strings.Contains(out.String()+diagnostic.String(), privateCanary) {
				t.Fatal("raw error or credential leaked")
			}
		})
	}
}

func TestCredentialAndOpenErrorsAreRedacted(t *testing.T) {
	for _, supplied := range []bool{false, true} {
		var out, diagnostic bytes.Buffer
		status := run(context.Background(), []string{"verify"}, &out, &diagnostic, dependencies{
			manifest: dbschema.Manifest,
			getenv: func(string) string {
				if supplied {
					return privateCanary
				}
				return ""
			},
			open: func(string, time.Duration) (schemaDatabase, error) { return nil, errors.New(privateCanary) },
		})
		if status == 0 || strings.Contains(out.String()+diagnostic.String(), privateCanary) {
			t.Fatal("error missing or not redacted")
		}
	}
}

func TestDedicatedEnvironmentRejectsMultilineAndOversized(t *testing.T) {
	for _, body := range []string{"", "one\ntwo", "one\r\ntwo", "one\x00two", strings.Repeat("x", 64*1024+1)} {
		if _, err := readCredentials("", func(string) string { return body }); err == nil {
			t.Fatal("invalid credential accepted")
		}
	}
}

func TestDriverConfigurationIsBoundedAndDoesNotLogRawErrors(t *testing.T) {
	config, err := databaseConfig("fixture-user:fixture-pass@tcp(127.0.0.1:9)/fixture?timeout=1h&readTimeout=1h&writeTimeout=1h", 10*time.Second)
	if err != nil {
		t.Fatal("valid DSN rejected")
	}
	if config.Timeout != 5*time.Second || config.ReadTimeout != 10*time.Second || config.WriteTimeout != 10*time.Second {
		t.Fatal("transport timeouts are unbounded")
	}
	// The installed driver accepts per-config Logger; this deliberately avoids
	// process-global logger mutation and all raw MySQL diagnostics.
	if fmt.Sprintf("%T", config.Logger) != "*mysql.NopLogger" {
		t.Fatal("driver logging may expose raw errors")
	}
	for _, suffix := range []string{"?multiStatements=true", "?allowAllFiles=true", "?allowCleartextPasswords=true", "?allowFallbackToPlaintext=true"} {
		if _, err := databaseConfig("user:pass@tcp(127.0.0.1:9)/fixture"+suffix, time.Minute); err == nil {
			t.Fatal("unsafe DSN flag accepted")
		}
	}
	for _, dsn := range []string{"invalid", "user:pass@tcp(127.0.0.1:9)/"} {
		if _, err := databaseConfig(dsn, time.Minute); err == nil {
			t.Fatal("invalid or unscoped DSN accepted")
		}
	}
}

func TestClockCommandsRequireExplicitIntentAndPreserveHistory(t *testing.T) {
	for _, args := range [][]string{
		{"init-clock"}, {"init-clock", "--notification-timezone", "UTC"},
		{"init-clock", "--confirm-no-writers", "--notification-timezone", "Local"},
		{"verify-clock"}, {"verify-clock", "--confirm-no-writers", "--notification-timezone", "UTC"},
		{"verify", "--notification-timezone", "UTC"},
	} {
		if _, err := parseOptions(args); err == nil {
			t.Fatal("unsafe clock options accepted")
		}
	}
	cases := []struct {
		name     string
		args     []string
		clockErr error
		want     int
		actions  []string
	}{
		{"read only", []string{"verify-clock", "--notification-timezone", "UTC"}, nil, 0, []string{"verify", "verify-clock", "close"}},
		{"initialize", []string{"init-clock", "--confirm-no-writers", "--notification-timezone", "Asia/Shanghai"}, nil, 0, []string{"verify", "init-clock", "verify-clock", "close"}},
		{"legacy rejected", []string{"init-clock", "--confirm-no-writers", "--notification-timezone", "UTC"}, dbschema.ErrClockHistory, 1, []string{"verify", "init-clock", "close"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := &fakeDatabase{clockError: tc.clockErr}
			var out, diagnostic bytes.Buffer
			code := run(context.Background(), tc.args, &out, &diagnostic, dependencies{
				manifest: dbschema.Manifest, getenv: func(string) string { return "fixture" }, open: func(string, time.Duration) (schemaDatabase, error) { return db, nil },
			})
			if code != tc.want || !reflect.DeepEqual(db.actions, tc.actions) {
				t.Fatalf("status=%d actions=%v", code, db.actions)
			}
		})
	}
}

func TestRolesPlanNeverReadsCredentialsOrOpensDatabase(t *testing.T) {
	var out, diagnostic bytes.Buffer
	args := []string{"roles", "--database", "cloudops", "--runtime-role", "app", "--runtime-role-host", "%", "--migrator-role", "migration", "--migrator-role-host", "%"}
	code := run(context.Background(), args, &out, &diagnostic, dependencies{
		manifest: dbschema.Manifest, getenv: func(string) string { t.Fatal("offline roles read credentials"); return "" },
		open: func(string, time.Duration) (schemaDatabase, error) {
			t.Fatal("offline roles opened database")
			return nil, nil
		},
	})
	if code != 0 || !strings.Contains(out.String(), "GRANT SELECT ON `cloudops`.`cloudops_schema_history`") || strings.Contains(out.String(), "IDENTIFIED BY") {
		t.Fatal("offline role output invalid")
	}
	for _, invalid := range [][]string{{"roles"}, append(append([]string{}, args...), "--dsn-file", privateCanary), {"verify", "--database", "cloudops"}} {
		if _, err := parseOptions(invalid); err == nil {
			t.Fatal("unsafe role arguments accepted")
		}
	}
}

func TestStructuredRolesPlanKeepsPrivilegeStagesSeparate(t *testing.T) {
	for _, section := range []string{"migrator", "runtime"} {
		var out, diagnostic bytes.Buffer
		args := []string{"roles", "--database", "cloudops", "--runtime-role", "app", "--runtime-role-host", "%", "--migrator-role", "migration", "--migrator-role-host", "%", "--section", section, "--format", "json"}
		code := run(context.Background(), args, &out, &diagnostic, dependencies{manifest: dbschema.Manifest,
			getenv: func(string) string { t.Fatal("offline JSON read credentials"); return "" },
			open: func(string, time.Duration) (schemaDatabase, error) {
				t.Fatal("offline JSON opened database")
				return nil, nil
			},
		})
		var payload struct {
			Version    int
			Database   string
			Section    string
			Statements []string
		}
		if code != 0 || json.Unmarshal(out.Bytes(), &payload) != nil || payload.Version != 1 || payload.Database != "cloudops" || payload.Section != section || len(payload.Statements) == 0 {
			t.Fatal("structured plan contract invalid")
		}
		expected, err := BuildRolePlan("cloudops", RoleAccount{"app", "%"}, RoleAccount{"migration", "%"})
		if err != nil {
			t.Fatal(err)
		}
		statements := expected.Migrator
		if section == "runtime" {
			statements = expected.Runtime
		}
		if !reflect.DeepEqual(payload.Statements, statements) {
			t.Fatal("privilege stages mixed")
		}
	}
}
