package config_test

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// 配置库只在配置边界使用，业务模块必须显式接收配置或动态策略。
func TestProductionViperImportsStayInsideConfigBoundary(t *testing.T) {
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate configuration boundary test")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(source), "../.."))
	for _, directory := range []string{"cmd", "internal", "pkg"} {
		err := filepath.WalkDir(filepath.Join(root, directory), func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			relative, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			if strings.HasPrefix(filepath.ToSlash(relative), "internal/config/") {
				return nil
			}
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
			if err != nil {
				return err
			}
			for _, imported := range file.Imports {
				importPath, err := strconv.Unquote(imported.Path.Value)
				if err != nil {
					return err
				}
				if importPath == "github.com/spf13/viper" {
					t.Errorf("%s imports Viper outside internal/config; inject explicit configuration or policy", relative)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}
