package schema

import (
	"context"
	"database/sql"
	"errors"
	"testing"
)

func TestAppliedPrefixOnlyAllowsCleanNonemptyReleasedHistory(t *testing.T) {
	manifest := append(validManifest(), SQLMigration(2, "second", "CREATE TABLE guard_example (id BIGINT)"))
	first := Record{Version: 1, Name: manifest[0].Name, Checksum: manifest[0].Checksum, State: "applied"}
	second := Record{Version: 2, Name: manifest[1].Name, Checksum: manifest[1].Checksum, State: "applied"}
	dirty := second
	dirty.State = "failed"
	drift := first
	drift.Checksum = Checksum([]byte("foreign"))
	for _, tc := range []struct {
		name    string
		records []Record
		failure string
		want    error
	}{
		{"empty", nil, "", ErrPending}, {"v1", []Record{first}, "", nil}, {"current", []Record{first, second}, "", nil},
		{"dirty tail", []Record{first, dirty}, "", ErrDirty}, {"edited baseline", []Record{drift}, "", ErrDrift}, {"unreadable", nil, "history", ErrPreflight},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state := &adoptionRecorder{records: tc.records, failure: tc.failure}
			db := sql.OpenDB(adoptionConnector{state})
			defer db.Close()
			conn, err := db.Conn(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close()
			err = VerifyAppliedPrefixConnection(context.Background(), conn, manifest)
			if !errors.Is(err, tc.want) {
				t.Fatalf("unexpected classification: %v", err)
			}
			if len(state.writes) != 0 {
				t.Fatal("prefix verifier wrote history")
			}
		})
	}
}
