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
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
)

// ClientConfig gRPC客户端配置
type ClientConfig struct {
	Address             string        `yaml:"address" json:"address"`                             // 服务地址
	MaxRetries          int           `yaml:"max_retries" json:"max_retries"`                     // 最大重试次数
	RetryInterval       time.Duration `yaml:"retry_interval" json:"retry_interval"`               // 重试间隔
	ConnectionTimeout   time.Duration `yaml:"connection_timeout" json:"connection_timeout"`       // 连接超时
	RequestTimeout      time.Duration `yaml:"request_timeout" json:"request_timeout"`             // 请求超时
	KeepAliveTime       time.Duration `yaml:"keepalive_time" json:"keepalive_time"`               // KeepAlive时间
	KeepAliveTimeout    time.Duration `yaml:"keepalive_timeout" json:"keepalive_timeout"`         // KeepAlive超时
	MaxMessageSize      int           `yaml:"max_message_size" json:"max_message_size"`           // 最大消息大小
	EnableLoadBalancing bool          `yaml:"enable_load_balancing" json:"enable_load_balancing"` // 启用负载均衡

	// 健康检查相关配置
	HealthCheckInterval  time.Duration `yaml:"health_check_interval" json:"health_check_interval"`   // 健康检查间隔
	HealthCheckTimeout   time.Duration `yaml:"health_check_timeout" json:"health_check_timeout"`     // 健康检查超时
	EnableAutoReconnect  bool          `yaml:"enable_auto_reconnect" json:"enable_auto_reconnect"`   // 启用自动重连
	MaxReconnectAttempts int           `yaml:"max_reconnect_attempts" json:"max_reconnect_attempts"` // 最大重连次数
	ReconnectInterval    time.Duration `yaml:"reconnect_interval" json:"reconnect_interval"`         // 重连间隔

	// 多实例支持配置
	Endpoints []string `yaml:"endpoints" json:"endpoints"` // 多个服务端点列表，如 ["host1:9000", "host2:9000"]

	// 监控配置
	EnableMetrics bool `yaml:"enable_metrics" json:"enable_metrics"` // 启用监控指标
}

// gRPC客户端监控指标
var (
	grpcRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "grpc_client_requests_total",
			Help: "Total number of gRPC requests",
		},
		[]string{"method", "status"},
	)

	grpcRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "grpc_client_request_duration_seconds",
			Help:    "Duration of gRPC requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method"},
	)

	grpcConnectionStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "grpc_client_connection_status",
			Help: "Status of gRPC connection (1 = connected, 0 = disconnected)",
		},
		[]string{"endpoint"},
	)

	grpcHealthCheckTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "grpc_client_health_checks_total",
			Help: "Total number of health checks",
		},
		[]string{"status"},
	)
)

// 注册Prometheus指标
func init() {
	prometheus.MustRegister(grpcRequestsTotal)
	prometheus.MustRegister(grpcRequestDuration)
	prometheus.MustRegister(grpcConnectionStatus)
	prometheus.MustRegister(grpcHealthCheckTotal)
}

// HealthStatus 健康状态信息
type HealthStatus struct {
	IsHealthy           bool               `json:"is_healthy"`           // 是否健康
	LastCheckTime       time.Time          `json:"last_check_time"`      // 最后检查时间
	LastHealthyTime     time.Time          `json:"last_healthy_time"`    // 最后健康时间
	ConnectionState     connectivity.State `json:"connection_state"`     // 连接状态
	ConsecutiveFailures int                `json:"consecutive_failures"` // 连续失败次数
	TotalChecks         int64              `json:"total_checks"`         // 总检查次数
	FailedChecks        int64              `json:"failed_checks"`        // 失败检查次数
	ReconnectAttempts   int                `json:"reconnect_attempts"`   // 重连尝试次数
	LastError           string             `json:"last_error,omitempty"` // 最后错误信息
	CurrentEndpoint     string             `json:"current_endpoint"`     // 当前连接的端点
}

// DefaultConfig 返回默认配置
func DefaultConfig() *ClientConfig {
	return &ClientConfig{
		Address:             "cloudops-aiops:9000",
		MaxRetries:          3,
		RetryInterval:       2 * time.Second,
		ConnectionTimeout:   10 * time.Second,
		RequestTimeout:      2 * time.Minute,
		KeepAliveTime:       30 * time.Second,
		KeepAliveTimeout:    5 * time.Second,
		MaxMessageSize:      4 * 1024 * 1024, // 4MB
		EnableLoadBalancing: true,

		// 健康检查默认配置
		HealthCheckInterval:  30 * time.Second, // 每30秒检查一次
		HealthCheckTimeout:   5 * time.Second,  // 健康检查5秒超时
		EnableAutoReconnect:  true,             // 启用自动重连
		MaxReconnectAttempts: 5,                // 最大重连5次
		ReconnectInterval:    10 * time.Second, // 重连间隔10秒

		// 多实例默认配置（为空表示使用Address字段）
		Endpoints: []string{},

		// 监控默认配置
		EnableMetrics: true, // 默认启用监控指标
	}
}

type AIOpsClient interface {
	HealthCheck(ctx context.Context, req *aiopsv1.HealthCheckRequest) (*aiopsv1.HealthCheckResponse, error)
	Chat(ctx context.Context, req *aiopsv1.ChatRequest, token string) (aiopsv1.AIOpsService_ChatClient, error)
	PredictLoad(ctx context.Context, req *aiopsv1.LoadPredictionRequest, token string) (*aiopsv1.LoadPredictionResponse, error)
	IsConnected() bool
	Close() error

	// 健康检查和重连相关方法
	StartHealthCheck(ctx context.Context) error // 启动健康检查
	StopHealthCheck()                           // 停止健康检查
	GetConnectionState() connectivity.State     // 获取连接状态
	GetHealthStatus() *HealthStatus             // 获取健康状态
	Reconnect(ctx context.Context) error        // 手动重连

	// 多端点支持方法
	GetEndpoints() []string // 获取所有端点
}

type aiopsClient struct {
	conn   *grpc.ClientConn
	client aiopsv1.AIOpsServiceClient
	config *ClientConfig
	logger *zap.Logger

	// 健康检查相关字段
	healthStatus  *HealthStatus
	healthMu      sync.RWMutex       // 健康状态锁
	healthCancel  context.CancelFunc // 健康检查取消函数
	healthStopped chan struct{}      // 健康检查停止信号
	reconnectMu   sync.Mutex         // 重连锁，防止并发重连

	// 监控相关字段
	currentEndpoint string // 当前连接的端点
}

func NewAIOpsClient(config *ClientConfig, logger *zap.Logger) (AIOpsClient, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// 设置gRPC连接选项
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                config.KeepAliveTime,
			Timeout:             config.KeepAliveTimeout,
			PermitWithoutStream: true,
		}),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(config.MaxMessageSize),
			grpc.MaxCallSendMsgSize(config.MaxMessageSize),
		),
	}

	// 确定目标地址
	var target string
	if len(config.Endpoints) > 0 {
		// 如果配置了多个endpoints，使用第一个作为目标，gRPC会自动负载均衡
		target = config.Endpoints[0]
		logger.Info("使用多端点配置", zap.Strings("endpoints", config.Endpoints))

		// 多端点时启用负载均衡
		opts = append(opts, grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`))
	} else {
		// 使用单一地址
		target = config.Address

		// 启用负载均衡（如果配置了）
		if config.EnableLoadBalancing {
			opts = append(opts, grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`))
		}
	}

	// 建立连接
	ctx, cancel := context.WithTimeout(context.Background(), config.ConnectionTimeout)
	defer cancel()

	conn, err := grpc.DialContext(ctx, target, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC server: %w", err)
	}

	client := &aiopsClient{
		conn:   conn,
		client: aiopsv1.NewAIOpsServiceClient(conn),
		config: config,
		logger: logger,
		healthStatus: &HealthStatus{
			IsHealthy:       true,
			LastCheckTime:   time.Now(),
			LastHealthyTime: time.Now(),
			ConnectionState: conn.GetState(),
			CurrentEndpoint: target,
		},
		healthStopped:   make(chan struct{}),
		currentEndpoint: target,
	}

	// 更新连接状态监控指标
	if config.EnableMetrics {
		grpcConnectionStatus.WithLabelValues(target).Set(1)
	}

	return client, nil
}

// withAuthMetadata 添加认证元数据
func (c *aiopsClient) withAuthMetadata(ctx context.Context, token string) context.Context {
	if token != "" {
		md := metadata.Pairs("authorization", "Bearer "+token)
		ctx = metadata.NewOutgoingContext(ctx, md)
	}
	return ctx
}

// executeWithRetry 执行带重试的操作
func (c *aiopsClient) executeWithRetry(ctx context.Context, fn func(context.Context) error) error {
	var lastErr error

	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// 等待重试间隔
			select {
			case <-time.After(c.config.RetryInterval):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		// 执行操作
		lastErr = fn(ctx)
		if lastErr == nil {
			return nil
		}

		c.logger.Warn("gRPC请求失败，正在重试",
			zap.Int("attempt", attempt+1),
			zap.Int("max_retries", c.config.MaxRetries),
			zap.Error(lastErr))
	}

	return fmt.Errorf("在%d次重试后仍然失败: %w", c.config.MaxRetries, lastErr)
}

// HealthCheck 健康检查
func (c *aiopsClient) HealthCheck(ctx context.Context, req *aiopsv1.HealthCheckRequest) (*aiopsv1.HealthCheckResponse, error) {
	var resp *aiopsv1.HealthCheckResponse
	start := time.Now()
	methodName := "HealthCheck"

	err := c.executeWithRetry(ctx, func(opCtx context.Context) error {
		var err error
		resp, err = c.client.HealthCheck(opCtx, req)
		return err
	})

	// 记录监控指标
	if c.config.EnableMetrics {
		duration := time.Since(start).Seconds()
		grpcRequestDuration.WithLabelValues(methodName).Observe(duration)

		status := "success"
		if err != nil {
			status = "error"
		}
		grpcRequestsTotal.WithLabelValues(methodName, status).Inc()
	}

	return resp, err
}

// Chat AI助手对话 - 流式
func (c *aiopsClient) Chat(ctx context.Context, req *aiopsv1.ChatRequest, token string) (aiopsv1.AIOpsService_ChatClient, error) {
	start := time.Now()
	methodName := "Chat"

	authCtx := c.withAuthMetadata(ctx, token)
	stream, err := c.client.Chat(authCtx, req)

	// 记录监控指标
	if c.config.EnableMetrics {
		duration := time.Since(start).Seconds()
		grpcRequestDuration.WithLabelValues(methodName).Observe(duration)

		status := "success"
		if err != nil {
			status = "error"
		}
		grpcRequestsTotal.WithLabelValues(methodName, status).Inc()
	}

	return stream, err
}

// PredictLoad 负载预测
func (c *aiopsClient) PredictLoad(ctx context.Context, req *aiopsv1.LoadPredictionRequest, token string) (*aiopsv1.LoadPredictionResponse, error) {
	var resp *aiopsv1.LoadPredictionResponse
	start := time.Now()
	methodName := "PredictLoad"

	err := c.executeWithRetry(ctx, func(opCtx context.Context) error {
		authCtx := c.withAuthMetadata(opCtx, token)

		var err error
		resp, err = c.client.PredictLoad(authCtx, req)
		return err
	})

	// 记录监控指标
	if c.config.EnableMetrics {
		duration := time.Since(start).Seconds()
		grpcRequestDuration.WithLabelValues(methodName).Observe(duration)

		status := "success"
		if err != nil {
			status = "error"
		}
		grpcRequestsTotal.WithLabelValues(methodName, status).Inc()
	}

	return resp, err
}

// IsConnected 检查连接状态
func (c *aiopsClient) IsConnected() bool {
	if c.conn == nil {
		return false
	}
	state := c.conn.GetState()
	return state == connectivity.Ready || state == connectivity.Idle
}

// Close 关闭连接
func (c *aiopsClient) Close() error {
	// 停止健康检查
	c.StopHealthCheck()

	// 更新连接状态监控指标
	if c.config.EnableMetrics && c.currentEndpoint != "" {
		grpcConnectionStatus.WithLabelValues(c.currentEndpoint).Set(0)
	}

	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// StartHealthCheck 启动健康检查
func (c *aiopsClient) StartHealthCheck(ctx context.Context) error {
	if c.config.HealthCheckInterval <= 0 {
		c.logger.Info("健康检查间隔未配置，跳过启动健康检查")
		return nil
	}

	// 创建可取消的上下文
	healthCtx, cancel := context.WithCancel(ctx)
	c.healthCancel = cancel

	go c.healthCheckLoop(healthCtx)
	c.logger.Info("已启动gRPC健康检查",
		zap.Duration("interval", c.config.HealthCheckInterval),
		zap.Bool("auto_reconnect", c.config.EnableAutoReconnect))

	return nil
}

// StopHealthCheck 停止健康检查
func (c *aiopsClient) StopHealthCheck() {
	if c.healthCancel != nil {
		c.healthCancel()
		c.healthCancel = nil

		// 等待健康检查goroutine结束
		select {
		case <-c.healthStopped:
		case <-time.After(5 * time.Second):
			c.logger.Warn("健康检查停止超时")
		}
	}
}

// GetConnectionState 获取连接状态
func (c *aiopsClient) GetConnectionState() connectivity.State {
	if c.conn == nil {
		return connectivity.Shutdown
	}
	return c.conn.GetState()
}

// GetHealthStatus 获取健康状态
func (c *aiopsClient) GetHealthStatus() *HealthStatus {
	c.healthMu.RLock()
	defer c.healthMu.RUnlock()

	// 创建副本避免外部修改
	status := *c.healthStatus
	status.ConnectionState = c.GetConnectionState()
	return &status
}

// Reconnect 手动重连
func (c *aiopsClient) Reconnect(ctx context.Context) error {
	c.reconnectMu.Lock()
	defer c.reconnectMu.Unlock()

	c.logger.Info("开始手动重连gRPC服务", zap.String("address", c.config.Address))

	return c.doReconnect(ctx, "manual")
}

// healthCheckLoop 健康检查循环
func (c *aiopsClient) healthCheckLoop(ctx context.Context) {
	defer close(c.healthStopped)

	ticker := time.NewTicker(c.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("健康检查已停止")
			return
		case <-ticker.C:
			c.performHealthCheck(ctx)
		}
	}
}

// performHealthCheck 执行健康检查
func (c *aiopsClient) performHealthCheck(ctx context.Context) {
	// 设置健康检查超时
	checkCtx, cancel := context.WithTimeout(ctx, c.config.HealthCheckTimeout)
	defer cancel()

	c.healthMu.Lock()
	c.healthStatus.TotalChecks++
	c.healthStatus.LastCheckTime = time.Now()
	c.healthStatus.ConnectionState = c.GetConnectionState()
	c.healthMu.Unlock()

	// 执行健康检查请求
	req := &aiopsv1.HealthCheckRequest{
		Service: "aiops",
	}

	_, err := c.client.HealthCheck(checkCtx, req)

	c.healthMu.Lock()
	if err != nil {
		c.healthStatus.FailedChecks++
		c.healthStatus.ConsecutiveFailures++
		c.healthStatus.IsHealthy = false
		c.healthStatus.LastError = err.Error()

		// 记录健康检查监控指标
		if c.config.EnableMetrics {
			grpcHealthCheckTotal.WithLabelValues("failed").Inc()
		}

		c.logger.Warn("健康检查失败",
			zap.Error(err),
			zap.Int("consecutive_failures", c.healthStatus.ConsecutiveFailures),
			zap.String("connection_state", c.healthStatus.ConnectionState.String()))

		// 触发自动重连
		if c.config.EnableAutoReconnect && c.healthStatus.ConsecutiveFailures >= 3 {
			c.healthMu.Unlock()
			go c.autoReconnect(ctx)
			return
		}
	} else {
		c.healthStatus.IsHealthy = true
		c.healthStatus.ConsecutiveFailures = 0
		c.healthStatus.LastHealthyTime = time.Now()
		c.healthStatus.LastError = ""

		// 记录健康检查监控指标
		if c.config.EnableMetrics {
			grpcHealthCheckTotal.WithLabelValues("success").Inc()
		}

		c.logger.Debug("健康检查成功",
			zap.String("connection_state", c.healthStatus.ConnectionState.String()))
	}
	c.healthMu.Unlock()
}

// autoReconnect 自动重连
func (c *aiopsClient) autoReconnect(ctx context.Context) {
	if !c.reconnectMu.TryLock() {
		c.logger.Debug("重连已在进行中，跳过此次重连")
		return
	}
	defer c.reconnectMu.Unlock()

	c.healthMu.RLock()
	attempts := c.healthStatus.ReconnectAttempts
	c.healthMu.RUnlock()

	if attempts >= c.config.MaxReconnectAttempts {
		c.logger.Error("已达到最大重连次数，停止重连",
			zap.Int("max_attempts", c.config.MaxReconnectAttempts))
		return
	}

	c.logger.Info("开始自动重连",
		zap.Int("attempt", attempts+1),
		zap.Int("max_attempts", c.config.MaxReconnectAttempts))

	// 等待重连间隔
	select {
	case <-ctx.Done():
		return
	case <-time.After(c.config.ReconnectInterval):
	}

	if err := c.doReconnect(ctx, "auto"); err != nil {
		c.logger.Error("自动重连失败", zap.Error(err))
	}
}

// doReconnect 执行重连逻辑
func (c *aiopsClient) doReconnect(ctx context.Context, reconnectType string) error {
	c.healthMu.Lock()
	c.healthStatus.ReconnectAttempts++
	attempts := c.healthStatus.ReconnectAttempts
	c.healthMu.Unlock()

	c.logger.Info("执行重连",
		zap.String("type", reconnectType),
		zap.Int("attempt", attempts))

	// 关闭旧连接
	if c.conn != nil {
		c.conn.Close()
	}

	// 创建新连接
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                c.config.KeepAliveTime,
			Timeout:             c.config.KeepAliveTimeout,
			PermitWithoutStream: true,
		}),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(c.config.MaxMessageSize),
			grpc.MaxCallSendMsgSize(c.config.MaxMessageSize),
		),
	}

	if c.config.EnableLoadBalancing {
		opts = append(opts, grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`))
	}

	connCtx, cancel := context.WithTimeout(ctx, c.config.ConnectionTimeout)
	defer cancel()

	conn, err := grpc.DialContext(connCtx, c.config.Address, opts...)
	if err != nil {
		c.healthMu.Lock()
		c.healthStatus.LastError = fmt.Sprintf("重连失败: %v", err)
		c.healthMu.Unlock()
		return fmt.Errorf("重连失败: %w", err)
	}

	// 更新连接和客户端
	c.conn = conn
	c.client = aiopsv1.NewAIOpsServiceClient(conn)

	// 重置健康状态
	c.healthMu.Lock()
	c.healthStatus.IsHealthy = true
	c.healthStatus.ConsecutiveFailures = 0
	c.healthStatus.LastHealthyTime = time.Now()
	c.healthStatus.ConnectionState = conn.GetState()
	c.healthStatus.LastError = ""
	// 重连成功后重置重连计数器
	if reconnectType == "manual" {
		c.healthStatus.ReconnectAttempts = 0
	}
	c.healthMu.Unlock()

	c.logger.Info("重连成功",
		zap.String("type", reconnectType),
		zap.String("address", c.config.Address))

	return nil
}

// GetEndpoints 获取所有端点
func (c *aiopsClient) GetEndpoints() []string {
	if len(c.config.Endpoints) > 0 {
		// 返回副本避免外部修改
		endpoints := make([]string, len(c.config.Endpoints))
		copy(endpoints, c.config.Endpoints)
		return endpoints
	}

	// 如果没有配置endpoints，返回Address
	return []string{c.config.Address}
}
