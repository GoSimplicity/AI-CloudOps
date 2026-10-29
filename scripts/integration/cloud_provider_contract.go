// The provider numbers are persisted and cross the Go/TypeScript API boundary.
// Run against the checked-out web revision used by the integration job.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

var enumBlock = regexp.MustCompile(`(?s)export\s+enum\s+CloudProvider\s*\{([^}]*)\}`)
var enumEntry = regexp.MustCompile(`^\s*([A-Za-z][A-Za-z0-9_]*)\s*=\s*([0-9]+)\s*,?\s*(?://.*)?$`)

func checkCloudProviders(source string) error {
	match := enumBlock.FindStringSubmatch(source)
	if len(match) != 2 {
		return fmt.Errorf("Web CloudProvider enum not found")
	}
	want := map[string]int{
		"AliCloud":     int(model.ProviderAliyun),
		"TencentCloud": int(model.ProviderTencent),
		"AWS":          int(model.ProviderAWS),
		"HuaweiCloud":  int(model.ProviderHuawei),
		"Azure":        int(model.ProviderAzure),
		"GCP":          int(model.ProviderGCP),
	}
	seen := make(map[string]bool, len(want))
	for _, line := range strings.Split(match[1], "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		entry := enumEntry.FindStringSubmatch(line)
		if len(entry) != 3 {
			return fmt.Errorf("unsupported CloudProvider declaration: %s", line)
		}
		expected, ok := want[entry[1]]
		if !ok || seen[entry[1]] {
			return fmt.Errorf("unexpected or repeated cloud provider: %s", entry[1])
		}
		actual, err := strconv.Atoi(entry[2])
		if err != nil || actual != expected {
			return fmt.Errorf("cloud provider %s differs between Go and Web", entry[1])
		}
		seen[entry[1]] = true
	}
	if len(seen) != len(want) {
		return fmt.Errorf("Web CloudProvider enum is incomplete")
	}
	return nil
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: go run cloud_provider_contract.go <web-repository>")
		os.Exit(2)
	}
	path := filepath.Join(os.Args[1], "apps/web-antd/src/api/core/tree/tree_account.ts")
	contents, err := os.ReadFile(path)
	if err == nil {
		err = checkCloudProviders(string(contents))
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("Go/Web cloud provider values match")
}
