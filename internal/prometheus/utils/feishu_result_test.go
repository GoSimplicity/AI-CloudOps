package utils

import (
	"errors"
	"strings"
	"testing"
)

func TestExplicitFeishuResult(t *testing.T) {
	for _, test := range []struct {
		name, body   string
		success      bool
		providerCode int
	}{
		{"explicit zero", `{"code":0}`, true, 0},
		{"unknown fields", `{"code":0,"msg":"PRIVATE","data":{"code":42},"StatusCode":42}`, true, 0},
		{"whitespace", `  { "code" : 0 }  `, true, 0},
		{"missing", `{}`, false, 0}, {"null code", `{"code":null}`, false, 0}, {"bool", `{"code":false}`, false, 0}, {"string", `{"code":"0"}`, false, 0},
		{"decimal", `{"code":0.0}`, false, 0}, {"exponent", `{"code":0e0}`, false, 0}, {"nonzero", `{"code":42,"msg":"PRIVATE"}`, false, 42},
		{"duplicate", `{"code":42,"code":0}`, false, 0}, {"duplicate identical", `{"code":0,"code":0}`, false, 0}, {"escaped duplicate", `{"code":0,"co\u0064e":0}`, false, 0},
		{"StatusCode is not code", `{"StatusCode":0}`, false, 0}, {"case variant is not code", `{"Code":0}`, false, 0},
		{"array", `[{"code":0}]`, false, 0}, {"scalar", `0`, false, 0}, {"null object", `null`, false, 0}, {"non JSON", `PRIVATE`, false, 0},
		{"two objects", `{"code":0}{"code":0}`, false, 0}, {"trailing scalar", `{"code":0} true`, false, 0}, {"overflow", `{"code":999999999999999999999999}`, false, 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := ConfirmFeishuResponse([]byte(test.body))
			if (err == nil) != test.success {
				t.Fatal("explicit result confirmation differs")
			}
			if err != nil && strings.Contains(err.Error(), "PRIVATE") {
				t.Fatal("provider body leaked in error")
			}
			if test.providerCode != 0 {
				var provider *FeishuProviderError
				if !errors.As(err, &provider) || provider.Code != test.providerCode {
					t.Fatal("safe numeric code classification missing")
				}
			} else if !test.success && !errors.Is(err, ErrFeishuResponse) {
				t.Fatal("malformed or ambiguous result was not classified")
			}
		})
	}
}
