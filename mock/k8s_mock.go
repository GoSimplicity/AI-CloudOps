// mock/k8s_client_mock.go

package mock

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/openkruise/kruise-api/client/clientset/versioned"
	"gorm.io/gorm"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	metricsClient "k8s.io/metrics/pkg/client/clientset/versioned"
)

// K8sClientMock 是 K8sClient 接口的手动 Mock 实现
type K8sClientMock struct {
	sync.RWMutex

	// 定义每个方法的返回值和行为
	InitClientFunc       func(ctx context.Context, clusterID int, kubeConfig *rest.Config) error
	GetKubeClientFunc    func(ctx context.Context, clusterID int) (*fake.Clientset, error)
	GetKruiseClientFunc  func(ctx context.Context, clusterID int) (*versioned.Clientset, error)
	GetMetricsClientFunc func(ctx context.Context, clusterID int) (*metricsClient.Clientset, error)
	GetDynamicClientFunc func(ctx context.Context, clusterID int) (*dynamic.DynamicClient, error)
	GetNamespacesFunc    func(ctx context.Context, clusterID int) ([]string, error)
	RecordProbeErrorFunc func(ctx context.Context, clusterID int, errMsg string)
	RefreshClientsFunc   func(ctx context.Context) error

	db *gorm.DB

	fakeClientsets map[int]*fake.Clientset
}

// NewK8sClientMock 创建一个新的 K8sClientMock 实例，并插入更多的 mock 数据
func NewK8sClientMock(db *gorm.DB) *K8sClientMock {
	mock := &K8sClientMock{
		db:             db,
		fakeClientsets: make(map[int]*fake.Clientset),
	}

	// 预先填充模拟数据
	//mock.populateMockData()

	return mock
}

// populateMockData 插入多个模拟集群和相关数据到数据库
func (m *K8sClientMock) populateMockData() {
	m.Lock()
	defer m.Unlock()

	// 示例：插入多个 K8sCluster
	clusters := []model.K8sCluster{
		{
			Name:          "Cluster-1",
			NameZh:        "集群-1",
			UserID:        1,
			CpuRequest:    "100m",
			CpuLimit:      "200m",
			MemoryRequest: "256Mi",
			MemoryLimit:   "512Mi",
			Env:           "prod",
			Version:       "v1.20.0",
			ApiServerAddr: "https://api.cluster1.example.com",
			KubeConfigContent: `"apiVersion": "v1"
"clusters":
- "cluster":
    "certificate-authority-data": "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJ2VENDQVVTZ0F3SUJBZ0lSQUttek1MY2lNL1BhS01PMVQrclAyQjh3Q2dZSUtvWkl6ajBFQXdNd0h6RWQKTUJzR0ExVUVBeE1VYTNWaVpYSnVaWFJsY3kxelpYSjJaWEl0WTJFd0lCY05NalF3T1RJMU1URXlPVFF6V2hnUApNakV5TkRBNU1qWXhNVEk1TkROYU1COHhIVEFiQmdOVkJBTVRGR3QxWW1WeWJtVjBaWE10YzJWeWRtVnlMV05oCk1IWXdFQVlIS29aSXpqMENBUVlGSzRFRUFDSURZZ0FFMDJUQnlGRk1tZ3ZqdFhXWTNwMjE4by83bytqaVVzSTYKVTZqbkI5bm5uVHZxRVNsV3V6Z2ZZc3JPR0xGUVFKMDdBVDdHWE5aSm5oSzhVWG9rZVdxeVNOaGlxaFg3ZVplVAowQlZwdXNxN056cGlaSU11b1drbGRLblQ4UFlGcWFJUW8wSXdRREFPQmdOVkhROEJBZjhFQkFNQ0FxUXdEd1lEClZSMFRBUUgvQkFVd0F3RUIvekFkQmdOVkhRNEVGZ1FVRXZIZzFjVnJpSlBoRlpIYzNhaGk5aUhic2RBd0NnWUkKS29aSXpqMEVBd01EWndBd1pBSXdjQkY5N1pGOGlsZVZlZFpDK1I2RlA2ak5wTnd0R0ZDWTIrejRQdURNbGJPcAo4NDgraCtWMzZqblNRbHRXN0FtNkFqQXlMR3k3dS9GdjZSKzhhdTAyMkNkT1d1T0NOampldDcwZW1xQi80ei9BCll0WTd3Ums2UWNwdUhTTTVmUW44dDBzPQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg=="
    "server": "https://43.154.130.129:6443"
  "name": "cluster.local"
"contexts":
- "context":
    "cluster": "cluster.local"
    "user": "master-user"
  "name": "cluster.local"
"current-context": "cluster.local"
"kind": "Config"
"preferences": {}
"users":
- "name": "master-user"
  "user":
    "client-certificate-data": "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUI0akNDQVdpZ0F3SUJBZ0lSQUprNnF0TGYxcklUYzN1dlV3Z0dqWUV3Q2dZSUtvWkl6ajBFQXdNd0h6RWQKTUJzR0ExVUVBeE1VYTNWaVpYSnVaWFJsY3kxamJHbGxiblF0WTJFd0lCY05NalF3T1RJMU1URXlPVFF6V2hnUApNakV5TkRBNU1qWXhNVEk1TkROYU1DOHhGekFWQmdOVkJBb1REbk41YzNSbGJUcHRZWE4wWlhKek1SUXdFZ1lEClZRUURFd3R0WVhOMFpYSXRkWE5sY2pCMk1CQUdCeXFHU000OUFnRUdCU3VCQkFBaUEySUFCTUVnNkhqVkZSV1EKOC85UDhVRm5BUk90WEtZVWROTTZEQkUxU0RpOW9wNm91dzRNTDZLZ0xuTGpWbkRTNWcxMDRvQ0JNazJjbHJpUQpuSzl4VjErSm5laWl2eVBVQjdUT2gxdzBKRW1PanU0TjdIaGRSUEs4ZHIvR245Q3RYR3dCNnFOV01GUXdEZ1lEClZSMFBBUUgvQkFRREFnV2dNQk1HQTFVZEpRUU1NQW9HQ0NzR0FRVUZCd01DTUF3R0ExVWRFd0VCL3dRQ01BQXcKSHdZRFZSMGpCQmd3Rm9BVVBqcFFSS1JpVzU4SVArVGo3QTRZeUJOOGpqSXdDZ1lJS29aSXpqMEVBd01EYUFBdwpaUUl3RitSTmV1MUNwZUdMc1MwZ2NEZnVMT0FEVGpJS01tRGg1TU5EeDZkaEluQ1RwUkJSZE16UWtNeEd3QWkzCkFOOUZBakVBaTl0dkVvaXJ2MEc4MTRhbXhaVEltdmxGSFNJUzNzYWlqR1lCZkY5blFTQUpRT1dXNlU1VTJRSDYKRnRqbWV1ZjUKLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo="
    "client-key-data": "LS0tLS1CRUdJTiBFQyBQUklWQVRFIEtFWS0tLS0tCk1JR2tBZ0VCQkRCVVFGZXc0eVZmRlp2RWxZZmVQMk5seWdSSGVkZ2hxQzFkQkdRWVRSRmVLbWpmVjhobHgrR3kKMStQVFZHc0JXdENnQndZRks0RUVBQ0toWkFOaUFBVEJJT2g0MVJVVmtQUC9UL0ZCWndFVHJWeW1GSFRUT2d3UgpOVWc0dmFLZXFMc09EQytpb0M1eTQxWncwdVlOZE9LQWdUSk5uSmE0a0p5dmNWZGZpWjNvb3I4ajFBZTB6b2RjCk5DUkpqbzd1RGV4NFhVVHl2SGEveHAvUXJWeHNBZW89Ci0tLS0tRU5EIEVDIFBSSVZBVEUgS0VZLS0tLS0K"
`,
			ActionTimeoutSeconds: 30,
		},
		{
			Name:          "Cluster-2",
			NameZh:        "集群-2",
			UserID:        2,
			CpuRequest:    "200m",
			CpuLimit:      "400m",
			MemoryRequest: "512Mi",
			MemoryLimit:   "1Gi",
			Env:           "dev",
			Version:       "v1.19.0",
			ApiServerAddr: "https://api.cluster2.example.com",
			KubeConfigContent: `
apiVersion: v1
kind: Config
clusters:
- name: cluster2
  cluster:
    server: https://10.0.0.2:6443
    certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURvVENDQVVTZ0F3SUJBZ0lRQUpBS0NBUUVBeU9YZzc4SHd0TXNQbEM3WjlQRkZ4eDl4ZDZ3bzlPV1ZEMGdreUYwMnJ4T3FqMHltCm9EQzJjWWhDWkp0MlhEckRVWjV1Wjd0dS9JcURUUFBnenFGYnJzL2RXcUM3VFFCODNzdlNLUUUzSDJ3L2llRTMKcmNmUU0vdjFMem51bDhwQzRIOE5SR1d5Q0hFWXRSNWsydjdYMHV5RXpRNWN2ckQra2dLS05ENnhxR2pJV3UzcApSNVNsVVFYUzE3VzEzTXJiOHgyMW15VlNLRHdTR2tDM3l5enRoTUFrTXJmNFZvWXdZbjdpRStsNytHVStEaVMrCjVieWU4ME9hYjFwK1h2UnJYK1BRcXV0ck1RdXFBN3FWRmNOTzRIcjZDRWw2a21tTzVlMnZob3ZDU0N5dE91a3IKbUR2cXhTN1NHNXRqYnRTTWhFVTd1OHRjbFp0U25DNDZDcFJBclFJREFRQUJBb0lCQUhlOUdjSnM4RitCYSs2aApJZzBveExNUzNaazdSa240a0NoRWtvL05adXRncUZheWs5bUVUeWxKMG5DT2JEUTZYZkhHb1AwNHJNVGVkYldtCjRjV21UTFBiZG80dW9KUWJoMUZ2WC9wS25DVWhhcGx1WDdZM0VyUDVKMENYb2NwY3BoR1NDRUZNNzVpeWQ4eHoKcWg4UXAyZ3l5bEJCa0N1ajZJYno5cFYvREhwWHN6OVVGdmhKRkYyRnMwMzRLemFxM0NmcXlzbGZRTHdzQlhiNQp0S0RFdW9yNE5OOW5sMVFKbXU3TXNVZUVHSUp1WlBYZUdSdmhYS3pxQ1FldXBzL0M4Mlh2VnNqUmdpL295WnVPCm9nL0hNL1dRU0dkMlp4Tnc5OERNaE5WUUdPclpQOUJQZnl5VFVZRC9jUWRRUkg1MEhXbUZRSU9iZmoxVEVJbjEKdE9QbjVjMENnWUVBMzl3UkhoeERJbVdkSUZPcGhnQ3Y5Z01lR0dvSGFrY3NRNEVWbkpqNDREaXFXWGpXN0todgpmNy9WODE3bUo2dmhSclhJQ1VnOC9yU3BTekJKUWxIaEo1Mkd4OGExN1RZWE5PdzBDaldmUzQ2US9SWVpUTThzClRkMTJuN3ZLS3d2U0dOc0kxR3JLN2wrTlBVbW96aTM4dG9peGNKa1JMaWpHQVJ6MjM0b2FhQ01DZ1lFQTViNGsKN2c4NHdvSDFBb2NEYmdmYnJLSFB0c0EwOFI0Ryt6cmxpOWN4dDN1cFF0cXI2S09XL1FaQk1wUFF0cmJ4c2R3SgpBcVk3VXhxWkNkbDNEVTVwZ3lmSGlXS2VJeTQweVY5R2lrRzlLSll5Yno4MEtxR0UrTGdxWHZVRUVrK1dtaHIvCjkxd1dvL3ZFWlN6YXAvcSs1STNvU1JoVk0xclZtNHg2eDlEcldPOENnWUFaQTRNYUpUaFBNS0dGQ2pRb0dBMlIKWkxuSktwZlhoVXBwNUpPZ291czBTc0NtTEwxL1JqYm5SRzFJdTNMbVBldDNOanE2NXNxQi8zZm5RZWI5dFI1KwppYmlVdkJ4NS9CQk54cEx1RFIzV21JQ1U5eEl1cGZ1aVc3dTBqNHhBa1JxUjBtL0RKSWUyYVJEa0owWG9lL1VBCkJIZ25SOE5Hc0NHbjRDK3B2TW5FbHdLQmdFdUc1QXNqRSt6VjNsOHpWWXhScHdVc3VPV0NjS1VuMHZHNm5nWUgKKzc4dk55alVUSm1SVml6ZVpvYWpFNFZOeFUxTVllWHVFaWl5NE1iZEtBZEcxT2NhSjczaG5zMC8vbmlKQ3Q2ZQoxL25FenRYRnVIZWZXK0NNWXRtT3dRVG9CMEdvU0tmZ0xVMUJrb0lVYWRtNVZCSTlHTFVXKzhPRFJCLzc0YzFZCndGWWZBb0dBT2pRd0RkeWg0L01YTzA5QWpOQ0hzRTVnNnJ4bHhtclRiTzFHZFVSckFOWUg4RnB5Q2gwUWxUWloKV0kyUldXdU50RUdPNDVsN08rVXJINGNkUFR6MDBmbkhlU3czVUVJeklxQ0I3UCtXc005R2tCNC9iMkFDQXJNZwpmandwK0h1eFNSemQ2M1pTT2dPYXZCb0cwVnF4TXhiTTNkNDdoNGFlM0pHNUJWM1o2T3M9Ci0tLS0tRU5EIFJTQSBQUklWQVRFIEtFWS0tLS0tCg==
contexts:
- name: context2
  context:
    cluster: cluster2
    user: user2
current-context: context2
users:
- name: user2
  user:
    client-certificate-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURvVENDQVVTZ0F3SUJBZ0lRQUpBS0NBUUVBeU9YZzc4SHd0TXNQbEM3WjlQRkZ4eDl4ZDZ3bzlPV1ZEMGdreUYwMnJ4T3FqMHltCm9EQzJjWWhDWkp0MlhEckRVWjV1Wjd0dS9JcURUUFBnenFGYnJzL2RXcUM3VFFCODNzdlNLUUUzSDJ3L2llRTMKcmNmUU0vdjFMem51bDhwQzRIOE5SR1d5Q0hFWXRSNWsydjdYMHV5RXpRNWN2ckQra2dLS05ENnhxR2pJV3UzcApSNVNsVVFYUzE3VzEzTXJiOHgyMW15VlNLRHdTR2tDM3l5enRoTUFrTXJmNFZvWXdZbjdpRStsNytHVStEaVMrCjVieWU4ME9hYjFwK1h2UnJYK1BRcXV0ck1RdXFBN3FWRmNOTzRIcjZDRWw2a21tTzVlMnZob3ZDU0N5dE91a3IKbUR2cXhTN1NHNXRqYnRTTWhFVTd1OHRjbFp0U25DNDZDcFJBclFJREFRQUJBb0lCQUhlOUdjSnM4RitCYSs2aApJZzBveExNUzNaazdSa240a0NoRWtvL05adXRncUZheWs5bUVUeWxKMG5DT2JEUTZYZkhHb1AwNHJNVGVkYldtCjRjV21UTFBiZG80dW9KUWJoMUZ2WC9wS25DVWhhcGx1WDdZM0VyUDVKMENYb2NwY3BoR1NDRUZNNzVpeWQ4eHoKcWg4UXAyZ3l5bEJCa0N1ajZJYno5cFYvREhwWHN6OVVGdmhKRkYyRnMwMzRLemFxM0NmcXlzbGZRTHdzQlhiNQp0S0RFdW9yNE5OOW5sMVFKbXU3TXNVZUVHSUp1WlBYZUdSdmhYS3pxQ1FldXBzL0M4Mlh2VnNqUmdpL295WnVPCm9nL0hNL1dRU0dkMlp4Tnc5OERNaE5WUUdPclpQOUJQZnl5VFVZRC9jUWRRUkg1MEhXbUZRSU9iZmoxVEVJbjEKdE9QbjVjMENnWUVBMzl3UkhoeERJbVdkSUZPcGhnQ3Y5Z01lR0dvSGFrY3NRNEVWbkpqNDREaXFXWGpXN0todgpmNy9WODE3bUo2dmhSclhJQ1VnOC9yU3BTekJKUWxIaEo1Mkd4OGExN1RZWE5PdzBDaldmUzQ2US9SWVpUTThzClRkMTJuN3ZLS3d2U0dOc0kxR3JLN2wrTlBVbW96aTM4dG9peGNKa1JMaWpHQVJ6MjM0b2FhQ01DZ1lFQTViNGsKN2c4NHdvSDFBb2NEYmdmYnJLSFB0c0EwOFI0Ryt6cmxpOWN4dDN1cFF0cXI2S09XL1FaQk1wUFF0cmJ4c2R3SgpBcVk3VXhxWkNkbDNEVTVwZ3lmSGlXS2VJeTQweVY5R2lrRzlLSll5Yno4MEtxR0UrTGdxWHZVRUVrK1dtaHIvCjkxd1dvL3ZFWlN6YXAvcSs1STNvU1JoVk0xclZtNHg2eDlEcldPOENnWUFaQTRNYUpUaFBNS0dGQ2pRb0dBMlIKWkxuSktwZlhoVXBwNUpPZ291czBTc0NtTEwxL1JqYm5SRzFJdTNMbVBldDNOanE2NXNxQi8zZm5RZWI5dFI1KwppYmlVdkJ4NS9CQk54cEx1RFIzV21JQ1U5eEl1cGZ1aVc3dTBqNHhBa1JxUjBtL0RKSWUyYVJEa0owWG9lL1VBCkJIZ25SOE5Hc0NHbjRDK3B2TW5FbHdLQmdFdUc1QXNqRSt6VjNsOHpWWXhScHdVc3VPV0NjS1VuMHZHNm5nWUgKKzc4dk55alVUSm1SVml6ZVpvYWpFNFZOeFUxTVllWHVFaWl5NE1iZEtBZEcxT2NhSjczaG5zMC8vbmlKQ3Q2ZQoxL25FenRYRnVIZWZXK0NNWXRtT3dRVG9CMEdvU0tmZ0xVMUJrb0lVYWRtNVZCSTlHTFVXKzhPRFJCLzc0YzFZCndGWWZBb0dBT2pRd0RkeWg0L01YTzA5QWpOQ0hzRTVnNnJ4bHhtclRiTzFHZFVSckFOWUg4RnB5Q2gwUWxUWloKV0kyUldXdU50RUdPNDVsN08rVXJINGNkUFR6MDBmbkhlU3czVUVJeklxQ0I3UCtXc005R2tCNC9iMkFDQXJNZwpmandwK0h1eFNSemQ2M1pTT2dPYXZCb0cwVnF4TXhiTTNkNDdoNGFlM0pHNUJWM1o2T3M9Ci0tLS0tRU5EIFJTQSBQUklWQVRFIEtFWS0tLS0tCg==
current-context: context2
users:
- name: user2
  user:
    client-key-data: LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFb2dJQkFBS0NBUUVBeU9YZzc4SHd0TXNQbEM3WjlQRkZ4eDl4ZDZ3bzlPV1ZEMGdreUYwMnJ4T3FqMHltCm9EQzJjWWhDWkp0MlhEckRVWjV1Wjd0dS9JcURUUFBnenFGYnJzL2RXcUM3VFFCODNzdlNLUUUzSDJ3L2llRTMKcmNmUU0vdjFMem51bDhwQzRIOE5SR1d5Q0hFWXRSNWsydjdYMHV5RXpRNWN2ckQra2dLS05ENnhxR2pJV3UzcApSNVNsVVFYUzE3VzEzTXJiOHgyMW15VlNLRHdTR2tDM3l5enRoTUFrTXJmNFZvWXdZbjdpRStsNytHVStEaVMrCjVieWU4ME9hYjFwK1h2UnJYK1BRcXV0ck1RdXFBN3FWRmNOTzRIcjZDRWw2a21tTzVlMnZob3ZDU0N5dE91a3IKbUR2cXhTN1NHNXRqYnRTTWhFVTd1OHRjbFp0U25DNDZDcFJBclFJREFRQUJBb0lCQUhlOUdjSnM4RitCYSs2aApJZzBveExNUzNaazdSa240a0NoRWtvL05adXRncUZheWs5bUVUeWxKMG5DT2JEUTZYZkhHb1AwNHJNVGVkYldtCjRjV21UTFBiZG80dW9KUWJoMUZ2WC9wS25DVWhhcGx1WDdZM0VyUDVKMENYb2NwY3BoR1NDRUZNNzVpeWQ4eHoKcWg4UXAyZ3l5bEJCa0N1ajZJYno5cFYvREhwWHN6OVVGdmhKRkYyRnMwMzRLemFxM0NmcXlzbGZRTHdzQlhiNQp0S0RFdW9yNE5OOW5sMVFKbXU3TXNVZUVHSUp1WlBYZUdSdmhYS3pxQ1FldXBzL0M4Mlh2VnNqUmdpL295WnVPCm9nL0hNL1dRU0dkMlp4Tnc5OERNaE5WUUdPclpQOUJQZnl5VFVZRC9jUWRRUkg1MEhXbUZRSU9iZmoxVEVJbjEKdE9QbjVjMENnWUVBMzl3UkhoeERJbVdkSUZPcGhnQ3Y5Z01lR0dvSGFrY3NRNEVWbkpqNDREaXFXWGpXN0todgpmNy9WODE3bUo2dmhSclhJQ1VnOC9yU3BTekJKUWxIaEo1Mkd4OGExN1RZWE5PdzBDaldmUzQ2US9SWVpUTThzClRkMTJuN3ZLS3d2U0dOc0kxR3JLN2wrTlBVbW96aTM4dG9peGNKa1JMaWpHQVJ6MjM0b2FhQ01DZ1lFQTViNGsKN2c4NHdvSDFBb2NEYmdmYnJLSFB0c0EwOFI0Ryt6cmxpOWN4dDN1cFF0cXI2S09XL1FaQk1wUFF0cmJ4c2R3SgpBcVk3VXhxWkNkbDNEVTVwZ3lmSGlXS2VJeTQweVY5R2lrRzlLSll5Yno4MEtxR0UrTGdxWHZVRUVrK1dtaHIvCjkxd1dvL3ZFWlN6YXAvcSs1STNvU1JoVk0xclZtNHg2eDlEcldPOENnWUFaQTRNYUpUaFBNS0dGQ2pRb0dBMlIKWkxuSktwZlhoVXBwNUpPZ291czBTc0NtTEwxL1JqYm5SRzFJdTNMbVBldDNOanE2NXNxQi8zZm5RZWI5dFI1KwppYmlVdkJ4NS9CQk54cEx1RFIzV21JQ1U5eEl1cGZ1aVc3dTBqNHhBa1JxUjBtL0RKSWUyYVJEa0owWG9lL1VBCkJIZ25SOE5Hc0NHbjRDK3B2TW5FbHdLQmdFdUc1QXNqRSt6VjNsOHpWWXhScHdVc3VPV0NjS1VuMHZHNm5nWUgKKzc4dk55alVUSm1SVml6ZVpvYWpFNFZOeFUxTVllWHVFaWl5NE1iZEtBZEcxT2NhSjczaG5zMC8vbmlKQ3Q2ZQoxL25FenRYRnVIZWZXK0NNWXRtT3dRVG9CMEdvU0tmZ0xVMUJrb0lVYWRtNVZCSTlHTFVXKzhPRFJCLzc0YzFZCndGWWZBb0dBT2pRd0RkeWg0L01YTzA5QWpOQ0hzRTVnNnJ4bHhtclRiTzFHZFVSckFOWUg4RnB5Q2gwUWxUWloKV0kyUldXdU50RUdPNDVsN08rVXJINGNkUFR6MDBmbkhlU3czVUVJeklxQ0I3UCtXc005R2tCNC9iMkFDQXJNZwpmandwK0h1eFNSemQ2M1pTT2dPYXZCb0cwVnF4TXhiTTNkNDdoNGFlM0pHNUJWM1o2T3M9Ci0tLS0tRU5EIFJTQSBQUklWQVRFIEtFWS0tLS0tCg==
`,
			ActionTimeoutSeconds: 30,
		},
		// 可以添加更多集群
	}

	for _, cluster := range clusters {
		if err := m.db.Where("name = ?", cluster.Name).FirstOrCreate(&cluster).Error; err != nil {
			log.Printf("populateMockData: 插入 K8sCluster 失败: %v\n", err)
			continue
		}

		log.Printf("populateMockData: 初始化 Kubernetes 集群成功，ClusterID: %d\n", cluster.ID)
	}
}

// InitClient 模拟 InitClient 方法，并插入模拟数据到数据库
func (m *K8sClientMock) InitClient(ctx context.Context, clusterID int, kubeConfig *rest.Config) error {
	m.Lock()
	defer m.Unlock()

	if m.InitClientFunc != nil {
		return m.InitClientFunc(ctx, clusterID, kubeConfig)
	}

	// 插入或更新 K8sCluster 模拟数据
	clusterName := fmt.Sprintf("Cluster-%d", clusterID)
	cluster := model.K8sCluster{
		Name:          clusterName,
		NameZh:        fmt.Sprintf("集群-%d", clusterID),
		UserID:        1,
		CpuRequest:    "100m",
		CpuLimit:      "200m",
		MemoryRequest: "256Mi",
		MemoryLimit:   "512Mi",
		Env:           "prod",
		Version:       "v1.20.0",
		ApiServerAddr: fmt.Sprintf("https://api.%s.example.com", clusterName),
		KubeConfigContent: `
"apiVersion": "v1"
"clusters":
- "cluster":
    "certificate-authority-data": "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJ2ekNDQVVTZ0F3SUJBZ0lSQVAwbjBySWxMWEJsMVU1TXEybjdBbWt3Q2dZSUtvWkl6ajBFQXdNd0h6RWQKTUJzR0ExVUVBeE1VYTNWaVpYSnVaWFJsY3kxelpYSjJaWEl0WTJFd0lCY05NalF3T1RJMU1UQTBPVE16V2hnUApNakV5TkRBNU1qWXhNRFE1TXpOYU1COHhIVEFiQmdOVkJBTVRGR3QxWW1WeWJtVjBaWE10YzJWeWRtVnlMV05oCk1IWXdFQVlIS29aSXpqMENBUVlGSzRFRUFDSURZZ0FFVzYvelJEKzZmRUVaWkZHV0ZZYk9sS1dFMmdlakRyeUwKL2U1VU9QbEVFSGNRNjU1TEpmMm5pYlN5NEhVQXdBWExpZjlHa0p0czNXbUVCRGRUbzcyRjFMbkhJQzdaaHlyLwpzeWY4Nmd6N3BmazIxdm1CdGdJMkY4ckZCSjRRRVQ1VW8wSXdRREFPQmdOVkhROEJBZjhFQkFNQ0FxUXdEd1lEClZSMFRBUUgvQkFVd0F3RUIvekFkQmdOVkhRNEVGZ1FVc1huNXRJNlkxWWZmcTFsZlVZRjFQKytJeThRd0NnWUkKS29aSXpqMEVBd01EYVFBd1pnSXhBTytzOVllU2M4YURybXhtL251eUU5WHhIdThPK3ZFNW1kMzJ4YTJ3eFF0cwpqODJ6UlJXenBZVkdLUi83RjU4THdRSXhBTTMyNk5PQ1pFMXczRSttM3dTdzFtOEhmV0FiZzJYWjhZWEVYK3pmCjlzQkdzQXpscDB1eTdRTkpUenJ4MUlHS1dRPT0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo="
    "server": "https://43.154.204.166:6443"
  "name": "cluster.local"
"contexts":
- "context":
    "cluster": "cluster.local"
    "user": "master-user"
  "name": "cluster.local"
"current-context": "cluster.local"
"kind": "Config"
"preferences": {}
"users":
- "name": "master-user"
  "user":
    "client-certificate-data": "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUI0VENDQVdlZ0F3SUJBZ0lRZGNMaStSM01GTndWK3VndXB0MkhrekFLQmdncWhrak9QUVFEQXpBZk1SMHcKR3dZRFZRUURFeFJyZFdKbGNtNWxkR1Z6TFdOc2FXVnVkQzFqWVRBZ0Z3MHlOREE1TWpVeE1EUTVNek5hR0E4eQpNVEkwTURreU5qRXdORGt6TTFvd0x6RVhNQlVHQTFVRUNoTU9jM2x6ZEdWdE9tMWhjM1JsY25NeEZEQVNCZ05WCkJBTVRDMjFoYzNSbGNpMTFjMlZ5TUhZd0VBWUhLb1pJemowQ0FRWUZLNEVFQUNJRFlnQUV0SVVRSWlNbEIvOVMKTHFlRzhIYnNJK05KRlBvTkxvMWg2SEZsNVBVT1VEWGpyREhDeUt0ejdKZW1INExJZVpBVGxaMWpzZjU5OGJWVgpQc3p5NnAyRVZjcCt6YUR3aTZQdTFNT2lsZE1NenJhTWtvajdIOCtiNkZBK2dueHRiTzNwbzFZd1ZEQU9CZ05WCkhROEJBZjhFQkFNQ0JhQXdFd1lEVlIwbEJBd3dDZ1lJS3dZQkJRVUhBd0l3REFZRFZSMFRBUUgvQkFJd0FEQWYKQmdOVkhTTUVHREFXZ0JTVEo0NzBPdXdXYVp2bkFCTWRPY0E4THFIRm16QUtCZ2dxaGtqT1BRUURBd05vQURCbApBakVBK0VxVXJ5RituQy9POVhZbWdpMVg4TFoxalhWWWhaVHZLdVJMdEYzK0RSRTRQWTVoalYxak43YkZlS0JHCjFCRm9BakFlaFZUNlpweGlPRE1HMzFFVDZhL1luNmdCT3BYWExxOHowQmFWbERQQ0s5bHVYL3d0NTdQZW9HRTQKbDlxSXdnUT0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo="
    "client-key-data": "LS0tLS1CRUdJTiBFQyBQUklWQVRFIEtFWS0tLS0tCk1JR2tBZ0VCQkRERkFEbXlaamFpcDJ3WTdQV0lTSmlGVnZsdnJVeVFRR3d5a21FbEdpMmtrRjUzQVRqVS9oNm4KSGVGQURkSHN4dXFnQndZRks0RUVBQ0toWkFOaUFBUzBoUkFpSXlVSC8xSXVwNGJ3ZHV3ajQwa1UrZzB1aldIbwpjV1hrOVE1UU5lT3NNY0xJcTNQc2w2WWZnc2g1a0JPVm5XT3gvbjN4dFZVK3pQTHFuWVJWeW43Tm9QQ0xvKzdVCnc2S1Ywd3pPdG95U2lQc2Z6NXZvVUQ2Q2ZHMXM3ZWs9Ci0tLS0tRU5EIEVDIFBSSVZBVEUgS0VZLS0tLS0K"
`,
		ActionTimeoutSeconds: 30,
	}

	if err := m.db.Where("name = ?", cluster.Name).FirstOrCreate(&cluster).Error; err != nil {
		log.Printf("InitClient: 插入 K8sCluster 失败: %v\n", err)
		return fmt.Errorf("插入 K8sCluster 失败: %w", err)
	}

	// 创建 Fake Clientset 并预先插入命名空间到 Fake Clientset 中
	fakeClient := fake.NewSimpleClientset()
	namespaces := []string{"default", "kube-system", "prod", "dev"}

	for _, ns := range namespaces {
		nsObj := &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
			},
		}
		_, err := fakeClient.CoreV1().Namespaces().Create(ctx, nsObj, metav1.CreateOptions{})
		if err != nil {
			log.Printf("InitClient: 插入 Namespace %s 到 Fake Clientset 失败: %v\n", ns, err)
			return fmt.Errorf("插入 Namespace %s 到 Fake Clientset 失败: %w", ns, err)
		}
		// 不在这里插入命名空间到数据库
	}

	// 将 Fake Clientset 存储到映射中
	m.fakeClientsets[clusterID] = fakeClient

	log.Printf("InitClient: 初始化 Kubernetes 客户端成功，ClusterID: %d\n", clusterID)
	return nil
}

// GetKubeClient 模拟 GetKubeClient 方法
func (m *K8sClientMock) GetKubeClient(ctx context.Context, clusterID int) (*fake.Clientset, error) {
	m.RLock()
	defer m.RUnlock()

	if m.GetKubeClientFunc != nil {
		return m.GetKubeClientFunc(ctx, clusterID)
	}

	client, exists := m.fakeClientsets[clusterID]
	if !exists {
		return nil, fmt.Errorf("mock: Cluster %d not initialized", clusterID)
	}

	return client, nil
}

// GetKruiseClient 模拟 GetKruiseClient 方法
func (m *K8sClientMock) GetKruiseClient(ctx context.Context, clusterID int) (*versioned.Clientset, error) {
	m.RLock()
	defer m.RUnlock()

	if m.GetKruiseClientFunc != nil {
		return m.GetKruiseClientFunc(ctx, clusterID)
	}

	// 返回一个空的 Kruise Clientset
	return &versioned.Clientset{}, nil
}

// GetMetricsClient 模拟 GetMetricsClient 方法
func (m *K8sClientMock) GetMetricsClient(ctx context.Context, clusterID int) (*metricsClient.Clientset, error) {
	m.RLock()
	defer m.RUnlock()

	if m.GetMetricsClientFunc != nil {
		return m.GetMetricsClientFunc(ctx, clusterID)
	}

	// 返回一个空的 Metrics Clientset
	return &metricsClient.Clientset{}, nil
}

// GetDynamicClient 模拟 GetDynamicClient 方法
func (m *K8sClientMock) GetDynamicClient(ctx context.Context, clusterID int) (*dynamic.DynamicClient, error) {
	m.RLock()
	defer m.RUnlock()

	if m.GetDynamicClientFunc != nil {
		return m.GetDynamicClientFunc(ctx, clusterID)
	}

	// 返回一个空的 Dynamic Client
	return &dynamic.DynamicClient{}, nil
}

// GetNamespaces 模拟 GetNamespaces 方法，并从 Fake Clientset 获取命名空间数据
func (m *K8sClientMock) GetNamespaces(ctx context.Context, clusterID int) ([]string, error) {
	m.RLock()
	defer m.RUnlock()

	if m.GetNamespacesFunc != nil {
		return m.GetNamespacesFunc(ctx, clusterID)
	}

	client, exists := m.fakeClientsets[clusterID]
	if !exists {
		return nil, fmt.Errorf("mock: Cluster %d not initialized", clusterID)
	}

	namespaces, err := client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Printf("GetNamespaces: 列出命名空间失败: %v\n", err)
		return nil, fmt.Errorf("获取命名空间失败: %w", err)
	}

	var nsList []string
	for _, ns := range namespaces.Items {
		nsList = append(nsList, ns.Name)
	}

	// 只插入不存在的命名空间到数据库
	for _, ns := range nsList {
		namespace := model.K8sNamespace{
			Name:      ns,
			ClusterID: clusterID,
		}
		if err := m.db.Where("name = ? AND cluster_id = ?", ns, clusterID).FirstOrCreate(&namespace).Error; err != nil {
			log.Printf("GetNamespaces: 插入 Namespace %s 失败: %v\n", ns, err)
			return nil, fmt.Errorf("插入 Namespace %s 失败: %w", ns, err)
		}
	}

	log.Printf("GetNamespaces: 获取并插入命名空间成功，ClusterID: %d\n", clusterID)
	return nsList, nil
}

// RecordProbeError 模拟 RecordProbeError 方法，并记录探针错误到数据库
func (m *K8sClientMock) RecordProbeError(ctx context.Context, clusterID int, errMsg string) {
	m.Lock()
	defer m.Unlock()

	if m.RecordProbeErrorFunc != nil {
		m.RecordProbeErrorFunc(ctx, clusterID, errMsg)
		return
	}

	// 插入或更新探针错误信息
	probeError := model.K8sProbeError{
		ClusterID: clusterID,
		ErrorMsg:  errMsg,
	}
	if err := m.db.Where("cluster_id = ?", clusterID).FirstOrCreate(&probeError).Error; err != nil {
		log.Printf("RecordProbeError: 插入 ProbeError 失败: %v\n", err)
		return
	}

	log.Printf("RecordProbeError: 记录探针错误成功，ClusterID: %d, ErrorMsg: %s\n", clusterID, errMsg)
}

// RefreshClients 模拟 RefreshClients 方法，并初始化多个 Cluster 的客户端数据
func (m *K8sClientMock) RefreshClients(ctx context.Context) error {
	m.Lock()
	defer m.Unlock()

	// 查询所有集群
	var clusters []model.K8sCluster
	if err := m.db.Find(&clusters).Error; err != nil {
		log.Printf("RefreshClients: 查询所有 K8sCluster 失败: %v\n", err)
		return err
	}

	for _, cluster := range clusters {
		// 假设每个集群都有一个 KubeConfigContent
		restConfig, err := clientcmd.RESTConfigFromKubeConfig([]byte(cluster.KubeConfigContent))
		if err != nil {
			log.Printf("RefreshClients: 解析 kubeconfig 失败, ClusterID: %d, Error: %v\n", cluster.ID, err)
			m.RecordProbeError(ctx, int(cluster.ID), "解析 kubeconfig 失败")
			continue
		}

		// 初始化客户端
		if err := m.InitClient(ctx, int(cluster.ID), restConfig); err != nil {
			log.Printf("RefreshClients: 初始化 Kubernetes 客户端失败, ClusterID: %d, Error: %v\n", cluster.ID, err)
			m.RecordProbeError(ctx, int(cluster.ID), "初始化 Kubernetes 客户端失败")
			continue
		}
	}

	log.Printf("RefreshClients: 所有集群的 Kubernetes 客户端刷新完成")
	return nil
}
