package action

import (
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestSignAndVerifyRoundTrip(t *testing.T) {
	expires := time.Now().Add(LinkTTL)
	sig, err := Sign("fixture-action-hmac", "silence", "abc", "1", expires)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(sig, "fixture-action-hmac") {
		t.Fatal("signature contained the key")
	}
	if err := Verify("fixture-action-hmac", "silence", "abc", "1", strconv.FormatInt(expires.Unix(), 10), sig, time.Now()); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyRejectsTamperExpiryAndEmptyKey(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	expires := now.Add(time.Hour)
	sig, err := Sign("fixture-action-hmac", "silence", "abc", "1", expires)
	if err != nil {
		t.Fatal(err)
	}
	exp := strconv.FormatInt(expires.Unix(), 10)
	cases := []struct {
		name, key, actionName, fingerprint, extra, exp, sig string
		at                                                  time.Time
	}{
		{"wrong key", "other-hmac", "silence", "abc", "1", exp, sig, now},
		{"other action", "fixture-action-hmac", "unsilence", "abc", "1", exp, sig, now},
		{"other fingerprint", "fixture-action-hmac", "silence", "zzz", "1", exp, sig, now},
		{"other hour", "fixture-action-hmac", "silence", "abc", "24", exp, sig, now},
		{"expired", "fixture-action-hmac", "silence", "abc", "1", exp, sig, expires},
		{"empty key", "", "silence", "abc", "1", exp, sig, now},
		{"bad sig", "fixture-action-hmac", "silence", "abc", "1", exp, "00", now},
		{"bad exp", "fixture-action-hmac", "silence", "abc", "1", "nope", sig, now},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := Verify(tc.key, tc.actionName, tc.fingerprint, tc.extra, tc.exp, tc.sig, tc.at); err == nil {
				t.Fatal("expected rejection")
			}
		})
	}
}

func TestSignRejectsEmptyKey(t *testing.T) {
	if _, err := Sign("  ", "silence", "abc", "1", time.Now().Add(time.Minute)); err == nil {
		t.Fatal("blank key must not sign")
	}
}
