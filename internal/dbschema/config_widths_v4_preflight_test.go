package dbschema

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/pkg/schema"
)

type widthPreflightState struct {
	tableCollation, tableCharset string
	columns                      []configWidthsV4Column
	reads, writes                int
	fail                         bool
	cancel                       context.CancelFunc
}
type widthPreflightConnector struct{ state *widthPreflightState }

func (c widthPreflightConnector) Connect(context.Context) (driver.Conn, error) {
	return widthPreflightConnection{c.state}, nil
}
func (c widthPreflightConnector) Driver() driver.Driver { return triggerGuardDriver{} }

type widthPreflightConnection struct{ state *widthPreflightState }

func (widthPreflightConnection) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("unexpected prepare")
}
func (widthPreflightConnection) Close() error { return nil }
func (widthPreflightConnection) Begin() (driver.Tx, error) {
	return nil, errors.New("unexpected transaction")
}
func (c widthPreflightConnection) ExecContext(context.Context, string, []driver.NamedValue) (driver.Result, error) {
	c.state.writes++
	return driver.RowsAffected(0), nil
}
func (c widthPreflightConnection) QueryContext(_ context.Context, q string, args []driver.NamedValue) (driver.Rows, error) {
	if strings.Contains(q, "collation_character_set_applicability") {
		return &triggerGuardRows{values: []driver.Value{c.state.tableCollation, c.state.tableCharset}}, nil
	}
	if !strings.Contains(q, "information_schema.columns") || len(args) != 2 {
		return nil, errors.New("unexpected width query")
	}
	c.state.reads++
	if c.state.reads == 4 {
		if c.state.fail {
			return nil, errors.New("private-width-canary")
		}
		if c.state.cancel != nil {
			c.state.cancel()
		}
	}
	for _, col := range c.state.columns {
		if args[0].Value == col.table && args[1].Value == col.name {
			var def, charset, collation driver.Value
			if col.defaultValue.Valid {
				def = col.defaultValue.String
			}
			if col.charset.Valid {
				charset = col.charset.String
			}
			if col.collation.Valid {
				collation = col.collation.String
			}
			return &triggerGuardRows{values: []driver.Value{col.dataType, col.length, col.nullable, def, col.comment, charset, collation, col.extra, col.generation}}, nil
		}
	}
	return nil, errors.New("missing width column")
}
func TestConfigWidthsV4ChecksAllAttributesBeforeFirstALTER(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*widthPreflightState)
		want   error
		writes int
	}{
		{"released source", nil, nil, 3},
		{"table collation", func(s *widthPreflightState) { s.tableCollation = "utf8mb4_unicode_ci" }, schema.ErrPreflight, 0},
		{"table charset", func(s *widthPreflightState) { s.tableCharset = "utf8mb3" }, schema.ErrPreflight, 0},
		{"last column changed", func(s *widthPreflightState) { s.columns[3].length = 179 }, schema.ErrPreflight, 0},
		{"collation", func(s *widthPreflightState) { s.columns[0].collation.String = "utf8mb4_unicode_ci" }, schema.ErrPreflight, 0},
		{"charset", func(s *widthPreflightState) { s.columns[0].charset.String = "utf8mb3" }, schema.ErrPreflight, 0},
		{"comment", func(s *widthPreflightState) { s.columns[1].comment = "private-width-canary" }, schema.ErrPreflight, 0},
		{"default", func(s *widthPreflightState) { s.columns[0].defaultValue = sql.NullString{String: "", Valid: true} }, schema.ErrPreflight, 0},
		{"nullable", func(s *widthPreflightState) { s.columns[1].nullable = "YES" }, schema.ErrPreflight, 0},
		{"type", func(s *widthPreflightState) { s.columns[2].dataType = "varchar" }, schema.ErrPreflight, 0},
		{"extra", func(s *widthPreflightState) { s.columns[0].extra = "INVISIBLE" }, schema.ErrPreflight, 0},
		{"generated", func(s *widthPreflightState) { s.columns[1].generation = "private-width-canary" }, schema.ErrPreflight, 0},
		{"read failure", func(s *widthPreflightState) { s.fail = true }, schema.ErrPreflight, 0},
		{"cancel at final read", nil, context.Canceled, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state := &widthPreflightState{columns: configWidthsV4SourceColumns(), tableCollation: "utf8mb4_0900_ai_ci", tableCharset: "utf8mb4"}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tc.mutate != nil {
				tc.mutate(state)
			}
			if tc.want == context.Canceled {
				state.cancel = cancel
			}
			db := sql.OpenDB(widthPreflightConnector{state})
			t.Cleanup(func() { _ = db.Close() })
			conn, err := db.Conn(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close()
			m, err := configWidthsV4Migration(configWidthsV4SQL, configWidthsV4Implementation)
			if err != nil {
				t.Fatal(err)
			}
			err = m.Apply(ctx, conn)
			if !errors.Is(err, tc.want) || state.writes != tc.writes {
				t.Fatalf("preflight outcome mismatch: err=%v writes=%d", err, state.writes)
			}
			if tc.writes > 0 && state.reads != 4 {
				t.Fatal("ALTER before full source inspection")
			}
			if err != nil && strings.Contains(err.Error(), "private-width-canary") {
				t.Fatal("error exposes private source metadata")
			}
		})
	}
}
