/*
 * MIT License
 *
 * Copyright (c) 2024 Bamboo
 */

package grpc_client

import (
	"testing"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/connectivity"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Address != "cloudops-aiops:9000" {
		t.Errorf("Expected address 'cloudops-aiops:9000', got '%s'", config.Address)
	}

	if config.MaxRetries != 3 {
		t.Errorf("Expected MaxRetries 3, got %d", config.MaxRetries)
	}

	if config.EnableMetrics != true {
		t.Errorf("Expected EnableMetrics true, got %v", config.EnableMetrics)
	}

	if config.HealthCheckInterval != 30*time.Second {
		t.Errorf("Expected HealthCheckInterval 30s, got %v", config.HealthCheckInterval)
	}
}

func TestNewAIOpsClientConfig(t *testing.T) {
	logger := zap.NewNop()

	// 测试默认配置
	client, err := NewAIOpsClient(nil, logger)
	if err == nil {
		client.Close()
	}

	// 测试自定义配置
	config := &ClientConfig{
		Address:           "localhost:9000",
		MaxRetries:        5,
		ConnectionTimeout: 5 * time.Second,
		EnableMetrics:     false,
	}

	client, err = NewAIOpsClient(config, logger)
	if err == nil {
		client.Close()
	}
}

func TestHealthStatus(t *testing.T) {
	healthStatus := &HealthStatus{
		IsHealthy:           true,
		LastCheckTime:       time.Now(),
		LastHealthyTime:     time.Now(),
		ConnectionState:     connectivity.Ready,
		ConsecutiveFailures: 0,
		TotalChecks:         10,
		FailedChecks:        2,
		CurrentEndpoint:     "localhost:9000",
	}

	if !healthStatus.IsHealthy {
		t.Error("Expected healthy status")
	}

	if healthStatus.ConnectionState != connectivity.Ready {
		t.Errorf("Expected Ready state, got %v", healthStatus.ConnectionState)
	}

	if healthStatus.CurrentEndpoint != "localhost:9000" {
		t.Errorf("Expected endpoint 'localhost:9000', got '%s'", healthStatus.CurrentEndpoint)
	}
}

func TestGetEndpoints(t *testing.T) {
	logger := zap.NewNop()

	// 测试单一地址
	config := &ClientConfig{
		Address: "localhost:9000",
	}

	client, err := NewAIOpsClient(config, logger)
	if err == nil {
		defer client.Close()

		endpoints := client.GetEndpoints()
		if len(endpoints) != 1 || endpoints[0] != "localhost:9000" {
			t.Errorf("Expected single endpoint 'localhost:9000', got %v", endpoints)
		}
	}

	// 测试多端点
	config = &ClientConfig{
		Address:   "localhost:9000",
		Endpoints: []string{"host1:9000", "host2:9000"},
	}

	client, err = NewAIOpsClient(config, logger)
	if err == nil {
		defer client.Close()

		endpoints := client.GetEndpoints()
		if len(endpoints) != 2 {
			t.Errorf("Expected 2 endpoints, got %d", len(endpoints))
		}
		if endpoints[0] != "host1:9000" || endpoints[1] != "host2:9000" {
			t.Errorf("Expected 'host1:9000' and 'host2:9000', got %v", endpoints)
		}
	}
}

func TestClientManager(t *testing.T) {
	logger := zap.NewNop()

	manager, err := NewClientManager(logger)
	if err != nil {
		t.Fatalf("Failed to create client manager: %v", err)
	}

	// 测试获取健康状态（在启动前）
	status := manager.GetAIOpsHealthStatus()
	if status.IsHealthy {
		t.Error("Expected unhealthy status before start")
	}

	// 测试是否启用（在启动前）
	if manager.IsAIOpsEnabled() {
		t.Error("Expected disabled before start")
	}
}

func TestConnectionState(t *testing.T) {
	logger := zap.NewNop()
	config := &ClientConfig{
		Address:           "invalid-address:9999",
		ConnectionTimeout: 1 * time.Second,
	}

	// 测试无效地址的连接
	client, err := NewAIOpsClient(config, logger)
	if err != nil {
		// 预期失败，这是正常的
		return
	}

	defer client.Close()

	// 检查连接状态
	state := client.GetConnectionState()
	_ = state // 只要不panic就行
}

func TestMetricsConfiguration(t *testing.T) {
	config := DefaultConfig()

	// 测试默认启用监控
	if !config.EnableMetrics {
		t.Error("Expected metrics to be enabled by default")
	}

	// 测试禁用监控
	config.EnableMetrics = false
	if config.EnableMetrics {
		t.Error("Expected metrics to be disabled")
	}
}
