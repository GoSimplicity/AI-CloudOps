package ssh

import (
	"crypto/ed25519"
	"crypto/rand"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
)

func testAuthorizedKey(t *testing.T) string {
	t.Helper()
	key, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	public, err := ssh.NewPublicKey(key)
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(ssh.MarshalAuthorizedKey(public)))
}

func TestAuthorizedKeyCommand(t *testing.T) {
	key := testAuthorizedKey(t)
	for _, tt := range []struct {
		name, input string
		invalid     bool
	}{
		{"key", key, false},
		{"comment", key + " owner's workstation", false},
		{"restricted key", `command="echo limited",no-port-forwarding ` + key, false},
		{"empty", "", true},
		{"invalid", "not-a-public-key", true},
		{"multiline", key + "\n" + key, true},
		{"command injection", key + "\n'; touch /tmp/invalid; #", true},
		{"oversized", strings.Repeat("a", 16385), true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			command, err := AuthorizedKeyCommand(tt.input)
			if (err != nil) != tt.invalid {
				t.Fatalf("unexpected validation result: %v", err)
			}
			if tt.invalid && command != "" {
				t.Fatal("invalid key produced executable command")
			}
		})
	}
}
