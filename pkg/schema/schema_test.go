package schema

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
)

func validManifest() []Migration {
	return []Migration{SQLMigration(1, "first", "CREATE TABLE example (id BIGINT)")}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name   string
		change func([]Migration) []Migration
		want   error
	}{
		{"valid", func(m []Migration) []Migration { return m }, nil},
		{"empty", func([]Migration) []Migration { return nil }, ErrManifest},
		{"gap", func(m []Migration) []Migration { m[0].Version = 2; return m }, ErrManifest},
		{"duplicate", func(m []Migration) []Migration { return append(m, m[0]) }, ErrManifest},
		{"empty name", func(m []Migration) []Migration { m[0].Name = ""; return m }, ErrManifest},
		{"long name", func(m []Migration) []Migration { m[0].Name = strings.Repeat("x", 191); return m }, ErrManifest},
		{"bad checksum", func(m []Migration) []Migration { m[0].Checksum = strings.Repeat("z", 64); return m }, ErrManifest},
		{"missing callback", func(m []Migration) []Migration { m[0].Apply = nil; return m }, ErrManifest},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := Validate(test.change(validManifest())); !errors.Is(err, test.want) {
				t.Fatalf("got %v want %v", err, test.want)
			}
		})
	}
}

func TestInspect(t *testing.T) {
	manifest := validManifest()
	good := Record{Version: 1, Name: manifest[0].Name, Checksum: manifest[0].Checksum, State: "applied"}
	tests := []struct {
		name    string
		records []Record
		want    error
	}{
		{"empty", nil, nil}, {"applied", []Record{good}, nil},
		{"future version", []Record{good, good}, ErrDrift},
	}
	for _, state := range []string{"started", "failed", "unknown"} {
		record := good
		record.State = state
		tests = append(tests, struct {
			name    string
			records []Record
			want    error
		}{state, []Record{record}, ErrDirty})
	}
	changed := good
	changed.Checksum = Checksum([]byte("changed"))
	tests = append(tests, struct {
		name    string
		records []Record
		want    error
	}{"edited implementation", []Record{changed}, ErrDrift})
	changed = good
	changed.Version = 2
	tests = append(tests, struct {
		name    string
		records []Record
		want    error
	}{"history gap", []Record{changed}, ErrDrift})
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := inspect(manifest, test.records); !errors.Is(err, test.want) {
				t.Fatalf("got %v want %v", err, test.want)
			}
		})
	}
}

func TestSQLMigrationChecksum(t *testing.T) {
	tests := []struct {
		name string
		a, b []string
		same bool
	}{
		{"same", []string{"AB", "C"}, []string{"AB", "C"}, true},
		{"boundaries", []string{"AB", "C"}, []string{"A", "BC"}, false},
		{"order", []string{"A", "B"}, []string{"B", "A"}, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			a := SQLMigration(1, "first", test.a...)
			b := SQLMigration(1, "first", test.b...)
			if (a.Checksum == b.Checksum) != test.same {
				t.Fatal("checksum does not encode ordered statement boundaries")
			}
		})
	}
}

func TestNilDatabase(t *testing.T) {
	tests := []struct {
		name string
		run  func(context.Context, *sql.DB, []Migration) error
	}{{"verify", Verify}, {"migrate", Migrate}}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if !errors.Is(test.run(context.Background(), nil, validManifest()), ErrPreflight) {
				t.Fatal("nil database not rejected")
			}
		})
	}
}

func TestCallbackPanicAndUnexpectedStateDoNotLeakValues(t *testing.T) {
	err := applyMigration(context.Background(), nil, func(context.Context, *sql.Conn) error { panic("private-panic-canary") })
	if !errors.Is(err, ErrDirty) || strings.Contains(err.Error(), "private-panic-canary") {
		t.Fatal("panic outcome was not sanitized and dirty")
	}
	manifest := validManifest()
	err = inspect(manifest, []Record{{Version: 1, Name: manifest[0].Name, Checksum: manifest[0].Checksum, State: "private-state"}})
	if !errors.Is(err, ErrDirty) || strings.Contains(err.Error(), "private-state") {
		t.Fatal("unexpected history state leaked")
	}
}
