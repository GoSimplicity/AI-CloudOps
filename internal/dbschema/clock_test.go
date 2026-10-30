package dbschema

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"
)

func TestValidateClockZone(t *testing.T) {
	tests := []struct {
		name, zone    string
		exists, valid bool
	}{
		{"explicit UTC", "UTC", true, true},
		{"IANA", "Asia/Shanghai", true, true},
		{"IANA alias", "Etc/UTC", true, true},
		{"empty", "", true, false},
		{"machine local", "Local", true, false},
		{"lowercase local", "local", true, false},
		{"surrounding spaces", " UTC ", true, false},
		{"missing zone", "Unknown/Zone", false, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			load := func(string) (*time.Location, error) {
				if !test.exists {
					return nil, errors.New("unavailable")
				}
				return time.FixedZone("fixture", 0), nil
			}
			err := validateClockZone(test.zone, load)
			if (err == nil) != test.valid {
				t.Fatalf("validation error = %v", err)
			}
		})
	}
}

func TestClockNilDatabase(t *testing.T) {
	tests := []struct {
		name      string
		operation func(context.Context, *sql.DB, string) error
	}{{"verify", VerifyClock}, {"initialize", InitializeFreshClock}}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if !errors.Is(test.operation(context.Background(), nil, "UTC"), ErrClockDatabase) {
				t.Fatal("nil database must fail closed")
			}
		})
	}
}
