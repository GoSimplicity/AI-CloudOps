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

package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	configDao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
	configService "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/service/config"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func TestMonitorConfigCredentialHTTPContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const stored = "" +
		"scrape_configs:\n" +
		"  - job_name: job-a\n" +
		"    authorization:\n" +
		"      credentials: fixture-job-a-token\n" +
		"    http_sd_configs:\n" +
		"      - url: http://fixture.example/sd-a\n" +
		"        authorization:\n" +
		"          credentials: fixture-sd-a-token\n"
	store := &credentialConfigStore{row: &model.MonitorConfig{
		Model: model.Model{ID: 7}, Name: "old-name", ConfigContent: stored, ConfigHash: "hash-kept",
	}}
	engine := gin.New()
	NewMonitorConfigHandler(configService.NewMonitorConfigService(zap.NewNop(), store)).RegisterRouters(engine)

	detail := httptest.NewRecorder()
	engine.ServeHTTP(detail, httptest.NewRequest(http.MethodGet, "/api/monitor/configs/detail/7", nil))
	if detail.Code != http.StatusOK || strings.Contains(detail.Body.String(), "fixture-job-a-token") || strings.Contains(detail.Body.String(), "fixture-sd-a-token") {
		t.Fatal("detail response exposed a credential or changed the HTTP status")
	}
	var detailBody struct {
		Code    int             `json:"code"`
		Data    json.RawMessage `json:"data"`
		Message string          `json:"message"`
	}
	if err := json.Unmarshal(detail.Body.Bytes(), &detailBody); err != nil || detailBody.Code != 0 || detailBody.Message != "请求成功" {
		t.Fatal("detail envelope changed")
	}
	var config struct {
		Name          string `json:"name"`
		ConfigContent string `json:"config_content"`
		ConfigHash    string `json:"config_hash"`
	}
	if err := json.Unmarshal(detailBody.Data, &config); err != nil || config.Name != "old-name" || config.ConfigHash != "hash-kept" || !strings.Contains(config.ConfigContent, "<secret>") {
		t.Fatal("detail data no longer exposes config_content")
	}

	update := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/api/monitor/configs/update/7", strings.NewReader(`{"name":"renamed","config_content":"scrape_configs:\n  - job_name: job-new\n    authorization:\n      credentials: <secret>\n"}`))
	request.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(update, request)
	assertCredentialError(t, update, "凭据占位符无法对应原值，请填写真实凭据")
	if store.updates != 0 || store.row.ConfigContent != stored || strings.Contains(update.Body.String(), "fixture-job-a-token") {
		t.Fatal("rejected update wrote the row or echoed a stored credential")
	}

	create := httptest.NewRecorder()
	createReq := httptest.NewRequest(http.MethodPost, "/api/monitor/configs/create", strings.NewReader(`{"name":"created","pool_id":1,"instance_ip":"127.0.0.1","config_type":1,"config_content":"authorization:\n  credentials: <secret>\n"}`))
	createReq.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(create, createReq)
	assertCredentialError(t, create, "创建配置不能包含凭据占位符，请填写真实凭据")
	if store.updates != 0 {
		t.Fatal("create wrote a credential placeholder")
	}
}

func assertCredentialError(t *testing.T, response *httptest.ResponseRecorder, message string) {
	t.Helper()
	if response.Code != http.StatusOK {
		t.Fatalf("HTTP status = %d", response.Code)
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil || len(envelope) != 3 {
		t.Fatal("error response is not the code/message/data envelope")
	}
	var code int
	var data json.RawMessage
	var got string
	if err := json.Unmarshal(envelope["code"], &code); err != nil || code != 1 {
		t.Fatal("error code is not 1")
	}
	if err := json.Unmarshal(envelope["data"], &data); err != nil || string(data) != "{}" {
		t.Fatalf("error data = %s", data)
	}
	if err := json.Unmarshal(envelope["message"], &got); err != nil || got != message {
		t.Fatalf("error message = %s", got)
	}
}

type credentialConfigStore struct {
	configDao.MonitorConfigDAO
	row     *model.MonitorConfig
	updates int
}

func (s *credentialConfigStore) GetMonitorConfigByID(context.Context, int) (*model.MonitorConfig, error) {
	return s.row, nil
}

func (s *credentialConfigStore) UpdateMonitorConfig(context.Context, *model.MonitorConfig) error {
	s.updates++
	return nil
}

func (s *credentialConfigStore) CreateMonitorConfig(context.Context, *model.MonitorConfig) error {
	s.updates++
	return nil
}
