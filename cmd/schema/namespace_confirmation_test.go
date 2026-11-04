package main

import (
	"bytes"
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/dbschema"
	"github.com/GoSimplicity/AI-CloudOps/pkg/schema"
)

func TestMigrationConfirmationsAreExplicitAndMutuallyExclusive(t *testing.T) {
	for _, tc := range []struct {
		name  string
		args  []string
		valid bool
	}{
		{"fresh", []string{"migrate", "--confirm-new-database"}, true},
		{"pending stopped", []string{"migrate", "--confirm-pending-migrations", "--confirm-no-writers"}, true},
		{"none", []string{"migrate"}, false},
		{"pending with writers", []string{"migrate", "--confirm-pending-migrations"}, false},
		{"both modes", []string{"migrate", "--confirm-new-database", "--confirm-pending-migrations", "--confirm-no-writers"}, false},
		{"pending outside migrate", []string{"verify", "--confirm-pending-migrations", "--confirm-no-writers"}, false},
		{"fresh flag keeps old no-writers rejection", []string{"migrate", "--confirm-new-database", "--confirm-no-writers"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseOptions(tc.args)
			if (err == nil) != tc.valid {
				t.Fatal("migration confirmation contract changed")
			}
		})
	}
}
func TestPendingAndFreshModesCannotAuthorizeEachOther(t *testing.T) {
	pending := []string{"migrate", "--confirm-pending-migrations", "--confirm-no-writers"}
	fresh := []string{"migrate", "--confirm-new-database"}
	for _, tc := range []struct {
		name                 string
		args                 []string
		verify               []error
		freshErr, pendingErr error
		status               int
		actions              []string
	}{
		{"fresh missing", fresh, []error{schema.ErrPreflight, nil}, nil, nil, 0, []string{"verify", "preflight", "migrate", "verify", "close"}},
		{"fresh empty ledger", fresh, []error{schema.ErrPending, nil}, nil, nil, 0, []string{"verify", "preflight", "migrate", "verify", "close"}},
		{"v1 cannot use fresh authority", fresh, []error{schema.ErrPending}, dbschema.ErrLegacy, nil, 1, []string{"verify", "preflight", "close"}},
		{"missing cannot use pending authority", pending, []error{schema.ErrPreflight}, nil, nil, 1, []string{"verify", "close"}},
		{"empty cannot use pending authority", pending, []error{schema.ErrPending}, nil, schema.ErrPending, 1, []string{"verify", "pending-preflight", "close"}},
		{"v1 stopped permits pending", pending, []error{schema.ErrPending, nil}, nil, nil, 0, []string{"verify", "pending-preflight", "migrate-pending", "verify", "close"}},
		{"pending dirty", pending, []error{schema.ErrDirty}, nil, nil, 1, []string{"verify", "close"}},
		{"pending drift", pending, []error{schema.ErrDrift}, nil, nil, 1, []string{"verify", "close"}},
		{"dirty appears during preflight", pending, []error{schema.ErrPending}, nil, schema.ErrDirty, 1, []string{"verify", "pending-preflight", "close"}},
		{"already current pending no DDL", pending, nil, nil, nil, 0, []string{"verify", "close"}},
		{"already current fresh no DDL", fresh, nil, nil, nil, 0, []string{"verify", "close"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := &fakeDatabase{verifyErrors: tc.verify, preflightError: tc.freshErr, pendingError: tc.pendingErr}
			var out, diagnostic bytes.Buffer
			code := run(context.Background(), tc.args, &out, &diagnostic, dependencies{manifest: dbschema.CurrentManifest, getenv: func(string) string { return "fixture" }, open: func(string, time.Duration) (schemaDatabase, error) { return db, nil }})
			if code != tc.status || !reflect.DeepEqual(db.actions, tc.actions) {
				t.Fatalf("code=%d actions=%v", code, db.actions)
			}
		})
	}
}
