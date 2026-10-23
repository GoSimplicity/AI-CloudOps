package di

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"testing"
)

// declaredPersistentModels 按真实TableName方法发现持久化模型，避免手写第二份清单
func declaredPersistentModels(t *testing.T) map[string]string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join("..", "..", "internal", "model", "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	models := map[string]string{}
	for _, path := range files {
		source, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range source.Decls {
			method, ok := decl.(*ast.FuncDecl)
			if !ok || method.Recv == nil || method.Name.Name != "TableName" {
				continue
			}
			receiver := method.Recv.List[0].Type
			if pointer, ok := receiver.(*ast.StarExpr); ok {
				receiver = pointer.X
			}
			name, ok := receiver.(*ast.Ident)
			if !ok {
				t.Fatal("无法识别持久化模型接收者")
			}
			if method.Body == nil || len(method.Body.List) != 1 {
				t.Fatal("TableName必须提供固定表名")
			}
			returned, ok := method.Body.List[0].(*ast.ReturnStmt)
			if !ok || len(returned.Results) != 1 {
				t.Fatal("无法读取持久化表名")
			}
			literal, ok := returned.Results[0].(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				t.Fatal("TableName必须返回固定字符串")
			}
			table, err := strconv.Unquote(literal.Value)
			if err != nil {
				t.Fatal(err)
			}
			models[name.Name] = table
		}
	}
	if len(models) < 30 {
		t.Fatal("未实际覆盖持久化模型目录")
	}
	return models
}
func TestStartupRegistersEveryPersistentModel(t *testing.T) {
	source, err := parser.ParseFile(token.NewFileSet(), "init.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	registered := map[string]bool{}
	for _, decl := range source.Decls {
		function, ok := decl.(*ast.FuncDecl)
		if !ok || function.Name.Name != "InitTables" {
			continue
		}
		ast.Inspect(function.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || selector.Sel.Name != "AutoMigrate" {
				return true
			}
			for _, argument := range call.Args {
				pointer, ok := argument.(*ast.UnaryExpr)
				if !ok || pointer.Op != token.AND {
					t.Fatal("启动迁移参数必须显式注册模型")
				}
				composite, ok := pointer.X.(*ast.CompositeLit)
				if !ok {
					t.Fatal("无法读取迁移模型")
				}
				modelType, ok := composite.Type.(*ast.SelectorExpr)
				if !ok {
					t.Fatal("迁移模型必须来自model包")
				}
				pkg, ok := modelType.X.(*ast.Ident)
				if !ok || pkg.Name != "model" {
					t.Fatal("迁移模型必须来自model包")
				}
				registered[modelType.Sel.Name] = true
			}
			return true
		})
	}
	for name, table := range declaredPersistentModels(t) {
		if !registered[name] {
			t.Errorf("启动迁移遗漏模型 %s (%s)", name, table)
		}
	}
}
