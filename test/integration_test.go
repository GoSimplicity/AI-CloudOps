/*
 * MIT License
 *
 * Copyright (c) 2024 Bamboo
 */

package test

import (
	"context"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/pkg/grpc_client"
	aiopsv1 "github.com/GoSimplicity/AI-CloudOps/proto/aiops/v1"
	"go.uber.org/zap"
)

// TestGrpcClientIntegration 测试gRPC客户端集成
func TestGrpcClientIntegration(t *testing.T) {
	logger := zap.NewNop()

	// 使用无效地址测试连接失败场景
	config := &grpc_client.ClientConfig{
		Address:           "localhost:9999", // 无效地址
		ConnectionTimeout: 2 * time.Second,
		EnableMetrics:     true,
	}

	client, err := grpc_client.NewAIOpsClient(config, logger)
	if err != nil {
		// 预期会失败，这是正常的
		t.Logf("Expected connection failure: %v", err)
		return
	}
	defer client.Close()

	// 测试基本方法
	endpoints := client.GetEndpoints()
	if len(endpoints) == 0 {
		t.Error("Expected at least one endpoint")
	}
}

// TestClientManagerIntegration 测试客户端管理器集成
func TestClientManagerIntegration(t *testing.T) {
	logger := zap.NewNop()

	manager, err := grpc_client.NewClientManager(logger)
	if err != nil {
		t.Fatalf("Failed to create client manager: %v", err)
	}

	// 测试启动和停止
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = manager.Start(ctx)
	if err != nil {
		t.Logf("Expected start failure due to missing config: %v", err)
	}

	// 测试健康状态
	status := manager.GetAIOpsHealthStatus()
	if status == nil {
		t.Error("Expected health status, got nil")
	}

	// 测试停止
	err = manager.Stop()
	if err != nil {
		t.Errorf("Failed to stop manager: %v", err)
	}
}

// TestHealthCheckFlow 测试健康检查流程
func TestHealthCheckFlow(t *testing.T) {
	logger := zap.NewNop()

	config := &grpc_client.ClientConfig{
		Address:             "localhost:9999",
		HealthCheckInterval: 1 * time.Second,
		HealthCheckTimeout:  500 * time.Millisecond,
		EnableMetrics:       true,
	}

	client, err := grpc_client.NewAIOpsClient(config, logger)
	if err != nil {
		t.Logf("Expected connection failure: %v", err)
		return
	}
	defer client.Close()

	// 获取健康状态
	status := client.GetHealthStatus()
	if status == nil {
		t.Error("Expected health status, got nil")
	}

	// 测试连接状态
	_ = client.GetConnectionState()
}

// TestMetricsIntegration 测试监控指标集成
func TestMetricsIntegration(t *testing.T) {
	logger := zap.NewNop()

	// 测试启用监控的配置
	config := &grpc_client.ClientConfig{
		Address:       "localhost:9999",
		EnableMetrics: true,
	}

	client, err := grpc_client.NewAIOpsClient(config, logger)
	if err != nil {
		t.Logf("Expected connection failure: %v", err)
		return
	}
	defer client.Close()

	// 测试健康检查（会记录监控指标）
	req := &aiopsv1.HealthCheckRequest{
		Service: "aiops",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	_, err = client.HealthCheck(ctx, req)
	// 不期望成功，但测试不会panic
	_ = err
}

// TestConfigurationValidation 测试配置验证
func TestConfigurationValidation(t *testing.T) {
	logger := zap.NewNop()

	// 测试默认配置
	defaultConfig := grpc_client.DefaultConfig()
	if defaultConfig == nil {
		t.Error("Expected default config, got nil")
	}

	client, err := grpc_client.NewAIOpsClient(defaultConfig, logger)
	if err != nil {
		t.Logf("Expected connection failure with default config: %v", err)
		return
	}
	defer client.Close()

	// 测试多端点配置
	multiEndpointConfig := &grpc_client.ClientConfig{
		Address:   "localhost:9000",
		Endpoints: []string{"localhost:9001", "localhost:9002"},
	}

	client2, err := grpc_client.NewAIOpsClient(multiEndpointConfig, logger)
	if err != nil {
		t.Logf("Expected connection failure with multi-endpoint config: %v", err)
		return
	}
	defer client2.Close()

	endpoints := client2.GetEndpoints()
	if len(endpoints) != 2 {
		t.Errorf("Expected 2 endpoints, got %d", len(endpoints))
	}
}

// TestErrorScenarios 测试错误场景
func TestErrorScenarios(t *testing.T) {
	logger := zap.NewNop()

	// 测试空配置
	_, err := grpc_client.NewAIOpsClient(nil, logger)
	if err != nil {
		t.Logf("Expected error with nil config: %v", err)
	}

	// 测试无效地址格式
	invalidConfig := &grpc_client.ClientConfig{
		Address: "invalid-address-format",
	}

	_, err = grpc_client.NewAIOpsClient(invalidConfig, logger)
	if err != nil {
		t.Logf("Expected error with invalid address: %v", err)
	}
}

// TestReconnectionScenario 测试重连场景
func TestReconnectionScenario(t *testing.T) {
	logger := zap.NewNop()

	config := &grpc_client.ClientConfig{
		Address:              "localhost:9999",
		EnableAutoReconnect:  true,
		MaxReconnectAttempts: 2,
		ReconnectInterval:    1 * time.Second,
	}

	client, err := grpc_client.NewAIOpsClient(config, logger)
	if err != nil {
		t.Logf("Expected connection failure: %v", err)
		return
	}
	defer client.Close()

	// 测试手动重连
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = client.Reconnect(ctx)
	// 预期会失败，但测试不会panic
	_ = err
}
