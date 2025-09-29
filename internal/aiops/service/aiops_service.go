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

package service

import (
	"context"
	"fmt"
	"sync"

	"github.com/GoSimplicity/AI-CloudOps/pkg/grpc_client"
	aiopsv1 "github.com/GoSimplicity/AI-CloudOps/proto/aiops/v1"
	"go.uber.org/zap"
)

type AIOpsService interface {
	Chat(ctx context.Context, req *aiopsv1.ChatRequest, token string) (aiopsv1.AIOpsService_ChatClient, error)
	PredictLoad(ctx context.Context, req *aiopsv1.LoadPredictionRequest, token string) (*aiopsv1.LoadPredictionResponse, error)
	HealthCheck(ctx context.Context, req *aiopsv1.HealthCheckRequest) (*aiopsv1.HealthCheckResponse, error)

	// 扩展接口 - 通过通用gRPC调用实现
	CallAIService(ctx context.Context, method string, request interface{}, token string) (interface{}, error)

	IsServiceAvailable() bool
}

type aiopsService struct {
	grpcManager *grpc_client.ClientManager
	logger      *zap.Logger
}

func NewAIOpsService(grpcManager *grpc_client.ClientManager, logger *zap.Logger) AIOpsService {
	return &aiopsService{
		grpcManager: grpcManager,
		logger:      logger,
	}
}

// Chat AI助手对话
func (s *aiopsService) Chat(ctx context.Context, req *aiopsv1.ChatRequest, token string) (aiopsv1.AIOpsService_ChatClient, error) {
	// 检查服务是否可用
	if !s.IsServiceAvailable() {
		return nil, fmt.Errorf("AI服务不可用")
	}

	// 获取gRPC客户端
	client, release, err := s.grpcManager.GetAIOpsClient(ctx)
	if err != nil {
		s.logger.Error("获取AIOps gRPC客户端失败", zap.Error(err))
		return nil, fmt.Errorf("获取客户端失败: %w", err)
	}

	s.logger.Info("开始AI助手对话流", zap.String("session_id", req.SessionId))

	// 调用gRPC方法
	stream, err := client.Chat(ctx, req, token)
	if err != nil {
		release() // 出错时释放连接
		s.logger.Error("AI助手对话失败", zap.Error(err))
		return nil, fmt.Errorf("对话失败: %w", err)
	}

	// 包装流以在流结束时自动释放连接
	wrappedStream := &wrappedChatStream{
		AIOpsService_ChatClient: stream,
		release:                 release,
		logger:                  s.logger,
		sessionID:               req.SessionId,
	}

	return wrappedStream, nil
}

// PredictLoad 负载预测
func (s *aiopsService) PredictLoad(ctx context.Context, req *aiopsv1.LoadPredictionRequest, token string) (*aiopsv1.LoadPredictionResponse, error) {
	// 检查服务是否可用
	if !s.IsServiceAvailable() {
		return nil, fmt.Errorf("AI服务不可用")
	}

	// 获取gRPC客户端
	client, release, err := s.grpcManager.GetAIOpsClient(ctx)
	if err != nil {
		s.logger.Error("获取AIOps gRPC客户端失败", zap.Error(err))
		return nil, fmt.Errorf("获取客户端失败: %w", err)
	}
	defer release()

	// 调用gRPC方法
	resp, err := client.PredictLoad(ctx, req, token)
	if err != nil {
		s.logger.Error("负载预测失败", zap.Error(err))
		return nil, fmt.Errorf("预测失败: %w", err)
	}

	return resp, nil
}

// HealthCheck 健康检查
func (s *aiopsService) HealthCheck(ctx context.Context, req *aiopsv1.HealthCheckRequest) (*aiopsv1.HealthCheckResponse, error) {
	// 检查服务是否可用
	if !s.IsServiceAvailable() {
		return nil, fmt.Errorf("AI服务不可用")
	}

	// 获取gRPC客户端
	client, release, err := s.grpcManager.GetAIOpsClient(ctx)
	if err != nil {
		s.logger.Error("获取AIOps gRPC客户端失败", zap.Error(err))
		return nil, fmt.Errorf("获取客户端失败: %w", err)
	}
	defer release()

	// 调用gRPC方法
	resp, err := client.HealthCheck(ctx, req)
	if err != nil {
		s.logger.Error("健康检查失败", zap.Error(err))
		return nil, fmt.Errorf("健康检查失败: %w", err)
	}

	return resp, nil
}

// CallAIService 通用AI服务调用
func (s *aiopsService) CallAIService(ctx context.Context, method string, request interface{}, token string) (interface{}, error) {
	// 检查服务是否可用
	if !s.IsServiceAvailable() {
		return nil, fmt.Errorf("AI服务不可用")
	}

	// 记录调用信息
	s.logger.Info("调用AI服务",
		zap.String("method", method),
		zap.Any("request", request))

	// 这里可以根据method调用不同的Python API
	// 暂时返回成功状态
	return map[string]interface{}{
		"status": "success",
		"method": method,
	}, nil
}

// IsServiceAvailable 检查服务是否可用
func (s *aiopsService) IsServiceAvailable() bool {
	if s.grpcManager == nil {
		return false
	}
	return s.grpcManager.IsAIOpsEnabled()
}

// wrappedChatStream 包装gRPC流，自动管理连接释放
type wrappedChatStream struct {
	aiopsv1.AIOpsService_ChatClient
	release   func()
	logger    *zap.Logger
	sessionID string
	once      sync.Once
}

// Recv 接收流数据，在流结束时自动释放连接
func (w *wrappedChatStream) Recv() (*aiopsv1.ChatResponse, error) {
	resp, err := w.AIOpsService_ChatClient.Recv()

	// 如果流结束（无论是EOF还是错误），释放连接
	if err != nil {
		w.once.Do(func() {
			w.logger.Info("AI助手对话流结束，释放连接",
				zap.String("session_id", w.sessionID),
				zap.String("reason", err.Error()))
			w.release()
		})
	}

	return resp, err
}
