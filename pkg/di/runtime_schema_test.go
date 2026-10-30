package di

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestRuntimeStartupOnlyVerifiesSchemaAndClock(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "gorm.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]bool{}
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Name.Name != "InitDB" {
			continue
		}
		ast.Inspect(function.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			switch callee := call.Fun.(type) {
			case *ast.Ident:
				if callee.Name == "InitTables" || callee.Name == "CheckNotificationTimeZone" {
					t.Fatal("runtime invoked a legacy initializer")
				}
			case *ast.SelectorExpr:
				if callee.Sel.Name == "AutoMigrate" || callee.Sel.Name == "Migrate" || callee.Sel.Name == "InitializeFreshClock" {
					t.Fatal("runtime attempted schema or clock mutation")
				}
				if pkg, ok := callee.X.(*ast.Ident); ok {
					found[pkg.Name+"."+callee.Sel.Name] = true
				}
			}
			return true
		})
	}
	if !found["schema.Verify"] || !found["dbschema.VerifyClock"] {
		t.Fatal("runtime must verify both ledger and clock")
	}
}
