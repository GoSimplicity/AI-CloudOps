/*
 * MIT License
 *
 * Copyright (c) 2024 Bamboo
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 *
 */

package config

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	configDao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"gopkg.in/yaml.v3"
)

const previewFixture = "" +
	"global:\n" +
	"  scrape_interval: 17s\n" +
	"extra:\n" +
	"  key: fixture-visible-key\n" +
	"scrape_configs:\n" +
	"  - job_name: job-a\n" +
	"    authorization:\n" +
	"      type: Bearer\n" +
	"      credentials: fixture-job-a-token\n" +
	"      credentials_file: /etc/fixture/job-a.token\n" +
	"    basic_auth:\n" +
	"      username: fixture-user\n" +
	"      password: fixture-job-a-password\n" +
	"      password_file: /etc/fixture/job-a.password\n" +
	"    bearer_token: fixture-job-a-bearer\n" +
	"    bearer_token_file: /etc/fixture/job-a.bearer\n" +
	"    oauth2:\n" +
	"      client_id: fixture-client\n" +
	"      client_secret: fixture-job-a-oauth\n" +
	"      client_secret_file: /etc/fixture/job-a.oauth\n" +
	"      tls_config:\n" +
	"        key: fixture-job-a-oauth-tls\n" +
	"        key_file: /etc/fixture/job-a.oauth.tls\n" +
	"    tls_config:\n" +
	"      key: fixture-job-a-tls\n" +
	"      key_file: /etc/fixture/job-a.tls\n" +
	"    http_sd_configs:\n" +
	"      - url: http://fixture.example/sd-a\n" +
	"        authorization:\n" +
	"          type: Bearer\n" +
	"          credentials: fixture-sd-a-token\n" +
	"          credentials_file: /etc/fixture/sd-a.token\n" +
	"      - url: http://fixture.example/sd-b\n" +
	"        authorization:\n" +
	"          credentials: fixture-sd-b-token\n" +
	"  - job_name: job-b\n" +
	"    authorization:\n" +
	"      credentials: fixture-job-b-token\n" +
	"    http_sd_configs:\n" +
	"      - url: http://fixture.example/sd-c\n" +
	"        basic_auth:\n" +
	"          password: fixture-sd-c-password\n" +
	"        bearer_token: fixture-sd-c-bearer\n" +
	"        oauth2:\n" +
	"          client_secret: fixture-sd-c-oauth\n" +
	"        tls_config:\n" +
	"          key: fixture-sd-c-tls\n" +
	"          key_file: /etc/fixture/sd-c.tls\n"

var previewSecrets = []string{
	"fixture-job-a-token",
	"fixture-job-a-password",
	"fixture-job-a-bearer",
	"fixture-job-a-oauth",
	"fixture-job-a-oauth-tls",
	"fixture-job-a-tls",
	"fixture-sd-a-token",
	"fixture-sd-b-token",
	"fixture-job-b-token",
	"fixture-sd-c-password",
	"fixture-sd-c-bearer",
	"fixture-sd-c-oauth",
	"fixture-sd-c-tls",
}

var previewKept = []string{
	"fixture-visible-key",
	"fixture-user",
	"fixture-client",
	"scrape_interval: 17s",
	"/etc/fixture/job-a.token",
	"/etc/fixture/job-a.password",
	"/etc/fixture/job-a.bearer",
	"/etc/fixture/job-a.oauth",
	"/etc/fixture/job-a.oauth.tls",
	"/etc/fixture/job-a.tls",
	"/etc/fixture/sd-a.token",
	"/etc/fixture/sd-c.tls",
	"http://fixture.example/sd-a",
	"http://fixture.example/sd-b",
	"http://fixture.example/sd-c",
	"job-a",
	"job-b",
}

func TestPreviewRedactsCredentialFields(t *testing.T) {
	svc, store := newPreviewService(previewFixture, nil)
	detail, err := svc.GetMonitorConfigByID(context.Background(), &model.GetMonitorConfigReq{ID: 7})
	if err != nil {
		t.Fatal(err)
	}
	list, err := svc.GetMonitorConfigList(context.Background(), &model.GetMonitorConfigListReq{})
	if err != nil || len(list.Items) != 1 {
		t.Fatal(err)
	}
	for _, body := range []string{detail.ConfigContent, list.Items[0].ConfigContent} {
		assertRedacted(t, body)
	}
	if store.stored.ConfigContent != previewFixture || detail.ConfigHash != "hash-kept" || detail.Name != "old-name" {
		t.Fatal("preview mutated the stored config or dropped unrelated fields")
	}
	refs := credentialValues(t, detail.ConfigContent)
	if len(refs) != len(previewSecrets) {
		t.Fatalf("credential count = %d, want %d", len(refs), len(previewSecrets))
	}
	for id, value := range refs {
		if value != redactedCredential {
			t.Fatalf("%s remained %q", id, value)
		}
	}
}

func TestPreviewKeepsConfigWithoutCredentials(t *testing.T) {
	const original = "# comment-kept\nscrape_configs:\n  - job_name: plain\n    metrics_path: /metrics\n    authorization:\n      credentials_file: /etc/fixture/only-path\n    tls_config:\n      key_file: /etc/fixture/only.key\n"
	for _, content := range []string{original, "", " \n"} {
		got, err := redactYAML(content)
		if err != nil || got != content {
			t.Fatalf("content without credentials changed: %q -> %q (%v)", content, got, err)
		}
	}
	svc, store := newPreviewService(original, nil)
	detail, err := svc.GetMonitorConfigByID(context.Background(), &model.GetMonitorConfigReq{ID: 7})
	if err != nil || detail.ConfigContent != original || store.stored.ConfigContent != original {
		t.Fatal("plain config was rewritten")
	}
}

func TestPreviewHidesUnparsedContent(t *testing.T) {
	const broken = "scrape_configs: [\n  credentials: fixture-unparsed-secret\n"
	core, logs := observer.New(zap.ErrorLevel)
	svc, store := newPreviewService(broken, zap.New(core))
	plain := &model.MonitorConfig{Model: model.Model{ID: 8}, Name: "plain", ConfigContent: "global: {}\n", ConfigHash: "plain-hash"}
	store.extra = plain

	detail, err := svc.GetMonitorConfigByID(context.Background(), &model.GetMonitorConfigReq{ID: 7})
	if err != nil {
		t.Fatal(err)
	}
	if detail.ConfigContent != hiddenConfigContent || strings.Contains(detail.ConfigContent, "fixture-unparsed-secret") {
		t.Fatal("unparsed detail returned the original body")
	}
	list, err := svc.GetMonitorConfigList(context.Background(), &model.GetMonitorConfigListReq{})
	if err != nil || len(list.Items) != 2 {
		t.Fatal(err)
	}
	if list.Items[0].ConfigContent != hiddenConfigContent || list.Items[1].ConfigContent != plain.ConfigContent {
		t.Fatal("list did not hide only the unparsed row")
	}
	if store.stored.ConfigContent != broken {
		t.Fatal("hide path mutated the stored body")
	}
	entries := logs.All()
	if len(entries) != 2 {
		t.Fatalf("hidden rows logged %d times", len(entries))
	}
	for _, entry := range entries {
		blob := entry.Message + fmt.Sprint(entry.ContextMap())
		if entry.Message != "监控配置解析失败，已隐藏内容" || strings.Contains(blob, "fixture-unparsed-secret") || strings.Contains(blob, "scrape_configs") {
			t.Fatalf("parse log leaked content: %s", blob)
		}
	}
}

func TestPreviewRedactsMultilineCredentials(t *testing.T) {
	const original = "" +
		"scrape_configs:\n" +
		"  - job_name: job-ml\n" +
		"    basic_auth:\n" +
		"      password: |\n" +
		"        fixture-multiline-one\n" +
		"        fixture-multiline-two\n" +
		"    authorization:\n" +
		"      credentials: >\n" +
		"        fixture-folded-one\n" +
		"        fixture-folded-two\n" +
		"    tls_config:\n" +
		"      key: |-\n" +
		"        fixture-block-key\n" +
		"      key_file: /etc/fixture/multiline.key\n"
	got, err := redactYAML(original)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"fixture-multiline-one", "fixture-multiline-two", "fixture-folded-one", "fixture-folded-two", "fixture-block-key"} {
		if strings.Contains(got, secret) {
			t.Fatalf("multiline secret %s leaked", secret)
		}
	}
	if !strings.Contains(got, "/etc/fixture/multiline.key") {
		t.Fatal("multiline key_file was redacted")
	}
	for _, value := range credentialValues(t, got) {
		if value != redactedCredential {
			t.Fatalf("multiline credential = %q", value)
		}
	}
	svc, store := newPreviewService(original, nil)
	detail, err := svc.GetMonitorConfigByID(context.Background(), &model.GetMonitorConfigReq{ID: 7})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.UpdateMonitorConfig(context.Background(), &model.UpdateMonitorConfigReq{ID: 7, Name: "renamed", ConfigContent: detail.ConfigContent}); err != nil {
		t.Fatal(err)
	}
	if store.stored.ConfigContent != original {
		t.Fatal("multiline round trip changed the stored body")
	}
}

func TestPreviewRedactsAnchorsAndAliases(t *testing.T) {
	const original = "" +
		"shared: &outside fixture-outside-token\n" +
		"auth: &auth_block\n" +
		"  type: Bearer\n" +
		"  credentials: fixture-block-token\n" +
		"merged: &merged_auth\n" +
		"  credentials: fixture-merge-token\n" +
		"scrape_configs:\n" +
		"  - job_name: job-anchor\n" +
		"    authorization:\n" +
		"      credentials: &fixture_token fixture-alias-token\n" +
		"  - job_name: job-alias\n" +
		"    authorization:\n" +
		"      credentials: *fixture_token\n" +
		"    http_sd_configs:\n" +
		"      - url: http://fixture.example/alias-sd\n" +
		"        authorization:\n" +
		"          credentials: *fixture_token\n" +
		"  - job_name: job-block\n" +
		"    authorization: *auth_block\n" +
		"  - job_name: job-outside\n" +
		"    authorization:\n" +
		"      credentials: *outside\n" +
		"  - job_name: job-merge\n" +
		"    authorization:\n" +
		"      <<: *merged_auth\n"
	got, err := redactYAML(original)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"fixture-outside-token", "fixture-block-token", "fixture-merge-token", "fixture-alias-token"} {
		if strings.Contains(got, secret) {
			t.Fatalf("alias secret %s leaked", secret)
		}
	}
	var expanded any
	if err := yaml.Unmarshal([]byte(got), &expanded); err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"fixture-outside-token", "fixture-block-token", "fixture-merge-token", "fixture-alias-token"} {
		if valueContains(expanded, secret) {
			t.Fatalf("expanded alias kept %s", secret)
		}
	}
	for _, value := range credentialValues(t, got) {
		if value != redactedCredential {
			t.Fatalf("alias credential = %q", value)
		}
	}
}

func TestUpdateNameOnlyKeepsStoredConfigContent(t *testing.T) {
	svc, store := newPreviewService(previewFixture, nil)
	detail, err := svc.GetMonitorConfigByID(context.Background(), &model.GetMonitorConfigReq{ID: 7})
	if err != nil {
		t.Fatal(err)
	}
	assertRedacted(t, detail.ConfigContent)
	if store.stored.ConfigContent != previewFixture {
		t.Fatal("detail mutated the stored body before update")
	}
	err = svc.UpdateMonitorConfig(context.Background(), &model.UpdateMonitorConfigReq{
		ID: 7, Name: "renamed-only", ConfigContent: detail.ConfigContent,
	})
	if err != nil {
		t.Fatal(err)
	}
	if store.updates != 1 || store.stored.Name != "renamed-only" || store.stored.ConfigContent != previewFixture || store.stored.ConfigHash != "hash-kept" {
		t.Fatal("renaming did not keep the stored config bytes")
	}
	if strings.Contains(store.written.ConfigContent, redactedCredential) {
		t.Fatal("placeholder was written")
	}
}

func TestUpdatePreservesCredentialsWhenOtherFieldsChange(t *testing.T) {
	svc, store := newPreviewService(previewFixture, nil)
	detail, err := svc.GetMonitorConfigByID(context.Background(), &model.GetMonitorConfigReq{ID: 7})
	if err != nil {
		t.Fatal(err)
	}
	edited, err := replaceScalarValue(detail.ConfigContent, "17s", "19s")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.UpdateMonitorConfig(context.Background(), &model.UpdateMonitorConfigReq{ID: 7, Name: "old-name", ConfigContent: edited}); err != nil {
		t.Fatal(err)
	}
	saved := credentialValues(t, store.stored.ConfigContent)
	original := credentialValues(t, previewFixture)
	if len(saved) != len(original) {
		t.Fatalf("saved credentials = %d, original = %d", len(saved), len(original))
	}
	for id, value := range original {
		if saved[id] != value {
			t.Fatalf("%s changed from %s to %s", id, value, saved[id])
		}
	}
	if !strings.Contains(store.stored.ConfigContent, "19s") || strings.Contains(store.stored.ConfigContent, redactedCredential) {
		t.Fatal("non-credential edit was dropped or a placeholder remained")
	}
}

func TestUpdateReplacesExplicitCredential(t *testing.T) {
	svc, store := newPreviewService(previewFixture, nil)
	detail, err := svc.GetMonitorConfigByID(context.Background(), &model.GetMonitorConfigReq{ID: 7})
	if err != nil {
		t.Fatal(err)
	}
	doc, err := parseYAMLDocument(detail.ConfigContent)
	if err != nil {
		t.Fatal(err)
	}
	replaced := false
	for _, ref := range collectCredentials(doc) {
		if ref.job == "job-a" && !ref.inSD && ref.path == "authorization.credentials" {
			ref.node.Value = "fixture-job-a-new-token"
			replaced = true
		}
	}
	if !replaced {
		t.Fatal("job-a authorization credential missing from preview")
	}
	out, err := yaml.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.UpdateMonitorConfig(context.Background(), &model.UpdateMonitorConfigReq{ID: 7, Name: "old-name", ConfigContent: string(out)}); err != nil {
		t.Fatal(err)
	}
	saved := credentialValues(t, store.stored.ConfigContent)
	if saved[credID("job-a", false, "", "authorization.credentials")] != "fixture-job-a-new-token" {
		t.Fatal("explicit credential was not saved")
	}
	if saved[credID("job-a", true, "http://fixture.example/sd-a", "authorization.credentials")] != "fixture-sd-a-token" {
		t.Fatal("untouched http_sd credential was not restored")
	}
	if saved[credID("job-b", false, "", "authorization.credentials")] != "fixture-job-b-token" {
		t.Fatal("untouched scrape credential was not restored")
	}
	if strings.Contains(store.stored.ConfigContent, redactedCredential) {
		t.Fatal("placeholder was written")
	}
}

func TestUpdateRejectsNewJobPlaceholder(t *testing.T) {
	svc, store := newPreviewService(previewFixture, nil)
	detail, err := svc.GetMonitorConfigByID(context.Background(), &model.GetMonitorConfigReq{ID: 7})
	if err != nil {
		t.Fatal(err)
	}
	doc, err := parseYAMLDocument(detail.ConfigContent)
	if err != nil {
		t.Fatal(err)
	}
	if !appendScrapeJob(doc, "job-new") {
		t.Fatal("scrape_configs missing")
	}
	out, err := yaml.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	err = svc.UpdateMonitorConfig(context.Background(), &model.UpdateMonitorConfigReq{ID: 7, Name: "old-name", ConfigContent: string(out)})
	if !errors.Is(err, errCredentialPlaceholderUnmapped) || store.updates != 0 || store.stored.ConfigContent != previewFixture {
		t.Fatal("new job placeholder was accepted or the message changed")
	}
}

func TestUpdateRestoresHTTPServiceDiscoveryByURLOrIndex(t *testing.T) {
	const original = "" +
		"scrape_configs:\n" +
		"  - job_name: job-sd\n" +
		"    http_sd_configs:\n" +
		"      - url: http://fixture.example/sd-a\n" +
		"        authorization:\n" +
		"          credentials: fixture-sd-a-token\n" +
		"      - url: http://fixture.example/sd-b\n" +
		"        authorization:\n" +
		"          credentials: fixture-sd-b-token\n" +
		"      - authorization:\n" +
		"          credentials: fixture-index-token\n" +
		"        refresh_interval: 30s\n"
	svc, store := newPreviewService(original, nil)
	swapped := "" +
		"scrape_configs:\n" +
		"  - job_name: job-sd\n" +
		"    http_sd_configs:\n" +
		"      - url: http://fixture.example/sd-b\n" +
		"        authorization:\n" +
		"          credentials: <secret>\n" +
		"      - url: http://fixture.example/sd-a\n" +
		"        authorization:\n" +
		"          credentials: <secret>\n" +
		"      - authorization:\n" +
		"          credentials: <secret>\n" +
		"        refresh_interval: 45s\n"
	if err := svc.UpdateMonitorConfig(context.Background(), &model.UpdateMonitorConfigReq{ID: 7, Name: "old-name", ConfigContent: swapped}); err != nil {
		t.Fatal(err)
	}
	saved := credentialValues(t, store.stored.ConfigContent)
	if saved[credID("job-sd", true, "http://fixture.example/sd-a", "authorization.credentials")] != "fixture-sd-a-token" ||
		saved[credID("job-sd", true, "http://fixture.example/sd-b", "authorization.credentials")] != "fixture-sd-b-token" ||
		saved["job-sd|index:2|authorization.credentials"] != "fixture-index-token" {
		t.Fatalf("http_sd credentials rebound incorrectly: %#v", saved)
	}
	if !strings.Contains(store.stored.ConfigContent, "45s") || strings.Contains(store.stored.ConfigContent, redactedCredential) {
		t.Fatal("index-matched http_sd dropped the non-credential edit or kept a placeholder")
	}
}

func TestCreateRejectsCredentialPlaceholder(t *testing.T) {
	svc, store := newPreviewService(previewFixture, nil)
	err := svc.CreateMonitorConfig(context.Background(), &model.CreateMonitorConfigReq{
		Name: "created", PoolID: 1, InstanceIP: "127.0.0.1", ConfigType: model.ConfigTypePrometheus,
		ConfigContent: "scrape_configs:\n  - job_name: job-new\n    authorization:\n      credentials: <secret>\n",
	})
	if !errors.Is(err, errCredentialPlaceholderOnCreate) || store.creates != 0 {
		t.Fatal("create accepted a credential placeholder")
	}
	plain := "scrape_configs:\n  - job_name: plain\n    metrics_path: /metrics\n"
	if err := svc.CreateMonitorConfig(context.Background(), &model.CreateMonitorConfigReq{
		Name: "created", PoolID: 1, InstanceIP: "127.0.0.1", ConfigType: model.ConfigTypePrometheus, ConfigContent: plain,
	}); err != nil || store.creates != 1 || store.created.ConfigContent != plain {
		t.Fatal("create without a placeholder changed the body")
	}
}

func TestUpdateEmptyContentKeepsStoredBehavior(t *testing.T) {
	svc, store := newPreviewService(previewFixture, nil)
	if err := svc.UpdateMonitorConfig(context.Background(), &model.UpdateMonitorConfigReq{ID: 7, Name: "renamed"}); err != nil {
		t.Fatal(err)
	}
	if store.gets != 0 || store.updates != 1 || store.written.ConfigContent != "" || store.stored.ConfigContent != previewFixture || store.stored.Name != "renamed" {
		t.Fatal("empty config_content no longer keeps the stored body")
	}
}

func TestUpdateHiddenPreviewKeepsStoredContent(t *testing.T) {
	const broken = "scrape_configs: [\n  credentials: fixture-unparsed-secret\n"
	svc, store := newPreviewService(broken, zap.NewNop())
	detail, err := svc.GetMonitorConfigByID(context.Background(), &model.GetMonitorConfigReq{ID: 7})
	if err != nil || detail.ConfigContent != hiddenConfigContent {
		t.Fatal(err)
	}
	if err := svc.UpdateMonitorConfig(context.Background(), &model.UpdateMonitorConfigReq{ID: 7, Name: "renamed", ConfigContent: detail.ConfigContent}); err != nil {
		t.Fatal(err)
	}
	if store.stored.ConfigContent != broken || store.stored.Name != "renamed" || strings.Contains(store.written.ConfigContent, hiddenConfigContent) {
		t.Fatal("hidden preview replaced the stored body")
	}
}

func TestUpdateRejectsAmbiguousPlaceholder(t *testing.T) {
	const original = "" +
		"scrape_configs:\n" +
		"  - job_name: same\n" +
		"    authorization:\n" +
		"      credentials: fixture-one\n" +
		"  - job_name: same\n" +
		"    authorization:\n" +
		"      credentials: fixture-two\n"
	svc, store := newPreviewService(original, nil)
	edited := "" +
		"scrape_configs:\n" +
		"  - job_name: same\n" +
		"    honor_labels: true\n" +
		"    authorization:\n" +
		"      credentials: <secret>\n" +
		"  - job_name: same\n" +
		"    authorization:\n" +
		"      credentials: <secret>\n"
	err := svc.UpdateMonitorConfig(context.Background(), &model.UpdateMonitorConfigReq{ID: 7, Name: "old-name", ConfigContent: edited})
	if !errors.Is(err, errCredentialPlaceholderUnmapped) || store.updates != 0 || store.stored.ConfigContent != original {
		t.Fatal("ambiguous placeholder was written")
	}
}

type previewStore struct {
	configDao.MonitorConfigDAO
	stored  *model.MonitorConfig
	extra   *model.MonitorConfig
	written *model.MonitorConfig
	created *model.MonitorConfig
	gets    int
	updates int
	creates int
}

func newPreviewService(content string, logger *zap.Logger) (MonitorConfigService, *previewStore) {
	if logger == nil {
		logger = zap.NewNop()
	}
	store := &previewStore{stored: &model.MonitorConfig{
		Model: model.Model{ID: 7}, Name: "old-name", ConfigContent: content, ConfigHash: "hash-kept",
		ConfigType: model.ConfigTypePrometheus, InstanceIP: "127.0.0.1", PoolID: 1,
	}}
	return NewMonitorConfigService(logger, store), store
}

func (s *previewStore) GetMonitorConfigList(context.Context, *model.GetMonitorConfigListReq) ([]*model.MonitorConfig, int64, error) {
	rows := []*model.MonitorConfig{s.stored}
	if s.extra != nil {
		rows = append(rows, s.extra)
	}
	return rows, int64(len(rows)), nil
}

func (s *previewStore) GetMonitorConfigByID(context.Context, int) (*model.MonitorConfig, error) {
	s.gets++
	return s.stored, nil
}

func (s *previewStore) CreateMonitorConfig(_ context.Context, row *model.MonitorConfig) error {
	s.creates++
	copied := *row
	s.created = &copied
	return nil
}

func (s *previewStore) UpdateMonitorConfig(_ context.Context, row *model.MonitorConfig) error {
	s.updates++
	copied := *row
	s.written = &copied
	if row.Name != "" {
		s.stored.Name = row.Name
	}
	if row.ConfigContent != "" {
		s.stored.ConfigContent = row.ConfigContent
	}
	return nil
}

func assertRedacted(t *testing.T, body string) {
	t.Helper()
	for _, secret := range previewSecrets {
		if strings.Contains(body, secret) {
			t.Fatalf("preview leaked %s", secret)
		}
	}
	for _, kept := range previewKept {
		if !strings.Contains(body, kept) {
			t.Fatalf("preview dropped %s", kept)
		}
	}
	if strings.Count(body, redactedCredential) < len(previewSecrets) {
		t.Fatal("preview missed a credential placeholder")
	}
}

func credentialValues(t *testing.T, content string) map[string]string {
	t.Helper()
	doc, err := parseYAMLDocument(content)
	if err != nil {
		t.Fatal(err)
	}
	out := make(map[string]string)
	for _, ref := range collectCredentials(doc) {
		id := credID(ref.job, ref.inSD, ref.sdURL, ref.path)
		if ref.inSD && ref.sdURL == "" {
			id = fmt.Sprintf("%s|index:%d|%s", ref.job, ref.sdIndex, ref.path)
		}
		if _, ok := out[id]; ok && out[id] != ref.node.Value {
			t.Fatalf("duplicate credential %s", id)
		}
		out[id] = ref.node.Value
	}
	return out
}

func credID(job string, inSD bool, sdURL, path string) string {
	if !inSD {
		return job + "|-|" + path
	}
	if sdURL == "" {
		return job + "|index|" + path
	}
	return job + "|" + sdURL + "|" + path
}

func replaceScalarValue(content, old, newValue string) (string, error) {
	doc, err := parseYAMLDocument(content)
	if err != nil {
		return "", err
	}
	if !replaceValue(doc, old, newValue) {
		return "", fmt.Errorf("scalar %s not found", old)
	}
	out, err := yaml.Marshal(doc)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func replaceValue(n *yaml.Node, old, newValue string) bool {
	current := followAlias(n)
	if current == nil {
		return false
	}
	if current.Kind == yaml.ScalarNode && current.Value == old {
		current.Value = newValue
		return true
	}
	for _, child := range current.Content {
		if replaceValue(child, old, newValue) {
			return true
		}
	}
	return false
}

func appendScrapeJob(doc *yaml.Node, jobName string) bool {
	sequence := findSequence(doc, "scrape_configs")
	if sequence == nil {
		return false
	}
	job := &yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{
		{Kind: yaml.ScalarNode, Value: "job_name"},
		{Kind: yaml.ScalarNode, Value: jobName, Tag: "!!str"},
		{Kind: yaml.ScalarNode, Value: "authorization"},
		{Kind: yaml.MappingNode, Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "credentials"},
			{Kind: yaml.ScalarNode, Value: redactedCredential, Tag: "!!str"},
		}},
	}}
	sequence.Content = append(sequence.Content, job)
	return true
}

func findSequence(n *yaml.Node, key string) *yaml.Node {
	current := followAlias(n)
	if current == nil {
		return nil
	}
	if current.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(current.Content); i += 2 {
			if current.Content[i].Value == key {
				if sequence := followAlias(current.Content[i+1]); sequence != nil && sequence.Kind == yaml.SequenceNode {
					return sequence
				}
			}
			if found := findSequence(current.Content[i+1], key); found != nil {
				return found
			}
		}
	}
	for _, child := range current.Content {
		if found := findSequence(child, key); found != nil {
			return found
		}
	}
	return nil
}

func valueContains(value any, needle string) bool {
	switch typed := value.(type) {
	case string:
		return strings.Contains(typed, needle)
	case map[string]any:
		for key, child := range typed {
			if strings.Contains(key, needle) || valueContains(child, needle) {
				return true
			}
		}
	case []any:
		for _, child := range typed {
			if valueContains(child, needle) {
				return true
			}
		}
	}
	return false
}
