package di

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/redis/go-redis/v9"
)

func TestProjectionProviderOwnsOnlyItsDedicatedPool(t *testing.T) {
	original := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", MaxRetries: 7})
	store, cleanup, err := ProvideProjectionStore(original)
	if err != nil || store == nil || cleanup == nil {
		t.Fatal("provider did not supply owned cleanup", err)
	}
	cleanup()
	cleanup()
	if err := store.Close(); !errors.Is(err, redis.ErrClosed) {
		t.Fatal("provider cleanup left dedicated pool open", err)
	}
	if err := original.Close(); err != nil {
		t.Fatal("provider cleanup closed the application's Redis pool", err)
	}
}
func TestProjectionProviderFailureDoesNotReturnCleanupOrCloseCaller(t *testing.T) {
	original := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", DB: 1})
	store, cleanup, err := ProvideProjectionStore(original)
	if err == nil || store != nil || cleanup != nil {
		t.Fatal("invalid provider dependency was accepted")
	}
	if err := original.Close(); err != nil {
		t.Fatal("failed construction closed caller pool", err)
	}
}

// Check the actual Wire-generated ownership graph. This guards failure branches
// added after the Store was allocated, not a hand-written copy of its cleanup.
func TestProjectionWireReleasesStoreOnEveryLaterConstructionFailure(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "wire_gen.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var fn *ast.FuncDecl
	for _, decl := range file.Decls {
		if value, ok := decl.(*ast.FuncDecl); ok && value.Name.Name == "ProvideApp" {
			fn = value
		}
	}
	if fn == nil {
		t.Fatal("generated injector absent")
	}
	cleanupName := ""
	skipProviderFailure := false
	failurePaths := 0
	returnedCleanup := false
	callsCleanup := func(node ast.Node) bool {
		found := false
		ast.Inspect(node, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			id, ok := call.Fun.(*ast.Ident)
			if ok && id.Name == cleanupName {
				found = true
			}
			return true
		})
		return found
	}
	for _, statement := range fn.Body.List {
		if assignment, ok := statement.(*ast.AssignStmt); ok {
			for _, rhs := range assignment.Rhs {
				call, ok := rhs.(*ast.CallExpr)
				if !ok {
					continue
				}
				name, ok := call.Fun.(*ast.Ident)
				if ok && name.Name == "ProvideProjectionStore" {
					if len(assignment.Lhs) != 3 {
						t.Fatal("Store ownership result missing")
					}
					cleanupName = assignment.Lhs[1].(*ast.Ident).Name
					skipProviderFailure = true
				}
			}
		}
		if cleanupName == "" {
			continue
		}
		if branch, ok := statement.(*ast.IfStmt); ok {
			hasReturn := false
			for _, item := range branch.Body.List {
				if _, ok := item.(*ast.ReturnStmt); ok {
					hasReturn = true
				}
			}
			if !hasReturn {
				continue
			}
			if skipProviderFailure {
				skipProviderFailure = false
				continue
			}
			failurePaths++
			if !callsCleanup(branch.Body) {
				t.Fatal("generated later construction failure leaked Store")
			}
		}
		if returned, ok := statement.(*ast.ReturnStmt); ok && len(returned.Results) == 3 {
			if closure, ok := returned.Results[1].(*ast.FuncLit); ok {
				returnedCleanup = callsCleanup(closure.Body)
			}
		}
	}
	if failurePaths == 0 || !returnedCleanup {
		t.Fatal("generated injector lost Store failure/success cleanup ownership")
	}
}
