package jwt

import (
	"strings"
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
)

func TestRejectInvalidAccessClaims(t *testing.T) {
	key := strings.Repeat("a", 32)
	h := mustNewJWTHandler(t, Config{Key1: key, Key2: strings.Repeat("b", 32), Issuer: "cloudops"})
	for _, tt := range []struct {
		name, issuer string
		method       jwtlib.SigningMethod
		expiry       *jwtlib.NumericDate
		valid        bool
	}{
		{"valid", "cloudops", jwtlib.SigningMethodHS512, jwtlib.NewNumericDate(time.Now().Add(time.Hour)), true},
		{"wrong issuer", "other", jwtlib.SigningMethodHS512, jwtlib.NewNumericDate(time.Now().Add(time.Hour)), false},
		{"missing expiry", "cloudops", jwtlib.SigningMethodHS512, nil, false},
		{"wrong algorithm", "cloudops", jwtlib.SigningMethodHS256, jwtlib.NewNumericDate(time.Now().Add(time.Hour)), false},
		{"expired", "cloudops", jwtlib.SigningMethodHS512, jwtlib.NewNumericDate(time.Now().Add(-time.Hour)), false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			token, err := jwtlib.NewWithClaims(tt.method, UserClaims{RegisteredClaims: jwtlib.RegisteredClaims{Issuer: tt.issuer, ExpiresAt: tt.expiry}}).SignedString([]byte(key))
			if err != nil {
				t.Fatal(err)
			}
			_, err = h.ParseUserClaims(token)
			if (err == nil) != tt.valid {
				t.Fatalf("unexpected validity: %v", err)
			}
		})
	}
}
