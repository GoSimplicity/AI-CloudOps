package jwt

import (
	"strings"
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
)

func TestRefreshTokenCarriesAndChecksIssuer(t *testing.T) {
	key1, key2 := strings.Repeat("a", 32), strings.Repeat("b", 32)
	current := NewJWTHandler(nil, Config{Key1: key1, Key2: key2, Issuer: "cloudops-new"})
	old := NewJWTHandler(nil, Config{Key1: key1, Key2: key2, Issuer: "cloudops-old"})

	t.Run("same-issuer-accepted", func(t *testing.T) {
		token, err := current.setRefreshToken(nil, 7, "admin", "ssid", 1)
		if err != nil {
			t.Fatal(err)
		}
		rc, err := current.ParseRefreshClaims(token)
		if err != nil {
			t.Fatalf("同 issuer 的刷新令牌应通过: %v", err)
		}
		if rc.Issuer != "cloudops-new" {
			t.Fatalf("刷新令牌必须写入 issuer, got %q", rc.Issuer)
		}
	})

	t.Run("different-issuer-rejected", func(t *testing.T) {
		token, err := old.setRefreshToken(nil, 7, "admin", "ssid", 1)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := current.ParseRefreshClaims(token); err == nil {
			t.Fatal("issuer 不同的刷新令牌必须被拒")
		}
	})

	t.Run("missing-issuer-rejected", func(t *testing.T) {
		// 修复前签发的刷新令牌不带 iss，同样必须被拒
		token, err := jwtlib.NewWithClaims(jwtlib.SigningMethodHS512, RefreshClaims{
			Uid: 7, Username: "admin", Ssid: "ssid",
			RegisteredClaims: jwtlib.RegisteredClaims{ExpiresAt: jwtlib.NewNumericDate(time.Now().Add(time.Hour))},
		}).SignedString([]byte(key2))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := current.ParseRefreshClaims(token); err == nil {
			t.Fatal("缺少 issuer 的刷新令牌必须被拒")
		}
	})
}
