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

package aws

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"go.uber.org/zap"
)

// ClientOptions AWS客户端选项配置
type ClientOptions struct {
	Endpoint        string        // 自定义端点
	Timeout         time.Duration // 请求超时时间
	ConnectTimeout  time.Duration // 连接超时时间
	ReadTimeout     time.Duration // 读取超时时间
	MaxRetries      int           // 最大重试次数
	RetryWaitMin    time.Duration // 重试最小等待时间
	RetryWaitMax    time.Duration // 重试最大等待时间
	UserAgent       string        // 用户代理
	EnableTracing   bool          // 是否启用链路追踪
	EnableAsyncMode bool          // 是否启用异步模式
}

// DefaultClientOptions 获取默认客户端选项
func DefaultClientOptions() *ClientOptions {
	return &ClientOptions{
		Timeout:        30 * time.Second,
		ConnectTimeout: 5 * time.Second,
		ReadTimeout:    10 * time.Second,
		MaxRetries:     3,
		RetryWaitMin:   100 * time.Millisecond,
		RetryWaitMax:   1 * time.Second,
		UserAgent:      "AI-CloudOps/1.0",
		EnableTracing:  true,
	}
}

// SDK AWS SDK客户端
type SDK struct {
	accessKeyId     string
	secretAccessKey string
	logger          *zap.Logger
	options         *ClientOptions
}

// Config SDK配置
type Config struct {
	AccessKeyId     string
	SecretAccessKey string
	Logger          *zap.Logger
	Options         *ClientOptions
}

// NewSDK 创建AWS SDK客户端
func NewSDK(accessKeyId, secretAccessKey string) *SDK {
	logger, _ := zap.NewProduction()
	return &SDK{
		accessKeyId:     accessKeyId,
		secretAccessKey: secretAccessKey,
		logger:          logger,
		options:         DefaultClientOptions(),
	}
}

// NewSDKWithConfig 使用配置创建AWS SDK客户端
func NewSDKWithConfig(config *Config) *SDK {
	var logger *zap.Logger
	if config.Logger != nil {
		logger = config.Logger
	} else {
		logger, _ = zap.NewProduction()
	}

	options := DefaultClientOptions()
	if config.Options != nil {
		options = config.Options
	}

	return &SDK{
		accessKeyId:     config.AccessKeyId,
		secretAccessKey: config.SecretAccessKey,
		logger:          logger,
		options:         options,
	}
}

// NewSDKWithLogger 创建带有自定义logger的AWS SDK客户端
func NewSDKWithLogger(accessKeyId, secretAccessKey string, logger *zap.Logger) *SDK {
	return &SDK{
		accessKeyId:     accessKeyId,
		secretAccessKey: secretAccessKey,
		logger:          logger,
		options:         DefaultClientOptions(),
	}
}

// getAWSConfig 获取AWS配置
func (s *SDK) getAWSConfig(ctx context.Context, region string) (aws.Config, error) {
	creds := credentials.NewStaticCredentialsProvider(s.accessKeyId, s.secretAccessKey, "")

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithCredentialsProvider(creds),
		config.WithRegion(region),
	)
	if err != nil {
		return aws.Config{}, err
	}

	return cfg, nil
}

// CreateEC2Client 创建EC2客户端
func (s *SDK) CreateEC2Client(ctx context.Context, region string) (*ec2.Client, error) {
	cfg, err := s.getAWSConfig(ctx, region)
	if err != nil {
		return nil, err
	}

	return ec2.NewFromConfig(cfg), nil
}

// CreateELBv2Client 创建ELB v2客户端
func (s *SDK) CreateELBv2Client(ctx context.Context, region string) (*elasticloadbalancingv2.Client, error) {
	cfg, err := s.getAWSConfig(ctx, region)
	if err != nil {
		return nil, err
	}

	return elasticloadbalancingv2.NewFromConfig(cfg), nil
}

// CreateSTSClient 创建STS客户端
func (s *SDK) CreateSTSClient(ctx context.Context, region string) (*sts.Client, error) {
	cfg, err := s.getAWSConfig(ctx, region)
	if err != nil {
		return nil, err
	}

	return sts.NewFromConfig(cfg), nil
}

// WithOptions 设置选项并返回SDK实例
func (s *SDK) WithOptions(options *ClientOptions) *SDK {
	if options != nil {
		s.options = options
	}
	return s
}

// GetLogger 获取日志器
func (s *SDK) GetLogger() *zap.Logger {
	return s.logger
}

// SetLogger 设置日志器
func (s *SDK) SetLogger(logger *zap.Logger) {
	if logger != nil {
		s.logger = logger
	}
}

// ValidateCredentials 验证AWS凭证
func (s *SDK) ValidateCredentials(ctx context.Context) error {
	s.logger.Info("验证AWS凭证", zap.String("accessKeyId", s.GetAccessKeyId()))

	if s.accessKeyId == "" || s.secretAccessKey == "" {
		return ErrInvalidCredentials
	}

	// 使用STS GetCallerIdentity API验证凭证
	stsClient, err := s.CreateSTSClient(ctx, "us-east-1") // STS服务使用us-east-1
	if err != nil {
		return err
	}

	_, err = stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		s.logger.Error("AWS凭证验证失败", zap.Error(err))
		return ErrInvalidCredentials
	}

	s.logger.Info("AWS凭证验证成功")
	return nil
}

// GetAccessKeyId 获取访问密钥ID（脱敏）
func (s *SDK) GetAccessKeyId() string {
	if len(s.accessKeyId) <= 8 {
		return s.accessKeyId
	}
	return s.accessKeyId[:8] + "***"
}

// IsInitialized 检查SDK是否已初始化
func (s *SDK) IsInitialized() bool {
	return s.accessKeyId != "" && s.secretAccessKey != ""
}
