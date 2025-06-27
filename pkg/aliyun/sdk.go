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

package aliyun

import (
	"context"
	"time"

	openapi "github.com/alibabacloud-go/darabonba-openapi/client"
	openapiv2 "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	ecs "github.com/alibabacloud-go/ecs-20140526/v2/client"
	slb "github.com/alibabacloud-go/slb-20140515/v3/client"
	"github.com/alibabacloud-go/tea/tea"
	vpc "github.com/alibabacloud-go/vpc-20160428/v2/client"
	"go.uber.org/zap"
)

type ClientOptions struct {
	Endpoint        string
	Timeout         int
	ConnectTimeout  int
	ReadTimeout     int
	MaxRetries      int
	RetryWaitMin    time.Duration
	RetryWaitMax    time.Duration
	UserAgent       string
	EnableTracing   bool
	EnableAsyncMode bool
}

func DefaultClientOptions() *ClientOptions {
	return &ClientOptions{
		Timeout:        30000,
		ConnectTimeout: 5000,
		ReadTimeout:    10000,
		MaxRetries:     3,
		RetryWaitMin:   100 * time.Millisecond,
		RetryWaitMax:   1 * time.Second,
		UserAgent:      "CloudOps/1.0",
		EnableTracing:  true,
	}
}

type SDK struct {
	accessKeyId     string
	accessKeySecret string
	logger          *zap.Logger
	options         *ClientOptions
}

type Config struct {
	AccessKeyId     string
	AccessKeySecret string
	Logger          *zap.Logger
	Options         *ClientOptions
}

func NewSDK(accessKeyId, accessKeySecret string) *SDK {
	logger, _ := zap.NewProduction()
	return &SDK{
		accessKeyId:     accessKeyId,
		accessKeySecret: accessKeySecret,
		logger:          logger,
		options:         DefaultClientOptions(),
	}
}

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
		accessKeySecret: config.AccessKeySecret,
		logger:          logger,
		options:         options,
	}
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

// applyRequestOptions 应用请求选项
func (s *SDK) applyRequestOptions(config *openapi.Config, options ...*ClientOptions) *openapi.Config {
	// 先应用SDK默认配置
	config.ConnectTimeout = tea.Int(s.options.ConnectTimeout)
	config.ReadTimeout = tea.Int(s.options.ReadTimeout)
	config.UserAgent = tea.String(s.options.UserAgent)

	// 再应用方法级配置（如果有）
	if len(options) > 0 && options[0] != nil {
		opt := options[0]
		if opt.Endpoint != "" {
			config.Endpoint = tea.String(opt.Endpoint)
		}
		if opt.ConnectTimeout > 0 {
			config.ConnectTimeout = tea.Int(opt.ConnectTimeout)
		}
		if opt.ReadTimeout > 0 {
			config.ReadTimeout = tea.Int(opt.ReadTimeout)
		}
		if opt.UserAgent != "" {
			config.UserAgent = tea.String(opt.UserAgent)
		}
	}
	return config
}

// applyRequestOptionsV2 应用请求选项 (V2版API)
func (s *SDK) applyRequestOptionsV2(config *openapiv2.Config, options ...*ClientOptions) *openapiv2.Config {
	// 先应用SDK默认配置
	config.ConnectTimeout = tea.Int(s.options.ConnectTimeout)
	config.ReadTimeout = tea.Int(s.options.ReadTimeout)
	config.UserAgent = tea.String(s.options.UserAgent)

	// 再应用方法级配置（如果有）
	if len(options) > 0 && options[0] != nil {
		opt := options[0]
		if opt.Endpoint != "" {
			config.Endpoint = tea.String(opt.Endpoint)
		}
		if opt.ConnectTimeout > 0 {
			config.ConnectTimeout = tea.Int(opt.ConnectTimeout)
		}
		if opt.ReadTimeout > 0 {
			config.ReadTimeout = tea.Int(opt.ReadTimeout)
		}
		if opt.UserAgent != "" {
			config.UserAgent = tea.String(opt.UserAgent)
		}
	}
	return config
}

// CreateEcsClient 创建ECS客户端
func (s *SDK) CreateEcsClient(region string, options ...*ClientOptions) (*ecs.Client, error) {
	config := &openapi.Config{
		AccessKeyId:     tea.String(s.accessKeyId),
		AccessKeySecret: tea.String(s.accessKeySecret),
		RegionId:        tea.String(region),
		Endpoint:        tea.String("ecs.aliyuncs.com"),
	}

	config = s.applyRequestOptions(config, options...)
	return ecs.NewClient(config)
}

// CreateEcsClientWithContext 创建带上下文的ECS客户端
func (s *SDK) CreateEcsClientWithContext(ctx context.Context, region string, options ...*ClientOptions) (*ecs.Client, error) {
	client, err := s.CreateEcsClient(region, options...)
	if err != nil {
		return nil, HandleError(err)
	}

	// 应用上下文
	if s.options.EnableTracing {
		// 这里可以添加从上下文提取tracing信息到客户端的逻辑
	}

	return client, nil
}

// CreateVpcClient 创建VPC客户端
func (s *SDK) CreateVpcClient(region string, options ...*ClientOptions) (*vpc.Client, error) {
	config := &openapiv2.Config{
		AccessKeyId:     tea.String(s.accessKeyId),
		AccessKeySecret: tea.String(s.accessKeySecret),
		RegionId:        tea.String(region),
		Endpoint:        tea.String("vpc.aliyuncs.com"),
	}

	config = s.applyRequestOptionsV2(config, options...)
	return vpc.NewClient(config)
}

// CreateVpcClientWithContext 创建带上下文的VPC客户端
func (s *SDK) CreateVpcClientWithContext(ctx context.Context, region string, options ...*ClientOptions) (*vpc.Client, error) {
	client, err := s.CreateVpcClient(region, options...)
	if err != nil {
		return nil, HandleError(err)
	}

	// 应用上下文
	if s.options.EnableTracing {
		// 这里可以添加从上下文提取tracing信息到客户端的逻辑
	}

	return client, nil
}

// CreateSlbClient 创建SLB客户端
func (s *SDK) CreateSlbClient(region string, options ...*ClientOptions) (*slb.Client, error) {
	config := &openapi.Config{
		AccessKeyId:     tea.String(s.accessKeyId),
		AccessKeySecret: tea.String(s.accessKeySecret),
		RegionId:        tea.String(region),
		Endpoint:        tea.String("slb.aliyuncs.com"),
	}

	config = s.applyRequestOptions(config, options...)
	return slb.NewClient(config)
}

// CreateSlbClientWithContext 创建带上下文的SLB客户端
func (s *SDK) CreateSlbClientWithContext(ctx context.Context, region string, options ...*ClientOptions) (*slb.Client, error) {
	client, err := s.CreateSlbClient(region, options...)
	if err != nil {
		return nil, HandleError(err)
	}

	// 应用上下文
	if s.options.EnableTracing {
		// 这里可以添加从上下文提取tracing信息到客户端的逻辑
	}

	return client, nil
}

// DoWithRetry 执行带重试的函数
func (s *SDK) DoWithRetry(ctx context.Context, fn func() error) error {
	var err error
	maxRetries := s.options.MaxRetries

	for i := 0; i <= maxRetries; i++ {
		err = fn()
		if err == nil {
			return nil
		}

		// 检查是否需要重试
		if !RetryableError(err) {
			return err
		}

		// 最后一次尝试失败后不再等待
		if i == maxRetries {
			break
		}

		// 计算退避时间
		wait := s.options.RetryWaitMin << uint(i)
		if wait > s.options.RetryWaitMax {
			wait = s.options.RetryWaitMax
		}

		// 使用context确保不会在已取消的上下文中进行等待
		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return ctx.Err()
		}

		s.logger.Info("正在重试请求", zap.Int("尝试次数", i+1), zap.Duration("等待时间", wait), zap.Error(err))
	}

	return err
}
