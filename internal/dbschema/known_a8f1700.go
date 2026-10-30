package dbschema

// This wrapper, validator, source snapshot and ordered SQL are immutable inputs
// for the explicitly supported a8f1700 -> version-1 upgrade. Add a new upgrade
// implementation instead of changing released input bytes or baseline_v1.
import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/pkg/schema"
)

const KnownA8Source = "a8f1700675bef0fe950bce5018893b9a2366f3b4"

var (
	ErrKnownA8Snapshot = errors.New("known-source upgrade snapshot is invalid")
	ErrKnownA8Schema   = errors.New("database does not match the explicitly supported source schema")
	ErrKnownA8Read     = errors.New("known-source upgrade inspection failed; details withheld")
	ErrKnownA8Provider = errors.New("enabled historical providers 4..6 require review and explicit disabling before upgrade")
	ErrKnownA8Time     = errors.New("historical DATETIME values require explicit time-zone attestation")
	ErrKnownA8Target   = errors.New("upgraded database does not match the complete target contract")
)

//go:embed known_a8f1700.json
var knownA8SourceJSON []byte

//go:embed known_a8f1700_sql.json
var knownA8SQLJSON []byte

//go:embed known_a8f1700.go
var knownA8Wrapper []byte

//go:embed known_a8f1700_validation.go
var knownA8Validation []byte

// The shared strict parser participates in this implementation's checksum too.
//
//go:embed legacy.go
var knownA8Parser []byte

type knownA8Snapshot struct {
	SourceRevision    string            `json:"source_revision"`
	MySQLVersion      string            `json:"mysql_version"`
	GORMVersion       string            `json:"gorm_version"`
	SourceFilesSHA256 map[string]string `json:"source_files_sha256"`
	Tables            map[string]string `json:"tables"`
}

// KnownA8Upgrade returns a fresh plan without connecting to a database. The
// engine's provenance checksum is distinct from the immutable target checksum.
func KnownA8Upgrade() (schema.BaselineUpgrade, error) {
	manifest, err := Manifest()
	if err != nil || len(manifest) != 1 || manifest[0].Version != 1 {
		return schema.BaselineUpgrade{}, ErrKnownA8Snapshot
	}
	source, err := knownA8Definitions()
	if err != nil {
		return schema.BaselineUpgrade{}, err
	}
	var statements []string
	if json.Unmarshal(knownA8SQLJSON, &statements) != nil || len(statements) != 15 {
		return schema.BaselineUpgrade{}, ErrKnownA8Snapshot
	}
	// Bind exact source/SQL bytes and every upgrade-specific executable wrapper.
	// Fresh baseline execution is never substituted with this implementation.
	material := ""
	for _, input := range [][]byte{knownA8SourceJSON, knownA8SQLJSON, knownA8Wrapper, knownA8Validation, knownA8Parser, []byte(manifest[0].Checksum)} {
		material += fmt.Sprintf("%d:%s", len(input), input)
	}
	checksum := schema.Checksum([]byte(material))
	plan := schema.BaselineUpgrade{
		Source: KnownA8Source, SourceChecksum: schema.Checksum(knownA8SourceJSON), ImplementationChecksum: checksum,
	}
	plan.ValidateSource = func(ctx context.Context, conn *sql.Conn, evidence schema.UpgradeEvidence) error {
		return validateKnownA8Source(ctx, conn, source, evidence)
	}
	plan.Apply = func(ctx context.Context, conn *sql.Conn, evidence schema.UpgradeEvidence) error {
		for _, statement := range statements {
			if _, err := conn.ExecContext(ctx, statement); err != nil {
				return ErrKnownA8Read
			}
		}
		// The caller explicitly attests the interpretation of unchanged historical
		// DATETIME values. Do not replay queues or enable legacy advanced triggers.
		if _, err := conn.ExecContext(ctx, "INSERT INTO cl_workorder_notification_clock (id,time_zone) VALUES (1,?)", evidence.NotificationZone); err != nil {
			return ErrKnownA8Read
		}
		return nil
	}
	plan.ValidateTarget = func(ctx context.Context, conn *sql.Conn, evidence schema.UpgradeEvidence) error {
		return validateKnownA8Target(ctx, conn, manifest[0], plan, evidence)
	}
	return plan, nil
}

func knownA8Definitions() (map[string]legacyDefinition, error) {
	var snapshot knownA8Snapshot
	if json.Unmarshal(knownA8SourceJSON, &snapshot) != nil || snapshot.SourceRevision != KnownA8Source || snapshot.MySQLVersion != "8.4.11" || snapshot.GORMVersion != "1.25.11" || len(snapshot.Tables) != 49 || len(snapshot.SourceFilesSHA256) == 0 {
		return nil, ErrKnownA8Snapshot
	}
	for _, digest := range snapshot.SourceFilesSHA256 {
		if !knownA8Digest.MatchString(digest) {
			return nil, ErrKnownA8Snapshot
		}
	}
	definitions := make(map[string]legacyDefinition, len(snapshot.Tables))
	for name, ddl := range snapshot.Tables {
		parsed, definition, err := parseLegacyCreate(ddl)
		if err != nil || parsed != name {
			return nil, ErrKnownA8Snapshot
		}
		definitions[name] = definition
	}
	return definitions, nil
}
