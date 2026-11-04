package dbschema

// Version 3 owns only the projection bookkeeping schema. Once released these
// exact SQL and wrapper inputs are immutable. Do not bind the mutable manifest
// aggregator or live GORM model definitions into a released checksum.
import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/pkg/schema"
)

//go:embed config_projection_v3.json
var configProjectionV3SQL []byte

//go:embed config_projection_v3.go
var configProjectionV3Implementation []byte

func configProjectionV3Migration(snapshot, implementation []byte) (schema.Migration, error) {
	var statements []string
	tables := []string{"cl_monitor_config_pool_controls", "cl_monitor_config_pool_operations", "cl_monitor_config_artifact_events"}
	if json.Unmarshal(snapshot, &statements) != nil || len(statements) != len(tables) || len(implementation) == 0 {
		return schema.Migration{}, schema.ErrManifest
	}
	for i, table := range tables {
		if !strings.HasPrefix(statements[i], "CREATE TABLE `"+table+"` (") || !strings.HasSuffix(statements[i], ") ENGINE=InnoDB") {
			return schema.Migration{}, schema.ErrManifest
		}
	}
	migration := schema.SQLMigration(3, "monitor_config_projection_journal", statements...)
	migration.Checksum = schema.Checksum([]byte(fmt.Sprintf("%d:%s%d:%s", len(implementation), implementation, len(migration.Checksum), migration.Checksum)))
	return migration, nil
}
