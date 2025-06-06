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
	logger          *zap.Logger
	accessKeyId     string
	accessKeySecret string
}

func NewSDK(logger *zap.Logger, accessKeyId, accessKeySecret string) *SDK {
	return &SDK{
		logger:          logger,
		accessKeyId:     accessKeyId,
		accessKeySecret: accessKeySecret,
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
