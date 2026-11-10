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

func updateNeedsStoredConfig(content string) bool {
	return content == hiddenConfigContent || strings.Contains(content, redactedCredential)
}

// restoreSubmittedConfig 将查看结果里的占位符按原位置还原。
// 提交内容与脱敏结果一致时返回库存原文，避免只改名称时重写 YAML。
func restoreSubmittedConfig(stored, submitted string) (string, error) {
	if submitted == "" {
		return "", nil
	}
	if submitted == hiddenConfigContent {
		return stored, nil
	}
	if !strings.Contains(submitted, redactedCredential) {
		return submitted, nil
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
	for _, ref := range collectCredentials(submittedDoc) {
		if ref.node.Value != redactedCredential {
			continue
		}
		value, ok := lookupCredential(collectCredentials(storedDoc), ref)
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

type credentialPos struct {
	job     string
	inJob   bool
	inSD    bool
	sdIndex int
	sdURL   string
	parent  string
	path    []string
}

type credentialRef struct {
	job     string
	inJob   bool
	inSD    bool
	sdIndex int
	sdURL   string
	path    string
	node    *yaml.Node
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
			next := credentialPos{job: mappingValue(child, "job_name"), inJob: true, sdIndex: -1}
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
			walkYAML(child, next, stack, refs)
		}
	default:
		for i, child := range current.Content {
			next := pos
			next.path = appendIndex(pos.path, i)
			next.parent = ""
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
		if credentialField(pos.parent, key) {
			if scalar := resolveScalar(valueNode); scalar != nil {
				segments := append(append([]string{}, pos.path...), key)
				*refs = append(*refs, credentialRef{
					job: pos.job, inJob: pos.inJob, inSD: pos.inSD,
					sdIndex: pos.sdIndex, sdURL: pos.sdURL,
					path: strings.Join(segments, "."), node: scalar,
				})
			}
		}
		next := pos
		next.parent = key
		next.path = append(append([]string{}, pos.path...), key)
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

func credentialField(parent, key string) bool {
	switch {
	case parent == "authorization" && key == "credentials":
		return true
	case parent == "basic_auth" && key == "password":
		return true
	case parent == "oauth2" && key == "client_secret":
		return true
	case parent == "tls_config" && key == "key":
		return true
	case key == "bearer_token":
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
