package utils

import (
	"strings"
	"testing"
)

func TestValidatePodFilePath(t *testing.T) {
	allowed := []string{
		"/",
		"/tmp/app",
		"/var/log/app.log",
		"tmp/app",
		"foo..bar",
		".../notes",
		"/tmp/~cache",
	}
	for _, path := range allowed {
		if err := ValidatePodFilePath(path); err != nil {
			t.Fatalf("path %q rejected: %v", path, err)
		}
	}

	rejected := []string{
		"",
		"..",
		"../etc/passwd",
		"/tmp/../etc/passwd",
		"foo/../../etc/passwd",
		"/var/log/../../etc/shadow",
		"foo/bar/..",
		`foo\..\etc\passwd`,
		"a\x00b",
		"foo\nbar",
		"foo\rbar",
		"~/secret",
		strings.Repeat("a", 4097),
	}
	for _, path := range rejected {
		if err := ValidatePodFilePath(path); err == nil {
			t.Fatalf("path %q accepted", path)
		}
	}
}

func TestIsAllowedPodExecShell(t *testing.T) {
	if !IsAllowedPodExecShell("sh") || !IsAllowedPodExecShell("bash") {
		t.Fatal("sh and bash must be allowed")
	}
	for _, shell := range []string{"", "Bash", "zsh", "fish", "ash", "/bin/sh", "bash -c echo", " sh", "sh\x00"} {
		if IsAllowedPodExecShell(shell) {
			t.Fatalf("shell %q allowed", shell)
		}
	}
}
