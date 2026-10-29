package utils

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	appmodel "github.com/GoSimplicity/AI-CloudOps/internal/model"
	sdkmodel "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/ecs/v2/model"
)

type huaweiFake func(context.Context, *sdkmodel.ListServersDetailsRequest) (*sdkmodel.ListServersDetailsResponse, error)

func (f huaweiFake) ListServersDetails(ctx context.Context, r *sdkmodel.ListServersDetailsRequest) (*sdkmodel.ListServersDetailsResponse, error) {
	return f(ctx, r)
}

type huaweiRoundTrip func(*http.Request) (*http.Response, error)

func (f huaweiRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func huaweiServer(i int) sdkmodel.ServerDetail {
	return sdkmodel.ServerDetail{Id: fmt.Sprintf("00000000-0000-0000-0000-%012d", i), Name: fmt.Sprintf("vm-%d", i), Status: "ACTIVE"}
}
func huaweiResponse(total int32, s []sdkmodel.ServerDetail) *sdkmodel.ListServersDetailsResponse {
	return &sdkmodel.ListServersDetailsResponse{Count: &total, Servers: &s}
}

func TestHuaweiPaginationAndFailureSafety(t *testing.T) {
	for _, tc := range []struct {
		name    string
		mode    string
		wantErr bool
	}{
		{"完整分页", "ok", false}, {"第二页失败", "error", true}, {"总数变化", "changed", true}, {"重复实例", "duplicate", true}, {"中途空页", "empty", true}, {"响应缺字段", "missing", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			first := []sdkmodel.ServerDetail{huaweiServer(1), huaweiServer(2)}
			client := &HuaweiClient{region: "cn-north-4", client: huaweiFake(func(ctx context.Context, r *sdkmodel.ListServersDetailsRequest) (*sdkmodel.ListServersDetailsResponse, error) {
				calls++
				if calls == 1 {
					return huaweiResponse(3, first), nil
				}
				if r.Marker == nil || *r.Marker != first[1].Id {
					t.Fatal("missing marker")
				}
				switch tc.mode {
				case "error":
					return nil, errors.New("failed")
				case "changed":
					return huaweiResponse(4, []sdkmodel.ServerDetail{huaweiServer(3)}), nil
				case "duplicate":
					return huaweiResponse(3, []sdkmodel.ServerDetail{first[1]}), nil
				case "empty":
					return huaweiResponse(3, []sdkmodel.ServerDetail{}), nil
				case "missing":
					return &sdkmodel.ListServersDetailsResponse{}, nil
				}
				return huaweiResponse(3, []sdkmodel.ServerDetail{huaweiServer(3)}), nil
			})}
			got, err := client.ListInstances(context.Background(), nil)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v", err)
			}
			if tc.wantErr && got != nil {
				t.Fatal("failed listing returned partial results")
			}
			if !tc.wantErr && (len(got) != 3 || calls != 2) {
				t.Fatalf("got %d instances in %d calls", len(got), calls)
			}
		})
	}
}
func TestHuaweiSpecifiedInstances(t *testing.T) {
	ids := make([]string, 101)
	for i := range ids {
		ids[i] = huaweiServer(i + 1).Id
	}
	calls := 0
	client := &HuaweiClient{client: huaweiFake(func(_ context.Context, r *sdkmodel.ListServersDetailsRequest) (*sdkmodel.ListServersDetailsResponse, error) {
		calls++
		if r.ServerId == nil || r.Marker != nil {
			t.Fatal("unexpected filter")
		}
		parts := strings.Split(*r.ServerId, ",")
		if len(parts) > 100 {
			t.Fatal("batch too large")
		}
		servers := make([]sdkmodel.ServerDetail, len(parts))
		for i, id := range parts {
			servers[i] = sdkmodel.ServerDetail{Id: id, Name: "vm", Status: "ACTIVE"}
		}
		return huaweiResponse(int32(len(parts)), servers), nil
	})}
	got, err := client.ListInstances(context.Background(), ids)
	if err != nil || len(got) != 101 || calls != 2 {
		t.Fatalf("got %d calls %d err %v", len(got), calls, err)
	}
	client.client = huaweiFake(func(_ context.Context, _ *sdkmodel.ListServersDetailsRequest) (*sdkmodel.ListServersDetailsResponse, error) {
		return huaweiResponse(1, []sdkmodel.ServerDetail{{Id: ids[0]}}), nil
	})
	got, err = client.ListInstances(context.Background(), ids[:1])
	if err == nil || got != nil {
		t.Fatal("missing server accepted")
	}
}
func TestHuaweiMapping(t *testing.T) {
	server := huaweiServer(1)
	server.Status = "SHUTOFF"
	server.Flavor = &sdkmodel.ServerFlavor{Id: "s6.large", Vcpus: "2", Ram: "4096", Disk: "40"}
	server.Metadata = map[string]string{"vpc_id": "vpc-test", "charging_mode": "1", "os_type": "Linux"}
	server.Image = &sdkmodel.ServerImage{Id: "image-test"}
	server.OSEXTAZavailabilityZone = "az1"
	e := sdkmodel.GetServerAddressOSEXTIPStypeEnum()
	server.Addresses = map[string][]sdkmodel.ServerAddress{"net": {{Addr: "10.0.0.2", OSEXTIPStype: &e.FIXED}, {Addr: "203.0.113.4", OSEXTIPStype: &e.FLOATING}}}
	r, err := mapHuaweiServer(server, "cn-north-4")
	if err != nil {
		t.Fatal(err)
	}
	if r.Status != appmodel.CloudResourceStopped || r.Cpu != 2 || r.Memory != 4 || r.Disk != 0 || r.PublicIP != "203.0.113.4" || r.PrivateIP != "10.0.0.2" || r.ChargeType != appmodel.ChargeTypePrePaid || r.ImageID != "image-test" || r.VpcID != "vpc-test" {
		t.Fatalf("bad mapping: %+v", r)
	}
}
func TestHuaweiSDKContextAndSigningOffline(t *testing.T) {
	for _, tc := range []struct {
		name   string
		cancel bool
	}{{"签名和项目路径", false}, {"取消传递到在途HTTP请求", true}} {
		t.Run(tc.name, func(t *testing.T) {
			entered := make(chan struct{})
			sdk := huaweiSDK{accessKey: "test-ak", secretKey: "test-sk", projectID: "test-project", region: "cn-north-4", transport: huaweiRoundTrip(func(r *http.Request) (*http.Response, error) {
				if r.URL.Host != "ecs.cn-north-4.myhuaweicloud.com" || !strings.Contains(r.URL.Path, "/test-project/cloudservers/detail") || r.Header.Get("Authorization") == "" {
					t.Error("missing expected endpoint, project or signing")
				}
				close(entered)
				if tc.cancel {
					<-r.Context().Done()
					return nil, r.Context().Err()
				}
				return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"count":0,"servers":[]}`)), Request: r}, nil
			})}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			done := make(chan error, 1)
			go func() { _, err := sdk.ListServersDetails(ctx, &sdkmodel.ListServersDetailsRequest{}); done <- err }()
			<-entered
			if tc.cancel {
				cancel()
			}
			err := <-done
			if tc.cancel && !errors.Is(err, context.Canceled) {
				t.Fatalf("expected cancel, got %v", err)
			}
			if !tc.cancel && err != nil {
				t.Fatal(err)
			}
		})
	}
}
func TestHuaweiConstructorAndCanceledBeforeRequest(t *testing.T) {
	for _, input := range [][4]string{{"", "sk", "project", "cn-north-4"}, {"ak", "", "project", "cn-north-4"}, {"ak", "sk", "../evil", "cn-north-4"}, {"ak", "sk", "project", "evil.example"}} {
		if _, err := NewHuaweiClient(input[0], input[1], input[2], input[3]); err == nil {
			t.Fatal("accepted invalid configuration")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	calls := 0
	c := &HuaweiClient{client: huaweiFake(func(context.Context, *sdkmodel.ListServersDetailsRequest) (*sdkmodel.ListServersDetailsResponse, error) {
		calls++
		return nil, nil
	})}
	got, err := c.ListInstances(ctx, nil)
	if !errors.Is(err, context.Canceled) || got != nil || calls != 0 {
		t.Fatal("cancellation did not stop request")
	}
}

func TestHuaweiRegionIsOnlyTheVerifiedProjectRegion(t *testing.T) {
	for _, tc := range []struct {
		name    string
		result  *sdkmodel.ListServersDetailsResponse
		wantErr bool
	}{
		{"empty project", huaweiResponse(0, []sdkmodel.ServerDetail{}), false},
		{"missing list", &sdkmodel.ListServersDetailsResponse{}, true},
		{"count without instances", huaweiResponse(1, []sdkmodel.ServerDetail{}), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := &HuaweiClient{region: "cn-north-4", client: huaweiFake(func(_ context.Context, r *sdkmodel.ListServersDetailsRequest) (*sdkmodel.ListServersDetailsResponse, error) {
				if r.EnterpriseProjectId == nil || *r.EnterpriseProjectId != "all_granted_eps" {
					t.Error("missing enterprise scope")
				}
				return tc.result, nil
			})}
			regions, err := client.GetAvailableRegions(context.Background())
			if (err != nil) != tc.wantErr {
				t.Fatalf("got error %v", err)
			}
			if tc.wantErr && regions != nil {
				t.Fatal("failed verification returned regions")
			}
			if !tc.wantErr && (len(regions) != 1 || regions[0].Region != "cn-north-4" || !regions[0].Available) {
				t.Fatal("expected only verified region")
			}
		})
	}
}

func TestHuaweiRejectsMismatchedProject(t *testing.T) {
	server := huaweiServer(1)
	server.TenantId = "other-project"
	client := &HuaweiClient{projectID: "expected-project", client: huaweiFake(func(context.Context, *sdkmodel.ListServersDetailsRequest) (*sdkmodel.ListServersDetailsResponse, error) {
		return huaweiResponse(1, []sdkmodel.ServerDetail{server}), nil
	})}
	if rows, err := client.ListInstances(context.Background(), nil); err == nil || rows != nil {
		t.Fatal("mismatched project accepted")
	}
	if err := client.VerifyCredentials(context.Background()); err == nil {
		t.Fatal("mismatched verification project accepted")
	}
}

func TestHuaweiMappingRejectsInvalidFields(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*sdkmodel.ServerDetail)
	}{
		{"名称超限", func(s *sdkmodel.ServerDetail) { s.Name = strings.Repeat("x", 101) }},
		{"镜像超限", func(s *sdkmodel.ServerDetail) { s.Image = &sdkmodel.ServerImage{Id: strings.Repeat("x", 101)} }},
		{"无效地址", func(s *sdkmodel.ServerDetail) {
			e := sdkmodel.GetServerAddressOSEXTIPStypeEnum()
			s.Addresses = map[string][]sdkmodel.ServerAddress{"x": {{Addr: "bad", OSEXTIPStype: &e.FIXED}}}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := huaweiServer(1)
			tc.change(&s)
			if _, err := mapHuaweiServer(s, "cn-north-4"); err == nil {
				t.Fatal("expected invalid instance to be rejected")
			}
		})
	}
}

func TestHuaweiSDKSpecifiedIDAndErrorRedaction(t *testing.T) {
	id := huaweiServer(1).Id
	sdk := huaweiSDK{accessKey: "test-ak", secretKey: "test-sk", projectID: "project-1", region: "cn-north-4", transport: huaweiRoundTrip(func(r *http.Request) (*http.Response, error) {
		if r.URL.Query().Get("server_id") != id {
			t.Error("specified ID not encoded in request")
		}
		return nil, errors.New("test-secret-sensitive-error")
	})}
	_, err := sdk.ListServersDetails(context.Background(), &sdkmodel.ListServersDetailsRequest{ServerId: &id})
	if err == nil || strings.Contains(err.Error(), "test-secret-sensitive-error") {
		t.Fatal("SDK error missing or exposed upstream details")
	}
}
