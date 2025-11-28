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

package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

type promHashPayload struct {
	Name                  string   `json:"name"`
	PrometheusInstances   []string `json:"prometheus_instances"`
	AlertManagerInstances []string `json:"alert_manager_instances"`
	ScrapeInterval        int      `json:"scrape_interval"`
	ScrapeTimeout         int      `json:"scrape_timeout"`
	RemoteTimeoutSeconds  int      `json:"remote_timeout_seconds"`
	SupportAlert          int8     `json:"support_alert"`
	SupportRecord         int8     `json:"support_record"`
	ExternalLabels        []string `json:"external_labels"`
	RemoteWriteUrl        string   `json:"remote_write_url"`
	RemoteReadUrl         string   `json:"remote_read_url"`
	AlertManagerUrl       string   `json:"alert_manager_url"`
	RecordFilePath        string   `json:"record_file_path"`
}

// CalculatePromHash 计算哈希值（不修改入参数据）
func CalculatePromHash(pool *model.MonitorScrapePool) string {
	instancesCopy := append([]string(nil), pool.PrometheusInstances...)
	labelsCopy := append([]string(nil), pool.Tags...)
	sort.Strings(instancesCopy)
	sort.Strings(labelsCopy)

	payload := promHashPayload{
		Name:                 pool.Name,
		PrometheusInstances:  instancesCopy,
		ScrapeInterval:       pool.ScrapeInterval,
		ScrapeTimeout:        pool.ScrapeTimeout,
		RemoteTimeoutSeconds: pool.RemoteTimeoutSeconds,
		SupportAlert:         pool.SupportAlert,
		SupportRecord:        pool.SupportRecord,
		ExternalLabels:       labelsCopy,
		RemoteWriteUrl:       pool.RemoteWriteUrl,
		RemoteReadUrl:        pool.RemoteReadUrl,
		AlertManagerUrl:      pool.AlertManagerUrl,
		RecordFilePath:       pool.RecordFilePath,
	}

	data, _ := json.Marshal(payload)

	hash := sha256.New()
	hash.Write(data)
	return hex.EncodeToString(hash.Sum(nil))
}

type alertHashPayload struct {
	Name                  string   `json:"name"`
	AlertManagerInstances []string `json:"alert_manager_instances"`
	ResolveTimeout        string   `json:"resolve_timeout"`
	GroupWait             string   `json:"group_wait"`
	GroupInterval         string   `json:"group_interval"`
	RepeatInterval        string   `json:"repeat_interval"`
	GroupBy               []string `json:"group_by"`
	Receiver              string   `json:"receiver"`
}

func CalculateAlertHash(pool *model.MonitorAlertManagerPool) string {
	instancesCopy := append([]string(nil), pool.AlertManagerInstances...)
	groupByCopy := append([]string(nil), pool.GroupBy...)
	sort.Strings(instancesCopy)
	sort.Strings(groupByCopy)

	payload := alertHashPayload{
		Name:                  pool.Name,
		AlertManagerInstances: instancesCopy,
		ResolveTimeout:        pool.ResolveTimeout,
		GroupWait:             pool.GroupWait,
		GroupInterval:         pool.GroupInterval,
		RepeatInterval:        pool.RepeatInterval,
		GroupBy:               groupByCopy,
		Receiver:              pool.Receiver,
	}

	data, _ := json.Marshal(payload)

	hash := sha256.New()
	hash.Write(data)
	return hex.EncodeToString(hash.Sum(nil))
}
