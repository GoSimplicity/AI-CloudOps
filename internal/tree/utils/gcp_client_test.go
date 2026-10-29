package utils

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	appmodel "github.com/GoSimplicity/AI-CloudOps/internal/model"
	"google.golang.org/api/compute/v1"
)

const gcpProject = "sample-project"
const gcpZone = "us-central1-a"
const gcpEmail = "robot@sample-project.iam.gserviceaccount.com"

func gcpFixtureKey(t *testing.T) []byte {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: raw})
}
func gcpFixtureCredentials(t *testing.T) []byte {
	t.Helper()
	value := map[string]string{"type": "service_account", "project_id": gcpProject, "client_email": gcpEmail, "private_key": string(gcpFixtureKey(t)), "private_key_id": "fixture-key-id", "token_uri": gcpTokenURL, "universe_domain": "googleapis.com"}
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
func gcpInstance(id uint64) *compute.Instance {
	name := fmt.Sprintf("vm-%d", id)
	root := "https://www.googleapis.com/compute/v1/projects/" + gcpProject + "/zones/" + gcpZone
	return &compute.Instance{Id: id, Name: name, Status: "RUNNING", Zone: root, SelfLink: root + "/instances/" + name}
}

type gcpFake func(context.Context, string) (*compute.InstanceList, error)

func (f gcpFake) List(ctx context.Context, token string) (*compute.InstanceList, error) {
	return f(ctx, token)
}

type gcpRT func(*http.Request) (*http.Response, error)

func (f gcpRT) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }
func gcpPage(instances []*compute.Instance, next string) *compute.InstanceList {
	return &compute.InstanceList{Kind: "compute#instanceList", Items: instances, NextPageToken: next}
}
func gcpStub(api gcpPageAPI) *GCPClient {
	return &GCPClient{projectID: gcpProject, zone: gcpZone, newAPI: func(context.Context) (gcpPageAPI, error) { return api, nil }}
}
func gcpHTTPReply(r *http.Request, body string) *http.Response {
	return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: r}
}

func TestGCPPaginationAndSpecifiedIDs(t *testing.T) {
	for _, tc := range []struct {
		name, mode string
		wantErr    bool
	}{
		{"完整两页", "ok", false}, {"后页失败", "error", true}, {"重复实例", "duplicate", true}, {"缺少指定实例", "missing", true}, {"跨项目", "project", true}, {"跨可用区", "zone", true}, {"重复分页标记", "repeat", true}, {"警告表示不完整", "warning", true}, {"响应缺失kind", "missing-kind", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			first, second := gcpInstance(1), gcpInstance(2)
			if tc.mode == "duplicate" {
				second = first
			}
			if tc.mode == "missing" {
				second = gcpInstance(3)
			}
			if tc.mode == "project" {
				second.SelfLink = strings.Replace(second.SelfLink, gcpProject, "other-project", 1)
			}
			if tc.mode == "zone" {
				second.Zone = strings.Replace(second.Zone, gcpZone, "us-central1-b", 1)
			}
			calls := 0
			client := gcpStub(gcpFake(func(_ context.Context, token string) (*compute.InstanceList, error) {
				calls++
				if calls == 1 {
					return gcpPage([]*compute.Instance{first}, "next"), nil
				}
				if token != "next" {
					t.Fatal("missing page token")
				}
				switch tc.mode {
				case "error":
					return nil, errors.New("secret-canary")
				case "repeat":
					return gcpPage([]*compute.Instance{second}, "next"), nil
				case "warning":
					p := gcpPage(nil, "")
					p.Warning = &compute.InstanceListWarning{Code: "UNREACHABLE"}
					return p, nil
				case "missing-kind":
					return &compute.InstanceList{}, nil
				}
				return gcpPage([]*compute.Instance{second}, ""), nil
			}))
			result, err := client.ListInstances(context.Background(), []string{"1", "2"})
			if (err != nil) != tc.wantErr {
				t.Fatalf("error %v", err)
			}
			if tc.wantErr && result != nil {
				t.Fatal("failure returned partial results")
			}
			if !tc.wantErr && (len(result) != 2 || result[0].Status != appmodel.CloudResourceRunning) {
				t.Fatal("unexpected result")
			}
			if err != nil && strings.Contains(err.Error(), "secret-canary") {
				t.Fatal("error leaked")
			}
		})
	}
}
func TestGCPServiceAccountValidation(t *testing.T) {
	valid := gcpFixtureCredentials(t)
	base := map[string]any{}
	if err := json.Unmarshal(valid, &base); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, key string
		value     any
	}{
		{"非服务账号", "type", "external_account"}, {"不可信令牌地址", "token_uri", "https://attacker.invalid/token"}, {"额外外部来源", "credential_source", map[string]string{"file": "/etc/passwd"}}, {"未知宇宙域", "universe_domain", "attacker.invalid"}, {"无效私钥", "private_key", "not-a-private-key"}, {"冒用邮箱", "client_email", "evil@attacker.invalid"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			copied := map[string]any{}
			for k, v := range base {
				copied[k] = v
			}
			copied[tc.key] = tc.value
			raw, _ := json.Marshal(copied)
			if _, err := parseGCPServiceAccount(raw); err == nil || strings.Contains(err.Error(), "attacker.invalid") {
				t.Fatal("accepted or exposed invalid credentials")
			}
		})
	}
	parsed, err := parseGCPServiceAccount(valid)
	if err != nil {
		t.Fatal(err)
	}
	redacted, err := json.Marshal(parsed)
	if err != nil || string(redacted) != "{}" || fmt.Sprintf("%v %#v", parsed, parsed) != "[REDACTED] [REDACTED]" {
		t.Fatal("service account exposed by formatting")
	}
}
func TestGCPTransportPolicy(t *testing.T) {
	calls := 0
	transport := gcpSafeTransport{ctx: context.Background(), base: gcpRT(func(r *http.Request) (*http.Response, error) { calls++; return gcpHTTPReply(r, `{}`), nil })}
	for _, tc := range []struct {
		method, url string
		allowed     bool
	}{
		{"POST", gcpTokenURL, true}, {"GET", "https://compute.googleapis.com/compute/v1/projects/a", true}, {"POST", "https://compute.googleapis.com/compute/v1/projects/a", false}, {"POST", "https://attacker.invalid/token", false}, {"GET", "http://compute.googleapis.com/compute/v1/projects/a", false}, {"POST", "https://oauth2.googleapis.com/not-token", false},
	} {
		req, _ := http.NewRequest(tc.method, tc.url, nil)
		_, err := transport.RoundTrip(req)
		if (err == nil) != tc.allowed {
			t.Fatalf("%s %s error %v", tc.method, tc.url, err)
		}
	}
	if calls != 2 {
		t.Fatal("blocked request reached transport")
	}
}
func TestGCPMappingAndUnavailableScope(t *testing.T) {
	instance := gcpInstance(1)
	instance.MachineType = instance.Zone + "/machineTypes/e2-medium"
	instance.NetworkInterfaces = []*compute.NetworkInterface{{NetworkIP: "10.0.0.2", AccessConfigs: []*compute.AccessConfig{{NatIP: "203.0.113.4"}}}}
	instance.Labels = map[string]string{"app": "demo"}
	c := gcpStub(nil)
	r, err := c.mapInstance(instance)
	if err != nil || r.InstanceType != "e2-medium" || r.PrivateIP != "10.0.0.2" || r.PublicIP != "203.0.113.4" || r.Cpu != 0 || r.Memory != 0 || r.Disk != 0 {
		t.Fatalf("bad mapping: %+v, %v", r, err)
	}
	if got, err := c.ListInstances(context.Background(), nil); err == nil || got != nil {
		t.Fatal("unfiltered list accepted")
	}
	for _, id := range []string{"0", "-1", "01", "1,2"} {
		if got, err := c.ListInstances(context.Background(), []string{id}); err == nil || got != nil {
			t.Fatal("bad id accepted")
		}
	}
}
func TestGCPActualSDKUsesFixedCredentialsAndCancel(t *testing.T) {
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "/nonexistent/private-canary")
	t.Setenv("GOOGLE_API_USE_CLIENT_CERTIFICATE", "true")
	t.Setenv("GOOGLE_API_USE_MTLS_ENDPOINT", "always")
	for _, cancelRequest := range []bool{false, true} {
		t.Run(fmt.Sprintf("cancel=%v", cancelRequest), func(t *testing.T) {
			calls := 0
			entered := make(chan struct{})
			rt := gcpRT(func(r *http.Request) (*http.Response, error) {
				calls++
				switch r.URL.Hostname() {
				case "oauth2.googleapis.com":
					if r.Method != "POST" || r.URL.String() != gcpTokenURL {
						t.Error("wrong token endpoint")
					}
					if err := r.ParseForm(); err != nil {
						t.Fatal(err)
					}
					assertion := strings.Split(r.Form.Get("assertion"), ".")
					if len(assertion) != 3 {
						t.Error("missing signed assertion")
						return nil, errors.New("invalid assertion")
					}
					claimsJSON, err := base64.RawURLEncoding.DecodeString(assertion[1])
					if err != nil {
						t.Fatal(err)
					}
					var claims map[string]any
					if err := json.Unmarshal(claimsJSON, &claims); err != nil {
						t.Fatal(err)
					}
					if claims["iss"] != gcpEmail || claims["aud"] != gcpTokenURL || claims["scope"] != gcpComputeScope {
						t.Error("unexpected assertion claims")
					}
					return gcpHTTPReply(r, `{"access_token":"fixture-token","token_type":"Bearer","expires_in":3600}`), nil
				case "compute.googleapis.com":
					if r.Method != "GET" || r.Header.Get("Authorization") != "Bearer fixture-token" || !strings.Contains(r.URL.Path, "/projects/"+gcpProject+"/zones/"+gcpZone+"/instances") {
						t.Error("wrong compute request")
					}
					if cancelRequest {
						close(entered)
						<-r.Context().Done()
						return nil, r.Context().Err()
					}
					inst := gcpInstance(1)
					return gcpHTTPReply(r, fmt.Sprintf(`{"kind":"compute#instanceList","items":[{"id":"1","name":%q,"status":"RUNNING","zone":%q,"selfLink":%q}]}`, inst.Name, inst.Zone, inst.SelfLink)), nil
				default:
					t.Error("unexpected host")
					return nil, errors.New("unexpected")
				}
			})
			c, err := newGCPClient(gcpFixtureCredentials(t), gcpProject, gcpZone, rt)
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			done := make(chan error, 1)
			go func() {
				rows, e := c.ListInstances(ctx, []string{"1"})
				if !cancelRequest && e == nil && len(rows) != 1 {
					e = errors.New("unexpected result count")
				}
				done <- e
			}()
			if cancelRequest {
				select {
				case <-entered:
					cancel()
				case err := <-done:
					t.Fatalf("request failed before compute: %v", err)
				case <-time.After(5 * time.Second):
					t.Fatal("compute request not reached")
				}
			}
			err = <-done
			if cancelRequest && !errors.Is(err, context.Canceled) {
				t.Fatalf("expected context cancellation, got %v", err)
			}
			if !cancelRequest && err != nil {
				t.Fatalf("SDK request failed: %v", err)
			}
			if calls != 2 {
				t.Fatalf("expected token and compute calls, got %d", calls)
			}
		})
	}
}

func TestGCPTokenRequestCancellation(t *testing.T) {
	entered := make(chan struct{})
	rt := gcpRT(func(r *http.Request) (*http.Response, error) {
		if r.URL.Hostname() != "oauth2.googleapis.com" {
			t.Error("unexpected request")
		}
		close(entered)
		<-r.Context().Done()
		return nil, r.Context().Err()
	})
	c, err := newGCPClient(gcpFixtureCredentials(t), gcpProject, gcpZone, rt)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { _, err := c.ListInstances(ctx, []string{"1"}); done <- err }()
	select {
	case <-entered:
		cancel()
	case err := <-done:
		t.Fatalf("request failed before token exchange: %v", err)
	case <-time.After(5 * time.Second):
		cancel()
		t.Fatal("token request not reached")
	}
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
}

func TestGCPRealRESTRequestsCarryPaginationToken(t *testing.T) {
	calls := 0
	transport := gcpRT(func(r *http.Request) (*http.Response, error) {
		if r.URL.Hostname() == "oauth2.googleapis.com" {
			return gcpHTTPReply(r, `{"access_token":"fixture-token","token_type":"Bearer","expires_in":3600}`), nil
		}
		if r.URL.Hostname() != "compute.googleapis.com" {
			t.Error("unexpected host")
			return nil, errors.New("unexpected")
		}
		calls++
		if r.URL.Query().Get("maxResults") != "500" {
			t.Error("missing page limit")
		}
		if calls == 1 && r.URL.Query().Get("pageToken") != "" {
			t.Error("unexpected first page token")
		}
		if calls == 2 && r.URL.Query().Get("pageToken") != "next page" {
			t.Error("missing next page token")
		}
		inst := gcpInstance(uint64(calls))
		next := ""
		if calls == 1 {
			next = `,"nextPageToken":"next page"`
		}
		return gcpHTTPReply(r, fmt.Sprintf(`{"kind":"compute#instanceList","items":[{"id":"%d","name":%q,"status":"RUNNING","zone":%q,"selfLink":%q}]%s}`, calls, inst.Name, inst.Zone, inst.SelfLink, next)), nil
	})
	c, err := newGCPClient(gcpFixtureCredentials(t), gcpProject, gcpZone, transport)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := c.ListInstances(context.Background(), []string{"1", "2"})
	if err != nil || len(rows) != 2 || calls != 2 {
		t.Fatalf("unexpected pagination result: %v, calls %d", err, calls)
	}
}
