package cache

import "encoding/json"

const projectionCompactWireTag = "pw2"

// Only transport is compacted. Lua reconstructs the complete canonical wire
// before running the existing role, TYPE, full-vector and frame-fingerprint
// checks. All KEYS remain explicit, and opaque ACK/intent strings stay intact.
func marshalProjectionWire(w projectionWire, keys []string) ([]byte, error) {
	if w.Roles == nil || w.Fences == nil || w.Artifacts == nil || w.Controls == nil || len(w.Roles) != len(keys) {
		return nil, ErrProjectionStore
	}
	roles := make([][5]string, len(w.Roles))
	for i, role := range w.Roles {
		// Elision must never conceal a disagreement with the actual command KEYS.
		if role.Key != keys[i] {
			return nil, ErrProjectionStore
		}
		roles[i] = [5]string{role.Role, role.Type, role.Kind, role.IP, role.Group}
	}
	fences := make([][4]interface{}, len(w.Fences))
	for i, fence := range w.Fences {
		fences[i] = [4]interface{}{fence.Key, fence.Version, fence.Nonce, fence.Signature}
	}
	artifacts := make([][6]interface{}, len(w.Artifacts))
	for i, artifact := range w.Artifacts {
		artifacts[i] = [6]interface{}{artifact.ID, artifact.Raw, artifact.Fence, artifact.Action, artifact.SHA1, artifact.Signature}
	}
	controls := make([][8]interface{}, len(w.Controls))
	for i, control := range w.Controls {
		// nil/null differs from an explicit empty collection in the Go contract.
		// Do not normalize malformed input while encoding it.
		if control.Desired == nil || control.Add == nil || control.Remove == nil {
			return nil, ErrProjectionStore
		}
		controls[i] = [8]interface{}{control.ID, control.Kind, control.Key, control.Desired, control.Add, control.Remove, control.Signature, control.RequireEmpty}
	}
	raw, err := json.Marshal([15]interface{}{projectionCompactWireTag, w.Mode, w.Family, w.Pool, w.Target, roles, fences, artifacts, controls, w.PoolFence, w.Ack, w.Legacy, w.LegacySHA, w.AckReceipt, w.AckSignature})
	if err != nil {
		return nil, ErrProjectionStore
	}
	return raw, nil
}
