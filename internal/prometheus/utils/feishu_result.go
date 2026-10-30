package utils

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

var ErrFeishuResponse = errors.New("Feishu response requires one explicit integer code")

type FeishuProviderError struct{ Code int }

func (err *FeishuProviderError) Error() string {
	return fmt.Sprintf("Feishu provider code %d", err.Code)
}

// ConfirmFeishuResponse confirms the project's recorded code==0 contract. It is
// a business protocol check, separate from the HTTP helper's status semantics.
// Unknown fields are allowed; missing/null/duplicate code and trailing JSON are
// rejected without including any provider body or message in the error.
func ConfirmFeishuResponse(body []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(body))
	first, err := decoder.Token()
	if err != nil || first != json.Delim('{') {
		return ErrFeishuResponse
	}
	seen := false
	code := 0
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return ErrFeishuResponse
		}
		name, ok := token.(string)
		if !ok {
			return ErrFeishuResponse
		}
		var raw json.RawMessage
		if err := decoder.Decode(&raw); err != nil {
			return ErrFeishuResponse
		}
		if name != "code" {
			continue
		}
		if seen || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			return ErrFeishuResponse
		}
		seen = true
		if json.Unmarshal(raw, &code) != nil {
			return ErrFeishuResponse
		}
	}
	closing, err := decoder.Token()
	if err != nil || closing != json.Delim('}') || !seen {
		return ErrFeishuResponse
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		return ErrFeishuResponse
	}
	if code != 0 {
		return &FeishuProviderError{Code: code}
	}
	return nil
}
