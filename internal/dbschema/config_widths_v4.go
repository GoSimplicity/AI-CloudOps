package dbschema

// Version 4 widens only released config key/name columns. Its exact SQL and
// wrapper are immutable release inputs; live model constants are not inputs.
import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/pkg/schema"
)

//go:embed config_widths_v4.json
var configWidthsV4SQL []byte

//go:embed config_widths_v4.go
var configWidthsV4Implementation []byte

func configWidthsV4Migration(snapshot, implementation []byte) (schema.Migration, error) {
	var statements []string
	tables := []string{"cl_monitor_configs", "cl_monitor_config_namespaces", "cl_monitor_config_artifact_events"}
	if json.Unmarshal(snapshot, &statements) != nil || len(statements) != len(tables) || len(implementation) == 0 {
		return schema.Migration{}, schema.ErrManifest
	}
	for i, table := range tables {
		if !strings.HasPrefix(statements[i], "ALTER TABLE `"+table+"` MODIFY COLUMN ") {
			return schema.Migration{}, schema.ErrManifest
		}
	}
	migration := schema.SQLMigration(4, "monitor_config_lossless_widths", statements...)
	migration.Checksum = schema.Checksum([]byte(fmt.Sprintf("%d:%s%d:%s", len(implementation), implementation, len(migration.Checksum), migration.Checksum)))
	apply := migration.Apply
	migration.Apply = func(ctx context.Context, conn *sql.Conn) error {
		// The released VARCHAR columns inherit this table default. Verify both
		// table and column attributes before MODIFY so inheritance preserves the
		// exact charset/collation, including SHOW CREATE's inherited form.
		var tableCollation, tableCharset string
		err := conn.QueryRowContext(ctx, `SELECT t.TABLE_COLLATION,c.CHARACTER_SET_NAME FROM information_schema.tables AS t JOIN information_schema.collation_character_set_applicability AS c ON c.COLLATION_NAME=t.TABLE_COLLATION WHERE t.TABLE_SCHEMA=DATABASE() AND BINARY t.TABLE_NAME=BINARY 'cl_monitor_configs'`).Scan(&tableCollation, &tableCharset)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil || tableCollation != "utf8mb4_0900_ai_ci" || tableCharset != "utf8mb4" {
			return fmt.Errorf("%w: config width table default differs from released schema", schema.ErrPreflight)
		}
		// Inspect every affected column before any ALTER; unsupported drift must
		// never be converted as a side effect of widening.
		for _, column := range configWidthsV4SourceColumns() {
			var actual configWidthsV4Column
			actual.table, actual.name = column.table, column.name
			err := conn.QueryRowContext(ctx, `SELECT DATA_TYPE,CHARACTER_MAXIMUM_LENGTH,IS_NULLABLE,COLUMN_DEFAULT,COLUMN_COMMENT,CHARACTER_SET_NAME,COLLATION_NAME,EXTRA,GENERATION_EXPRESSION FROM information_schema.columns WHERE table_schema=DATABASE() AND BINARY table_name=BINARY ? AND BINARY column_name=BINARY ?`, column.table, column.name).Scan(&actual.dataType, &actual.length, &actual.nullable, &actual.defaultValue, &actual.comment, &actual.charset, &actual.collation, &actual.extra, &actual.generation)
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if err != nil || actual != column {
				return fmt.Errorf("%w: config width source column differs from released schema", schema.ErrPreflight)
			}
		}
		return apply(ctx, conn)
	}
	return migration, nil
}

type configWidthsV4Column struct {
	table, name, dataType            string
	length                           int64
	nullable, comment                string
	defaultValue, charset, collation sql.NullString
	extra, generation                string
}

func configWidthsV4SourceColumns() []configWidthsV4Column {
	utf8 := sql.NullString{String: "utf8mb4", Valid: true}
	collation := sql.NullString{String: "utf8mb4_0900_ai_ci", Valid: true}
	return []configWidthsV4Column{
		{table: "cl_monitor_configs", name: "name", dataType: "varchar", length: 100, nullable: "NO", comment: "配置名称", charset: utf8, collation: collation},
		{table: "cl_monitor_configs", name: "instance_ip", dataType: "varchar", length: 45, nullable: "NO", comment: "实例IP地址", charset: utf8, collation: collation},
		{table: "cl_monitor_config_namespaces", name: "instance_key", dataType: "varbinary", length: 180, nullable: "NO"},
		{table: "cl_monitor_config_artifact_events", name: "instance_key", dataType: "varbinary", length: 180, nullable: "NO"},
	}
}
