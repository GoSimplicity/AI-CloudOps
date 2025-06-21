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

package huawei

import (
	"go.uber.org/zap"

	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
	ecs "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/ecs/v2"
	ecsregion "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/ecs/v2/region"
	evs "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/evs/v2"
	evsregion "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/evs/v2/region"
	vpc "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/vpc/v3"
	vpcregion "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/vpc/v3/region"
)

// SDK 华为云SDK客户端
type SDK struct {
	logger    *zap.Logger
	accessKey string
	secretKey string
}

// NewSDK 创建华为云SDK客户端
func NewSDK(logger *zap.Logger, accessKey, secretKey string) *SDK {
	return &SDK{
		logger:    logger,
		accessKey: accessKey,
		secretKey: secretKey,
	}
}

// CreateEcsClient 创建ECS客户端
func (s *SDK) CreateEcsClient(regionId, projectId string) (*ecs.EcsClient, error) {
	auth := basic.NewCredentialsBuilder().
		WithAk(s.accessKey).
		WithSk(s.secretKey).
		WithProjectId(projectId).
		Build()

	r := ecsregion.ValueOf(regionId)
	hcClient := ecs.EcsClientBuilder().
		WithRegion(r).
		WithCredential(auth).
		Build()

	return ecs.NewEcsClient(hcClient), nil
}

// CreateEvsClient 创建EVS客户端
func (s *SDK) CreateEvsClient(regionId, projectId string) (*evs.EvsClient, error) {
	auth := basic.NewCredentialsBuilder().
		WithAk(s.accessKey).
		WithSk(s.secretKey).
		WithProjectId(projectId).
		Build()

	r := evsregion.ValueOf(regionId)
	hcClient := evs.EvsClientBuilder().
		WithRegion(r).
		WithCredential(auth).
		Build()

	return evs.NewEvsClient(hcClient), nil
}

// CreateVpcClient 创建VPC客户端
func (s *SDK) CreateVpcClient(regionId, projectId string) (*vpc.VpcClient, error) {
	auth := basic.NewCredentialsBuilder().
		WithAk(s.accessKey).
		WithSk(s.secretKey).
		WithProjectId(projectId).
		Build()

	r := vpcregion.ValueOf(regionId)
	hcClient := vpc.VpcClientBuilder().
		WithRegion(r).
		WithCredential(auth).
		Build()

	return vpc.NewVpcClient(hcClient), nil
}
