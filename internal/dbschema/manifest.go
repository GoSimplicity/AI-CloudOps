// Package dbschema contains immutable, forward-only schema releases. Production
// migration code does not depend on live GORM models or the legacy initializer.
package dbschema

import "github.com/GoSimplicity/AI-CloudOps/pkg/schema"

// Manifest returns a fresh manifest so callers cannot mutate shared migration
// state. Append new immutable versions here; never edit released baseline files.
func Manifest() ([]schema.Migration, error) {
	migration, err := baselineV1Migration(baselineV1SQL, baselineV1Implementation)
	if err != nil {
		return nil, err
	}
	return []schema.Migration{migration}, nil
}
