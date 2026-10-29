package utils

import (
	"context"
	"errors"
	"net"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	appmodel "github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/config"
	sdkregion "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/region"
	ecs "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/ecs/v2"
	sdkmodel "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/ecs/v2/model"
)

// Huawei's SDK does not expose a context argument; this adapter binds each
// operation's context to its HTTP request without changing any shared client.
type huaweiContextTransport struct {
	ctx  context.Context
	base http.RoundTripper
}

func (t huaweiContextTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.base.RoundTrip(req.Clone(t.ctx))
}

type huaweiECSAPI interface {
	ListServersDetails(context.Context, *sdkmodel.ListServersDetailsRequest) (*sdkmodel.ListServersDetailsResponse, error)
}
type huaweiSDK struct {
	accessKey, secretKey, projectID, region string
	transport                               http.RoundTripper
}

func (s huaweiSDK) ListServersDetails(ctx context.Context, req *sdkmodel.ListServersDetailsRequest) (*sdkmodel.ListServersDetailsResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	credential, err := basic.NewCredentialsBuilder().WithAk(s.accessKey).WithSk(s.secretKey).WithProjectId(s.projectID).SafeBuild()
	if err != nil {
		return nil, errors.New("华为云凭据配置无效")
	}
	// Construct the official hostname from a validated region, avoiding SDK
	// environment-based endpoint overrides and accidental credential redirection.
	endpoint := "https://ecs." + s.region + ".myhuaweicloud.com"
	conf := config.DefaultHttpConfig().WithTimeout(30 * time.Second).WithHttpRoundTripper(huaweiContextTransport{ctx: ctx, base: s.transport})
	client, err := ecs.EcsClientBuilder().WithRegion(sdkregion.NewRegion(s.region, endpoint)).WithCredential(credential).WithHttpConfig(conf).SafeBuild()
	if err != nil {
		return nil, errors.New("华为云客户端配置无效")
	}
	resp, err := ecs.NewEcsClient(client).ListServersDetails(req)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err != nil {
		return nil, errors.New("华为云ECS请求失败")
	}
	return resp, nil
}

type HuaweiClient struct {
	client    huaweiECSAPI
	region    string
	projectID string
}

var huaweiRegionPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)+$`)
var huaweiProjectPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)
var huaweiIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func NewHuaweiClient(accessKey, secretKey, projectID, region string) (*HuaweiClient, error) {
	if strings.TrimSpace(accessKey) == "" || strings.TrimSpace(secretKey) == "" || !huaweiProjectPattern.MatchString(projectID) || !huaweiRegionPattern.MatchString(region) || len(region) > 50 {
		return nil, errors.New("华为云访问密钥、项目ID和有效地域不能为空")
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	return &HuaweiClient{client: huaweiSDK{accessKey: accessKey, secretKey: secretKey, projectID: projectID, region: region, transport: transport}, region: region, projectID: projectID}, nil
}

func (c *HuaweiClient) VerifyCredentials(ctx context.Context) error {
	limit := int32(1)
	allGranted := "all_granted_eps"
	response, err := c.client.ListServersDetails(ctx, &sdkmodel.ListServersDetailsRequest{Limit: &limit, EnterpriseProjectId: &allGranted})
	if err != nil {
		return err
	}
	if response == nil || response.Servers == nil || response.Count == nil || *response.Count < 0 || int(*response.Count) < len(*response.Servers) || (*response.Count > 0 && len(*response.Servers) == 0) {
		return errors.New("华为云返回的实例清单不完整")
	}
	for _, server := range *response.Servers {
		if !huaweiIDPattern.MatchString(server.Id) || (c.projectID != "" && server.TenantId != c.projectID) {
			return errors.New("华为云实例不属于配置项目")
		}
	}
	return ctx.Err()
}

// The configured project ID applies only to this region. Return only the
// region actually verified; this is not a catalog of all account regions.
func (c *HuaweiClient) GetAvailableRegions(ctx context.Context) ([]appmodel.AvailableRegion, error) {
	if err := c.VerifyCredentials(ctx); err != nil {
		return nil, err
	}
	return []appmodel.AvailableRegion{{Region: c.region, RegionName: c.region, Available: true}}, nil
}

func (c *HuaweiClient) ListInstances(ctx context.Context, ids []string) ([]*appmodel.TreeCloudResource, error) {
	batches, err := instanceIDBatches(ids, 100)
	if err != nil {
		return nil, err
	}
	result := make([]*appmodel.TreeCloudResource, 0)
	seen := make(map[string]bool)
	for _, batch := range batches {
		expected := -1
		marker := ""
		markers := make(map[string]bool)
		requested := make(map[string]bool, len(batch))
		for _, id := range batch {
			if !huaweiIDPattern.MatchString(id) {
				return nil, errors.New("华为云实例ID格式无效")
			}
			requested[strings.ToLower(id)] = true
		}
		for {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			limit := int32(200)
			req := &sdkmodel.ListServersDetailsRequest{Limit: &limit}
			if len(batch) == 0 {
				allGranted := "all_granted_eps"
				req.EnterpriseProjectId = &allGranted
			}
			if len(batch) > 0 {
				joined := strings.Join(batch, ",")
				req.ServerId = &joined
			} else if marker != "" {
				req.Marker = &marker
			}
			response, err := c.client.ListServersDetails(ctx, req)
			if err != nil {
				return nil, err
			}
			if err = ctx.Err(); err != nil {
				return nil, err
			}
			if response == nil || response.Count == nil || response.Servers == nil || *response.Count < 0 {
				return nil, errors.New("华为云实例清单响应不完整")
			}
			if expected < 0 {
				expected = int(*response.Count)
			} else if expected != int(*response.Count) {
				return nil, errors.New("华为云实例清单在分页期间发生变化")
			}
			servers := *response.Servers
			if len(servers) > int(limit) || len(result)+len(servers) > 100000 {
				return nil, errors.New("华为云实例清单超过安全上限")
			}
			for _, server := range servers {
				id := strings.ToLower(server.Id)
				if !huaweiIDPattern.MatchString(server.Id) || server.Name == "" || server.Status == "" || (c.projectID != "" && server.TenantId != c.projectID) || seen[id] || (len(batch) > 0 && !requested[id]) {
					return nil, errors.New("华为云实例清单包含缺失、重复或不匹配的实例")
				}
				seen[id] = true
				resource, err := mapHuaweiServer(server, c.region)
				if err != nil {
					return nil, err
				}
				result = append(result, resource)
			}
			if len(batch) > 0 {
				if len(servers) != len(requested) || expected != len(requested) {
					return nil, errors.New("华为云部分指定实例不可用")
				}
				break
			}
			if len(result) == expected {
				break
			}
			if len(result) > expected || len(servers) == 0 {
				return nil, errors.New("华为云实例清单未完整分页")
			}
			marker = servers[len(servers)-1].Id
			if markers[marker] {
				return nil, errors.New("华为云实例分页未推进")
			}
			markers[marker] = true
		}
	}
	return result, nil
}

func mapHuaweiServer(server sdkmodel.ServerDetail, region string) (*appmodel.TreeCloudResource, error) {
	if len([]rune(server.Name)) > 100 {
		return nil, errors.New("华为云实例名称超出存储限制")
	}
	status := appmodel.CloudResourceUnknown
	switch server.Status {
	case "ACTIVE":
		status = appmodel.CloudResourceRunning
	case "SHUTOFF", "SHELVED_OFFLOADED", "SUSPENDED":
		status = appmodel.CloudResourceStopped
	case "BUILD":
		status = appmodel.CloudResourceStarting
	case "DELETED", "SOFT_DELETED":
		status = appmodel.CloudResourceDeleted
	}
	r := &appmodel.TreeCloudResource{Name: server.Name, InstanceID: server.Id, ResourceType: appmodel.ResourceTypeECS, Status: status, Region: region, ZoneID: server.OSEXTAZavailabilityZone, Currency: appmodel.CurrencyCNY, VpcID: server.Metadata["vpc_id"], OSType: server.Metadata["os_type"], ImageName: server.Metadata["image_name"]}
	if len([]rune(r.ZoneID)) > 50 || len([]rune(r.VpcID)) > 100 || len([]rune(r.OSType)) > 50 || len([]rune(r.ImageName)) > 100 {
		return nil, errors.New("华为云实例字段超出存储限制")
	}
	if server.Description != nil {
		r.Description = *server.Description
	}
	if server.Image != nil {
		if len([]rune(server.Image.Id)) > 100 {
			return nil, errors.New("华为云镜像ID超出存储限制")
		}
		r.ImageID = server.Image.Id
	}
	if server.Flavor != nil {
		if len([]rune(server.Flavor.Id)) > 100 {
			return nil, errors.New("华为云实例规格超出存储限制")
		}
		r.InstanceType = server.Flavor.Id
		cpu, e := strconv.Atoi(server.Flavor.Vcpus)
		if e == nil && cpu >= 0 {
			r.Cpu = cpu
		}
		ram, e := strconv.Atoi(server.Flavor.Ram)
		if e == nil && ram >= 0 && ram%1024 == 0 {
			r.Memory = ram / 1024
		}
	}
	switch server.Metadata["charging_mode"] {
	case "0":
		r.ChargeType = appmodel.ChargeTypePostPaid
	case "1":
		r.ChargeType = appmodel.ChargeTypePrePaid
	}
	// flavor.disk is documented as invalid here; never claim disk capacity.
	var private, public []string
	for _, addresses := range server.Addresses {
		for _, a := range addresses {
			if a.OSEXTIPStype == nil || a.Addr == "" {
				continue
			}
			if net.ParseIP(a.Addr) == nil || len(a.Addr) > 45 {
				return nil, errors.New("华为云实例IP格式无效")
			}
			switch a.OSEXTIPStype.Value() {
			case "fixed":
				private = append(private, a.Addr)
			case "floating":
				public = append(public, a.Addr)
			}
		}
	}
	sort.Strings(private)
	sort.Strings(public)
	if len(private) > 0 {
		r.PrivateIP = private[0]
	}
	if len(public) > 0 {
		r.PublicIP = public[0]
	}
	if server.Tags != nil {
		for _, tag := range *server.Tags {
			parts := strings.SplitN(tag, "=", 2)
			kv := appmodel.KeyValue{Key: parts[0]}
			if len(parts) == 2 {
				kv.Value = parts[1]
			}
			r.Tags = append(r.Tags, kv)
		}
	}
	return r, nil
}
