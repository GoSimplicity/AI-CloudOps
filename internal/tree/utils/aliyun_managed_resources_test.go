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
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	kv "github.com/aliyun/alibaba-cloud-sdk-go/services/r_kvstore"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/rds"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/slb"
)

type rdsFixture func(*rds.DescribeDBInstancesRequest) (*rds.DescribeDBInstancesResponse, error)

func (f rdsFixture) DescribeDBInstances(r *rds.DescribeDBInstancesRequest) (*rds.DescribeDBInstancesResponse, error) {
	return f(r)
}

type slbFixture func(*slb.DescribeLoadBalancersRequest) (*slb.DescribeLoadBalancersResponse, error)

func (f slbFixture) DescribeLoadBalancers(r *slb.DescribeLoadBalancersRequest) (*slb.DescribeLoadBalancersResponse, error) {
	return f(r)
}

type cacheFixture func(*kv.DescribeInstancesRequest) (*kv.DescribeInstancesResponse, error)

func (f cacheFixture) DescribeInstances(r *kv.DescribeInstancesRequest) (*kv.DescribeInstancesResponse, error) {
	return f(r)
}

func fakeManagedClient(t *testing.T, calls *int, failID string) *AliyunManagedResourceClient {
	t.Helper()
	c := &AliyunManagedResourceClient{region: "cn-hangzhou"}
	check := func(id, region, scheme string) {
		t.Helper()
		*calls++
		if region != "cn-hangzhou" || scheme != "https" {
			t.Fatal("wrong region or transport")
		}
	}
	c.rds = rdsFixture(func(r *rds.DescribeDBInstancesRequest) (*rds.DescribeDBInstancesResponse, error) {
		check(r.DBInstanceId, r.RegionId, r.Scheme)
		if err := requests.InitParams(r); err != nil || r.GetQueryParams()["DBInstanceId"] != r.DBInstanceId || r.GetQueryParams()["PageSize"] != "100" {
			t.Fatal("RDS SDK query contract mismatch")
		}
		if r.DBInstanceId == failID {
			return nil, errors.New("private-cloud-error-canary")
		}
		return &rds.DescribeDBInstancesResponse{PageNumber: 1, PageRecordCount: 1, TotalRecordCount: 1, Items: rds.ItemsInDescribeDBInstances{DBInstance: []rds.DBInstance{{DBInstanceId: r.DBInstanceId, RegionId: "cn-hangzhou", DBInstanceDescription: "数据库", Engine: "MySQL", EngineVersion: "8.0", DBInstanceClass: "rds.test", DBInstanceStatus: "Running"}}}}, nil
	})
	c.slb = slbFixture(func(r *slb.DescribeLoadBalancersRequest) (*slb.DescribeLoadBalancersResponse, error) {
		check(r.LoadBalancerId, r.RegionId, r.Scheme)
		if err := requests.InitParams(r); err != nil || r.GetQueryParams()["LoadBalancerId"] != r.LoadBalancerId || r.GetQueryParams()["PageSize"] != "100" {
			t.Fatal("SLB SDK query contract mismatch")
		}
		if r.LoadBalancerId == failID {
			return nil, errors.New("private-cloud-error-canary")
		}
		return &slb.DescribeLoadBalancersResponse{PageNumber: 1, PageSize: 100, TotalCount: 1, LoadBalancers: slb.LoadBalancers{LoadBalancer: []slb.LoadBalancer{{LoadBalancerId: r.LoadBalancerId, RegionId: "cn-hangzhou", LoadBalancerName: "负载均衡", LoadBalancerStatus: "active", AddressType: "intranet"}}}}, nil
	})
	c.cache = cacheFixture(func(r *kv.DescribeInstancesRequest) (*kv.DescribeInstancesResponse, error) {
		check(r.InstanceIds, r.RegionId, r.Scheme)
		if err := requests.InitParams(r); err != nil || r.GetQueryParams()["InstanceIds"] != r.InstanceIds || r.GetQueryParams()["PageSize"] != "50" {
			t.Fatal("cache SDK query contract mismatch")
		}
		if strings.HasPrefix(r.InstanceIds, "[") {
			t.Fatal("InstanceIds must be a comma-separated string, not JSON")
		}
		if r.InstanceIds == failID {
			return nil, errors.New("private-cloud-error-canary")
		}
		return &kv.DescribeInstancesResponse{PageNumber: 1, PageSize: 100, TotalCount: 1, Instances: kv.InstancesInDescribeInstances{KVStoreInstance: []kv.KVStoreInstance{{InstanceId: r.InstanceIds, RegionId: "cn-hangzhou", InstanceName: "缓存", InstanceStatus: "Normal", InstanceType: "Redis", EngineVersion: "6.0", InstanceClass: "redis.test"}}}}, nil
	})
	return c
}

func TestAliyunManagedExactLookupsAndSafeProjection(t *testing.T) {
	for _, kind := range []string{"rds", "slb", "cache"} {
		t.Run(kind, func(t *testing.T) {
			calls := 0
			c := fakeManagedClient(t, &calls, "")
			ids := []string{"id-one", "id-two"}
			var value any
			var err error
			switch kind {
			case "rds":
				value, err = c.ListRDS(context.Background(), ids)
			case "slb":
				value, err = c.ListSLB(context.Background(), ids)
			case "cache":
				value, err = c.ListCache(context.Background(), ids)
			}
			if err != nil || calls != 2 {
				t.Fatalf("calls=%d err=%v", calls, err)
			}
			encoded, err := json.Marshal(value)
			if err != nil {
				t.Fatal(err)
			}
			for _, sensitive := range []string{"password", "secret", "access_key", "connection_string", "username"} {
				if strings.Contains(strings.ToLower(string(encoded)), sensitive) {
					t.Fatal("credential field leaked")
				}
			}
			if !strings.Contains(string(encoded), "id-one") || !strings.Contains(string(encoded), "id-two") {
				t.Fatal("missing requested IDs")
			}
		})
	}
}

func TestAliyunManagedInvalidOrPartialResponsesFailClosed(t *testing.T) {
	for _, kind := range []string{"rds", "slb", "cache"} {
		t.Run(kind, func(t *testing.T) {
			calls := 0
			c := fakeManagedClient(t, &calls, "id-two")
			check := func(ids []string) (any, error) {
				switch kind {
				case "rds":
					return c.ListRDS(context.Background(), ids)
				case "slb":
					return c.ListSLB(context.Background(), ids)
				default:
					return c.ListCache(context.Background(), ids)
				}
			}
			result, err := check([]string{"id-one", "id-two"})
			if err == nil || strings.Contains(err.Error(), "private-cloud-error-canary") {
				t.Fatalf("unsafe error=%v", err)
			}
			data, _ := json.Marshal(result)
			if string(data) != "null" {
				t.Fatal("partial result escaped")
			}
			tooMany := make([]string, 101)
			for i := range tooMany {
				tooMany[i] = fmt.Sprintf("id-%d", i)
			}
			for _, ids := range [][]string{nil, {}, {"id-one", "id-one"}, {"../bad"}, {strings.Repeat("a", 101)}, tooMany} {
				before := calls
				_, err := check(ids)
				if err == nil || calls != before {
					t.Fatal("invalid ID reached provider")
				}
			}
		})
	}
}

func TestAliyunManagedRejectsMalformedPaginationAndWrongRegion(t *testing.T) {
	for _, kind := range []string{"rds", "slb", "cache"} {
		for _, mode := range []string{"empty", "extra", "wrong_id", "wrong_region", "missing_region", "page", "metadata"} {
			t.Run(kind+"/"+mode, func(t *testing.T) {
				calls := 0
				c := fakeManagedClient(t, &calls, "")
				switch kind {
				case "rds":
					base := c.rds
					c.rds = rdsFixture(func(r *rds.DescribeDBInstancesRequest) (*rds.DescribeDBInstancesResponse, error) {
						v, e := base.DescribeDBInstances(r)
						switch mode {
						case "empty":
							v.Items.DBInstance = nil
						case "extra":
							v.TotalRecordCount = 2
						case "wrong_id":
							v.Items.DBInstance[0].DBInstanceId = "other"
						case "wrong_region":
							v.Items.DBInstance[0].RegionId = "cn-other"
						case "missing_region":
							v.Items.DBInstance[0].RegionId = ""
						case "page":
							v.NextToken = "next"
						case "metadata":
							v.PageRecordCount = 0
						}
						return v, e
					})
					value, err := c.ListRDS(context.Background(), []string{"id-one"})
					if err == nil || value != nil {
						t.Fatal("invalid RDS response accepted")
					}
				case "slb":
					base := c.slb
					c.slb = slbFixture(func(r *slb.DescribeLoadBalancersRequest) (*slb.DescribeLoadBalancersResponse, error) {
						v, e := base.DescribeLoadBalancers(r)
						switch mode {
						case "empty":
							v.LoadBalancers.LoadBalancer = nil
						case "extra":
							v.TotalCount = 2
						case "wrong_id":
							v.LoadBalancers.LoadBalancer[0].LoadBalancerId = "other"
						case "wrong_region":
							v.LoadBalancers.LoadBalancer[0].RegionId = "cn-other"
						case "missing_region":
							v.LoadBalancers.LoadBalancer[0].RegionId = ""
						case "page":
							v.PageNumber = 2
						case "metadata":
							v.PageSize = 0
						}
						return v, e
					})
					value, err := c.ListSLB(context.Background(), []string{"id-one"})
					if err == nil || value != nil {
						t.Fatal("invalid SLB response accepted")
					}
				case "cache":
					base := c.cache
					c.cache = cacheFixture(func(r *kv.DescribeInstancesRequest) (*kv.DescribeInstancesResponse, error) {
						v, e := base.DescribeInstances(r)
						switch mode {
						case "empty":
							v.Instances.KVStoreInstance = nil
						case "extra":
							v.TotalCount = 2
						case "wrong_id":
							v.Instances.KVStoreInstance[0].InstanceId = "other"
						case "wrong_region":
							v.Instances.KVStoreInstance[0].RegionId = "cn-other"
						case "missing_region":
							v.Instances.KVStoreInstance[0].RegionId = ""
						case "page":
							v.PageNumber = 2
						case "metadata":
							v.PageSize = 0
						}
						return v, e
					})
					value, err := c.ListCache(context.Background(), []string{"id-one"})
					if err == nil || value != nil {
						t.Fatal("invalid cache response accepted")
					}
				}
			})
		}
	}
}

func TestAliyunManagedCancellationAndConstructor(t *testing.T) {
	calls := 0
	c := fakeManagedClient(t, &calls, "")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := c.ListRDS(ctx, []string{"id-one"}); !errors.Is(err, context.Canceled) || calls != 0 {
		t.Fatal("cancellation reached provider")
	}
	ctx, cancel = context.WithCancel(context.Background())
	base := c.cache
	c.cache = cacheFixture(func(r *kv.DescribeInstancesRequest) (*kv.DescribeInstancesResponse, error) {
		v, e := base.DescribeInstances(r)
		cancel()
		return v, e
	})
	if result, err := c.ListCache(ctx, []string{"id-one"}); !errors.Is(err, context.Canceled) || result != nil {
		t.Fatal("cancelled result escaped")
	}
	ctx, cancel = context.WithCancel(context.Background())
	c.rds = rdsFixture(func(*rds.DescribeDBInstancesRequest) (*rds.DescribeDBInstancesResponse, error) {
		cancel()
		return nil, errors.New("private fixture")
	})
	if result, err := c.ListRDS(ctx, []string{"id-one"}); !errors.Is(err, context.Canceled) || result != nil {
		t.Fatal("cancelled provider error was not classified")
	}
	if _, err := NewAliyunManagedResourceClient(CloudCredentials{AccessKey: "test", SecretKey: "test"}, "../invalid"); err == nil {
		t.Fatal("invalid region accepted")
	}
	if _, err := NewAliyunManagedResourceClient(CloudCredentials{AccessKey: "test", SecretKey: "test", CredentialType: "temporary", SessionToken: "test", ExpiresAt: func() *time.Time { v := time.Now().Add(time.Hour); return &v }()}, "cn-hangzhou"); err == nil {
		t.Fatal("unsupported temporary credentials accepted")
	}
}

type managedRoundTrip func(*http.Request) (*http.Response, error)

func (f managedRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestAliyunManagedRealSDKSerializationWithInMemoryTransport(t *testing.T) {
	client, err := NewAliyunManagedResourceClient(CloudCredentials{AccessKey: "test", SecretKey: "test"}, "cn-hangzhou")
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	fake := managedRoundTrip(func(req *http.Request) (*http.Response, error) {
		calls++
		if err := req.ParseForm(); err != nil {
			return nil, errors.New("bad request encoding")
		}
		if req.Form.Get("RegionId") != "cn-hangzhou" {
			return nil, errors.New("region missing")
		}
		var body string
		switch req.Form.Get("Action") {
		case "DescribeDBInstances":
			if req.Form.Get("DBInstanceId") != "rm-test" || req.Form.Get("PageSize") != "100" {
				return nil, errors.New("RDS request mismatch")
			}
			body = `{"PageNumber":1,"PageRecordCount":1,"TotalRecordCount":1,"Items":{"DBInstance":[{"DBInstanceId":"rm-test","RegionId":"cn-hangzhou","Engine":"MySQL"}]}}`
		case "DescribeLoadBalancers":
			if req.Form.Get("LoadBalancerId") != "lb-test" || req.Form.Get("PageSize") != "100" {
				return nil, errors.New("SLB request mismatch")
			}
			body = `{"PageNumber":1,"PageSize":100,"TotalCount":1,"LoadBalancers":{"LoadBalancer":[{"LoadBalancerId":"lb-test","RegionId":"cn-hangzhou"}]}}`
		case "DescribeInstances":
			if req.Form.Get("InstanceIds") != "r-test" || req.Form.Get("PageSize") != "50" {
				return nil, errors.New("cache request mismatch")
			}
			body = `{"PageNumber":1,"PageSize":50,"TotalCount":1,"Instances":{"KVStoreInstance":[{"InstanceId":"r-test","RegionId":"cn-hangzhou","InstanceType":"Redis"}]}}`
		default:
			return nil, errors.New("unexpected API action")
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
	})
	client.rds.(*rds.Client).SetTransport(fake)
	client.slb.(*slb.Client).SetTransport(fake)
	client.cache.(*kv.Client).SetTransport(fake)
	if values, err := client.ListRDS(context.Background(), []string{"rm-test"}); err != nil || len(values) != 1 {
		t.Fatalf("RDS response failed: %v", err)
	}
	if values, err := client.ListSLB(context.Background(), []string{"lb-test"}); err != nil || len(values) != 1 {
		t.Fatalf("SLB response failed: %v", err)
	}
	if values, err := client.ListCache(context.Background(), []string{"r-test"}); err != nil || len(values) != 1 {
		t.Fatalf("cache response failed: %v", err)
	}
	if calls != 3 {
		t.Fatalf("SDK calls=%d", calls)
	}
}
