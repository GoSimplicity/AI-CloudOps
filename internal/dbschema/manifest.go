// Package dbschema contains immutable, forward-only schema releases. Production
// migration code does not depend on live GORM models or the legacy initializer.
package dbschema

import "github.com/GoSimplicity/AI-CloudOps/pkg/schema"

// Manifest preserves the released sole-v1 target used by immutable legacy
// validators and upgrade closures. New runtime and migration consumers must use
// CurrentManifest, which appends later immutable versions.
func Manifest() ([]schema.Migration, error) {
	migration, err := baselineV1Migration(baselineV1SQL, baselineV1Implementation)
	if err != nil {
		return nil, err
	}
	return []schema.Migration{migration}, nil
}

// CurrentManifest is the full runtime/migration target. Manifest intentionally
// remains the released v1-only compatibility entry for frozen legacy validators
// and upgrade closures, whose exact source and sole-v1 target are audit inputs.
func CurrentManifest() ([]schema.Migration, error) {
	baseline, err := Manifest()
	if err != nil {
		return nil, err
	}
	guard, err := configNamespaceV2Migration(configNamespaceV2SQL, configNamespaceV2Implementation)
	if err != nil {
		return nil, err
	}
	return append(baseline, guard), nil
}
