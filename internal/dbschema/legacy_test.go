package dbschema

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

const legacySmallDDL = "CREATE TABLE `fixture` (\n  `id` bigint NOT NULL AUTO_INCREMENT,\n  `name` varchar(20) DEFAULT NULL,\n  PRIMARY KEY (`id`),\n  KEY `name_index` (`name`)\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci"

func TestLegacyCreateParser(t *testing.T) {
	tests := []struct {
		name, ddl string
		valid     bool
	}{
		{"captured format", legacySmallDDL, true},
		{"growing auto increment", strings.Replace(legacySmallDDL, "ENGINE=InnoDB ", "ENGINE=InnoDB AUTO_INCREMENT=4000 ", 1), true},
		{"unknown table option", legacySmallDDL + " ROW_FORMAT=DYNAMIC", false},
		{"conditional DDL", strings.Replace(legacySmallDDL, "CREATE TABLE ", "CREATE TABLE IF NOT EXISTS ", 1), false},
		{"missing separator", strings.Replace(legacySmallDDL, "AUTO_INCREMENT,", "AUTO_INCREMENT", 1), false},
		{"unknown clause", strings.Replace(legacySmallDDL, "KEY `name_index` (`name`)", "CHECK (`id` > 0)", 1), false},
		{"empty", "", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, _, err := parseLegacyCreate(test.ddl)
			if (err == nil) != test.valid {
				t.Fatalf("parser error %v", err)
			}
		})
	}
}

func TestLegacyCanonicalComparison(t *testing.T) {
	_, baseline, err := parseLegacyCreate(legacySmallDDL)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name, ddl string
		equal     bool
	}{
		{"same", legacySmallDDL, true},
		{"independent index order", strings.Replace(legacySmallDDL, "  PRIMARY KEY (`id`),\n  KEY `name_index` (`name`)", "  KEY `name_index` (`name`),\n  PRIMARY KEY (`id`)", 1), true},
		{"auto increment value", strings.Replace(legacySmallDDL, "ENGINE=InnoDB ", "ENGINE=InnoDB AUTO_INCREMENT=4000 ", 1), true},
		{"column type", strings.Replace(legacySmallDDL, "varchar(20)", "varchar(21)", 1), false},
		{"column order", strings.Replace(legacySmallDDL, "  `id` bigint NOT NULL AUTO_INCREMENT,\n  `name` varchar(20) DEFAULT NULL,", "  `name` varchar(20) DEFAULT NULL,\n  `id` bigint NOT NULL AUTO_INCREMENT,", 1), false},
		{"index definition", strings.Replace(legacySmallDDL, "KEY `name_index` (`name`)", "KEY `name_index` (`name` DESC)", 1), false},
		{"engine", strings.Replace(legacySmallDDL, "ENGINE=InnoDB", "ENGINE=MyISAM", 1), false},
		{"collation", strings.Replace(legacySmallDDL, "utf8mb4_0900_ai_ci", "utf8mb4_unicode_ci", 1), false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, actual, err := parseLegacyCreate(test.ddl)
			if err != nil {
				t.Fatal(err)
			}
			if (baseline.canonical() == actual.canonical()) != test.equal {
				t.Fatal("canonical comparison ignored semantic structure or counter normalization")
			}
		})
	}
}

func TestLegacySnapshotParser(t *testing.T) {
	var original []string
	if err := json.Unmarshal(baselineV1SQL, &original); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		change func([]string) []string
		valid  bool
	}{
		{"released snapshot", func(s []string) []string { return s }, true},
		{"missing table", func(s []string) []string { return s[1:] }, false},
		{"duplicate table", func(s []string) []string { s[1] = s[0]; return s }, false},
		{"missing foreign key", func(s []string) []string { return append(s[:58], s[59:]...) }, false},
		{"wrong foreign key target", func(s []string) []string {
			s[58] = strings.Replace(s[58], "REFERENCES `", "REFERENCES `unknown_", 1)
			return s
		}, false},
		{"mutated data repair", func(s []string) []string { s[len(s)-1] += " /* changed */"; return s }, false},
		{"arbitrary SQL", func(s []string) []string { s[0] = "DROP DATABASE anything"; return s }, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			raw, err := json.Marshal(test.change(append([]string(nil), original...)))
			if err != nil {
				t.Fatal(err)
			}
			tables, err := expectedLegacyDefinitions(raw)
			if (err == nil) != test.valid {
				t.Fatalf("snapshot error %v", err)
			}
			if err == nil && len(tables) != 58 {
				t.Fatal("incomplete baseline inventory")
			}
		})
	}
}

func TestLegacyNilConnection(t *testing.T) {
	if !errors.Is(ValidateLegacyCurrent(context.Background(), nil), ErrLegacyRead) {
		t.Fatal("nil connection not rejected")
	}
}
