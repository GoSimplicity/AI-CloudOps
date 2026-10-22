package utils

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
	regionapi "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/region/v20220627"
)

type awsStub struct {
	instances func(context.Context, *ec2.DescribeInstancesInput) (*ec2.DescribeInstancesOutput, error)
	regions   func(context.Context, *ec2.DescribeRegionsInput) (*ec2.DescribeRegionsOutput, error)
}

func (s awsStub) DescribeInstances(ctx context.Context, in *ec2.DescribeInstancesInput, _ ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error) {
	return s.instances(ctx, in)
}
func (s awsStub) DescribeRegions(ctx context.Context, in *ec2.DescribeRegionsInput, _ ...func(*ec2.Options)) (*ec2.DescribeRegionsOutput, error) {
	return s.regions(ctx, in)
}
func awsPage(id, next string) *ec2.DescribeInstancesOutput {
	return &ec2.DescribeInstancesOutput{NextToken: aws.String(next), Reservations: []types.Reservation{{Instances: []types.Instance{{InstanceId: aws.String(id)}}}}}
}

type tencentStub struct {
	instances func(context.Context, *cvm.DescribeInstancesRequest) (*cvm.DescribeInstancesResponse, error)
	regions   func(context.Context, *regionapi.DescribeRegionsRequest) (*regionapi.DescribeRegionsResponse, error)
}

func (s tencentStub) DescribeInstancesWithContext(ctx context.Context, in *cvm.DescribeInstancesRequest) (*cvm.DescribeInstancesResponse, error) {
	return s.instances(ctx, in)
}
func (s tencentStub) DescribeRegionsWithContext(ctx context.Context, in *regionapi.DescribeRegionsRequest) (*regionapi.DescribeRegionsResponse, error) {
	return s.regions(ctx, in)
}
func tencentPage(total int64, ids ...string) *cvm.DescribeInstancesResponse {
	page := &cvm.DescribeInstancesResponseParams{TotalCount: common.Int64Ptr(total)}
	for _, id := range ids {
		page.InstanceSet = append(page.InstanceSet, &cvm.Instance{InstanceId: common.StringPtr(id)})
	}
	return &cvm.DescribeInstancesResponse{Response: page}
}

func TestAWSPaginationAndFiltering(t *testing.T) {
	for _, filtered := range []bool{false, true} {
		t.Run(fmt.Sprintf("filtered=%v", filtered), func(t *testing.T) {
			calls := 0
			var ids []string
			if filtered {
				ids = []string{"i-one", "i-two"}
			}
			c := &AWSClient{region: "us-east-1", client: awsStub{instances: func(_ context.Context, in *ec2.DescribeInstancesInput) (*ec2.DescribeInstancesOutput, error) {
				calls++
				if filtered && in.MaxResults != nil {
					t.Fatal("AWS forbids IDs with MaxResults")
				}
				if !filtered && aws.ToInt32(in.MaxResults) != 1000 {
					t.Fatal("unbounded listing")
				}
				if calls == 1 {
					return awsPage("i-one", "next"), nil
				}
				if aws.ToString(in.NextToken) != "next" {
					t.Fatal("token lost")
				}
				return awsPage("i-two", ""), nil
			}}}
			result, err := c.ListInstances(context.Background(), ids)
			if err != nil || len(result) != 2 || calls != 2 {
				t.Fatalf("calls=%d len=%d err=%v", calls, len(result), err)
			}
		})
	}
}

func TestAWSFailedPagesDiscardSnapshot(t *testing.T) {
	for _, mode := range []string{"denied", "repeated", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			calls := 0
			c := &AWSClient{client: awsStub{instances: func(context.Context, *ec2.DescribeInstancesInput) (*ec2.DescribeInstancesOutput, error) {
				calls++
				if calls == 1 {
					return awsPage("i-one", "next"), nil
				}
				switch mode {
				case "denied":
					return nil, errors.New("UnauthorizedOperation")
				case "cancel":
					cancel()
					return awsPage("i-two", ""), nil
				default:
					return awsPage("i-one", "next"), nil
				}
			}}}
			result, err := c.ListInstances(ctx, nil)
			if err == nil || result != nil {
				t.Fatal("partial AWS snapshot escaped")
			}
		})
	}
}

func TestTencentPaginationAndFiltering(t *testing.T) {
	for _, filtered := range []bool{false, true} {
		t.Run(fmt.Sprintf("filtered=%v", filtered), func(t *testing.T) {
			calls := 0
			var ids []string
			if filtered {
				for i := 0; i < 101; i++ {
					ids = append(ids, fmt.Sprintf("ins-%d", i))
				}
			}
			c := &TencentClient{region: "ap-guangzhou", client: tencentStub{instances: func(_ context.Context, in *cvm.DescribeInstancesRequest) (*cvm.DescribeInstancesResponse, error) {
				calls++
				if deref(in.Limit) != 100 || len(in.InstanceIds) > 100 {
					t.Fatal("invalid CVM limit")
				}
				if filtered {
					resp := tencentPage(int64(len(in.InstanceIds)))
					for _, id := range in.InstanceIds {
						resp.Response.InstanceSet = append(resp.Response.InstanceSet, &cvm.Instance{InstanceId: id})
					}
					return resp, nil
				}
				if deref(in.Offset) == 0 {
					return tencentPage(2, "ins-one"), nil
				}
				if deref(in.Offset) != 1 {
					t.Fatal("offset must advance by returned count")
				}
				return tencentPage(2, "ins-two"), nil
			}}}
			result, err := c.ListInstances(context.Background(), ids)
			want := 2
			if filtered {
				want = 101
			}
			if err != nil || len(result) != want || calls != 2 {
				t.Fatalf("calls=%d len=%d err=%v", calls, len(result), err)
			}
		})
	}
}

func TestTencentIncompleteListingFailsClosed(t *testing.T) {
	for _, mode := range []string{"empty page", "permission", "cancel", "duplicate", "changed total"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			calls := 0
			c := &TencentClient{client: tencentStub{instances: func(context.Context, *cvm.DescribeInstancesRequest) (*cvm.DescribeInstancesResponse, error) {
				calls++
				if calls == 1 {
					return tencentPage(2, "ins-one"), nil
				}
				if mode == "permission" {
					return nil, errors.New("AuthFailure")
				}
				if mode == "duplicate" {
					return tencentPage(2, "ins-one"), nil
				}
				if mode == "changed total" {
					return tencentPage(3, "ins-two"), nil
				}
				if mode == "cancel" {
					cancel()
				}
				return tencentPage(2), nil
			}}}
			result, err := c.ListInstances(ctx, nil)
			if err == nil || result != nil {
				t.Fatal("partial CVM snapshot escaped")
			}
		})
	}
}

func TestProviderVerificationPermissionAndCancellation(t *testing.T) {
	denied := errors.New("permission denied")
	clients := []CloudComputeClient{
		&AWSClient{client: awsStub{instances: func(_ context.Context, in *ec2.DescribeInstancesInput) (*ec2.DescribeInstancesOutput, error) {
			if aws.ToInt32(in.MaxResults) != 5 {
				t.Fatal("AWS validation read not bounded")
			}
			return nil, denied
		}}},
		&TencentClient{client: tencentStub{instances: func(_ context.Context, in *cvm.DescribeInstancesRequest) (*cvm.DescribeInstancesResponse, error) {
			if deref(in.Limit) != 1 {
				t.Fatal("CVM validation read not bounded")
			}
			return nil, denied
		}}},
	}
	for _, client := range clients {
		if err := client.VerifyCredentials(context.Background()); !errors.Is(err, denied) {
			t.Fatal("authorization failure hidden")
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if err := client.VerifyCredentials(ctx); !errors.Is(err, context.Canceled) {
			t.Fatal("canceled verification contacted provider")
		}
		if _, err := client.ListInstances(ctx, nil); !errors.Is(err, context.Canceled) {
			t.Fatal("canceled listing contacted provider")
		}
	}
	if _, err := NewAWSClient("ASIA-temporary", "secret", "us-east-1"); err == nil {
		t.Fatal("temporary credentials accepted without session token")
	}
}

func TestProviderRegionsAndMappings(t *testing.T) {
	awsClient := &AWSClient{region: "us-east-1", client: awsStub{regions: func(_ context.Context, in *ec2.DescribeRegionsInput) (*ec2.DescribeRegionsOutput, error) {
		if !aws.ToBool(in.AllRegions) {
			t.Fatal("opt-in state cannot be distinguished")
		}
		return &ec2.DescribeRegionsOutput{Regions: []types.Region{{RegionName: aws.String("us-east-1"), OptInStatus: aws.String("opt-in-not-required")}, {RegionName: aws.String("ap-east-1"), OptInStatus: aws.String("not-opted-in")}}}, nil
	}}}
	regions, err := awsClient.GetAvailableRegions(context.Background())
	if err != nil || len(regions) != 2 || !regions[0].Available || regions[1].Available {
		t.Fatal("AWS opt-in mapping failed")
	}
	awsResource := awsClient.convertInstance(types.Instance{InstanceId: aws.String("i-one"), Tags: []types.Tag{{Key: aws.String("Name"), Value: aws.String("example")}}, CpuOptions: &types.CpuOptions{CoreCount: aws.Int32(2), ThreadsPerCore: aws.Int32(2)}, State: &types.InstanceState{Name: types.InstanceStateNameRunning}, Platform: types.PlatformValuesWindows})
	if awsResource.Name != "example" || awsResource.Cpu != 4 || awsResource.Memory != 0 || awsResource.Status != model.CloudResourceRunning || awsResource.OSType != "windows" {
		t.Fatal("AWS mapping invented or lost metadata")
	}
	tc := &TencentClient{region: "ap-guangzhou", regions: tencentStub{regions: func(_ context.Context, in *regionapi.DescribeRegionsRequest) (*regionapi.DescribeRegionsResponse, error) {
		if deref(in.Product) != "cvm" {
			t.Fatal("must query current region API for CVM")
		}
		return &regionapi.DescribeRegionsResponse{Response: &regionapi.DescribeRegionsResponseParams{RegionSet: []*regionapi.RegionInfo{{Region: common.StringPtr("ap-guangzhou"), RegionState: common.StringPtr("AVAILABLE")}, {Region: common.StringPtr("closed"), RegionState: common.StringPtr("UNAVAILABLE")}}}}, nil
	}}}
	regions, err = tc.GetAvailableRegions(context.Background())
	if err != nil || len(regions) != 2 || !regions[0].Available || regions[1].Available {
		t.Fatal("Tencent region state mapping failed")
	}
	resource := tc.convertInstance(&cvm.Instance{InstanceId: common.StringPtr("ins-one"), CPU: common.Int64Ptr(2), Memory: common.Int64Ptr(8), InstanceState: common.StringPtr("RUNNING"), InstanceChargeType: common.StringPtr("PREPAID"), ExpiredTime: common.StringPtr("2027-01-01T00:00:00Z"), SystemDisk: &cvm.SystemDisk{DiskSize: common.Int64Ptr(50)}, Tags: []*cvm.Tag{{Key: common.StringPtr("env"), Value: common.StringPtr("test")}}})
	if resource.Memory != 8 || resource.Cpu != 2 || resource.Disk != 50 || resource.ExpireTime == nil || resource.ChargeType != model.ChargeTypePrePaid || len(resource.Tags) != 1 {
		t.Fatal("Tencent mapping failed")
	}
}
