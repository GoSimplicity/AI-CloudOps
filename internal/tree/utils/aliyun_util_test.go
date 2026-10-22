package utils

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"go.uber.org/zap"
)

type ecsStub struct {
	instances func(*ecs.DescribeInstancesRequest) (*ecs.DescribeInstancesResponse, error)
	regions   func(*ecs.DescribeRegionsRequest) (*ecs.DescribeRegionsResponse, error)
}

func (s ecsStub) DescribeInstances(r *ecs.DescribeInstancesRequest) (*ecs.DescribeInstancesResponse, error) {
	return s.instances(r)
}
func (s ecsStub) DescribeRegions(r *ecs.DescribeRegionsRequest) (*ecs.DescribeRegionsResponse, error) {
	return s.regions(r)
}
func instancePage(id, next string) *ecs.DescribeInstancesResponse {
	return &ecs.DescribeInstancesResponse{NextToken: next, TotalCount: 0, Instances: ecs.InstancesInDescribeInstances{Instance: []ecs.Instance{{InstanceId: id}}}}
}

func TestAliyunTokenPagination(t *testing.T) {
	for _, tt := range []struct {
		name         string
		repeat, fail bool
		invalid      bool
	}{
		{"short page with zero total continues", false, false, false},
		{"repeated token fails closed", true, false, true},
		{"later page failure returns no partial snapshot", false, true, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			client := &AliyunClient{region: "cn-hangzhou", logger: zap.NewNop(), client: ecsStub{instances: func(r *ecs.DescribeInstancesRequest) (*ecs.DescribeInstancesResponse, error) {
				calls++
				if string(r.MaxResults) != "100" || r.PageNumber != "" || r.PageSize != "" || r.RegionId != "cn-hangzhou" || r.Scheme != "https" {
					t.Fatal("incorrect pagination/region/transport contract")
				}
				if calls == 1 {
					return instancePage("i-one", "next"), nil
				}
				if r.NextToken != "next" {
					t.Fatal("next token not forwarded")
				}
				if tt.fail {
					return nil, errors.New("provider failure")
				}
				if tt.repeat {
					return instancePage("i-two", "next"), nil
				}
				return instancePage("i-two", ""), nil
			}}}
			resources, err := client.ListECSInstances(context.Background(), nil)
			if (err != nil) != tt.invalid {
				t.Fatalf("unexpected error %v", err)
			}
			if tt.invalid && resources != nil {
				t.Fatal("partial snapshot escaped on failure")
			}
			if !tt.invalid && (len(resources) != 2 || calls != 2) {
				t.Fatalf("resources=%d calls=%d", len(resources), calls)
			}
		})
	}
}

func TestAliyunInstanceFilterBatching(t *testing.T) {
	ids := make([]string, 101)
	for i := range ids {
		ids[i] = fmt.Sprintf("i-%d", i)
	}
	ids[0] = `i-quote"escaped`
	calls := 0
	received := []string{}
	client := &AliyunClient{region: "cn-hangzhou", logger: zap.NewNop(), client: ecsStub{instances: func(r *ecs.DescribeInstancesRequest) (*ecs.DescribeInstancesResponse, error) {
		calls++
		var batch []string
		if err := json.Unmarshal([]byte(r.InstanceIds), &batch); err != nil {
			t.Fatal("invalid InstanceIds JSON")
		}
		if len(batch) > 100 {
			t.Fatal("provider filter limit exceeded")
		}
		received = append(received, batch...)
		return &ecs.DescribeInstancesResponse{}, nil
	}}}
	if _, err := client.ListECSInstances(context.Background(), ids); err != nil {
		t.Fatal(err)
	}
	if calls != 2 || len(received) != 101 || received[0] != ids[0] {
		t.Fatal("filters lost or not split into batches")
	}
}

func TestAliyunCredentialVerificationUsesRequiredECSRead(t *testing.T) {
	calls := 0
	client := &AliyunClient{region: "cn-hangzhou", logger: zap.NewNop(), client: ecsStub{instances: func(r *ecs.DescribeInstancesRequest) (*ecs.DescribeInstancesResponse, error) {
		calls++
		if string(r.MaxResults) != "10" {
			t.Fatal("verification read must be bounded")
		}
		return &ecs.DescribeInstancesResponse{}, nil
	}}}
	if err := client.VerifyCredentials(context.Background()); err != nil {
		t.Fatal(err)
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := client.VerifyCredentials(canceled); !errors.Is(err, context.Canceled) {
		t.Fatal("canceled request was accepted")
	}
	if calls != 1 {
		t.Fatal("canceled verification contacted provider")
	}
}

func TestAliyunRegionAvailability(t *testing.T) {
	client := &AliyunClient{logger: zap.NewNop(), client: ecsStub{regions: func(r *ecs.DescribeRegionsRequest) (*ecs.DescribeRegionsResponse, error) {
		if r.ResourceType != "instance" || r.AcceptLanguage != "zh-CN" {
			t.Fatal("unexpected region request")
		}
		return &ecs.DescribeRegionsResponse{Regions: ecs.Regions{Region: []ecs.Region{{RegionId: "one", Status: "available"}, {RegionId: "two", Status: "soldOut"}}}}, nil
	}}}
	regions, err := client.GetAvailableRegions(context.Background())
	if err != nil || len(regions) != 2 || !regions[0].Available || regions[1].Available {
		t.Fatalf("incorrect region availability: %v", err)
	}
}

func TestAliyunExpiryAndEIP(t *testing.T) {
	for _, expiry := range []string{"2026-10-22T09:30:00Z", "2026-10-22T09:30Z"} {
		client := &AliyunClient{region: "cn-hangzhou"}
		resource := client.convertECSToResource(&ecs.Instance{InstanceChargeType: "PrePaid", ExpiredTime: expiry, EipAddress: ecs.EipAddressInDescribeInstances{IpAddress: "192.0.2.1"}})
		if resource.ExpireTime == nil || resource.ExpireTime.Hour() != 9 || resource.PublicIP != "192.0.2.1" {
			t.Fatal("lost expiry or elastic public IP")
		}
	}
	req := ecs.CreateDescribeInstancesRequest()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := prepareAliyunRequest(ctx, req); err != nil {
		t.Fatal(err)
	}
	if req.GetReadTimeout() > time.Second || req.GetConnectTimeout() > time.Second {
		t.Fatal("request timeout exceeds caller deadline")
	}
}
