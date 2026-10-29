package utils

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v8"
	appmodel "github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/google/uuid"
)

// AzureCredentials is request-scoped and must never be returned to API callers.
type AzureCredentials struct {
	TenantID       string `json:"-"`
	ClientID       string `json:"-"`
	ClientSecret   string `json:"-"`
	SubscriptionID string `json:"-"`
}

func (AzureCredentials) String() string   { return "[REDACTED]" }
func (AzureCredentials) GoString() string { return "[REDACTED]" }

// SDK authentication may POST only to the fixed Microsoft identity endpoint;
// resource queries may GET only from Azure Resource Manager. This guard also
// prevents a malicious nextLink from receiving a bearer token.
type azureSafeTransport struct {
	base   policy.Transporter
	tenant string
}

func (t azureSafeTransport) Do(req *http.Request) (*http.Response, error) {
	u := req.URL
	if u == nil || u.Scheme != "https" || u.User != nil || u.Port() != "" || u.Fragment != "" {
		return nil, errors.New("Azure 请求目标不受信任")
	}
	switch strings.ToLower(u.Hostname()) {
	case "management.azure.com":
		if req.Method != http.MethodGet {
			return nil, errors.New("Azure 只允许资源查询")
		}
	case "login.microsoftonline.com":
		if req.Method != http.MethodGet && req.Method != http.MethodPost {
			return nil, errors.New("Azure 身份请求方法无效")
		}
		if !strings.HasPrefix(strings.ToLower(u.Path), "/"+strings.ToLower(t.tenant)+"/") {
			return nil, errors.New("Azure 身份请求租户无效")
		}
	default:
		return nil, errors.New("Azure 请求目标不受信任")
	}
	return t.base.Do(req)
}

type azurePager interface {
	More() bool
	NextPage(context.Context) (armcompute.VirtualMachinesClientListByLocationResponse, error)
}
type azurePagerFactory func(string) azurePager

type AzureClient struct {
	subscriptionID, region string
	newPager               azurePagerFactory
}

func NewAzureClient(tenantID, clientID, clientSecret, subscriptionID, region string) (*AzureClient, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	client := &http.Client{Timeout: 30 * time.Second, Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	return newAzureClient(AzureCredentials{TenantID: tenantID, ClientID: clientID, ClientSecret: clientSecret, SubscriptionID: subscriptionID}, region, client)
}
func newAzureClient(creds AzureCredentials, region string, transport policy.Transporter) (*AzureClient, error) {
	for _, id := range []string{creds.TenantID, creds.ClientID, creds.SubscriptionID} {
		if parsed, err := uuid.Parse(id); err != nil || parsed.String() != strings.ToLower(id) {
			return nil, errors.New("Azure 租户、应用和订阅 ID 必须是 UUID")
		}
	}
	if strings.TrimSpace(creds.ClientSecret) == "" || len(creds.ClientSecret) > 8192 || !azureRegionPattern.MatchString(region) || len(region) > 50 || transport == nil {
		return nil, errors.New("Azure 凭据或地域配置无效")
	}
	safe := azureSafeTransport{base: transport, tenant: creds.TenantID}
	options := policy.ClientOptions{Cloud: cloud.AzurePublic, Transport: safe, Retry: policy.RetryOptions{MaxRetries: 1}}
	credential, err := azidentity.NewClientSecretCredential(creds.TenantID, creds.ClientID, creds.ClientSecret, &azidentity.ClientSecretCredentialOptions{ClientOptions: options, DisableInstanceDiscovery: true})
	if err != nil {
		return nil, errors.New("Azure 身份凭据配置无效")
	}
	client, err := armcompute.NewVirtualMachinesClient(creds.SubscriptionID, credential, &arm.ClientOptions{ClientOptions: options, DisableRPRegistration: true})
	if err != nil {
		return nil, errors.New("Azure 计算客户端配置无效")
	}
	return &AzureClient{subscriptionID: creds.SubscriptionID, region: region, newPager: func(location string) azurePager { return client.NewListByLocationPager(location, nil) }}, nil
}

var azureRegionPattern = regexp.MustCompile(`^[a-z0-9]+$`)

func (c *AzureClient) VerifyCredentials(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	pager := c.newPager(c.region)
	if pager == nil || !pager.More() {
		return errors.New("Azure 未返回实例查询页")
	}
	page, err := pager.NextPage(ctx)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if err != nil {
		return errors.New("Azure 虚拟机读取权限验证失败")
	}
	if page.Value == nil {
		return errors.New("Azure 实例清单响应不完整")
	}
	return nil
}

// Only this explicitly configured location was checked. This is not a list
// of all subscription locations or proof of subscription-wide permissions.
func (c *AzureClient) GetAvailableRegions(ctx context.Context) ([]appmodel.AvailableRegion, error) {
	if err := c.VerifyCredentials(ctx); err != nil {
		return nil, err
	}
	return []appmodel.AvailableRegion{{Region: c.region, RegionName: c.region, Available: true}}, nil
}

func (c *AzureClient) ListInstances(ctx context.Context, ids []string) ([]*appmodel.TreeCloudResource, error) {
	// Azure can omit resources outside the caller's RBAC scope. Do not expose an
	// unfiltered listing as a complete snapshot without proven subscription-wide access.
	if len(ids) == 0 {
		return nil, errors.New("Azure 仅支持指定实例 ID 的查询，无法确认订阅权限覆盖范围")
	}
	requested := make(map[string]bool, len(ids))
	for _, id := range ids {
		parsed, err := uuid.Parse(id)
		if err != nil {
			return nil, errors.New("Azure 实例 ID 必须是 UUID")
		}
		requested[parsed.String()] = true
	}
	if len(requested) > 1000 {
		return nil, errors.New("Azure 指定实例数量超过上限")
	}
	pager := c.newPager(c.region)
	if pager == nil || !pager.More() {
		return nil, errors.New("Azure 未返回实例查询页")
	}
	found := map[string]*appmodel.TreeCloudResource{}
	seen := map[string]bool{}
	links := map[string]bool{}
	pages := 0
	for pager.More() {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		pages++
		if pages > 1000 {
			return nil, errors.New("Azure 实例分页超过安全上限")
		}
		page, err := pager.NextPage(ctx)
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if err != nil {
			return nil, errors.New("Azure 实例查询失败")
		}
		if page.Value == nil {
			return nil, errors.New("Azure 实例清单响应不完整")
		}
		for _, vm := range page.Value {
			resource, err := c.mapVirtualMachine(vm)
			if err != nil {
				return nil, err
			}
			id := strings.ToLower(resource.InstanceID)
			if seen[id] {
				return nil, errors.New("Azure 实例分页包含重复 ID")
			}
			seen[id] = true
			if len(seen) > 100000 {
				return nil, errors.New("Azure 实例清单超过安全上限")
			}
			if requested[id] {
				found[id] = resource
			}
		}
		if page.NextLink != nil && *page.NextLink != "" {
			link := *page.NextLink
			if links[link] || !c.validNextLink(link) {
				return nil, errors.New("Azure 实例分页链接无效或重复")
			}
			links[link] = true
		}
	}
	if len(found) != len(requested) {
		return nil, errors.New("部分 Azure 指定实例未找到或无读取权限")
	}
	out := make([]*appmodel.TreeCloudResource, 0, len(requested))
	for _, id := range ids {
		parsed, _ := uuid.Parse(id)
		key := parsed.String()
		if r, ok := found[key]; ok {
			out = append(out, r)
			delete(found, key)
		}
	}
	return out, nil
}

func (c *AzureClient) validNextLink(link string) bool {
	u, err := url.Parse(link)
	if err != nil || u.User != nil || u.Fragment != "" {
		return false
	}
	if u.IsAbs() && (u.Scheme != "https" || !strings.EqualFold(u.Host, "management.azure.com")) {
		return false
	}
	if !u.IsAbs() && u.Host != "" {
		return false
	}
	expected := "/subscriptions/" + c.subscriptionID + "/providers/Microsoft.Compute/locations/" + c.region + "/virtualMachines"
	return strings.EqualFold(u.Path, expected)
}
func (c *AzureClient) mapVirtualMachine(vm *armcompute.VirtualMachine) (*appmodel.TreeCloudResource, error) {
	if vm == nil || vm.Location == nil || !strings.EqualFold(*vm.Location, c.region) || vm.Name == nil || *vm.Name == "" || len([]rune(*vm.Name)) > 100 || vm.ID == nil || vm.Properties == nil || vm.Properties.VMID == nil {
		return nil, errors.New("Azure 实例缺少有效身份或地域")
	}
	parsed, err := uuid.Parse(*vm.Properties.VMID)
	if err != nil {
		return nil, errors.New("Azure 实例 ID 无效")
	}
	parts := strings.Split(strings.Trim(*vm.ID, "/"), "/")
	if len(parts) != 8 || !strings.EqualFold(parts[0], "subscriptions") || !strings.EqualFold(parts[1], c.subscriptionID) || !strings.EqualFold(parts[2], "resourceGroups") || parts[3] == "" || !strings.EqualFold(parts[4], "providers") || !strings.EqualFold(parts[5], "Microsoft.Compute") || !strings.EqualFold(parts[6], "virtualMachines") || !strings.EqualFold(parts[7], *vm.Name) {
		return nil, errors.New("Azure 实例不属于配置订阅")
	}
	result := &appmodel.TreeCloudResource{Name: *vm.Name, InstanceID: parsed.String(), Region: c.region, ResourceType: appmodel.ResourceTypeECS, Status: appmodel.CloudResourceUnknown, Currency: appmodel.CurrencyUSD}
	if vm.Properties.HardwareProfile != nil && vm.Properties.HardwareProfile.VMSize != nil {
		result.InstanceType = string(*vm.Properties.HardwareProfile.VMSize)
		if len([]rune(result.InstanceType)) > 100 {
			return nil, errors.New("Azure 实例规格超出存储限制")
		}
	}
	if len(vm.Zones) == 1 && vm.Zones[0] != nil {
		if len(*vm.Zones[0]) > 50 {
			return nil, errors.New("Azure 可用区超出存储限制")
		}
		result.ZoneID = *vm.Zones[0]
	}
	if vm.Properties.InstanceView != nil {
		for _, s := range vm.Properties.InstanceView.Statuses {
			if s == nil || s.Code == nil {
				continue
			}
			switch strings.ToLower(*s.Code) {
			case "powerstate/running":
				result.Status = appmodel.CloudResourceRunning
			case "powerstate/stopped", "powerstate/deallocated":
				result.Status = appmodel.CloudResourceStopped
			case "powerstate/starting":
				result.Status = appmodel.CloudResourceStarting
			case "powerstate/stopping", "powerstate/deallocating":
				result.Status = appmodel.CloudResourceStopping
			}
		}
	}
	keys := make([]string, 0, len(vm.Tags))
	for key := range vm.Tags {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if vm.Tags[key] != nil {
			result.Tags = append(result.Tags, appmodel.KeyValue{Key: key, Value: *vm.Tags[key]})
		}
	}
	return result, nil
}
