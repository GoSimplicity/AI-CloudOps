package dbschema

// These version-2 inputs become immutable on release. The new lock-address
// table deliberately adds no uniqueness constraint or repair to legacy metadata.
import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/pkg/schema"
)

//go:embed config_namespace_v2.json
var configNamespaceV2SQL []byte

//go:embed config_namespace_v2.go
var configNamespaceV2Implementation []byte

func configNamespaceV2Migration(snapshot, implementation []byte) (schema.Migration, error) {
	var statements []string
	if json.Unmarshal(snapshot, &statements) != nil || len(statements) != 1 || len(implementation) == 0 || !strings.HasPrefix(statements[0], "CREATE TABLE `cl_monitor_config_namespaces` (") {
		return schema.Migration{}, schema.ErrManifest
	}
	migration := schema.SQLMigration(2, "monitor_config_namespace_guards", statements...)
	migration.Checksum = schema.Checksum([]byte(fmt.Sprintf("%d:%s%d:%s", len(implementation), implementation, len(migration.Checksum), migration.Checksum)))
	return migration, nil
}
