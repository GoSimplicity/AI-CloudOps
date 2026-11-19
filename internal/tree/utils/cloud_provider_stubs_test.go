package utils

import (
	"os"
	"strings"
	"testing"
)

func TestRetiredCloudHelperStubsStayAbsent(t *testing.T) {
	banned := []string{
		"func VerifyHuaweiCredentials",
		"func VerifyAzureCredentials",
		"func VerifyGCPCredentials",
		"func SyncHuaweiResources",
		"func SyncAzureResources",
		"func SyncGCPResources",
		"华为云凭证验证功能暂未实现",
		"Azure凭证验证功能暂未实现",
		"GCP凭证验证功能暂未实现",
		"华为云资源同步功能暂未实现",
		"Azure资源同步功能暂未实现",
		"GCP资源同步功能暂未实现",
	}
	for _, name := range []string{"cloud_account_util.go", "tree_cloud_util.go"} {
		body, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		text := string(body)
		for _, marker := range banned {
			if strings.Contains(text, marker) {
				t.Fatalf("%s still contains unused helper stub %q", name, marker)
			}
		}
	}
	factory, err := os.ReadFile("cloud_client.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(factory)
	for _, marker := range []string{"NewHuaweiClient", "NewAzureClient", "NewGCPClient", "云厂商 %d 暂未实现计算资源API"} {
		if !strings.Contains(text, marker) {
			t.Fatalf("cloud client factory lost %q", marker)
		}
	}
}
