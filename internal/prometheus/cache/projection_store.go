package cache

import (
	"context"
	"crypto/sha1"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	configdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
	"github.com/redis/go-redis/v9"
)

var (
	ErrProjectionStore = errors.New("projection store operation failed; reconcile durable receipt before retry")
	ErrProjectionFence = errors.New("projection fence rejected stale or conflicting material")
)

//go:embed projection_fence.lua
var projectionFenceLua string

// ProjectionStore owns its dedicated no-retry connection pool. Closing it never
// closes the application client's pool. This protocol targets standalone Redis.
type ProjectionStore struct{ client *redis.Client }

func NewProjectionStore(client *redis.Client) (*ProjectionStore, error) {
	if client == nil || client.Options().DB != 0 {
		return nil, ErrProjectionStore
	}
	options := *client.Options()
	options.MaxRetries = -1
	return &ProjectionStore{client: redis.NewClient(&options)}, nil
}
func (s *ProjectionStore) Close() error { return s.client.Close() }

type PreparedProjection struct {
	store     *ProjectionStore
	lease     configdao.ProjectionLease
	wire      projectionWire
	keys      []string
	materials map[string][]byte
	proof     *configdao.ProjectionAckProof
}

func (s *ProjectionStore) Prepare(ctx context.Context, lease configdao.ProjectionLease) (*PreparedProjection, error) {
	if lease == nil {
		return nil, ErrProjectionStore
	}
	if err := lease.RecheckCurrent(ctx); err != nil {
		return nil, err
	}
	snapshot, err := lease.Snapshot()
	if err != nil {
		return nil, err
	}
	handle, err := compileProjection(s, lease, snapshot)
	if err != nil {
		return nil, err
	}
	if err := handle.execute(ctx, "prepare", "", nil); err != nil {
		return nil, err
	}
	return handle, nil
}
func (p *PreparedProjection) ApplyArtifact(ctx context.Context, eventID uint64) error {
	id := strconv.FormatUint(eventID, 10)
	body, ok := p.materials[id]
	if !ok {
		return ErrProjectionStore
	}
	return p.execute(ctx, "artifact", id, body)
}
func (p *PreparedProjection) ApplyMembers(ctx context.Context, controlID uint64) error {
	return p.execute(ctx, "members", strconv.FormatUint(controlID, 10), nil)
}
func (p *PreparedProjection) AddIntents(ctx context.Context, controlID uint64) error {
	return p.execute(ctx, "intent_add", strconv.FormatUint(controlID, 10), nil)
}
func (p *PreparedProjection) RemoveIntents(ctx context.Context, controlID uint64) error {
	return p.execute(ctx, "intent_remove", strconv.FormatUint(controlID, 10), nil)
}
func (p *PreparedProjection) Acknowledge(ctx context.Context) (configdao.ProjectionAckProof, error) {
	if p.proof == nil {
		return configdao.ProjectionAckProof{}, ErrProjectionStore
	}
	if err := p.execute(ctx, "ack", "", nil); err != nil {
		return configdao.ProjectionAckProof{}, err
	}
	proof := *p.proof
	proof.ArtifactVector = append([]configdao.ProjectionVectorEntry(nil), proof.ArtifactVector...)
	proof.RequiredControls = append([]configdao.ProjectionControlDigest(nil), proof.RequiredControls...)
	return proof, nil
}
func (p *PreparedProjection) execute(ctx context.Context, mode, target string, body []byte) error {
	// Never renew a receipt on another SQL connection. The original sealed lease
	// must still own its control/namespace locks immediately before every command.
	if err := p.lease.RecheckCurrent(ctx); err != nil {
		return err
	}
	wire := p.wire
	wire.Mode = mode
	wire.Target = target
	raw, err := json.Marshal(wire)
	if err != nil {
		return ErrProjectionStore
	}
	result, err := p.store.client.Eval(ctx, projectionFenceLua, p.keys, string(raw), string(body)).Text()
	if err != nil {
		return errors.Join(ErrProjectionStore, ctx.Err())
	}
	if result != "OK" {
		return ErrProjectionFence
	}
	return nil
}

type projectionRole struct {
	Role  string `json:"role"`
	Key   string `json:"key"`
	Type  string `json:"type"`
	Kind  string `json:"kind"`
	IP    string `json:"ip"`
	Group string `json:"group"`
}
type projectionFence struct {
	Key       int    `json:"key"`
	Version   string `json:"version"`
	Nonce     string `json:"nonce"`
	Signature string `json:"signature"`
}
type projectionArtifact struct {
	ID        string `json:"id"`
	Raw       int    `json:"raw"`
	Fence     int    `json:"fence"`
	Action    string `json:"action"`
	SHA1      string `json:"sha1"`
	Signature string `json:"signature"`
}
type projectionControl struct {
	ID           string   `json:"id"`
	Kind         string   `json:"kind"`
	Key          int      `json:"key"`
	Desired      []string `json:"desired"`
	Add          []string `json:"add"`
	Remove       []string `json:"remove"`
	Signature    string   `json:"signature"`
	RequireEmpty bool     `json:"require_empty"`
}
type projectionWire struct {
	Mode         string               `json:"mode"`
	Family       string               `json:"family"`
	Pool         string               `json:"pool"`
	Target       string               `json:"target"`
	Roles        []projectionRole     `json:"roles"`
	Fences       []projectionFence    `json:"fences"`
	Artifacts    []projectionArtifact `json:"artifacts"`
	Controls     []projectionControl  `json:"controls"`
	PoolFence    int                  `json:"pool_fence"`
	Ack          int                  `json:"ack"`
	Legacy       int                  `json:"legacy"`
	LegacySHA    string               `json:"legacy_sha"`
	AckReceipt   string               `json:"ack_receipt"`
	AckSignature string               `json:"ack_signature"`
}

func projectionVersion(value uint64) string { return fmt.Sprintf("%020d", value) }
func projectionDigest(value interface{}) string {
	raw, _ := json.Marshal(value)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
func projectionPoolKey(family int8, poolID int, suffix string) string {
	return fmt.Sprintf("%s:projection:v1:pool:%d:%d:%s", redisPrefix, family, poolID, suffix)
}
func projectionNamespaceKey(kind int8, key string) string {
	return fmt.Sprintf("%s:projection:v1:namespace:%d:%s", redisPrefix, kind, hex.EncodeToString([]byte(key)))
}
func projectionRawKey(binding configdao.ProjectionBinding) (string, error) {
	raw, err := json.Marshal(binding)
	if err != nil {
		return "", ErrProjectionStore
	}
	validated, err := configdao.ParseProjectionBinding(string(raw))
	if err != nil {
		return "", err
	}
	switch validated.ConfigType {
	case 1:
		return buildRedisKeyPrometheusMain(validated.InstanceIP), nil
	case 2:
		return buildRedisKeyAlertManagerMain(validated.InstanceIP), nil
	case 3:
		return buildRedisKeyAlertRule(validated.InstanceIP), nil
	case 4:
		return buildRedisKeyRecordRule(validated.InstanceIP), nil
	case 5:
		return buildRedisKeyWebhookFile(validated.InstanceIP, *validated.GroupID), nil
	}
	return "", ErrProjectionStore
}
func projectionMemberKey(family int8, poolID int) (string, error) {
	switch family {
	case 1:
		return buildRedisSetKeyPrometheusMainPoolIPs(poolID), nil
	case 2:
		return buildRedisSetKeyAlertManagerMainPoolIPs(poolID), nil
	case 3:
		return buildRedisSetKeyAlertRulePoolIPs(poolID), nil
	case 4:
		return buildRedisSetKeyRecordRulePoolIPs(poolID), nil
	}
	return "", ErrProjectionStore
}
func projectionLegacyKey(family int8, poolID int) (string, error) {
	switch family {
	case 1:
		return promPublicationKey(poolID), nil
	case 2:
		return alertPublicationStateKey(poolID), nil
	case 3:
		return alertRulePublicationKey(poolID), nil
	case 4:
		return recordPublicationKey(poolID), nil
	}
	return "", ErrProjectionStore
}

func compileProjection(store *ProjectionStore, lease configdao.ProjectionLease, snapshot configdao.ProjectionLeaseSnapshot) (*PreparedProjection, error) {
	handle := &PreparedProjection{store: store, lease: lease, materials: map[string][]byte{}}
	wire := &handle.wire
	wire.Roles = []projectionRole{}
	wire.Fences = []projectionFence{}
	wire.Artifacts = []projectionArtifact{}
	wire.Controls = []projectionControl{}
	addKey := func(key, role, kind string) (int, error) {
		for i, known := range wire.Roles {
			if known.Key == key {
				if known.Role != role || known.Type != kind {
					return 0, ErrProjectionStore
				}
				return i + 1, nil
			}
		}
		wire.Roles = append(wire.Roles, projectionRole{Role: role, Key: key, Type: kind})
		handle.keys = append(handle.keys, key)
		return len(handle.keys), nil
	}
	nonce := ""
	if snapshot.Pool != nil {
		receipt := snapshot.Pool
		if receipt.ManifestVersion != 2 || !receipt.CompleteControls {
			return nil, ErrProjectionStore
		}
		ref := receipt.Reservation
		wire.Family = strconv.Itoa(int(ref.Family))
		wire.Pool = strconv.Itoa(ref.PoolID)
		nonce = hex.EncodeToString(ref.Nonce[:])
		index, err := addKey(projectionPoolKey(ref.Family, ref.PoolID, "fence"), "pool_fence", "hash")
		if err != nil {
			return nil, err
		}
		wire.PoolFence = index
		wire.Fences = append(wire.Fences, projectionFence{index, projectionVersion(ref.Generation), nonce, projectionPoolDigest(*receipt)})
		wire.Ack, err = addKey(projectionPoolKey(ref.Family, ref.PoolID, "ack"), "ack", "string")
		if err != nil {
			return nil, err
		}
		for _, control := range receipt.Controls {
			item := projectionControl{ID: strconv.FormatUint(control.ControlID, 10), Kind: string(control.Kind), Signature: hex.EncodeToString(control.Digest[:]), Desired: []string{}, Add: []string{}, Remove: []string{}}
			switch control.Kind {
			case configdao.ProjectionMembersFull:
				key, err := projectionMemberKey(ref.Family, ref.PoolID)
				if err != nil || control.Members == nil {
					return nil, ErrProjectionStore
				}
				item.Key, err = addKey(key, "members", "set")
				if err != nil {
					return nil, err
				}
				item.Desired = append([]string{}, control.Members.DesiredIPs...)
			case configdao.ProjectionWebhookIntents:
				if ref.Family != model.ConfigPublicationFamilyAlertManager || control.Intents == nil {
					return nil, ErrProjectionStore
				}
				var err error
				item.Key, err = addKey(alertRetirementIntentKey(ref.PoolID), "intents", "set")
				if err != nil {
					return nil, err
				}
				item.RequireEmpty = control.Intents.RequireEmpty
				for _, change := range control.Intents.Add {
					raw, err := json.Marshal(change.Owner)
					if err != nil {
						return nil, ErrProjectionStore
					}
					item.Add = append(item.Add, string(raw))
				}
				for _, change := range control.Intents.Remove {
					raw, err := json.Marshal(change.Owner)
					if err != nil {
						return nil, ErrProjectionStore
					}
					item.Remove = append(item.Remove, string(raw))
				}
			case configdao.ProjectionACKFull, configdao.ProjectionACKWebhookSubset:
				if control.Ack == nil {
					return nil, ErrProjectionStore
				}
				proof := configdao.ProjectionAckProof{Reservation: ref, SourceHash: receipt.SourceHash, PlanDigest: receipt.PlanDigest, ArtifactVector: control.Ack.ArtifactVector, RequiredControls: control.Ack.RequiredControls}
				handle.proof = &proof
				raw, err := json.Marshal(proof)
				if err != nil {
					return nil, ErrProjectionStore
				}
				wire.AckReceipt = string(raw)
				wire.AckSignature = controlDigest(control)
				if control.Kind == configdao.ProjectionACKFull {
					if control.Ack.LegacySHA == nil {
						return nil, ErrProjectionStore
					}
					key, err := projectionLegacyKey(ref.Family, ref.PoolID)
					if err != nil {
						return nil, err
					}
					wire.Legacy, err = addKey(key, "legacy_ack", "string")
					if err != nil {
						return nil, err
					}
					wire.LegacySHA = hex.EncodeToString(control.Ack.LegacySHA[:])
				} else if ref.Family != 5 {
					return nil, ErrProjectionStore
				}
			default:
				return nil, ErrProjectionStore
			}
			wire.Controls = append(wire.Controls, item)
		}
	} else if len(snapshot.Artifacts) != 1 {
		return nil, ErrProjectionStore
	}
	for _, material := range snapshot.Artifacts {
		artifact := material.Artifact
		if snapshot.Pool == nil && (artifact.OperationID != nil || artifact.SourceID != 0 || artifact.Action == model.ConfigProjectionRender) {
			return nil, ErrProjectionStore
		}
		key, err := projectionRawKey(material.Binding)
		if err != nil {
			return nil, err
		}
		raw, err := addKey(key, "artifact", "string")
		if err != nil {
			return nil, err
		}
		fence, err := addKey(projectionNamespaceKey(artifact.ConfigType, artifact.InstanceKey), "namespace_fence", "hash")
		if err != nil {
			return nil, err
		}
		for _, index := range []int{raw, fence} {
			wire.Roles[index-1].Kind = strconv.Itoa(int(material.Binding.ConfigType))
			wire.Roles[index-1].IP = material.Binding.InstanceIP
			if material.Binding.GroupID != nil {
				wire.Roles[index-1].Group = strconv.Itoa(*material.Binding.GroupID)
			}
		}
		signature := projectionDigest(struct {
			Artifact configdao.ArtifactReceipt
			Binding  configdao.ProjectionBinding
			Plan     string
		}{artifact, material.Binding, wire.AckSignature})
		artifactNonce := nonce
		if snapshot.Pool == nil {
			artifactNonce = projectionDigest(artifact)[:32]
		}
		wire.Fences = append(wire.Fences, projectionFence{fence, projectionVersion(artifact.EventID), artifactNonce, signature})
		id := strconv.FormatUint(artifact.EventID, 10)
		sum := sha1.Sum(material.Body)
		wire.Artifacts = append(wire.Artifacts, projectionArtifact{id, raw, fence, artifact.Action, hex.EncodeToString(sum[:]), signature})
		handle.materials[id] = append([]byte(nil), material.Body...)
	}
	return handle, nil
}
func controlDigest(control configdao.IssuedProjectionControl) string {
	return hex.EncodeToString(control.Digest[:])
}

func projectionPoolDigest(receipt configdao.ProjectionReceipt) string {
	receipt.Phase = ""
	return projectionDigest(receipt)
}
