package utils

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v8"
	appmodel "github.com/GoSimplicity/AI-CloudOps/internal/model"
)

const azTenant = "11111111-1111-1111-1111-111111111111"
const azClient = "22222222-2222-2222-2222-222222222222"
const azSubscription = "33333333-3333-3333-3333-333333333333"

func azPtr[T any](v T) *T { return &v }
func azVM(i int) *armcompute.VirtualMachine {
	name := fmt.Sprintf("vm-%d", i)
	id := fmt.Sprintf("00000000-0000-0000-0000-%012d", i)
	location := "eastus"
	resource := fmt.Sprintf("/subscriptions/%s/resourceGroups/rg/providers/Microsoft.Compute/virtualMachines/%s", azSubscription, name)
	return &armcompute.VirtualMachine{Name: &name, ID: &resource, Location: &location, Properties: &armcompute.VirtualMachineProperties{VMID: &id}}
}

type azureFakePager struct {
	pages   []armcompute.VirtualMachinesClientListByLocationResponse
	at      int
	failAt  int
	ctxSeen context.Context
}

func (p *azureFakePager) More() bool { return p.at < len(p.pages) }
func (p *azureFakePager) NextPage(ctx context.Context) (armcompute.VirtualMachinesClientListByLocationResponse, error) {
	p.ctxSeen = ctx
	p.at++
	if p.failAt == p.at {
		return armcompute.VirtualMachinesClientListByLocationResponse{}, errors.New("sensitive upstream")
	}
	return p.pages[p.at-1], nil
}
func azPage(v []*armcompute.VirtualMachine, next string) armcompute.VirtualMachinesClientListByLocationResponse {
	return armcompute.VirtualMachinesClientListByLocationResponse{VirtualMachineListResult: armcompute.VirtualMachineListResult{Value: v, NextLink: azPtr(next)}}
}
func azStub(p *azureFakePager) *AzureClient {
	return &AzureClient{subscriptionID: azSubscription, region: "eastus", newPager: func(string) azurePager { return p }}
}

type azTransport func(*http.Request) (*http.Response, error)

func (f azTransport) Do(req *http.Request) (*http.Response, error) { return f(req) }
func azReply(r *http.Request, body string) *http.Response {
	return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: r}
}
func azLink(page string) string {
	return "https://management.azure.com/subscriptions/" + azSubscription + "/providers/Microsoft.Compute/locations/eastus/virtualMachines?page=" + page
}

func TestAzureSpecifiedIDsAndPagination(t *testing.T) {
	for _, tc := range []struct {
		name     string
		scenario string
		wantErr  bool
	}{
		{"两页完整匹配", "ok", false}, {"后页故障", "fail", true}, {"重复ID", "duplicate", true}, {"缺失ID", "missing", true}, {"项目不符", "wrong-subscription", true}, {"地区不符", "wrong-region", true}, {"缺少VM ID", "missing-vmid", true}, {"不安全分页地址", "bad-link", true}, {"重复分页地址", "repeat-link", true}, {"响应缺少value", "missing-value", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			first, second := azVM(1), azVM(2)
			link := azLink("2")
			if tc.scenario == "duplicate" {
				second = first
			}
			if tc.scenario == "missing" {
				second = azVM(3)
			}
			if tc.scenario == "wrong-region" {
				second.Location = azPtr("westus")
			}
			if tc.scenario == "wrong-subscription" {
				second.ID = azPtr(strings.ReplaceAll(*second.ID, azSubscription, azTenant))
			}
			if tc.scenario == "missing-vmid" {
				second.Properties.VMID = nil
			}
			if tc.scenario == "bad-link" {
				link = "https://attacker.invalid/collect"
			}
			pages := []armcompute.VirtualMachinesClientListByLocationResponse{azPage([]*armcompute.VirtualMachine{first}, link), azPage([]*armcompute.VirtualMachine{second}, "")}
			if tc.scenario == "repeat-link" {
				pages[1].NextLink = azPtr(link)
				pages = append(pages, azPage([]*armcompute.VirtualMachine{}, ""))
			}
			if tc.scenario == "missing-value" {
				pages[1].Value = nil
			}
			p := &azureFakePager{pages: pages}
			if tc.scenario == "fail" {
				p.failAt = 2
			}
			c := azStub(p)
			got, err := c.ListInstances(context.Background(), []string{*azVM(1).Properties.VMID, *azVM(2).Properties.VMID})
			if (err != nil) != tc.wantErr {
				t.Fatalf("error %v", err)
			}
			if tc.wantErr && got != nil {
				t.Fatal("incomplete response returned")
			}
			if !tc.wantErr && (len(got) != 2 || got[0].Status != appmodel.CloudResourceUnknown) {
				t.Fatalf("bad result: %+v", got)
			}
			if err != nil && strings.Contains(err.Error(), "sensitive upstream") {
				t.Fatal("upstream error leaked")
			}
		})
	}
}
func TestAzureReadOnlyTransportAndNextLinks(t *testing.T) {
	calls := 0
	transport := azureSafeTransport{tenant: azTenant, base: azTransport(func(r *http.Request) (*http.Response, error) { calls++; return azReply(r, `{}`), nil })}
	for _, tc := range []struct {
		method, url string
		allowed     bool
	}{
		{"GET", "https://management.azure.com/subscriptions/x", true}, {"POST", "https://management.azure.com/subscriptions/x", false}, {"GET", "https://attacker.invalid/", false}, {"GET", "http://management.azure.com/", false}, {"POST", "https://login.microsoftonline.com/" + azTenant + "/oauth2/v2.0/token", true}, {"POST", "https://login.microsoftonline.com/" + azClient + "/oauth2/v2.0/token", false},
	} {
		req, _ := http.NewRequest(tc.method, tc.url, nil)
		_, err := transport.Do(req)
		if (err == nil) != tc.allowed {
			t.Fatalf("%s %s: %v", tc.method, tc.url, err)
		}
	}
	if calls != 2 {
		t.Fatalf("unsafe request dispatched: %d", calls)
	}
	c := azStub(&azureFakePager{})
	if !c.validNextLink(azLink("2")) || c.validNextLink("https://attacker.invalid/?token=x") || c.validNextLink(strings.ReplaceAll(azLink("2"), azSubscription, azClient)) {
		t.Fatal("invalid nextLink policy")
	}
}
func TestAzureConstructorValidationAndScope(t *testing.T) {
	transport := azTransport(func(*http.Request) (*http.Response, error) {
		t.Error("unexpected request")
		return nil, errors.New("unexpected")
	})
	for _, v := range []AzureCredentials{{TenantID: "", ClientID: azClient, ClientSecret: "x", SubscriptionID: azSubscription}, {TenantID: azTenant, ClientID: azClient, ClientSecret: "", SubscriptionID: azSubscription}} {
		if _, err := newAzureClient(v, "eastus", transport); err == nil {
			t.Fatal("accepted incomplete credentials")
		}
	}
	good := AzureCredentials{TenantID: azTenant, ClientID: azClient, ClientSecret: "secret-test", SubscriptionID: azSubscription}
	if _, err := newAzureClient(good, "../evil", transport); err == nil {
		t.Fatal("accepted invalid region")
	}
	encoded, _ := json.Marshal(good)
	if string(encoded) != "{}" {
		t.Fatal("credential JSON exposed private fields")
	}
	if fmt.Sprintf("%v %#v", good, good) != "[REDACTED] [REDACTED]" {
		t.Fatal("credential formatting exposed secret")
	}
	c := azStub(&azureFakePager{})
	if got, err := c.ListInstances(context.Background(), nil); err == nil || got != nil {
		t.Fatal("unfiltered list was accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c = azStub(&azureFakePager{pages: []armcompute.VirtualMachinesClientListByLocationResponse{azPage([]*armcompute.VirtualMachine{}, "")}})
	if got, err := c.ListInstances(ctx, []string{*azVM(1).Properties.VMID}); !errors.Is(err, context.Canceled) || got != nil {
		t.Fatal("cancellation did not propagate")
	}
}
func TestAzureCredentialsVerifyOneRegion(t *testing.T) {
	p := &azureFakePager{pages: []armcompute.VirtualMachinesClientListByLocationResponse{azPage([]*armcompute.VirtualMachine{}, "")}}
	c := azStub(p)
	regions, err := c.GetAvailableRegions(context.Background())
	if err != nil || len(regions) != 1 || regions[0].Region != "eastus" || !regions[0].Available {
		t.Fatal("expected one checked location")
	}
}

func TestAzureActualSDKUsesExplicitCredentialAndContext(t *testing.T) {
	t.Setenv("AZURE_CLIENT_ID", azTenant)
	t.Setenv("AZURE_CLIENT_SECRET", "ambient-secret")
	calls := []string{}
	transport := azTransport(func(r *http.Request) (*http.Response, error) {
		calls = append(calls, r.Method+" "+r.URL.Hostname()+r.URL.Path)
		switch {
		case r.URL.Hostname() == "login.microsoftonline.com" && strings.Contains(r.URL.Path, "openid-configuration"):
			endpoint := "https://login.microsoftonline.com/" + azTenant + "/oauth2/v2.0/token"
			return azReply(r, fmt.Sprintf(`{"token_endpoint":%q,"authorization_endpoint":"https://login.microsoftonline.com/%s/oauth2/v2.0/authorize","issuer":"https://login.microsoftonline.com/%s/v2.0"}`, endpoint, azTenant, azTenant)), nil
		case r.URL.Hostname() == "login.microsoftonline.com" && strings.HasSuffix(r.URL.Path, "/token"):
			if err := r.ParseForm(); err != nil {
				t.Fatal(err)
			}
			if r.Form.Get("client_id") != azClient || r.Form.Get("client_secret") != "fixture-secret" {
				t.Error("wrong explicit service principal")
			}
			return azReply(r, `{"access_token":"fixture-token","token_type":"Bearer","expires_in":3600}`), nil
		case r.URL.Hostname() == "management.azure.com":
			if r.Method != "GET" || r.Header.Get("Authorization") != "Bearer fixture-token" || !strings.Contains(r.URL.Path, "/locations/eastus/virtualMachines") {
				t.Error("wrong ARM request")
			}
			id := *azVM(1).Properties.VMID
			name := *azVM(1).Name
			rid := *azVM(1).ID
			return azReply(r, fmt.Sprintf(`{"value":[{"id":%q,"name":%q,"location":"eastus","properties":{"vmId":%q}}]}`, rid, name, id)), nil
		default:
			t.Errorf("unexpected request host %s", r.URL.Hostname())
			return nil, errors.New("unexpected request")
		}
	})
	c, err := newAzureClient(AzureCredentials{TenantID: azTenant, ClientID: azClient, ClientSecret: "fixture-secret", SubscriptionID: azSubscription}, "eastus", transport)
	if err != nil {
		t.Fatal(err)
	}
	result, err := c.ListInstances(context.Background(), []string{*azVM(1).Properties.VMID})
	if err != nil || len(result) != 1 {
		t.Fatalf("SDK request failed: %v, calls %v", err, calls)
	}
}

func TestAzureActualSDKCancellationReachesTransport(t *testing.T) {
	entered := make(chan struct{})
	transport := azTransport(func(r *http.Request) (*http.Response, error) {
		if strings.Contains(r.URL.Path, "openid-configuration") {
			endpoint := "https://login.microsoftonline.com/" + azTenant + "/oauth2/v2.0/token"
			return azReply(r, fmt.Sprintf(`{"token_endpoint":%q,"authorization_endpoint":"https://login.microsoftonline.com/%s/oauth2/v2.0/authorize","issuer":"https://login.microsoftonline.com/%s/v2.0"}`, endpoint, azTenant, azTenant)), nil
		}
		if strings.HasSuffix(r.URL.Path, "/token") {
			return azReply(r, `{"access_token":"fixture-token","token_type":"Bearer","expires_in":3600}`), nil
		}
		if r.URL.Hostname() == "management.azure.com" {
			close(entered)
			<-r.Context().Done()
			return nil, r.Context().Err()
		}
		return nil, errors.New("unexpected destination")
	})
	c, err := newAzureClient(AzureCredentials{TenantID: azTenant, ClientID: azClient, ClientSecret: "fixture-secret", SubscriptionID: azSubscription}, "eastus", transport)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { _, e := c.ListInstances(ctx, []string{*azVM(1).Properties.VMID}); done <- e }()
	<-entered
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("expected canceled request, got %v", err)
	}
}
