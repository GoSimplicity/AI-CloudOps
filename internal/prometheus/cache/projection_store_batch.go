package cache

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"strconv"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

const projectionArtifactBatchMaxItems = 16
const projectionArtifactBatchMaxRequestBytes int64 = 1 << 20

type projectionArtifactSelection struct {
	eventID uint64
	id      string
	body    []byte
}
type projectionArtifactCommand struct {
	selections    []projectionArtifactSelection
	selectionJSON string
	single        bool
}

// ApplyArtifacts validates the complete selection before the first command.
// Budget-based single-command fallback is chosen before sending, never after a
// Redis error. Old single actions and raw body bytes remain compatible.
func (p *PreparedProjection) ApplyArtifacts(ctx context.Context, eventIDs []uint64) error {
	if p == nil || p.lease == nil {
		return ErrProjectionStore
	}
	if err := p.lease.RecheckCurrent(ctx); err != nil {
		return err
	}
	selected, err := p.selectArtifacts(append([]uint64(nil), eventIDs...))
	if err != nil {
		return err
	}
	if len(selected) == 0 {
		return nil
	}
	wire := p.wire
	wire.Mode = "artifact_batch"
	wire.Target = ""
	encoded, err := json.Marshal(wire)
	if err != nil {
		return ErrProjectionStore
	}
	commands, err := p.artifactCommands(string(encoded), selected)
	if err != nil {
		return err
	}
	for _, command := range commands {
		if command.single {
			if err := p.ApplyArtifact(ctx, command.selections[0].eventID); err != nil {
				return err
			}
			continue
		}
		args := make([]string, 0, len(command.selections)+2)
		args = append(args, string(encoded), command.selectionJSON)
		for _, item := range command.selections {
			args = append(args, string(item.body))
		}
		if err := p.executeArguments(ctx, args); err != nil {
			return err
		}
	}
	return nil
}
func (p *PreparedProjection) selectArtifacts(ids []uint64) ([]projectionArtifactSelection, error) {
	byID := make(map[string]projectionArtifact, len(p.wire.Artifacts))
	for _, artifact := range p.wire.Artifacts {
		if _, duplicate := byID[artifact.ID]; duplicate {
			return nil, ErrProjectionStore
		}
		byID[artifact.ID] = artifact
	}
	if p.wire.PoolFence == 0 && len(ids) > 1 {
		return nil, ErrProjectionStore
	}
	selected := make([]projectionArtifactSelection, 0, len(ids))
	seen := make(map[uint64]bool, len(ids))
	for _, eventID := range ids {
		if eventID == 0 || seen[eventID] {
			return nil, ErrProjectionStore
		}
		seen[eventID] = true
		id := strconv.FormatUint(eventID, 10)
		artifact, exists := byID[id]
		body, hasBody := p.materials[id]
		if !exists || !hasBody {
			return nil, ErrProjectionStore
		}
		if artifact.Raw < 1 || artifact.Raw > len(p.wire.Roles) || p.wire.Roles[artifact.Raw-1].Role != "artifact" {
			return nil, ErrProjectionStore
		}
		if p.wire.PoolFence != 0 {
			kind := p.wire.Roles[artifact.Raw-1].Kind
			if p.wire.Family == "2" {
				if kind != "2" && kind != "5" {
					return nil, ErrProjectionStore
				}
			} else if kind != p.wire.Family {
				return nil, ErrProjectionStore
			}
		}

		switch artifact.Action {
		case model.ConfigProjectionRender:
			if p.wire.PoolFence == 0 {
				return nil, ErrProjectionStore
			}
			digest := sha1.Sum(body)
			if hex.EncodeToString(digest[:]) != artifact.SHA1 {
				return nil, ErrProjectionStore
			}
		case model.ConfigProjectionRetire, model.ConfigProjectionInvalidate:
			if len(body) != 0 || p.wire.Family == "5" {
				return nil, ErrProjectionStore
			}
		default:
			return nil, ErrProjectionStore
		}
		selected = append(selected, projectionArtifactSelection{eventID: eventID, id: id, body: body})
	}
	return selected, nil
}
func (p *PreparedProjection) artifactCommands(wire string, selected []projectionArtifactSelection) ([]projectionArtifactCommand, error) {
	commands := []projectionArtifactCommand{}
	pending := []projectionArtifactSelection{}
	flush := func() error {
		if len(pending) == 0 {
			return nil
		}
		copy := append([]projectionArtifactSelection(nil), pending...)
		encoded, err := projectionSelectionJSON(copy)
		if err != nil {
			return err
		}
		commands = append(commands, projectionArtifactCommand{selections: copy, selectionJSON: encoded, single: len(copy) == 1 || p.wire.PoolFence == 0})
		pending = nil
		return nil
	}
	fits := func(items []projectionArtifactSelection) (bool, error) {
		if len(items) > projectionArtifactBatchMaxItems || p.wire.PoolFence == 0 {
			return false, nil
		}
		selection, err := projectionSelectionJSON(items)
		if err != nil {
			return false, err
		}
		lengths := []int{len(wire), len(selection)}
		for _, item := range items {
			lengths = append(lengths, len(item.body))
		}
		return projectionEvalRESPBytesForLengths(p.keys, lengths) <= projectionArtifactBatchMaxRequestBytes, nil
	}
	for _, item := range selected {
		candidate := append(pending, item)
		ok, err := fits(candidate)
		if err != nil {
			return nil, err
		}
		if ok {
			pending = candidate
			continue
		}
		if err := flush(); err != nil {
			return nil, err
		}
		ok, err = fits([]projectionArtifactSelection{item})
		if err != nil {
			return nil, err
		}
		if ok {
			pending = []projectionArtifactSelection{item}
		} else {
			commands = append(commands, projectionArtifactCommand{selections: []projectionArtifactSelection{item}, single: true})
		}
	}
	if err := flush(); err != nil {
		return nil, err
	}
	return commands, nil
}
func projectionSelectionJSON(items []projectionArtifactSelection) (string, error) {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.id)
	}
	raw, err := json.Marshal(ids)
	if err != nil {
		return "", ErrProjectionStore
	}
	return string(raw), nil
}

// Exact EVAL RESP-array bytes, including the current script, numkeys, all KEYS,
// JSON arguments, raw bodies, and every bulk/array framing byte. This bounds
// only batch eligibility; a legal oversized item keeps the original single path.
func projectionEvalRESPBytes(keys []string, args []string) int64 {
	lengths := make([]int, len(args))
	for i, value := range args {
		lengths[i] = len(value)
	}
	return projectionEvalRESPBytesForLengths(keys, lengths)
}
func projectionEvalRESPBytesForLengths(keys []string, lengths []int) int64 {
	total := int64(len(strconv.Itoa(3+len(keys)+len(lengths))) + 3)
	for _, size := range []int{len("eval"), len(projectionFenceLua), len(strconv.Itoa(len(keys)))} {
		total = projectionRESPAdd(total, projectionRESPBulk(size))
	}
	for _, key := range keys {
		total = projectionRESPAdd(total, projectionRESPBulk(len(key)))
	}
	for _, size := range lengths {
		total = projectionRESPAdd(total, projectionRESPBulk(size))
	}
	return total
}
func projectionRESPBulk(size int) int64 {
	if size < 0 {
		return 1<<63 - 1
	}
	return projectionRESPAdd(int64(size), int64(len(strconv.Itoa(size))+5))
}
func projectionRESPAdd(total, extra int64) int64 {
	const maximum int64 = 1<<63 - 1
	if total < 0 || extra < 0 || total > maximum-extra {
		return maximum
	}
	return total + extra
}
