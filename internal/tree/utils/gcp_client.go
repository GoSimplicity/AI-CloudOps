package utils

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	appmodel "github.com/GoSimplicity/AI-CloudOps/internal/model"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/jwt"
	"google.golang.org/api/compute/v1"
)

const gcpTokenURL = "https://oauth2.googleapis.com/token"
const gcpComputeScope = "https://www.googleapis.com/auth/compute.readonly"

var gcpProjectPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{4,28}[a-z0-9]$`)
var gcpZonePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{1,48}-[a-z]$`)
var gcpEmailPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,62}@[a-z0-9][a-z0-9-]{1,62}\.iam\.gserviceaccount\.com$`)

type gcpServiceAccount struct {
	Type                       string `json:"type"`
	ProjectID                  string `json:"project_id"`
	PrivateKeyID               string `json:"private_key_id"`
	PrivateKey                 string `json:"private_key"`
	ClientEmail                string `json:"client_email"`
	ClientID                   string `json:"client_id"`
	AuthURI                    string `json:"auth_uri"`
	TokenURI                   string `json:"token_uri"`
	AuthProviderCertificateURL string `json:"auth_provider_x509_cert_url"`
	ClientCertificateURL       string `json:"client_x509_cert_url"`
	UniverseDomain             string `json:"universe_domain"`
}

func (gcpServiceAccount) String() string               { return "[REDACTED]" }
func (gcpServiceAccount) GoString() string             { return "[REDACTED]" }
func (gcpServiceAccount) MarshalJSON() ([]byte, error) { return []byte("{}"), nil }

func parseGCPServiceAccount(raw []byte) (gcpServiceAccount, error) {
	var cred gcpServiceAccount
	if len(raw) == 0 || len(raw) > 16384 {
		return cred, errors.New("GCP 服务账号凭据无效")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cred); err != nil {
		return gcpServiceAccount{}, errors.New("GCP 服务账号凭据格式无效")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return gcpServiceAccount{}, errors.New("GCP 服务账号凭据格式无效")
	}
	if cred.Type != "service_account" || cred.TokenURI != gcpTokenURL || !gcpProjectPattern.MatchString(cred.ProjectID) || !gcpEmailPattern.MatchString(cred.ClientEmail) || (cred.UniverseDomain != "" && cred.UniverseDomain != "googleapis.com") || len(cred.PrivateKeyID) > 200 {
		return gcpServiceAccount{}, errors.New("GCP 服务账号凭据字段无效")
	}
	block, rest := pem.Decode([]byte(cred.PrivateKey))
	if block == nil || block.Type != "PRIVATE KEY" || len(bytes.TrimSpace(rest)) != 0 {
		return gcpServiceAccount{}, errors.New("GCP 服务账号私钥格式无效")
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return gcpServiceAccount{}, errors.New("GCP 服务账号私钥格式无效")
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok || key.N.BitLen() < 2048 || key.N.BitLen() > 4096 || key.Validate() != nil {
		return gcpServiceAccount{}, errors.New("GCP 服务账号私钥无效")
	}
	return cred, nil
}

// Binds both OAuth and Compute HTTP requests to the caller's context, enforces
// fixed Google endpoints and prevents redirects or credential exfiltration.
type gcpSafeTransport struct {
	ctx  context.Context
	base http.RoundTripper
}

func (t gcpSafeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	u := req.URL
	if u == nil || u.Scheme != "https" || u.User != nil || u.Port() != "" || u.Fragment != "" {
		return nil, errors.New("GCP 请求目标不受信任")
	}
	switch strings.ToLower(u.Hostname()) {
	case "oauth2.googleapis.com":
		if req.Method != http.MethodPost || u.Path != "/token" || u.RawQuery != "" {
			return nil, errors.New("GCP 身份请求无效")
		}
	case "compute.googleapis.com":
		if req.Method != http.MethodGet || !strings.HasPrefix(u.Path, "/compute/v1/projects/") {
			return nil, errors.New("GCP 仅允许计算资源查询")
		}
	default:
		return nil, errors.New("GCP 请求目标不受信任")
	}
	return t.base.RoundTrip(req.Clone(t.ctx))
}

type gcpPageAPI interface {
	List(context.Context, string) (*compute.InstanceList, error)
}
type gcpSDK struct {
	client        *http.Client
	project, zone string
}

func (s gcpSDK) List(ctx context.Context, token string) (*compute.InstanceList, error) {
	endpoint := "https://compute.googleapis.com/compute/v1/projects/" + s.project + "/zones/" + s.zone + "/instances"
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, errors.New("GCP 实例请求配置无效")
	}
	query := u.Query()
	query.Set("maxResults", "500")
	if token != "" {
		query.Set("pageToken", token)
	}
	u.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, errors.New("GCP 实例请求配置无效")
	}
	resp, err := s.client.Do(req)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err != nil {
		return nil, errors.New("GCP 实例查询失败")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("GCP 实例查询失败")
	}
	// Avoid parsing unbounded remote responses. Reaching the limit is a failure,
	// never an excuse to return a partial list.
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8*1024*1024+1))
	if err != nil || len(body) > 8*1024*1024 {
		return nil, errors.New("GCP 实例响应过大或读取失败")
	}
	var page compute.InstanceList
	if err = json.Unmarshal(body, &page); err != nil {
		return nil, errors.New("GCP 实例响应格式无效")
	}
	return &page, nil
}

type GCPClient struct {
	account         gcpServiceAccount
	projectID, zone string
	base            http.RoundTripper
	newAPI          func(context.Context) (gcpPageAPI, error)
}

func (*GCPClient) String() string   { return "[REDACTED]" }
func (*GCPClient) GoString() string { return "[REDACTED]" }

// NewGCPClient accepts only a service-account key, never ADC, a file path, a
// metadata server, or an externally configured token endpoint.
func NewGCPClient(serviceAccountJSON []byte, projectID, zone string) (*GCPClient, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	return newGCPClient(serviceAccountJSON, projectID, zone, transport)
}
func newGCPClient(raw []byte, projectID, zone string, base http.RoundTripper) (*GCPClient, error) {
	if !gcpProjectPattern.MatchString(projectID) || !gcpZonePattern.MatchString(zone) || base == nil {
		return nil, errors.New("GCP 项目或可用区配置无效")
	}
	account, err := parseGCPServiceAccount(raw)
	if err != nil {
		return nil, err
	}
	c := &GCPClient{account: account, projectID: projectID, zone: zone, base: base}
	c.newAPI = c.createAPI
	return c, nil
}
func (c *GCPClient) createAPI(ctx context.Context) (gcpPageAPI, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	safe := gcpSafeTransport{ctx: ctx, base: c.base}
	plain := &http.Client{Transport: safe, Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	tokenContext := context.WithValue(ctx, oauth2.HTTPClient, plain)
	conf := jwt.Config{Email: c.account.ClientEmail, PrivateKey: []byte(c.account.PrivateKey), PrivateKeyID: c.account.PrivateKeyID, TokenURL: gcpTokenURL, Scopes: []string{gcpComputeScope}}
	httpClient := &http.Client{Transport: &oauth2.Transport{Source: conf.TokenSource(tokenContext), Base: safe}, Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	return gcpSDK{client: httpClient, project: c.projectID, zone: c.zone}, nil
}
func (c *GCPClient) VerifyCredentials(ctx context.Context) error {
	api, err := c.newAPI(ctx)
	if err != nil {
		return err
	}
	page, err := api.List(ctx, "")
	if err != nil {
		return err
	}
	if err = validateGCPPage(page); err != nil {
		return err
	}
	return ctx.Err()
}

// A verified zone is not a full project-region catalog or proof of complete IAM scope.
func (c *GCPClient) GetAvailableRegions(ctx context.Context) ([]appmodel.AvailableRegion, error) {
	if err := c.VerifyCredentials(ctx); err != nil {
		return nil, err
	}
	return []appmodel.AvailableRegion{{Region: c.zone, RegionName: c.zone, Available: true}}, nil
}
func validateGCPPage(page *compute.InstanceList) error {
	if page == nil || page.Kind != "compute#instanceList" || len(page.Items) > 500 {
		return errors.New("GCP 实例列表响应不完整")
	}
	if page.Warning != nil && (page.Warning.Code != "NO_RESULTS_ON_PAGE" || len(page.Items) != 0) {
		return errors.New("GCP 实例列表含未确认的警告")
	}
	if len(page.NextPageToken) > 4096 {
		return errors.New("GCP 实例分页标记无效")
	}
	return nil
}
func (c *GCPClient) ListInstances(ctx context.Context, ids []string) ([]*appmodel.TreeCloudResource, error) {
	if len(ids) == 0 {
		return nil, errors.New("GCP 仅支持指定实例 ID 的查询，无法确认项目权限覆盖范围")
	}
	requested := make(map[string]bool, len(ids))
	for _, id := range ids {
		n, err := strconv.ParseUint(id, 10, 64)
		if err != nil || n == 0 || strconv.FormatUint(n, 10) != id {
			return nil, errors.New("GCP 实例 ID 必须是有效数字")
		}
		requested[id] = true
	}
	if len(requested) > 1000 {
		return nil, errors.New("GCP 指定实例数量超过上限")
	}
	api, err := c.newAPI(ctx)
	if err != nil {
		return nil, err
	}
	found := map[string]*appmodel.TreeCloudResource{}
	seen := map[string]bool{}
	tokens := map[string]bool{}
	token := ""
	for pages := 0; ; pages++ {
		if pages >= 1000 {
			return nil, errors.New("GCP 实例分页超过安全上限")
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		page, err := api.List(ctx, token)
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if err != nil {
			return nil, errors.New("GCP 实例查询失败")
		}
		if err = validateGCPPage(page); err != nil {
			return nil, err
		}
		for _, instance := range page.Items {
			r, err := c.mapInstance(instance)
			if err != nil {
				return nil, err
			}
			if seen[r.InstanceID] {
				return nil, errors.New("GCP 实例分页包含重复 ID")
			}
			seen[r.InstanceID] = true
			if len(seen) > 100000 {
				return nil, errors.New("GCP 实例清单超过安全上限")
			}
			if requested[r.InstanceID] {
				found[r.InstanceID] = r
			}
		}
		if page.NextPageToken == "" {
			break
		}
		if tokens[page.NextPageToken] {
			return nil, errors.New("GCP 实例分页标记重复")
		}
		tokens[page.NextPageToken] = true
		token = page.NextPageToken
	}
	if len(found) != len(requested) {
		return nil, errors.New("部分 GCP 指定实例未找到或无读取权限")
	}
	out := make([]*appmodel.TreeCloudResource, 0, len(found))
	for _, id := range ids {
		if r, ok := found[id]; ok {
			out = append(out, r)
			delete(found, id)
		}
	}
	return out, nil
}
func (c *GCPClient) validResourceURL(raw, path string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.User != nil || u.Port() != "" || u.RawQuery != "" || u.Fragment != "" {
		return false
	}
	if u.Hostname() != "www.googleapis.com" && u.Hostname() != "compute.googleapis.com" {
		return false
	}
	return u.EscapedPath() == path
}
func (c *GCPClient) mapInstance(instance *compute.Instance) (*appmodel.TreeCloudResource, error) {
	if instance == nil || instance.Id == 0 || instance.Name == "" || len([]rune(instance.Name)) > 100 {
		return nil, errors.New("GCP 实例身份无效")
	}
	root := "/compute/v1/projects/" + c.projectID + "/zones/" + c.zone
	if !c.validResourceURL(instance.Zone, root) || !c.validResourceURL(instance.SelfLink, root+"/instances/"+instance.Name) {
		return nil, errors.New("GCP 实例不属于配置项目或可用区")
	}
	r := &appmodel.TreeCloudResource{Name: instance.Name, InstanceID: strconv.FormatUint(instance.Id, 10), ResourceType: appmodel.ResourceTypeECS, Region: c.zone, ZoneID: c.zone, Status: appmodel.CloudResourceUnknown, Currency: appmodel.CurrencyUSD}
	switch instance.Status {
	case "RUNNING":
		r.Status = appmodel.CloudResourceRunning
	case "PROVISIONING", "STAGING":
		r.Status = appmodel.CloudResourceStarting
	case "STOPPING", "SUSPENDING":
		r.Status = appmodel.CloudResourceStopping
	case "TERMINATED", "SUSPENDED":
		r.Status = appmodel.CloudResourceStopped
	}
	if instance.MachineType != "" {
		u, err := url.Parse(instance.MachineType)
		if err != nil || !c.validResourceURL(instance.MachineType, root+"/machineTypes/"+lastGCPPath(u.Path)) {
			return nil, errors.New("GCP 实例规格无效")
		}
		r.InstanceType = lastGCPPath(u.Path)
		if len([]rune(r.InstanceType)) > 100 {
			return nil, errors.New("GCP 实例规格超出存储限制")
		}
	}
	var private, public []string
	for _, iface := range instance.NetworkInterfaces {
		if iface == nil {
			continue
		}
		if iface.NetworkIP != "" {
			if net.ParseIP(iface.NetworkIP) == nil {
				return nil, errors.New("GCP 私网地址无效")
			}
			private = append(private, iface.NetworkIP)
		}
		for _, access := range iface.AccessConfigs {
			if access != nil && access.NatIP != "" {
				if net.ParseIP(access.NatIP) == nil {
					return nil, errors.New("GCP 公网地址无效")
				}
				public = append(public, access.NatIP)
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
	keys := make([]string, 0, len(instance.Labels))
	for key := range instance.Labels {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		r.Tags = append(r.Tags, appmodel.KeyValue{Key: key, Value: instance.Labels[key]})
	}
	return r, nil
}
func lastGCPPath(path string) string { parts := strings.Split(path, "/"); return parts[len(parts)-1] }
