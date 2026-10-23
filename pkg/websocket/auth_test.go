package websocket

import "testing"

func TestTokenFromProtocols(t *testing.T) {
	for _, tt := range []struct{ name, header, expected string }{
		{"missing", "", ""},
		{"version only", Protocol, ""},
		{"credential", Protocol + ", " + CredentialProtocolPrefix + "test-token", "test-token"},
		{"empty", Protocol + ", " + CredentialProtocolPrefix, ""},
		{"duplicate", CredentialProtocolPrefix + "first, " + CredentialProtocolPrefix + "second", ""},
		{"wrong prefix", "unknown.test-token", ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if token := TokenFromProtocols(tt.header); token != tt.expected {
				t.Fatal("unexpected credential extraction")
			}
		})
	}
}
