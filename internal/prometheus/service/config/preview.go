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
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"gopkg.in/yaml.v3"
)

const (
	redactedCredential  = "<secret>"
	hiddenConfigContent = "配置解析失败，已隐藏内容"
)

var (
	errCredentialPlaceholderUnmapped = errors.New("凭据占位符无法对应原值，请填写真实凭据")
	errCredentialPlaceholderOnCreate = errors.New("创建配置不能包含凭据占位符，请填写真实凭据")
	errHiddenConfigContent           = errors.New("配置内容为隐藏占位，请填写完整配置")
)

// redactConfigPreviews 只遮盖返回给查看接口的副本，不改调用方持有的原对象。
func redactConfigPreviews(list []*model.MonitorConfig) []*model.MonitorConfig {
	if list == nil {
		return nil
	}
	out := make([]*model.MonitorConfig, len(list))
	for i, item := range list {
		out[i] = redactConfigPreview(item)
	}
	return out
}

func redactConfigPreview(item *model.MonitorConfig) *model.MonitorConfig {
	if item == nil {
		return nil
	}
	copied := *item
	redacted, err := redactYAML(item.ConfigContent)
	if err != nil {
		copied.ConfigContent = hiddenConfigContent
		return &copied
	}
	copied.ConfigContent = redacted
	return &copied
}

func redactYAML(content string) (string, error) {
	if strings.TrimSpace(content) == "" {
		return content, nil
	}
	if err := singleYAMLDocument(content); err != nil {
		return "", err
	}
	doc, err := parseYAMLDocument(content)
	if err != nil {
		return "", err
	}
	changed := false
	for _, ref := range collectCredentials(doc) {
		if ref.node.Value == "" || ref.node.Value == redactedCredential {
			continue
		}
		ref.node.Value = redactedCredential
		ref.node.Tag = "!!str"
		ref.node.Style = 0
		changed = true
	}
	if !changed {
		return content, nil
	}
	out, err := yaml.Marshal(doc)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func rejectCreateCredentialPlaceholder(content string) error {
	if strings.Contains(content, redactedCredential) {
		return errCredentialPlaceholderOnCreate
	}
	return nil
}

func rejectHiddenConfigContent(content string) error {
	if strings.TrimSpace(content) == hiddenConfigContent {
		return errHiddenConfigContent
	}
	return nil
}

func updateNeedsStoredConfig(content string) bool {
	return strings.TrimSpace(content) == hiddenConfigContent || strings.Contains(content, redactedCredential)
}

// restoreSubmittedConfig 将查看结果里的占位符按原位置还原。
// 提交内容与脱敏结果一致，或正文是解析失败占位时，返回库存原文。
func restoreSubmittedConfig(stored, submitted string) (string, error) {
	if submitted == "" {
		return "", nil
	}
	if strings.TrimSpace(submitted) == hiddenConfigContent {
		return stored, nil
	}
	if !strings.Contains(submitted, redactedCredential) {
		return submitted, nil
	}
	if err := singleYAMLDocument(submitted); err != nil {
		return "", errCredentialPlaceholderUnmapped
	}
	if err := singleYAMLDocument(stored); err != nil {
		return "", errCredentialPlaceholderUnmapped
	}
	if redacted, err := redactYAML(stored); err == nil && submitted == redacted {
		if err := ensureCredentialPlaceholderAbsent(stored); err != nil {
			return "", err
		}
		return stored, nil
	}
	submittedDoc, err := parseYAMLDocument(submitted)
	if err != nil {
		return "", errCredentialPlaceholderUnmapped
	}
	storedDoc, err := parseYAMLDocument(stored)
	if err != nil {
		return "", errCredentialPlaceholderUnmapped
	}
	assigned := make(map[*yaml.Node]string)
	replaced := false
	storedRefs := collectCredentials(storedDoc)
	for _, ref := range collectCredentials(submittedDoc) {
		if ref.node.Value != redactedCredential {
			continue
		}
		if !destinationsMatch(storedDoc, ref) {
			return "", errCredentialPlaceholderUnmapped
		}
		value, ok := lookupCredential(storedRefs, ref)
		if !ok || value == redactedCredential {
			return "", errCredentialPlaceholderUnmapped
		}
		if previous, seen := assigned[ref.node]; seen && previous != value {
			return "", errCredentialPlaceholderUnmapped
		}
		assigned[ref.node] = value
		ref.node.Value = value
		ref.node.Tag = "!!str"
		ref.node.Style = 0
		replaced = true
	}
	if !replaced {
		return submitted, nil
	}
	out, err := yaml.Marshal(submittedDoc)
	if err != nil {
		return "", errCredentialPlaceholderUnmapped
	}
	if err := ensureCredentialPlaceholderAbsent(string(out)); err != nil {
		return "", err
	}
	return string(out), nil
}

func ensureCredentialPlaceholderAbsent(content string) error {
	if !strings.Contains(content, redactedCredential) {
		return nil
	}
	doc, err := parseYAMLDocument(content)
	if err != nil {
		return errCredentialPlaceholderUnmapped
	}
	for _, ref := range collectCredentials(doc) {
		if ref.node.Value == redactedCredential {
			return errCredentialPlaceholderUnmapped
		}
	}
	return nil
}

func parseYAMLDocument(content string) (*yaml.Node, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(content), &doc); err != nil {
		return nil, err
	}
	return &doc, nil
}

func singleYAMLDocument(content string) error {
	decoder := yaml.NewDecoder(strings.NewReader(content))
	count := 0
	for {
		var doc yaml.Node
		err := decoder.Decode(&doc)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		count++
		if count > 1 {
			return errors.New("多个 YAML 文档")
		}
	}
	return nil
}

type credentialPos struct {
	job        string
	inJob      bool
	inSD       bool
	sdIndex    int
	sdURL      string
	parent     string
	container  string
	path       []string
	scopeKind  string
	scopeIndex int
	scope      *yaml.Node
}

type credentialRef struct {
	job        string
	inJob      bool
	inSD       bool
	sdIndex    int
	sdURL      string
	path       string
	node       *yaml.Node
	scopeKind  string
	scopeIndex int
	scope      *yaml.Node
}

func collectCredentials(root *yaml.Node) []credentialRef {
	var refs []credentialRef
	walkYAML(root, credentialPos{sdIndex: -1}, nil, &refs)
	return refs
}

func walkYAML(n *yaml.Node, pos credentialPos, stack []*yaml.Node, refs *[]credentialRef) {
	if n == nil || nodeOnStack(stack, n) {
		return
	}
	stack = appendNode(stack, n)
	current := n
	if n.Kind == yaml.AliasNode {
		if n.Alias == nil || nodeOnStack(stack, n.Alias) {
			return
		}
		current = n.Alias
		stack = appendNode(stack, current)
	}
	switch current.Kind {
	case yaml.DocumentNode:
		for _, child := range current.Content {
			walkYAML(child, pos, stack, refs)
		}
	case yaml.SequenceNode:
		walkSequence(current, pos, stack, refs)
	case yaml.MappingNode:
		walkMapping(current, pos, stack, refs)
	}
}

func walkSequence(current *yaml.Node, pos credentialPos, stack []*yaml.Node, refs *[]credentialRef) {
	switch pos.parent {
	case "scrape_configs":
		for _, child := range current.Content {
			resolved := followAlias(child)
			next := credentialPos{job: mappingValue(child, "job_name"), inJob: true, sdIndex: -1, scopeKind: "job", scopeIndex: -1, scope: resolved}
			walkYAML(child, next, stack, refs)
		}
	case "receivers":
		for _, child := range current.Content {
			resolved := followAlias(child)
			next := credentialPos{job: mappingValue(child, "name"), inJob: true, sdIndex: -1, scopeKind: "receiver", scopeIndex: -1, scope: resolved}
			walkYAML(child, next, stack, refs)
		}
	case "remote_write", "remote_read":
		for i, child := range current.Content {
			next := pos
			next.scopeKind = pos.parent
			next.scopeIndex = i
			next.scope = followAlias(child)
			next.inJob = false
			next.inSD = false
			next.sdIndex = -1
			next.path = appendIndex(pos.path, i)
			next.parent = ""
			next.container = pos.parent
			walkYAML(child, next, stack, refs)
		}
	case "http_sd_configs":
		for i, child := range current.Content {
			next := pos
			next.inSD = true
			next.sdIndex = i
			next.sdURL = mappingValue(child, "url")
			next.path = nil
			next.parent = ""
			next.container = "http_sd_configs"
			walkYAML(child, next, stack, refs)
		}
	default:
		for i, child := range current.Content {
			next := pos
			next.path = appendIndex(pos.path, i)
			next.parent = ""
			next.container = pos.parent
			walkYAML(child, next, stack, refs)
		}
	}
}

func walkMapping(current *yaml.Node, pos credentialPos, stack []*yaml.Node, refs *[]credentialRef) {
	for i := 0; i+1 < len(current.Content); i += 2 {
		keyNode, valueNode := current.Content[i], current.Content[i+1]
		key := keyNode.Value
		if key == "<<" {
			walkMerge(valueNode, pos, stack, refs)
			continue
		}
		if credentialField(pos.parent, key, pos.container) {
			if scalar := resolveScalar(valueNode); scalar != nil {
				segments := append(append([]string{}, pos.path...), key)
				*refs = append(*refs, credentialRef{
					job: pos.job, inJob: pos.inJob, inSD: pos.inSD,
					sdIndex: pos.sdIndex, sdURL: pos.sdURL,
					path: strings.Join(segments, "."), node: scalar,
					scopeKind: pos.scopeKind, scopeIndex: pos.scopeIndex, scope: pos.scope,
				})
			}
		}
		next := pos
		next.parent = key
		next.path = append(append([]string{}, pos.path...), key)
		if key == "global" && pos.scopeKind == "" {
			next.scopeKind = "global"
			next.scopeIndex = -1
			next.scope = followAlias(valueNode)
		}
		walkYAML(valueNode, next, stack, refs)
	}
}

func walkMerge(n *yaml.Node, pos credentialPos, stack []*yaml.Node, refs *[]credentialRef) {
	resolved := followAlias(n)
	if resolved != nil && resolved.Kind == yaml.SequenceNode {
		for _, child := range resolved.Content {
			walkYAML(child, pos, stack, refs)
		}
		return
	}
	walkYAML(n, pos, stack, refs)
}

func lookupCredential(stored []credentialRef, want credentialRef) (string, bool) {
	var matches []credentialRef
	for _, item := range stored {
		if sameCredentialPlace(item, want) {
			matches = append(matches, item)
		}
	}
	if len(matches) == 0 {
		return "", false
	}
	value := matches[0].node.Value
	for _, item := range matches[1:] {
		if item.node.Value != value {
			return "", false
		}
	}
	return value, true
}

func sameCredentialPlace(item, want credentialRef) bool {
	if item.path != want.path || item.inJob != want.inJob || item.job != want.job || item.inSD != want.inSD {
		return false
	}
	if !want.inSD {
		return true
	}
	if want.sdURL != "" && item.sdURL == want.sdURL {
		return true
	}
	return want.sdURL == "" && item.sdIndex == want.sdIndex
}

func credentialField(parent, key, container string) bool {
	switch key {
	case "bearer_token", "smtp_auth_password", "smtp_auth_secret", "slack_api_url",
		"wechat_api_secret", "opsgenie_api_key", "victorops_api_key", "routing_key",
		"service_key", "bot_token":
		return true
	}
	if strings.HasSuffix(key, "_api_url") {
		return true
	}
	switch {
	case parent == "authorization" && key == "credentials":
		return true
	case parent == "basic_auth" && key == "password":
		return true
	case parent == "oauth2" && key == "client_secret":
		return true
	case parent == "tls_config" && key == "key":
		return true
	case key == "url" && (container == "webhook_configs" || parent == "webhook_configs"):
		return true
	case key == "api_url" && secretAPIURLContainer(container):
		return true
	case key == "api_secret" && container == "wechat_configs":
		return true
	case key == "api_key" && (container == "opsgenie_configs" || container == "victorops_configs" || container == "pagerduty_configs"):
		return true
	case (key == "auth_password" || key == "auth_secret") && container == "email_configs":
		return true
	default:
		return false
	}
}

func secretAPIURLContainer(container string) bool {
	switch container {
	case "slack_configs", "opsgenie_configs", "wechat_configs", "pagerduty_configs",
		"telegram_configs", "victorops_configs", "msteams_configs", "sns_configs",
		"discord_configs", "webex_configs", "rocketchat_configs", "pushover_configs":
		return true
	default:
		return false
	}
}

func destinationsMatch(storedRoot *yaml.Node, want credentialRef) bool {
	if want.scopeKind == "" {
		return true
	}
	storedScope, ok := findScope(storedRoot, want)
	if !ok || want.scope == nil || storedScope == nil {
		return false
	}
	return mappingDestinationEqual(want.scope, storedScope)
}

func findScope(root *yaml.Node, want credentialRef) (*yaml.Node, bool) {
	switch want.scopeKind {
	case "job":
		return uniqueNamedItem(sequenceByKey(root, "scrape_configs"), "job_name", want.job)
	case "receiver":
		return uniqueNamedItem(sequenceByKey(root, "receivers"), "name", want.job)
	case "remote_write", "remote_read":
		items := sequenceByKey(root, want.scopeKind)
		if want.scopeIndex < 0 || want.scopeIndex >= len(items) {
			return nil, false
		}
		return followAlias(items[want.scopeIndex]), true
	case "global":
		node := mappingByKey(root, "global")
		return node, node != nil
	default:
		return nil, false
	}
}

func uniqueNamedItem(items []*yaml.Node, key, name string) (*yaml.Node, bool) {
	var found *yaml.Node
	for _, item := range items {
		if mappingValue(item, key) != name {
			continue
		}
		if found != nil {
			return nil, false
		}
		found = followAlias(item)
	}
	return found, found != nil
}

func sequenceByKey(root *yaml.Node, key string) []*yaml.Node {
	node := mappingByKey(root, key)
	if node == nil || node.Kind != yaml.SequenceNode {
		return nil
	}
	return node.Content
}

func mappingByKey(root *yaml.Node, key string) *yaml.Node {
	mapping := documentMapping(root)
	if mapping == nil {
		return nil
	}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return followAlias(mapping.Content[i+1])
		}
	}
	return nil
}

func documentMapping(root *yaml.Node) *yaml.Node {
	node := followAlias(root)
	if node != nil && node.Kind == yaml.DocumentNode && len(node.Content) == 1 {
		node = followAlias(node.Content[0])
	}
	if node != nil && node.Kind == yaml.MappingNode {
		return node
	}
	return nil
}

func mappingDestinationEqual(submitted, stored *yaml.Node) bool {
	return mappingDestinationEqualIn(submitted, stored, "")
}

func mappingDestinationEqualIn(submitted, stored *yaml.Node, container string) bool {
	submitted, stored = followAlias(submitted), followAlias(stored)
	if submitted == nil || stored == nil || submitted.Kind != yaml.MappingNode || stored.Kind != yaml.MappingNode {
		return submitted == nil && stored == nil
	}
	left := destinationPairs(submitted)
	right := destinationPairs(stored)
	if len(left) != len(right) {
		return false
	}
	for key, value := range left {
		other, ok := right[key]
		if !ok || !destinationValueEqual(key, value, other, container) {
			return false
		}
	}
	return true
}

func destinationPairs(node *yaml.Node) map[string]*yaml.Node {
	pairs := make(map[string]*yaml.Node)
	if node == nil || node.Kind != yaml.MappingNode {
		return pairs
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		key := node.Content[i].Value
		if isDestinationKey(key) || shouldDescendDestination(key) {
			pairs[key] = node.Content[i+1]
		}
	}
	return pairs
}

func destinationValueEqual(key string, submitted, stored *yaml.Node, container string) bool {
	if key == "oauth2" {
		return mappingValue(submitted, "token_url") == mappingValue(stored, "token_url")
	}
	if shouldDescendDestination(key) {
		return descendedDestinationEqual(submitted, stored, key)
	}
	return treeEqualValue(submitted, stored, "", key, container)
}

func descendedDestinationEqual(submitted, stored *yaml.Node, container string) bool {
	submitted, stored = followAlias(submitted), followAlias(stored)
	if submitted == nil || stored == nil || submitted.Kind != stored.Kind {
		return false
	}
	if submitted.Kind == yaml.SequenceNode {
		if len(submitted.Content) != len(stored.Content) {
			return false
		}
		for i := range submitted.Content {
			if !mappingDestinationEqualIn(submitted.Content[i], stored.Content[i], container) {
				return false
			}
		}
		return true
	}
	return mappingDestinationEqualIn(submitted, stored, container)
}

func treeEqualValue(submitted, stored *yaml.Node, parent, key, container string) bool {
	submitted, stored = followAlias(submitted), followAlias(stored)
	if submitted == nil || stored == nil {
		return submitted == nil && stored == nil
	}
	if submitted.Kind != stored.Kind {
		return false
	}
	switch submitted.Kind {
	case yaml.ScalarNode:
		if credentialField(parent, key, container) && submitted.Value == redactedCredential {
			return true
		}
		return submitted.Value == stored.Value
	case yaml.SequenceNode:
		if len(submitted.Content) != len(stored.Content) {
			return false
		}
		for i := range submitted.Content {
			if !treeEqualNode(submitted.Content[i], stored.Content[i], key, container) {
				return false
			}
		}
		return true
	case yaml.MappingNode:
		return mappingTreeEqual(submitted, stored, key, container)
	default:
		return false
	}
}

func treeEqualNode(submitted, stored *yaml.Node, containerKey, container string) bool {
	submitted, stored = followAlias(submitted), followAlias(stored)
	if submitted == nil || stored == nil || submitted.Kind != stored.Kind {
		return submitted == nil && stored == nil
	}
	if submitted.Kind == yaml.MappingNode {
		nextContainer := container
		if containerKey != "" {
			nextContainer = containerKey
		}
		return mappingTreeEqual(submitted, stored, "", nextContainer)
	}
	return treeEqualValue(submitted, stored, "", "", container)
}

func mappingTreeEqual(submitted, stored *yaml.Node, parent, container string) bool {
	left := mappingPairs(submitted)
	right := mappingPairs(stored)
	if len(left) != len(right) {
		return false
	}
	for key, value := range left {
		other, ok := right[key]
		if !ok || !treeEqualValue(value, other, parent, key, container) {
			return false
		}
	}
	return true
}

func mappingPairs(node *yaml.Node) map[string]*yaml.Node {
	pairs := make(map[string]*yaml.Node)
	if node == nil || node.Kind != yaml.MappingNode {
		return pairs
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		pairs[node.Content[i].Value] = node.Content[i+1]
	}
	return pairs
}

func isDestinationKey(key string) bool {
	if strings.HasSuffix(key, "_sd_configs") || strings.HasSuffix(key, "_api_url") {
		return true
	}
	switch key {
	case "targets", "labels", "static_configs", "scheme", "proxy_url", "proxy_from_environment",
		"relabel_configs", "url", "api_url", "smtp_smarthost", "smarthost", "to", "host",
		"api_server", "role", "namespaces", "oauth2":
		return true
	default:
		return false
	}
}

func shouldDescendDestination(key string) bool {
	switch key {
	case "http_config", "webhook_configs", "slack_configs", "email_configs", "pagerduty_configs",
		"opsgenie_configs", "wechat_configs", "victorops_configs", "telegram_configs",
		"pushover_configs", "sns_configs", "msteams_configs", "discord_configs",
		"webex_configs", "rocketchat_configs":
		return true
	default:
		return false
	}
}

func mappingValue(n *yaml.Node, key string) string {
	current := followAlias(n)
	if current == nil || current.Kind != yaml.MappingNode {
		return ""
	}
	for i := 0; i+1 < len(current.Content); i += 2 {
		if current.Content[i].Value == key {
			if scalar := resolveScalar(current.Content[i+1]); scalar != nil {
				return scalar.Value
			}
		}
	}
	return ""
}

func resolveScalar(n *yaml.Node) *yaml.Node {
	current := followAlias(n)
	if current == nil || current.Kind != yaml.ScalarNode {
		return nil
	}
	return current
}

func followAlias(n *yaml.Node) *yaml.Node {
	seen := make(map[*yaml.Node]struct{})
	for n != nil && n.Kind == yaml.AliasNode {
		if _, ok := seen[n]; ok {
			return nil
		}
		seen[n] = struct{}{}
		n = n.Alias
	}
	return n
}

func appendIndex(path []string, index int) []string {
	suffix := fmt.Sprintf("[%d]", index)
	if len(path) == 0 {
		return []string{suffix}
	}
	out := append([]string{}, path...)
	out[len(out)-1] += suffix
	return out
}

func nodeOnStack(stack []*yaml.Node, n *yaml.Node) bool {
	for _, item := range stack {
		if item == n {
			return true
		}
	}
	return false
}

func appendNode(stack []*yaml.Node, n *yaml.Node) []*yaml.Node {
	next := make([]*yaml.Node, len(stack)+1)
	copy(next, stack)
	next[len(stack)] = n
	return next
}
