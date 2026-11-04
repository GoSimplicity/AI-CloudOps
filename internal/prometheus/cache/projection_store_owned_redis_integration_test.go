//go:build integration

package cache

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

var errProjectionRemoteDocker = errors.New("owned projection tests require a local Unix Docker endpoint without environment overrides")

func projectionLocalDockerEndpoint(ctx context.Context) (string, error) {
	if os.Getenv("DOCKER_HOST") != "" || os.Getenv("DOCKER_CONTEXT") != "" {
		return "", errProjectionRemoteDocker
	}
	// Context inspection reads local CLI metadata; no Docker daemon operation
	// occurs until its endpoint has been checked and explicitly selected.
	out, err := exec.CommandContext(ctx, "docker", "context", "inspect", "--format", "{{.Endpoints.docker.Host}}").Output()
	if err != nil {
		return "", errors.New("cannot read local Docker context metadata")
	}
	endpoint := strings.TrimSpace(string(out))
	parsed, err := url.Parse(endpoint)
	if err != nil || !strings.HasPrefix(endpoint, "unix:///") || parsed.Scheme != "unix" || parsed.Host != "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || !filepath.IsAbs(parsed.Path) {
		return "", errProjectionRemoteDocker
	}
	info, err := os.Stat(parsed.Path)
	if err != nil || info.Mode()&os.ModeSocket == 0 {
		return "", errors.New("local Docker endpoint is not an available Unix socket")
	}
	return parsed.String(), nil
}

func TestProjectionOwnedDockerRejectsRemoteOverrides(t *testing.T) {
	for _, key := range []string{"DOCKER_HOST", "DOCKER_CONTEXT"} {
		t.Run(key, func(t *testing.T) {
			t.Setenv("DOCKER_HOST", "")
			t.Setenv("DOCKER_CONTEXT", "")
			t.Setenv("PATH", "")
			t.Setenv(key, "remote-fixture-must-never-be-used")
			if _, err := projectionLocalDockerEndpoint(context.Background()); !errors.Is(err, errProjectionRemoteDocker) {
				t.Fatal("remote override was not rejected before invoking Docker")
			}
		})
	}
}

// This entry owns a fresh Redis instance. It never discovers or connects to a
// developer's prepared Redis service or inherits a supplied connection file.
func TestProjectionStoreOwnedRedisProtocol(t *testing.T) {
	if os.Getenv("CLOUDOPS_TEST_PROJECTION_STORE") != "true" {
		t.Skip("requires explicit owned projection Redis acceptance flag")
	}
	endpointContext, endpointCancel := context.WithTimeout(context.Background(), 10*time.Second)
	endpoint, endpointErr := projectionLocalDockerEndpoint(endpointContext)
	endpointCancel()
	if endpointErr != nil {
		t.Fatal(endpointErr)
	}
	privateDir := t.TempDir()
	if err := os.Chmod(privateDir, 0700); err != nil {
		t.Fatal("cannot secure owned fixture directory")
	}
	randomHex := func(size int) string {
		t.Helper()
		bytes := make([]byte, size)
		if _, err := rand.Read(bytes); err != nil {
			t.Fatal("cannot generate private fixture identity")
		}
		return hex.EncodeToString(bytes)
	}
	token, password := randomHex(24), randomHex(32)
	name := "cloudops-projection-owned-" + randomHex(8)
	const label = "com.cloudops.projection-store.owner"
	const image = "redis:7.4.11-alpine"
	const ownerKey = "cloudops:test:projection-store:owner"
	const username = "projection_test"
	command := func(args ...string) (string, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		out, err := exec.CommandContext(ctx, "docker", append([]string{"--host", endpoint}, args...)...).Output()
		return strings.TrimSpace(string(out)), err
	}
	writePrivate := func(name string, body []byte) string {
		t.Helper()
		path := filepath.Join(privateDir, name)
		if err := os.WriteFile(path, body, 0600); err != nil {
			t.Fatal("cannot write private owned fixture configuration")
		}
		return path
	}
	allowed := []string{"hello", "auth", "ping", "select", "quit", "client|setinfo", "client|setname", "get", "set", "del", "exists", "type", "hget", "hmget", "hset", "hgetall", "hexists", "smembers", "scard", "sadd", "srem", "eval", "evalsha", "script|load", "script|exists", "dump", "pttl", "ttl", "info"}
	acl := "user default off\nuser " + username + " on >" + password + " ~aiops:monitor:config:* ~monitor:webhook_file:* ~" + ownerKey
	for _, name := range allowed {
		acl += " +" + name
	}
	writePrivate("users.acl", []byte(acl+"\n"))
	writePrivate("redis.conf", []byte("bind 0.0.0.0\nport 6379\nprotected-mode yes\nsave \"\"\nappendonly no\naclfile /run/projection-acl/users.acl\ndir /data\n"))
	uid, gid := strconv.Itoa(os.Getuid()), strconv.Itoa(os.Getgid())
	t.Cleanup(func() {
		identity, err := command("inspect", "--format", `{{.Id}} {{index .Config.Labels "com.cloudops.projection-store.owner"}}`, name)
		if err != nil {
			remaining, probeErr := command("ps", "--all", "--filter", "label="+label+"="+token, "--format", "{{.ID}}")
			if probeErr != nil || remaining != "" {
				t.Error("could not verify cleanup after an uncertain owned Redis start")
			}
			return
		}
		fields := strings.Fields(identity)
		if len(fields) != 2 || len(fields[0]) != 64 || fields[1] != token {
			t.Error("owned Redis identity mismatch; cleanup refused")
			return
		}
		if _, err := command("rm", "--force", fields[0]); err != nil {
			t.Error("could not remove owned Redis instance")
			return
		}
		remaining, err := command("ps", "--all", "--filter", "label="+label+"="+token, "--format", "{{.ID}}")
		if err != nil || remaining != "" {
			t.Error("owned Redis cleanup not verified")
		}
	})
	container, err := command("run", "--detach", "--pull=never", "--name", name, "--label", label+"="+token,
		"--read-only", "--cap-drop=ALL", "--security-opt=no-new-privileges", "--user", uid+":"+gid,
		"--tmpfs", "/data:rw,nosuid,noexec,size=16m,uid="+uid+",gid="+gid+",mode=700", "--memory=128m", "--pids-limit=64",
		"--publish", "127.0.0.1::6379", "--mount", "type=bind,source="+privateDir+",target=/run/projection-acl,readonly",
		"--entrypoint", "redis-server", image, "/run/projection-acl/redis.conf")
	if err != nil {
		t.Fatal("could not start a fresh owned projection Redis instance")
	}
	if len(container) != 64 {
		t.Fatal("unexpected owned container identity")
	}

	portText, err := command("port", container, "6379/tcp")
	if err != nil {
		t.Fatal("could not resolve owned Redis loopback port")
	}
	host, port, err := net.SplitHostPort(portText)
	portNumber, parseErr := strconv.Atoi(port)
	if err != nil || parseErr != nil || host != "127.0.0.1" || portNumber <= 0 || portNumber > 65535 || portNumber == 16379 || portNumber == 6379 {
		t.Fatal("owned Redis must use a new dynamic loopback port")
	}
	client := redis.NewClient(&redis.Options{Addr: portText, Username: username, Password: password, MaxRetries: -1})
	t.Cleanup(func() { client.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for client.Ping(ctx).Err() != nil {
		select {
		case <-ticker.C:
		case <-ctx.Done():
			t.Fatal("owned Redis did not become ready")
		}
	}
	if client.Set(ctx, ownerKey, token, 0).Err() != nil {
		t.Fatal("cannot set owned Redis marker")
	}
	cfg := projectionRedisConnection{Address: portText, Username: username, Password: password, OwnerKey: ownerKey, OwnerToken: token}
	encoded, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal("cannot encode private fixture connection")
	}
	path := writePrivate("connection.json", encoded)
	t.Setenv("CLOUDOPS_PROJECTION_STORE_REDIS_CONNECTION_FILE", path)
	groups := []struct {
		name string
		run  func(*testing.T)
	}{
		{"protocol", TestProjectionStoreNativeRedis},
		{"queued_full_plan", TestProjectionStoreReviewQueuedFullPlanCannotRevertHigherPrepare},
		{"same_plan_sha", TestIndependentProjectionStoreLegacySHAIsEqualPayload},
		{"same_plan_binding", TestIndependentProjectionStoreEqualPlanCannotRedirectArtifactBinding},
		{"record_publishers", TestRecordProjectionNativeLatePacketsCannotDamageNewOwner},
		{"record_legacy_owner", TestRecordProjectionNativeLegacyOwnerCanFinishAfterReplacement},
		{"record_source_snapshot", TestRecordProjectionNativeOlderSourceCannotReserveAfterNewerPublication},
		{"record_same_hash_metadata", TestRecordProjectionNativeSameHashFindsNewDatabaseOnlyRetirement},
		{"alert_publishers", TestAlertRuleProjectionNativeLatePacketsCannotDamageNewOwner},
		{"alert_legacy_owner", TestAlertRuleProjectionNativeLegacyOwnerCanFinishAfterReplacement},
		{"alert_same_hash_metadata", TestAlertRuleProjectionNativeSameHashFindsNewDatabaseOnlyRetirement},
		{"alert_source_snapshot", TestAlertRuleProjectionNativeOlderSourceCannotReserveAfterNewerPublication},
		{"scrape_rule_family_isolation", TestScrapeRuleProjectionNativeFamiliesShareDatabaseWithoutCrossWrites},
		{"prom_publishers", TestPromMainProjectionNativeLatePacketsCannotDamageNewOwner},
		{"prom_legacy_owner", TestPromMainProjectionNativeLegacyOwnerCanFinishAfterReplacement},
		{"prom_same_hash_metadata", TestPromMainProjectionNativeSameHashFindsNewDatabaseOnlyRetirement},
		{"prom_source_snapshot", TestPromMainProjectionNativeOlderSourceCannotReplaceNewer},
		{"prom_credential_rotation", TestPromMainProjectionNativeCredentialRotationUsesOperationalBytes},
		{"prom_deleted_pool", TestPromMainProjectionNativeDeletedPoolRetiresWithoutJobRead},
		{"am_subset_isolation", TestAMProjectionNativeSubsetIsolationAndStableNoop},
		{"am_empty_groups", TestAMProjectionNativeEmptyGroupsRetireOnlyWebhook},
		{"am_unsaved_subset", TestAMProjectionNativeRejectsUnsavedSubsetPool},
		{"am_source_snapshot", TestAMProjectionNativeOlderSourceCannotReplacePeerPublication},
		{"am_failed_reservation", TestAMProjectionNativeFailedReservationDoesNotInvalidateLastGood},
		{"am_delayed_subset", TestAMProjectionNativeDelayedSubsetCannotRevertFullPublication},
		{"am_same_hash_metadata", TestAMProjectionNativeSameHashFindsDatabaseOnlyRetirement},
		{"am_unissued_ack", TestAMProjectionNativeCoverageRejectsUnissuedACKBytes},
		{"am_delayed_full", TestAMProjectionNativeDelayedFullCannotRevertSubset},
		{"am_rename_aba", TestAMProjectionNativeRepeatedRenameAndSecondABARecover},
		{"am_malformed_before_writes", TestAMProjectionNativeMalformedIntentFailsBeforeReplayWrites},
		{"recovery_scrape_restart", TestProjectionRecoveryNativeScrapeFamiliesRestart},
		{"recovery_am_peer", TestProjectionRecoveryNativeAMPeerConverges},
		{"recovery_poison_page", TestProjectionRecoveryNativePoisonDoesNotBlockLaterPool},
		{"scale_cost", TestProjectionStoreNativeScaleCost},
		{"batch_preflight", TestProjectionBatchNativeRejectsBeforeAnyArtifactWrite},
		{"batch_lost_reply", TestProjectionBatchNativeLostReplyRecoversWithoutRetry},
		{"batch_expired_lease", TestProjectionBatchNativeExpiredLeaseSendsNoEval},
		{"batch_independent_review", TestProjectionBatchIndependentNative},
		{"batch_raw_bytes", TestProjectionBatchNativeRawBodyBytesRemainExact},
	}
	for _, group := range groups {
		skipped := false
		passed := t.Run(group.name, func(child *testing.T) {
			defer func() { skipped = child.Skipped() }()
			group.run(child)
		})
		if skipped {
			t.Errorf("owned protocol group %s skipped despite prepared fixture", group.name)
		}
		if !passed {
			t.Errorf("owned protocol group %s failed", group.name)
		}
	}
}
