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
	openapi "github.com/alibabacloud-go/darabonba-openapi/client"
	openapiv2 "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	ecs "github.com/alibabacloud-go/ecs-20140526/v2/client"
	"github.com/alibabacloud-go/tea/tea"
	vpc "github.com/alibabacloud-go/vpc-20160428/v2/client"
	"go.uber.org/zap"
)

type SDK struct {
	accessKeyId     string
	accessKeySecret string
	logger          *zap.Logger
}

func NewSDK(accessKeyId, accessKeySecret string) *SDK {
	logger, _ := zap.NewProduction()
	return &SDK{
		accessKeyId:     accessKeyId,
		accessKeySecret: accessKeySecret,
		logger:          logger,
	}
}

func NewSDKWithLogger(accessKeyId, accessKeySecret string, logger *zap.Logger) *SDK {
	return &SDK{
		accessKeyId:     accessKeyId,
		accessKeySecret: accessKeySecret,
		logger:          logger,
	}
}

// CreateEcsClient 创建ECS客户端
func (s *SDK) CreateEcsClient(region string) (*ecs.Client, error) {
	config := &openapi.Config{
		AccessKeyId:     tea.String(s.accessKeyId),
		AccessKeySecret: tea.String(s.accessKeySecret),
		RegionId:        tea.String(region),
		Endpoint:        tea.String("ecs.aliyuncs.com"),
	}
	return ecs.NewClient(config)
}

// CreateVpcClient 创建VPC客户端
func (s *SDK) CreateVpcClient(region string) (*vpc.Client, error) {
	config := &openapiv2.Config{
		AccessKeyId:     tea.String(s.accessKeyId),
		AccessKeySecret: tea.String(s.accessKeySecret),
		RegionId:        tea.String(region),
	}
	return vpc.NewClient(config)
}
