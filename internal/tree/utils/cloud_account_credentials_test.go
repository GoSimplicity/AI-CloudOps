package utils

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
)

type credentialRoundTrip func(*http.Request) (*http.Response, error)

func (f credentialRoundTrip) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestTemporaryCredentialsReachSDKWithoutAmbientCredentials(t *testing.T) {
	expiry := time.Now().UTC().Add(time.Hour)
	cred := CloudCredentials{AccessKey: "ASIAFIXTUREKEY", SecretKey: "secret-fixture", SessionToken: "token-fixture", ExpiresAt: &expiry, CredentialType: "temporary"}
	awsClient, err := NewAWSClientWithCredentials(cred, "us-east-1")
	if err != nil {
		t.Fatal(err)
	}
	got, err := awsClient.client.(*ec2.Client).Options().Credentials.Retrieve(context.Background())
	if err != nil || got.AccessKeyID != cred.AccessKey || got.SecretAccessKey != cred.SecretKey || got.SessionToken != cred.SessionToken || !got.CanExpire || !got.Expires.Equal(expiry) {
		t.Fatal("AWS SDK credential tuple differs")
	}
	tencentClient, err := NewTencentClientWithCredentials(cred, "ap-test")
	if err != nil {
		t.Fatal(err)
	}
	id, secret, token := tencentClient.client.(*cvm.Client).GetCredential().GetCredential()
	if id != cred.AccessKey || secret != cred.SecretKey || token != cred.SessionToken {
		t.Fatal("Tencent SDK credential tuple differs")
	}
}

func TestTemporaryCredentialsRejectInvalidBeforeClientCreation(t *testing.T) {
	future, past := time.Now().UTC().Add(time.Hour), time.Now().UTC().Add(-time.Hour)
	for _, tc := range []struct {
		name     string
		provider model.CloudProvider
		input    CloudCredentials
	}{
		{"missing token", model.ProviderAWS, CloudCredentials{AccessKey: "ASIAFIXTURE", SecretKey: "secret", CredentialType: "temporary", ExpiresAt: &future}},
		{"missing expiry", model.ProviderTencent, CloudCredentials{AccessKey: "fixture", SecretKey: "secret", CredentialType: "temporary", SessionToken: "token"}},
		{"expired", model.ProviderAWS, CloudCredentials{AccessKey: "ASIAFIXTURE", SecretKey: "secret", CredentialType: "temporary", SessionToken: "token", ExpiresAt: &past}},
		{"unmarked AWS temporary", model.ProviderAWS, CloudCredentials{AccessKey: "ASIAFIXTURE", SecretKey: "secret"}},
		{"unexpected token on long-lived", model.ProviderTencent, CloudCredentials{AccessKey: "fixture", SecretKey: "secret", SessionToken: "token"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := NewCloudComputeClientWithCredentials(tc.provider, tc.input, "test", nil); err == nil {
				t.Fatal("invalid credentials accepted")
			}
		})
	}
}

func TestTemporaryCredentialsAreSignedInSDKHeadersNotURL(t *testing.T) {
	expiry := time.Now().UTC().Add(time.Hour)
	cred := CloudCredentials{AccessKey: "ASIAFIXTUREKEY", SecretKey: "secret-fixture", SessionToken: "token-fixture", ExpiresAt: &expiry, CredentialType: "temporary"}
	for _, provider := range []string{"aws", "tencent"} {
		t.Run(provider, func(t *testing.T) {
			calls := 0
			transport := credentialRoundTrip(func(req *http.Request) (*http.Response, error) {
				calls++
				if strings.Contains(req.URL.String(), cred.SessionToken) || strings.Contains(req.URL.String(), cred.SecretKey) {
					t.Error("credential in URL")
				}
				header := "X-Amz-Security-Token"
				body := `<DescribeInstancesResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/"><reservationSet/></DescribeInstancesResponse>`
				if provider == "tencent" {
					header = "X-Tc-Token"
					body = `{"Response":{"TotalCount":0,"InstanceSet":[],"RequestId":"fixture"}}`
				}
				if req.Header.Get(header) != cred.SessionToken {
					t.Error("signed request omitted session token")
				}
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
			})
			if provider == "aws" {
				client, err := NewAWSClientWithCredentials(cred, "us-east-1")
				if err != nil {
					t.Fatal(err)
				}
				client.client.(*ec2.Client).Options().HTTPClient.(*http.Client).Transport = transport
				if err := client.VerifyCredentials(context.Background()); err != nil {
					t.Fatal("AWS fixture request failed")
				}
			} else {
				client, err := NewTencentClientWithCredentials(cred, "ap-guangzhou")
				if err != nil {
					t.Fatal(err)
				}
				client.client.(*cvm.Client).WithHttpTransport(transport)
				if err := client.VerifyCredentials(context.Background()); err != nil {
					t.Fatal("Tencent fixture request failed")
				}
			}
			if calls != 1 {
				t.Fatalf("unexpected calls: %d", calls)
			}
		})
	}
}
