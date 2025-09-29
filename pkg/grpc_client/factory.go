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
 */

package grpc_client

import (
	"context"
	"fmt"
	"sync"
	"time"

	aiopsv1 "github.com/GoSimplicity/AI-CloudOps/proto/aiops/v1"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"google.golang.org/grpc/connectivity"
)

type ClientManager struct {
	aiopsClient AIOpsClient
	config      *GrpcConfig
	logger      *zap.Logger
	mu          sync.RWMutex
	started     bool
}

type GrpcConfig struct {
	AIOps struct {
		Enabled bool         `yaml:"enabled"`
		Client  ClientConfig `yaml:",inline"`
	} `yaml:"aiops"`
}

func NewClientManager(logger *zap.Logger) (*ClientManager, error) {
	config := &GrpcConfig{}

	if err := loadConfigFromViper(config); err != nil {
		return nil, fmt.Errorf("failed to load gRPC config: %w", err)
	}

	manager := &ClientManager{
		config: config,
		logger: logger,
	}

	return manager, nil
}

func loadConfigFromViper(config *GrpcConfig) error {
	// AIOps基础配置
	config.AIOps.Enabled = viper.GetBool("grpc.aiops.enabled")
	config.AIOps.Client.Address = viper.GetString("grpc.aiops.address")
	config.AIOps.Client.MaxRetries = viper.GetInt("grpc.aiops.max_retries")
	config.AIOps.Client.RetryInterval = viper.GetDuration("grpc.aiops.retry_interval")
	config.AIOps.Client.ConnectionTimeout = viper.GetDuration("grpc.aiops.connection_timeout")
	config.AIOps.Client.RequestTimeout = viper.GetDuration("grpc.aiops.request_timeout")
	config.AIOps.Client.KeepAliveTime = viper.GetDuration("grpc.aiops.keepalive_time")
	config.AIOps.Client.KeepAliveTimeout = viper.GetDuration("grpc.aiops.keepalive_timeout")
	config.AIOps.Client.MaxMessageSize = viper.GetInt("grpc.aiops.max_message_size")
	config.AIOps.Client.EnableLoadBalancing = viper.GetBool("grpc.aiops.enable_load_balancing")

	// 健康检查配置
	config.AIOps.Client.HealthCheckInterval = viper.GetDuration("grpc.aiops.health_check_interval")
	config.AIOps.Client.HealthCheckTimeout = viper.GetDuration("grpc.aiops.health_check_timeout")
	config.AIOps.Client.EnableAutoReconnect = viper.GetBool("grpc.aiops.enable_auto_reconnect")
	config.AIOps.Client.MaxReconnectAttempts = viper.GetInt("grpc.aiops.max_reconnect_attempts")
	config.AIOps.Client.ReconnectInterval = viper.GetDuration("grpc.aiops.reconnect_interval")

	// 多端点配置
	config.AIOps.Client.Endpoints = viper.GetStringSlice("grpc.aiops.endpoints")

	// 监控配置
	config.AIOps.Client.EnableMetrics = viper.GetBool("grpc.aiops.enable_metrics")

	// 设置默认值
	if config.AIOps.Client.Address == "" {
		config.AIOps.Client.Address = "cloudops-aiops:9000"
	}
	if config.AIOps.Client.MaxRetries == 0 {
		config.AIOps.Client.MaxRetries = 3
	}
	if config.AIOps.Client.RetryInterval == 0 {
		config.AIOps.Client.RetryInterval = 2 * time.Second
	}
	if config.AIOps.Client.ConnectionTimeout == 0 {
		config.AIOps.Client.ConnectionTimeout = 10 * time.Second
	}
	if config.AIOps.Client.RequestTimeout == 0 {
		config.AIOps.Client.RequestTimeout = 2 * time.Minute
	}
	if config.AIOps.Client.MaxMessageSize == 0 {
		config.AIOps.Client.MaxMessageSize = 4 * 1024 * 1024 // 4MB
	}

	// 健康检查默认值
	if config.AIOps.Client.HealthCheckInterval == 0 {
		config.AIOps.Client.HealthCheckInterval = 30 * time.Second
	}
	if config.AIOps.Client.HealthCheckTimeout == 0 {
		config.AIOps.Client.HealthCheckTimeout = 5 * time.Second
	}
	if config.AIOps.Client.MaxReconnectAttempts == 0 {
		config.AIOps.Client.MaxReconnectAttempts = 5
	}
	if config.AIOps.Client.ReconnectInterval == 0 {
		config.AIOps.Client.ReconnectInterval = 10 * time.Second
	}
	// EnableAutoReconnect 默认true（如果没有显式设置为false）
	if !viper.IsSet("grpc.aiops.enable_auto_reconnect") {
		config.AIOps.Client.EnableAutoReconnect = true
	}
	// EnableMetrics 默认true（如果没有显式设置为false）
	if !viper.IsSet("grpc.aiops.enable_metrics") {
		config.AIOps.Client.EnableMetrics = true
	}

	return nil
}

func (m *ClientManager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.started {
		return nil
	}

	if !m.config.AIOps.Enabled {
		m.logger.Info("AIOps gRPC客户端已禁用")
		m.started = true
		return nil
	}

	// 创建AIOps客户端
	client, err := NewAIOpsClient(&m.config.AIOps.Client, m.logger)
	if err != nil {
		return fmt.Errorf("failed to create AIOps client: %w", err)
	}

	m.aiopsClient = client
	m.started = true

	// 启动健康检查
	if err := client.StartHealthCheck(ctx); err != nil {
		m.logger.Warn("启动健康检查失败", zap.Error(err))
		// 不影响客户端启动，仅记录警告
	}

	m.logger.Info("gRPC客户端管理器启动成功")
	return nil
}

func (m *ClientManager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.started {
		return nil
	}

	var err error
	if m.aiopsClient != nil {
		if closeErr := m.aiopsClient.Close(); closeErr != nil {
			err = fmt.Errorf("failed to close AIOps client: %w", closeErr)
		}
		m.aiopsClient = nil
	}

	m.started = false
	m.logger.Info("gRPC客户端管理器已停止")
	return err
}

func (m *ClientManager) GetAIOpsClient(ctx context.Context) (AIOpsClient, func(), error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.started {
		return nil, nil, fmt.Errorf("client manager not started")
	}

	if !m.config.AIOps.Enabled {
		return nil, nil, fmt.Errorf("AIOps client is disabled")
	}

	if m.aiopsClient == nil {
		return nil, nil, fmt.Errorf("AIOps client not available")
	}

	if !m.aiopsClient.IsConnected() {
		return nil, nil, fmt.Errorf("AIOps client is not connected")
	}

	// 直接返回客户端
	release := func() {
		// 无需释放逻辑
	}

	return m.aiopsClient, release, nil
}

func (m *ClientManager) CheckAIOpsHealth(ctx context.Context) error {
	if !m.config.AIOps.Enabled {
		return nil // 如果禁用了gRPC，认为是健康的
	}

	client, release, err := m.GetAIOpsClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to get AIOps client for health check: %w", err)
	}
	defer release()

	// 执行健康检查
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req := &aiopsv1.HealthCheckRequest{
		Service: "aiops",
	}

	_, err = client.HealthCheck(checkCtx, req)
	if err != nil {
		return fmt.Errorf("AIOps health check failed: %w", err)
	}

	return nil
}

func (m *ClientManager) WarmUp(ctx context.Context) error {
	if !m.config.AIOps.Enabled {
		m.logger.Info("AIOps gRPC客户端已禁用，跳过预热")
		return nil
	}

	m.logger.Info("开始预热gRPC连接...")

	// 执行健康检查来预热连接
	if err := m.CheckAIOpsHealth(ctx); err != nil {
		m.logger.Warn("gRPC连接预热失败", zap.Error(err))
		return err
	}

	m.logger.Info("gRPC连接预热完成")
	return nil
}

func (m *ClientManager) IsAIOpsEnabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.AIOps.Enabled && m.started
}

func (m *ClientManager) UpdateConfig() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 重新加载配置
	newConfig := &GrpcConfig{}
	if err := loadConfigFromViper(newConfig); err != nil {
		return fmt.Errorf("failed to reload config: %w", err)
	}

	m.config = newConfig
	m.logger.Info("gRPC配置已更新")
	return nil
}

// GetAIOpsHealthStatus 获取AIOps客户端健康状态
func (m *ClientManager) GetAIOpsHealthStatus() *HealthStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.started || !m.config.AIOps.Enabled || m.aiopsClient == nil {
		return &HealthStatus{
			IsHealthy:       false,
			LastCheckTime:   time.Now(),
			ConnectionState: connectivity.Shutdown,
			LastError:       "client not available",
		}
	}

	return m.aiopsClient.GetHealthStatus()
}

// ReconnectAIOpsClient 手动重连AIOps客户端
func (m *ClientManager) ReconnectAIOpsClient(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.started || !m.config.AIOps.Enabled || m.aiopsClient == nil {
		return fmt.Errorf("AIOps client not available")
	}

	return m.aiopsClient.Reconnect(ctx)
}
