package cache

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"strings"
)

// Test-only inspection of canonical old/new envelopes. It grants no permission
// and is not a substitute for the actual Lua role/fence/CAS checks. As in Lua,
// empty {} and [] collections are equivalent; fixed tuple arities are not.
func decodeProjectionTestWire(raw string, keys []string) (projectionWire, error) {
	var result projectionWire
	text := strings.TrimSpace(raw)
	if text == "" {
		return result, fmt.Errorf("empty wire")
	}
	var top []json.RawMessage
	var err error
	if text[0] == '{' {
		top, err = projectionTestLegacyTuple([]byte(text), keys)
	} else {
		top, err = projectionTestArray(json.RawMessage(text))
	}
	if err != nil {
		return result, err
	}
	if len(top) != 15 {
		return result, fmt.Errorf("wire tuple arity")
	}
	var marker string
	if err := projectionTestValue(top[0], &marker); err != nil || marker != projectionCompactWireTag {
		return result, fmt.Errorf("wire codec tag")
	}
	for i, dst := range []interface{}{&result.Mode, &result.Family, &result.Pool, &result.Target} {
		if err := projectionTestValue(top[i+1], dst); err != nil {
			return result, err
		}
	}
	collections := make([][]json.RawMessage, 4)
	for i := range collections {
		collections[i], err = projectionTestArray(top[i+5])
		if err != nil {
			return result, err
		}
	}
	if len(collections[0]) != len(keys) {
		return result, fmt.Errorf("role/key count")
	}
	result.Roles = make([]projectionRole, len(collections[0]))
	result.Fences = make([]projectionFence, len(collections[1]))
	result.Artifacts = make([]projectionArtifact, len(collections[2]))
	result.Controls = make([]projectionControl, len(collections[3]))
	for i, raw := range collections[0] {
		row, err := projectionTestTuple(raw, 5)
		if err != nil {
			return result, err
		}
		role := &result.Roles[i]
		role.Key = keys[i]
		for j, dst := range []interface{}{&role.Role, &role.Type, &role.Kind, &role.IP, &role.Group} {
			if err := projectionTestValue(row[j], dst); err != nil {
				return result, err
			}
		}
	}
	for i, raw := range collections[1] {
		row, err := projectionTestTuple(raw, 4)
		if err != nil {
			return result, err
		}
		f := &result.Fences[i]
		if f.Key, err = projectionTestReference(row[0], len(keys), false); err != nil {
			return result, err
		}
		for j, dst := range []interface{}{&f.Version, &f.Nonce, &f.Signature} {
			if err := projectionTestValue(row[j+1], dst); err != nil {
				return result, err
			}
		}
	}
	for i, raw := range collections[2] {
		row, err := projectionTestTuple(raw, 6)
		if err != nil {
			return result, err
		}
		a := &result.Artifacts[i]
		if err := projectionTestValue(row[0], &a.ID); err != nil {
			return result, err
		}
		if a.Raw, err = projectionTestReference(row[1], len(keys), false); err != nil {
			return result, err
		}
		if a.Fence, err = projectionTestReference(row[2], len(keys), false); err != nil {
			return result, err
		}
		for j, dst := range []interface{}{&a.Action, &a.SHA1, &a.Signature} {
			if err := projectionTestValue(row[j+3], dst); err != nil {
				return result, err
			}
		}
	}
	for i, raw := range collections[3] {
		row, err := projectionTestTuple(raw, 8)
		if err != nil {
			return result, err
		}
		c := &result.Controls[i]
		if err := projectionTestValue(row[0], &c.ID); err != nil {
			return result, err
		}
		if err := projectionTestValue(row[1], &c.Kind); err != nil {
			return result, err
		}
		if c.Key, err = projectionTestReference(row[2], len(keys), true); err != nil {
			return result, err
		}
		lists := []*[]string{&c.Desired, &c.Add, &c.Remove}
		for j, dst := range lists {
			value, err := projectionTestStrings(row[j+3])
			if err != nil {
				return result, err
			}
			*dst = value
		}
		if err := projectionTestValue(row[6], &c.Signature); err != nil {
			return result, err
		}
		if err := projectionTestValue(row[7], &c.RequireEmpty); err != nil {
			return result, err
		}
	}
	for i, dst := range []*int{&result.PoolFence, &result.Ack, &result.Legacy} {
		value, err := projectionTestReference(top[i+9], len(keys), true)
		if err != nil {
			return result, err
		}
		*dst = value
	}
	for i, dst := range []interface{}{&result.LegacySHA, &result.AckReceipt, &result.AckSignature} {
		if err := projectionTestValue(top[i+12], dst); err != nil {
			return result, err
		}
	}
	return result, nil
}
func projectionTestArray(raw json.RawMessage) ([]json.RawMessage, error) {
	text := bytes.TrimSpace(raw)
	if len(text) == 0 || bytes.Equal(text, []byte("null")) {
		return nil, fmt.Errorf("missing/null collection")
	}
	if text[0] == '{' {
		var object map[string]json.RawMessage
		if err := json.Unmarshal(text, &object); err != nil || object == nil || len(object) != 0 {
			return nil, fmt.Errorf("nonempty object in array position")
		}
		return []json.RawMessage{}, nil
	}
	if text[0] != '[' {
		return nil, fmt.Errorf("non-array collection")
	}
	var values []json.RawMessage
	if err := json.Unmarshal(text, &values); err != nil || values == nil {
		return nil, fmt.Errorf("invalid array")
	}
	return values, nil
}
func projectionTestTuple(raw json.RawMessage, arity int) ([]json.RawMessage, error) {
	row, err := projectionTestArray(raw)
	if err != nil || len(row) != arity {
		return nil, fmt.Errorf("tuple arity")
	}
	return row, nil
}
func projectionTestValue(raw json.RawMessage, dst interface{}) error {
	if len(raw) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return fmt.Errorf("missing/null field")
	}
	return json.Unmarshal(raw, dst)
}
func projectionTestStrings(raw json.RawMessage) ([]string, error) {
	values, err := projectionTestArray(raw)
	if err != nil {
		return nil, err
	}
	result := make([]string, len(values))
	for i, value := range values {
		if err := projectionTestValue(value, &result[i]); err != nil {
			return nil, err
		}
	}
	return result, nil
}
func projectionTestReference(raw json.RawMessage, maximum int, zero bool) (int, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value interface{}
	if err := decoder.Decode(&value); err != nil {
		return 0, err
	}
	number, ok := value.(json.Number)
	if !ok {
		return 0, fmt.Errorf("reference is not a number")
	}
	n, err := number.Float64()
	minimum := 1
	if zero {
		minimum = 0
	}
	if err != nil || math.IsNaN(n) || math.IsInf(n, 0) || n != math.Trunc(n) || n < float64(minimum) || n > float64(maximum) {
		return 0, fmt.Errorf("invalid reference")
	}
	return int(n), nil
}
func projectionTestLegacyTuple(raw []byte, keys []string) ([]json.RawMessage, error) {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil || object == nil {
		return nil, fmt.Errorf("invalid object wire")
	}
	top := make([]json.RawMessage, 15)
	top[0] = json.RawMessage(`"pw2"`)
	fields := []string{"mode", "family", "pool", "target", "roles", "fences", "artifacts", "controls", "pool_fence", "ack", "legacy", "legacy_sha", "ack_receipt", "ack_signature"}
	for i, key := range fields {
		value, ok := object[key]
		if !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return nil, fmt.Errorf("missing/null legacy field")
		}
		top[i+1] = value
	}
	rowFields := [][]string{{"role", "type", "kind", "ip", "group"}, {"key", "version", "nonce", "signature"}, {"id", "raw", "fence", "action", "sha1", "signature"}, {"id", "kind", "key", "desired", "add", "remove", "signature", "require_empty"}}
	for collection, names := range rowFields {
		rows, err := projectionTestArray(top[collection+5])
		if err != nil {
			return nil, err
		}
		if collection == 0 && len(rows) != len(keys) {
			return nil, fmt.Errorf("legacy role/key count")
		}
		converted := make([][]json.RawMessage, len(rows))
		for i, rawRow := range rows {
			var row map[string]json.RawMessage
			if err := json.Unmarshal(rawRow, &row); err != nil || row == nil {
				return nil, fmt.Errorf("legacy row is not object")
			}
			if collection == 0 {
				var key string
				if err := projectionTestValue(row["key"], &key); err != nil || key != keys[i] {
					return nil, fmt.Errorf("legacy role key mismatch")
				}
			}
			converted[i] = make([]json.RawMessage, len(names))
			for j, name := range names {
				value, ok := row[name]
				if !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
					return nil, fmt.Errorf("legacy row missing/null field")
				}
				converted[i][j] = value
			}
		}
		encoded, err := json.Marshal(converted)
		if err != nil {
			return nil, err
		}
		top[collection+5] = encoded
	}
	return top, nil
}
