package dbschema

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"regexp"
	"sort"
	"strings"
)

var (
	ErrLegacyStructure  = errors.New("legacy schema differs from the immutable current release")
	ErrLegacyData       = errors.New("legacy data does not satisfy current repair postconditions")
	ErrLegacyVisibility = errors.New("legacy preflight requires direct schema-wide SELECT or REFERENCES metadata visibility")
	ErrLegacyRead       = errors.New("legacy preflight could not read complete metadata or data; database details withheld")
	ErrLegacySnapshot   = errors.New("legacy preflight does not recognize the immutable release format")
)

var (
	legacyCreateHeader = regexp.MustCompile("^CREATE TABLE `([A-Za-z0-9_]+)` \\($")
	legacyColumn       = regexp.MustCompile("^`[A-Za-z0-9_]+` .+$")
	legacyIndex        = regexp.MustCompile("^(PRIMARY KEY|(?:UNIQUE )?KEY `[A-Za-z0-9_]+`) \\(.+$")
	legacyForeignKey   = regexp.MustCompile("^CONSTRAINT `[A-Za-z0-9_]+` FOREIGN KEY \\(.+\\) REFERENCES `([A-Za-z0-9_]+)` \\(.+\\)(?: ON DELETE (?:RESTRICT|CASCADE|SET NULL|NO ACTION))?(?: ON UPDATE (?:RESTRICT|CASCADE|SET NULL|NO ACTION))?$")
	legacyAlter        = regexp.MustCompile("^ALTER TABLE `([A-Za-z0-9_]+)` ADD (CONSTRAINT .+)$")
	legacyOptions      = regexp.MustCompile("^ENGINE=([A-Za-z0-9_]+)(?: AUTO_INCREMENT=[0-9]+)? DEFAULT CHARSET=([A-Za-z0-9_]+) COLLATE=([A-Za-z0-9_]+)$")
)

type legacyDefinition struct {
	columns []string
	keys    []string
	options string
}

func (definition legacyDefinition) canonical() string {
	keys := append([]string(nil), definition.keys...)
	sort.Strings(keys)
	return strings.Join(definition.columns, "\n") + "\n" + strings.Join(keys, "\n") + "\n" + definition.options
}

// parseLegacyCreate accepts only the fixed multiline SHOW CREATE format captured
// in version 1. It never executes DDL. Column order and every definition byte are
// significant; only independent key declarations and AUTO_INCREMENT counters
// are normalized. Unknown clauses or options are rejected conservatively.
func parseLegacyCreate(ddl string) (string, legacyDefinition, error) {
	lines := strings.Split(ddl, "\n")
	definition := legacyDefinition{}
	if len(lines) < 3 {
		return "", definition, ErrLegacySnapshot
	}
	header := legacyCreateHeader.FindStringSubmatch(lines[0])
	if header == nil {
		return "", definition, ErrLegacySnapshot
	}
	if !strings.HasPrefix(lines[len(lines)-1], ") ") {
		return "", definition, ErrLegacySnapshot
	}
	options := legacyOptions.FindStringSubmatch(strings.TrimPrefix(lines[len(lines)-1], ") "))
	if options == nil {
		return "", definition, ErrLegacySnapshot
	}
	definition.options = "ENGINE=" + options[1] + " DEFAULT CHARSET=" + options[2] + " COLLATE=" + options[3]
	seenKey := false
	for i, line := range lines[1 : len(lines)-1] {
		if !strings.HasPrefix(line, "  ") || strings.HasPrefix(line, "   ") {
			return "", definition, ErrLegacySnapshot
		}
		content := strings.TrimPrefix(line, "  ")
		shouldHaveComma := i < len(lines)-3
		if strings.HasSuffix(content, ",") != shouldHaveComma {
			return "", definition, ErrLegacySnapshot
		}
		content = strings.TrimSuffix(content, ",")
		switch {
		case legacyColumn.MatchString(content):
			if seenKey {
				return "", definition, ErrLegacySnapshot
			}
			definition.columns = append(definition.columns, content)
		case legacyIndex.MatchString(content), legacyForeignKey.MatchString(content):
			seenKey = true
			definition.keys = append(definition.keys, content)
		default:
			return "", definition, ErrLegacySnapshot
		}
	}
	if len(definition.columns) == 0 {
		return "", definition, ErrLegacySnapshot
	}
	return header[1], definition, nil
}

func expectedLegacyDefinitions(snapshot []byte) (map[string]legacyDefinition, error) {
	var statements []string
	if json.Unmarshal(snapshot, &statements) != nil {
		return nil, ErrLegacySnapshot
	}
	tables := make(map[string]legacyDefinition)
	phase := 0
	foreignKeys, repairs := 0, 0
	for _, statement := range statements {
		switch {
		case strings.HasPrefix(statement, "CREATE TABLE "):
			if phase != 0 {
				return nil, ErrLegacySnapshot
			}
			name, definition, err := parseLegacyCreate(statement)
			if err != nil {
				return nil, err
			}
			if _, exists := tables[name]; exists {
				return nil, ErrLegacySnapshot
			}
			for _, key := range definition.keys {
				if strings.HasPrefix(key, "CONSTRAINT ") {
					return nil, ErrLegacySnapshot
				}
			}
			tables[name] = definition
		case strings.HasPrefix(statement, "ALTER TABLE "):
			if phase > 1 {
				return nil, ErrLegacySnapshot
			}
			phase = 1
			alter := legacyAlter.FindStringSubmatch(statement)
			if alter == nil {
				return nil, ErrLegacySnapshot
			}
			foreign := legacyForeignKey.FindStringSubmatch(alter[2])
			if foreign == nil {
				return nil, ErrLegacySnapshot
			}
			definition, exists := tables[alter[1]]
			if !exists {
				return nil, ErrLegacySnapshot
			}
			if _, exists := tables[foreign[1]]; !exists {
				return nil, ErrLegacySnapshot
			}
			definition.keys = append(definition.keys, alter[2])
			tables[alter[1]] = definition
			foreignKeys++
		case statement == legacyAuditRepair || statement == legacyDefaultRepair:
			phase = 2
			repairs++
			if (repairs == 1 && statement != legacyAuditRepair) || (repairs == 2 && statement != legacyDefaultRepair) {
				return nil, ErrLegacySnapshot
			}
		default:
			return nil, ErrLegacySnapshot
		}
	}
	if len(tables) != 58 || foreignKeys != 34 || repairs != 2 {
		return nil, ErrLegacySnapshot
	}
	return tables, nil
}

// ValidateLegacyCurrent performs SELECT/SHOW only. It neither creates the
// ledger nor claims a schema version. The caller must stop all writers and hold
// schema.WithMigrationLock on this connection throughout validation and any
// later adoption. Direct schema-wide metadata visibility is mandatory: a
// table-scoped account cannot prove that hidden extra tables do not exist.
// This validator recognizes version 1 only; future manifests require a new,
// reviewed legacy validation contract rather than silently adopting new code.
func ValidateLegacyCurrent(ctx context.Context, conn *sql.Conn) error {
	if conn == nil {
		return ErrLegacyRead
	}
	manifest, err := Manifest()
	if err != nil || len(manifest) != 1 || manifest[0].Version != 1 {
		return ErrLegacySnapshot
	}
	expected, err := expectedLegacyDefinitions(baselineV1SQL)
	if err != nil {
		return err
	}
	var database sql.NullString
	var account string
	if err := conn.QueryRowContext(ctx, "SELECT DATABASE(), CURRENT_USER()").Scan(&database, &account); err != nil || !database.Valid || database.String == "" {
		return ErrLegacyRead
	}
	separator := strings.LastIndex(account, "@")
	if separator < 1 {
		return ErrLegacyVisibility
	}
	grantee := "'" + account[:separator] + "'@'" + account[separator+1:] + "'"
	var visible bool
	if err := conn.QueryRowContext(ctx, `SELECT EXISTS(
 SELECT 1 FROM information_schema.SCHEMA_PRIVILEGES WHERE TABLE_SCHEMA=DATABASE() AND GRANTEE=? AND PRIVILEGE_TYPE IN ('SELECT','REFERENCES')
 UNION ALL SELECT 1 FROM information_schema.USER_PRIVILEGES WHERE GRANTEE=? AND PRIVILEGE_TYPE IN ('SELECT','REFERENCES')
 )`, grantee, grantee).Scan(&visible); err != nil {
		return ErrLegacyRead
	}
	if !visible {
		return ErrLegacyVisibility
	}
	rows, err := conn.QueryContext(ctx, "SELECT TABLE_NAME, TABLE_TYPE FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() ORDER BY TABLE_NAME LIMIT 59")
	if err != nil {
		return ErrLegacyRead
	}
	names := make([]string, 0, len(expected))
	validInventory := true
	for rows.Next() {
		var name, kind string
		if err := rows.Scan(&name, &kind); err != nil {
			rows.Close()
			return ErrLegacyRead
		}
		if _, exists := expected[name]; !exists || kind != "BASE TABLE" {
			validInventory = false
		}
		names = append(names, name)
	}
	readErr := rows.Err()
	rows.Close()
	if readErr != nil {
		return ErrLegacyRead
	}
	if !validInventory || len(names) != len(expected) {
		return ErrLegacyStructure
	}
	for _, name := range names {
		// Names have already matched the fixed snapshot's safe identifier grammar.
		var returned, ddl string
		if err := conn.QueryRowContext(ctx, "SHOW CREATE TABLE `"+name+"`").Scan(&returned, &ddl); err != nil {
			return ErrLegacyRead
		}
		parsed, actual, err := parseLegacyCreate(ddl)
		if err != nil || returned != name || parsed != name || actual.canonical() != expected[name].canonical() {
			return ErrLegacyStructure
		}
	}
	var invalidDefaults, unrepairedAudit uint64
	if err := conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM cl_workorder_notification WHERE is_default = 0").Scan(&invalidDefaults); err != nil {
		return ErrLegacyRead
	}
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM cl_tree_cloud_resource_change_log
 WHERE LOWER(TRIM(field_name)) IN ('key','password')
 AND (old_value <> '[REDACTED]' OR new_value <> '[REDACTED]')`).Scan(&unrepairedAudit); err != nil {
		return ErrLegacyRead
	}
	if invalidDefaults != 0 || unrepairedAudit != 0 {
		return ErrLegacyData
	}
	return nil
}

// These exact immutable statements also validate the parser's repair section.
// The read-only postconditions use their WHERE semantics, including SQL NULL's
// three-valued comparisons and the lack of any soft-delete exclusion.
const legacyAuditRepair = "UPDATE `cl_tree_cloud_resource_change_log` SET `new_value`='[REDACTED]',`old_value`='[REDACTED]' WHERE LOWER(TRIM(field_name)) IN ('key','password') AND (old_value <> '[REDACTED]' OR new_value <> '[REDACTED]')"
const legacyDefaultRepair = "UPDATE `cl_workorder_notification` SET `is_default`=2 WHERE is_default = 0"
