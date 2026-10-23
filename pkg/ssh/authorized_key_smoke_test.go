//go:build smoke

package ssh

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestAuthorizedKeyShellEscaping(t *testing.T) {
	home := t.TempDir()
	key := testAuthorizedKey(t) + ` owner's key'; touch "$HOME/pwned"; #`
	command, err := AuthorizedKeyCommand(key)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("sh", "-c", command)
	cmd.Env = append(os.Environ(), "HOME="+home)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("isolated shell failed: %v %s", err, output)
	}
	if _, err := os.Stat(filepath.Join(home, "pwned")); !os.IsNotExist(err) {
		t.Fatal("public key comment executed as shell code")
	}
	saved, err := os.ReadFile(filepath.Join(home, ".ssh", "authorized_keys"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(saved)) != key {
		t.Fatal("authorized key comment or options were changed")
	}
	info, err := os.Stat(filepath.Join(home, ".ssh", "authorized_keys"))
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("authorized_keys is not private")
	}
}
