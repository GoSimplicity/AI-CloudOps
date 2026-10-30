package utils

import (
	"encoding/base64"
	"strings"
	"testing"

	base "github.com/GoSimplicity/AI-CloudOps/pkg/base"
	"github.com/spf13/viper"
)

func TestPasswordCipherOpensLegacySyntheticVector(t *testing.T) {
	// Captured from the original Viper-reading helper before this refactor. Both
	// key and plaintext are synthetic test material, never deployment credentials.
	encryptionKey := strings.Repeat("0f", 16)
	const plaintext = "legacy-typed-credential"
	const legacyVector = "QoQ+yTgfNxhhqfuERfpkAn6mSyycyz6Rz1x0rUj/eDe3YrK7X+7/qP9Slfo0Y/FEJgVQ"
	actual, err := DecryptPassword(legacyVector, encryptionKey)
	if err != nil || actual != plaintext {
		t.Fatal("legacy AES-GCM format was not preserved")
	}
	first, err := EncryptPassword(plaintext, encryptionKey)
	if err != nil {
		t.Fatal(err)
	}
	second, err := EncryptPassword(plaintext, encryptionKey)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("random nonce was not preserved")
	}
	decoded, err := base64.StdEncoding.DecodeString(first)
	if err != nil || len(decoded) != 12+len(plaintext)+16 {
		t.Fatal("nonce prefix, GCM tag or StdBase64 format changed")
	}
	// Independent old lower-level oracle catches plaintext/key argument inversion.
	opened, err := base.DecryptSecretKey(first, []byte(encryptionKey))
	if err != nil || opened != plaintext {
		t.Fatal("new writer did not use the supplied raw 32-byte key")
	}
}

func TestPasswordExplicitKeyErrorsNeverReadGlobalFallback(t *testing.T) {
	old := viper.Get("tree.password_encryption_key")
	viper.Set("tree.password_encryption_key", strings.Repeat("v", 32))
	t.Cleanup(func() { viper.Set("tree.password_encryption_key", old) })
	key := strings.Repeat("0f", 16)
	encoded, err := EncryptPassword("synthetic-plaintext-canary", key)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, input, key string
		encrypt, valid   bool
	}{
		{"empty encrypt bypasses missing key", "", "", true, true}, {"empty decrypt bypasses invalid key", "", "bad", false, true},
		{"missing encryption key", "synthetic-plaintext-canary", "", true, false}, {"wrong encryption key length", "synthetic-plaintext-canary", "short-key-canary", true, false},
		{"missing decryption key", encoded, "", false, false}, {"wrong decryption key length", encoded, "short-key-canary", false, false},
		{"wrong decryption key", encoded, strings.Repeat("f0", 16), false, false},
		{"hex spelling remains opaque", encoded, strings.ToUpper(key), false, false}, {"malformed ciphertext", "ciphertext-private-canary", key, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var err error
			if tc.encrypt {
				_, err = EncryptPassword(tc.input, tc.key)
			} else {
				_, err = DecryptPassword(tc.input, tc.key)
			}
			if (err == nil) != tc.valid {
				t.Fatal("explicit cipher contract mismatch")
			}
			if err != nil {
				for _, value := range []string{tc.input, tc.key} {
					if value != "" && strings.Contains(err.Error(), value) {
						t.Fatal("cipher error disclosed an input")
					}
				}
			}
		})
	}
}
