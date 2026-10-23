package jwt

import (
	"errors"
	"testing"
)

func mustNewJWTHandler(t *testing.T, cfg Config) Handler {
	t.Helper()
	h, err := NewJWTHandler(nil, cfg)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

// issuer 为空或仅空白时拒绝创建 Handler
func TestNewJWTHandlerRequiresIssuer(t *testing.T) {
	for _, issuer := range []string{"", "  ", "\t"} {
		h, err := NewJWTHandler(nil, Config{Key1: "k1", Key2: "k2", Issuer: issuer})
		if !errors.Is(err, ErrEmptyIssuer) || h != nil {
			t.Fatalf("issuer=%q 应返回 ErrEmptyIssuer, got h=%v err=%v", issuer, h, err)
		}
	}
	if _, err := NewJWTHandler(nil, Config{Key1: "k1", Key2: "k2", Issuer: "cloudops"}); err != nil {
		t.Fatalf("非空 issuer 应成功: %v", err)
	}
}
