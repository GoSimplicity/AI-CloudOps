package main

import (
	"bytes"
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/dbschema"
	"github.com/GoSimplicity/AI-CloudOps/pkg/schema"
)

type legacyTargetCapture struct {
	*fakeDatabase
	versions []uint64
}

func (d *legacyTargetCapture) AdoptBaseline(ctx context.Context, m []schema.Migration, note string) error {
	for _, x := range m {
		d.versions = append(d.versions, x.Version)
	}
	return d.fakeDatabase.AdoptBaseline(ctx, m, note)
}
func (d *legacyTargetCapture) UpgradeBaseline(ctx context.Context, m []schema.Migration, e schema.UpgradeEvidence) error {
	for _, x := range m {
		d.versions = append(d.versions, x.Version)
	}
	return d.fakeDatabase.UpgradeBaseline(ctx, m, e)
}
func TestModernCLIPlanAndMainUseCurrentManifest(t *testing.T) {
	var out, diagnostic bytes.Buffer
	if run(context.Background(), []string{"plan"}, &out, &diagnostic, dependencies{manifest: dbschema.CurrentManifest}) != 0 || !strings.Contains(out.String(), "version=1 ") || !strings.Contains(out.String(), "version=2 ") {
		t.Fatal("current offline plan omitted a released version")
	}
	file, err := parser.ParseFile(token.NewFileSet(), "main.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "main" {
			continue
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			kv, ok := n.(*ast.KeyValueExpr)
			if !ok {
				return true
			}
			key, ok := kv.Key.(*ast.Ident)
			if !ok || key.Name != "manifest" {
				return true
			}
			sel, ok := kv.Value.(*ast.SelectorExpr)
			if ok && sel.Sel.Name == "CurrentManifest" {
				found = true
			}
			return true
		})
	}
	if !found {
		t.Fatal("actual CLI entry no longer loads current manifest")
	}
}
func TestLegacyCLIStillPassesOnlyReleasedV1ToFrozenEngines(t *testing.T) {
	note, proof := upgradeFiles(t)
	cases := [][]string{{"adopt", "--confirm-legacy-schema", "--confirm-no-writers", "--adoption-note-file", note}, {knownUpgradeCommand, "--execute", "--confirm-no-writers", "--upgrade-note-file", note, "--backup-proof-file", proof, "--notification-timezone", "UTC", "--attest-historical-time"}}
	for _, args := range cases {
		t.Run(args[0], func(t *testing.T) {
			db := &legacyTargetCapture{fakeDatabase: &fakeDatabase{}}
			var out, diagnostic bytes.Buffer
			status := run(context.Background(), args, &out, &diagnostic, dependencies{manifest: dbschema.CurrentManifest, getenv: func(string) string { return "private-fixture" }, open: func(string, time.Duration) (schemaDatabase, error) { return db, nil }})
			if status != 0 || len(db.versions) != 1 || db.versions[0] != 1 || !strings.Contains(out.String(), "migrate --confirm-pending-migrations --confirm-no-writers") {
				t.Fatal("legacy command adopted/upgraded later versions or omitted next-step boundary")
			}
		})
	}
}
func TestNamespaceGuardRoleCannotDeleteOrAlterAddresses(t *testing.T) {
	plan, err := BuildRolePlan("cloudopsTest", RoleAccount{Name: "runtime", Host: "localhost"}, RoleAccount{Name: "migrator", Host: "localhost"})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, grant := range plan.Runtime {
		if strings.Contains(grant, "`.`"+configNamespaceTable+"`") {
			found = true
			if !strings.HasPrefix(grant, "GRANT SELECT, INSERT, UPDATE ON ") || strings.Contains(grant, "DELETE") || strings.Contains(grant, "ALTER") {
				t.Fatal("runtime guard grant exceeds locking address writes")
			}
		}
	}
	if !found {
		t.Fatal("runtime cannot acquire guard rows")
	}
}
