package utils

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type awsEC2API interface {
	DescribeInstances(context.Context, *ec2.DescribeInstancesInput, ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error)
	DescribeRegions(context.Context, *ec2.DescribeRegionsInput, ...func(*ec2.Options)) (*ec2.DescribeRegionsOutput, error)
}
type AWSClient struct {
	client awsEC2API
	region string
}

func NewAWSClient(accessKey, secretKey, region string) (*AWSClient, error) {
	if strings.TrimSpace(accessKey) == "" || strings.TrimSpace(secretKey) == "" || strings.TrimSpace(region) == "" {
		return nil, fmt.Errorf("AWS访问密钥和地域不能为空")
	}
	if strings.HasPrefix(accessKey, "ASIA") {
		return nil, fmt.Errorf("AWS临时凭据需要session_token；当前账户仅支持长期AccessKey")
	}
	// Explicit credentials prevent ambient profiles/instance metadata credentials
	// from silently substituting a different cloud account.
	cfg := aws.Config{Region: region, Credentials: aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")), HTTPClient: &http.Client{Timeout: 30 * time.Second}}
	return &AWSClient{client: ec2.NewFromConfig(cfg), region: region}, nil
}
func (c *AWSClient) VerifyCredentials(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	response, err := c.client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{MaxResults: aws.Int32(5)})
	if err != nil {
		return fmt.Errorf("AWS EC2读取权限验证失败: %w", err)
	}
	if response == nil {
		return fmt.Errorf("AWS返回空响应")
	}
	return ctx.Err()
}
func (c *AWSClient) GetAvailableRegions(ctx context.Context) ([]model.AvailableRegion, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	response, err := c.client.DescribeRegions(ctx, &ec2.DescribeRegionsInput{AllRegions: aws.Bool(true)})
	if err != nil {
		return nil, fmt.Errorf("AWS地域查询失败: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if response == nil {
		return nil, fmt.Errorf("AWS返回空地域响应")
	}
	regions := make([]model.AvailableRegion, 0, len(response.Regions))
	for _, r := range response.Regions {
		status := aws.ToString(r.OptInStatus)
		regions = append(regions, model.AvailableRegion{Region: aws.ToString(r.RegionName), RegionName: aws.ToString(r.RegionName), Available: status == "opt-in-not-required" || status == "opted-in"})
	}
	return regions, nil
}
func (c *AWSClient) ListInstances(ctx context.Context, ids []string) ([]*model.TreeCloudResource, error) {
	batches, err := instanceIDBatches(ids, 100)
	if err != nil {
		return nil, err
	}
	resources := make([]*model.TreeCloudResource, 0)
	seenIDs := make(map[string]bool)
	for _, batch := range batches {
		input := &ec2.DescribeInstancesInput{InstanceIds: batch}
		if len(batch) == 0 {
			input.MaxResults = aws.Int32(1000)
		} // AWS forbids MaxResults with InstanceIds.
		tokens := make(map[string]bool)
		for {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			response, err := c.client.DescribeInstances(ctx, input)
			if err != nil {
				return nil, fmt.Errorf("AWS实例查询失败: %w", err)
			}
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			if response == nil {
				return nil, fmt.Errorf("AWS返回空实例响应")
			}
			for _, reservation := range response.Reservations {
				for _, instance := range reservation.Instances {
					id := aws.ToString(instance.InstanceId)
					if id == "" {
						return nil, fmt.Errorf("AWS实例响应缺少ID，拒绝不完整同步")
					}
					if seenIDs[id] {
						return nil, fmt.Errorf("AWS分页返回重复实例，请重新同步")
					}
					resources = append(resources, c.convertInstance(instance))
					seenIDs[id] = true
				}
			}
			next := aws.ToString(response.NextToken)
			if next == "" {
				break
			}
			if tokens[next] {
				return nil, fmt.Errorf("AWS返回重复分页令牌")
			}
			tokens[next] = true
			input.NextToken = aws.String(next)
		}
	}
	return resources, nil
}
func (c *AWSClient) convertInstance(instance types.Instance) *model.TreeCloudResource {
	r := &model.TreeCloudResource{Name: aws.ToString(instance.InstanceId), InstanceID: aws.ToString(instance.InstanceId), ResourceType: model.ResourceTypeECS, InstanceType: string(instance.InstanceType), Region: c.region, VpcID: aws.ToString(instance.VpcId), PrivateIP: aws.ToString(instance.PrivateIpAddress), PublicIP: aws.ToString(instance.PublicIpAddress), ImageID: aws.ToString(instance.ImageId), OSName: aws.ToString(instance.PlatformDetails), ChargeType: model.ChargeTypePostPaid, Currency: model.CurrencyUSD, Port: 22, Status: model.CloudResourceUnknown}
	if instance.Placement != nil {
		r.ZoneID = aws.ToString(instance.Placement.AvailabilityZone)
	}
	if instance.CpuOptions != nil {
		r.Cpu = int(aws.ToInt32(instance.CpuOptions.CoreCount) * aws.ToInt32(instance.CpuOptions.ThreadsPerCore))
	}
	if instance.State != nil {
		switch instance.State.Name {
		case types.InstanceStateNameRunning:
			r.Status = model.CloudResourceRunning
		case types.InstanceStateNameStopped:
			r.Status = model.CloudResourceStopped
		case types.InstanceStateNamePending:
			r.Status = model.CloudResourceStarting
		case types.InstanceStateNameStopping, types.InstanceStateNameShuttingDown:
			r.Status = model.CloudResourceStopping
		case types.InstanceStateNameTerminated:
			r.Status = model.CloudResourceDeleted
		}
	}
	if instance.Platform == types.PlatformValuesWindows {
		r.OSType = "windows"
	} else if strings.Contains(strings.ToLower(r.OSName), "linux") {
		r.OSType = "linux"
	}
	for _, tag := range instance.Tags {
		key, value := aws.ToString(tag.Key), aws.ToString(tag.Value)
		r.Tags = append(r.Tags, model.KeyValue{Key: key, Value: value})
		if key == "Name" && value != "" {
			r.Name = value
		}
	}
	return r
}
