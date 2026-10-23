package model

import (
	"encoding/json"
	"strings"
)

const CredentialAuditRedaction = "[REDACTED]"

// Redacted 只保留凭据变更事件，不记录旧值、新值或密文
func (c CloudResourceChangeLog) Redacted() CloudResourceChangeLog {
	switch strings.ToLower(strings.TrimSpace(c.FieldName)) {
	case "key", "password":
		c.OldValue = CredentialAuditRedaction
		c.NewValue = CredentialAuditRedaction
	}
	return c
}

// MarshalJSON 为升级前的历史记录提供相同的读取边界
func (c CloudResourceChangeLog) MarshalJSON() ([]byte, error) {
	type public CloudResourceChangeLog
	return json.Marshal(public(c.Redacted()))
}
