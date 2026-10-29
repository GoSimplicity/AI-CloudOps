package utils

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	kv "github.com/aliyun/alibaba-cloud-sdk-go/services/r_kvstore"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/rds"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/slb"
)

// Each API is explicitly independent; ECS permission is not evidence of access.
type aliyunRDSAPI interface {
	DescribeDBInstances(*rds.DescribeDBInstancesRequest) (*rds.DescribeDBInstancesResponse, error)
}
type aliyunSLBAPI interface {
	DescribeLoadBalancers(*slb.DescribeLoadBalancersRequest) (*slb.DescribeLoadBalancersResponse, error)
}
type aliyunCacheAPI interface {
	DescribeInstances(*kv.DescribeInstancesRequest) (*kv.DescribeInstancesResponse, error)
}

type AliyunManagedResourceClient struct {
	region string
	rds    aliyunRDSAPI
	slb    aliyunSLBAPI
	cache  aliyunCacheAPI
}

var aliyunManagedID = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-]{1,99}$`)
var aliyunManagedRegion = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-]{1,49}$`)

// NewAliyunManagedResourceClient builds SDK clients only; it never probes the cloud.
func NewAliyunManagedResourceClient(credentials CloudCredentials, region string) (*AliyunManagedResourceClient, error) {
	if err := ValidateCloudCredentials(model.ProviderAliyun, credentials, time.Now().UTC()); err != nil {
		return nil, errors.New("阿里云凭据无效")
	}
	if !aliyunManagedRegion.MatchString(region) {
		return nil, errors.New("阿里云地域无效")
	}
	database, err := rds.NewClientWithAccessKey(region, credentials.AccessKey, credentials.SecretKey)
	if err != nil {
		return nil, errors.New("RDS客户端初始化失败")
	}
	balance, err := slb.NewClientWithAccessKey(region, credentials.AccessKey, credentials.SecretKey)
	if err != nil {
		return nil, errors.New("SLB客户端初始化失败")
	}
	cache, err := kv.NewClientWithAccessKey(region, credentials.AccessKey, credentials.SecretKey)
	if err != nil {
		return nil, errors.New("缓存客户端初始化失败")
	}
	return &AliyunManagedResourceClient{region: region, rds: database, slb: balance, cache: cache}, nil
}

func managedIDs(ids []string) error {
	if len(ids) == 0 || len(ids) > 100 {
		return errors.New("必须指定1至100个资源ID")
	}
	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		if !aliyunManagedID.MatchString(id) || seen[id] {
			return errors.New("资源ID无效或重复")
		}
		seen[id] = true
	}
	return nil
}
func validManagedFields(values ...string) bool {
	for _, v := range values {
		if len([]rune(v)) > 500 || strings.ContainsAny(v, "\x00\r\n") {
			return false
		}
	}
	return true
}
func checkManagedRegion(expected, actual string) bool { return actual == expected }

// ListRDS requires a successful exact lookup for every ID; no partial result is returned.
func (c *AliyunManagedResourceClient) ListRDS(ctx context.Context, ids []string) ([]model.AliyunRDSInstance, error) {
	if c == nil || c.rds == nil || !aliyunManagedRegion.MatchString(c.region) {
		return nil, errors.New("RDS客户端不可用")
	}
	if err := managedIDs(ids); err != nil {
		return nil, err
	}
	out := make([]model.AliyunRDSInstance, 0, len(ids))
	for _, id := range ids {
		request := rds.CreateDescribeDBInstancesRequest()
		request.Scheme = "https"
		request.RegionId = c.region
		request.DBInstanceId = id
		request.PageNumber = requests.NewInteger(1)
		request.PageSize = requests.NewInteger(100)
		if err := prepareAliyunRequest(ctx, request); err != nil {
			return nil, err
		}
		response, err := c.rds.DescribeDBInstances(request)
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if err != nil {
			return nil, errors.New("RDS读取失败")
		}
		if response == nil || response.TotalRecordCount != 1 || response.PageRecordCount != 1 || response.PageNumber != 1 || response.NextToken != "" || len(response.Items.DBInstance) != 1 {
			return nil, errors.New("RDS响应不完整或包含额外结果")
		}
		item := response.Items.DBInstance[0]
		if item.DBInstanceId != id || !checkManagedRegion(c.region, item.RegionId) || !validManagedFields(item.DBInstanceDescription, item.ZoneId, item.DBInstanceStatus, item.Engine, item.EngineVersion, item.DBInstanceClass) {
			return nil, errors.New("RDS响应数据不匹配")
		}
		out = append(out, model.AliyunRDSInstance{InstanceID: id, Name: item.DBInstanceDescription, Region: c.region, Zone: item.ZoneId, Status: item.DBInstanceStatus, Engine: item.Engine, EngineVersion: item.EngineVersion, InstanceClass: item.DBInstanceClass})
	}
	return out, nil
}

// ListSLB covers classic SLB only; ALB and NLB are not included.
func (c *AliyunManagedResourceClient) ListSLB(ctx context.Context, ids []string) ([]model.AliyunLoadBalancer, error) {
	if c == nil || c.slb == nil || !aliyunManagedRegion.MatchString(c.region) {
		return nil, errors.New("SLB客户端不可用")
	}
	if err := managedIDs(ids); err != nil {
		return nil, err
	}
	out := make([]model.AliyunLoadBalancer, 0, len(ids))
	for _, id := range ids {
		request := slb.CreateDescribeLoadBalancersRequest()
		request.Scheme = "https"
		request.RegionId = c.region
		request.LoadBalancerId = id
		request.PageNumber = requests.NewInteger(1)
		request.PageSize = requests.NewInteger(100)
		if err := prepareAliyunRequest(ctx, request); err != nil {
			return nil, err
		}
		response, err := c.slb.DescribeLoadBalancers(request)
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if err != nil {
			return nil, errors.New("SLB读取失败")
		}
		if response == nil || response.TotalCount != 1 || response.PageNumber != 1 || response.PageSize < 1 || len(response.LoadBalancers.LoadBalancer) != 1 {
			return nil, errors.New("SLB响应不完整或包含额外结果")
		}
		item := response.LoadBalancers.LoadBalancer[0]
		if item.LoadBalancerId != id || !checkManagedRegion(c.region, item.RegionId) || !validManagedFields(item.LoadBalancerName, item.LoadBalancerStatus, item.AddressType, item.VpcId) {
			return nil, errors.New("SLB响应数据不匹配")
		}
		out = append(out, model.AliyunLoadBalancer{InstanceID: id, Name: item.LoadBalancerName, Region: c.region, Status: item.LoadBalancerStatus, AddressType: item.AddressType, VPCID: item.VpcId})
	}
	return out, nil
}

func (c *AliyunManagedResourceClient) ListCache(ctx context.Context, ids []string) ([]model.AliyunCacheInstance, error) {
	if c == nil || c.cache == nil || !aliyunManagedRegion.MatchString(c.region) {
		return nil, errors.New("缓存客户端不可用")
	}
	if err := managedIDs(ids); err != nil {
		return nil, err
	}
	out := make([]model.AliyunCacheInstance, 0, len(ids))
	for _, id := range ids {
		request := kv.CreateDescribeInstancesRequest()
		request.Scheme = "https"
		request.RegionId = c.region
		request.InstanceIds = id
		request.PageNumber = requests.NewInteger(1)
		request.PageSize = requests.NewInteger(50)
		if err := prepareAliyunRequest(ctx, request); err != nil {
			return nil, err
		}
		response, err := c.cache.DescribeInstances(request)
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if err != nil {
			return nil, errors.New("缓存读取失败")
		}
		if response == nil || response.TotalCount != 1 || response.PageNumber != 1 || response.PageSize < 1 || len(response.Instances.KVStoreInstance) != 1 {
			return nil, errors.New("缓存响应不完整或包含额外结果")
		}
		item := response.Instances.KVStoreInstance[0]
		if item.InstanceId != id || !checkManagedRegion(c.region, item.RegionId) || !validManagedFields(item.InstanceName, item.ZoneId, item.InstanceStatus, item.InstanceType, item.EngineVersion, item.InstanceClass) {
			return nil, errors.New("缓存响应数据不匹配")
		}
		out = append(out, model.AliyunCacheInstance{InstanceID: id, Name: item.InstanceName, Region: c.region, Zone: item.ZoneId, Status: item.InstanceStatus, InstanceType: item.InstanceType, EngineVersion: item.EngineVersion, InstanceClass: item.InstanceClass})
	}
	return out, nil
}
