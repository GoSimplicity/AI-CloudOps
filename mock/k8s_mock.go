package mock

import (
	"context"
	"fmt"
	"gopkg.in/yaml.v3"
	"log"
	"sync"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"gorm.io/gorm"
	"k8s.io/client-go/rest"
)

type K8sClientMock struct {
	sync.RWMutex

	// 定义每个方法的返回值和行为
	InitClientFunc func(ctx context.Context, clusterID int, kubeConfig *rest.Config) error
	db             *gorm.DB
}

func NewK8sClientMock(db *gorm.DB) *K8sClientMock {
	mock := &K8sClientMock{
		db: db,
	}

	mock.populateMockData()

	return mock
}

// populateMockData 插入多个模拟集群和相关数据到数据库
func (m *K8sClientMock) populateMockData() {
	m.Lock()
	defer m.Unlock()

	// 示例：插入多个 K8sCluster
	clusters := []model.K8sCluster{
		{
			Name:                 "Cluster-1",
			NameZh:               "集群-1",
			UserID:               1,
			CpuRequest:           "100m",
			CpuLimit:             "200m",
			MemoryRequest:        "256Mi",
			MemoryLimit:          "512Mi",
			Env:                  "prod",
			Version:              "v1.20.0",
			ApiServerAddr:        "https://api.cluster1.example.com",
			KubeConfigContent:    m.generateKubeConfigContent("Cluster-1"),
			ActionTimeoutSeconds: 30,
		},
		{
			Name:                 "Cluster-2",
			NameZh:               "集群-2",
			UserID:               2,
			CpuRequest:           "200m",
			CpuLimit:             "400m",
			MemoryRequest:        "512Mi",
			MemoryLimit:          "1Gi",
			Env:                  "dev",
			Version:              "v1.19.0",
			ApiServerAddr:        "https://api.cluster2.example.com",
			KubeConfigContent:    m.generateKubeConfigContent("Cluster-2"),
			ActionTimeoutSeconds: 30,
		},
		// 可以添加更多集群
	}

	for _, cluster := range clusters {
		// 插入或更新 K8sCluster
		if err := m.db.Where("name = ?", cluster.Name).FirstOrCreate(&cluster).Error; err != nil {
			log.Printf("populateMockData: 插入 K8sCluster 失败: %v\n", err)
			continue
		}

		log.Printf("populateMockData: 初始化 Kubernetes 集群成功，ClusterID: %d\n", cluster.ID)

		// 为每个集群插入模拟节点
		var nodes []*model.K8sNode

		for i := 0; i < 30; i++ {
			node := &model.K8sNode{
				Name:      fmt.Sprintf("%s-mock-node-%d", cluster.Name, i),
				ClusterID: cluster.ID,
				// 根据需要添加其他节点字段
			}
			nodes = append(nodes, node)
		}

		if err := m.db.Create(&nodes).Error; err != nil {
			log.Printf("populateMockData: 插入 Node 失败: %v\n", err)
			continue
		}

		log.Printf("populateMockData: 为集群 %d 插入节点成功\n", cluster.ID)
	}
}

// InitClient 模拟 InitClient 方法，并插入模拟数据到数据库
func (m *K8sClientMock) InitClient(ctx context.Context, clusterID int, kubeConfig *rest.Config) error {
	m.Lock()
	defer m.Unlock()

	// 如果提供了自定义的 InitClientFunc，则优先使用它
	if m.InitClientFunc != nil {
		return m.InitClientFunc(ctx, clusterID, kubeConfig)
	}

	// 构建模拟的 K8sCluster 对象
	clusterName := fmt.Sprintf("Cluster-%d", clusterID)
	cluster := model.K8sCluster{
		Name:                 clusterName,
		NameZh:               fmt.Sprintf("集群-%d", clusterID),
		UserID:               1,
		CpuRequest:           "100m",
		CpuLimit:             "200m",
		MemoryRequest:        "256Mi",
		MemoryLimit:          "512Mi",
		Env:                  "prod",
		Version:              "v1.20.0",
		ApiServerAddr:        fmt.Sprintf("https://api.%s.example.com", clusterName),
		KubeConfigContent:    m.generateKubeConfigContent(clusterName), // 生成 kubeconfig 的内容
		ActionTimeoutSeconds: 30,
	}

	if err := m.db.Where("name = ?", cluster.Name).FirstOrCreate(&cluster).Error; err != nil {
		log.Printf("InitClient: 插入 K8sCluster 失败: %v\n", err)
		return fmt.Errorf("插入 K8sCluster 失败: %w", err)
	}

	log.Printf("InitClient: 初始化 Kubernetes 集群成功，ClusterID: %d\n", cluster.ID)

	// 为初始化的集群插入模拟节点
	var nodes []*model.K8sNode

	for i := 0; i < 30; i++ {
		node := &model.K8sNode{
			Name:      fmt.Sprintf("%s-mock-node-%d", cluster.Name, i),
			ClusterID: cluster.ID,
			// 根据需要添加其他节点字段
		}
		nodes = append(nodes, node)
	}

	if err := m.db.Create(&nodes).Error; err != nil {
		log.Printf("InitClient: 插入 Node 失败: %v\n", err)
		return fmt.Errorf("插入 Node 失败: %w", err)
	}

	log.Printf("InitClient: 为集群 %d 插入节点成功\n", cluster.ID)

	return nil
}

type KubeConfig struct {
	APIVersion     string                 `yaml:"apiVersion"`
	Kind           string                 `yaml:"kind"`
	Clusters       []NamedCluster         `yaml:"clusters"`
	Contexts       []NamedContext         `yaml:"contexts"`
	CurrentContext string                 `yaml:"current-context"`
	Preferences    map[string]interface{} `yaml:"preferences"`
	Users          []NamedAuthInfo        `yaml:"users"`
}

type NamedCluster struct {
	Name    string  `yaml:"name"`
	Cluster Cluster `yaml:"cluster"`
}

type Cluster struct {
	Server                   string `yaml:"server"`
	CertificateAuthorityData string `yaml:"certificate-authority-data"`
}

type NamedContext struct {
	Name    string  `yaml:"name"`
	Context Context `yaml:"context"`
}

type Context struct {
	Cluster string `yaml:"cluster"`
	User    string `yaml:"user"`
}

type NamedAuthInfo struct {
	Name     string   `yaml:"name"`
	AuthInfo AuthInfo `yaml:"user"`
}

type AuthInfo struct {
	ClientCertificateData string `yaml:"client-certificate-data"`
	ClientKeyData         string `yaml:"client-key-data"`
}

func (m *K8sClientMock) generateKubeConfigContent(clusterName string) string {
	kubeConfig := KubeConfig{
		APIVersion: "v1",
		Kind:       "Config",
		Clusters: []NamedCluster{
			{
				Name: "default",
				Cluster: Cluster{
					Server:                   fmt.Sprintf("https://%s.example.com:6443", clusterName),
					CertificateAuthorityData: "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJlRENDQVIyZ0F3SUJBZ0lCQURBS0JnZ3Foa2pPUFFRREFqQWpNU0V3SHdZRFZRUUREQmhyTTNNdGMyVnkKZG1WeUxXTmhRREUzTWpjek1qVTNNelF3SGhjTk1qUXdPVEkyTURRME1qRTBXaGNOTXpRd09USTBNRFEwTWpFMApXakFqTVNFd0h3WURWUVFEREJock0zTXRjMlZ5ZG1WeUxXTmhRREUzTWpjek1qVTNNelF3V1RBVEJnY3Foa2pPClBRSUJCZ2dxaGtqT1BRTUJCd05DQUFRNVBieGE0b2U1VHcxSVV3U3FGQXBEN1dBbFZjT1dFbitVM0owTXNQd3QKc3hBbEtjMFZnTGhGVU5zMlIwdUc3cUZzZ04wYWZxc2RkYWVtOEZvcnF2clFvMEl3UURBT0JnTlZIUThCQWY4RQpCQU1DQXFRd0R3WURWUjBUQVFIL0JBVXdBd0VCL3pBZEJnTlZIUTRFRmdRVXV1SGFOV1RRbmw2ZDNzZVBWUWpyCnpBZjZHYm93Q2dZSUtvWkl6ajBFQXdJRFNRQXdSZ0loQU1SdHhNZm84NVlNcFFITGcvTXhKTUJZRFhFSmVIY1oKWkE2S3E1Y1czOFVSQWlFQTVFajBiVlN6WGY3TFpiZ1lUb25WZ1FLaGI2elJDNkNxTXFzQXdERGx5Mjg9Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K",
				},
			},
		},
		Contexts: []NamedContext{
			{
				Name: "default",
				Context: Context{
					Cluster: "default",
					User:    "default",
				},
			},
		},
		CurrentContext: "default",
		Preferences:    map[string]interface{}{},
		Users: []NamedAuthInfo{
			{
				Name: "default",
				AuthInfo: AuthInfo{
					ClientCertificateData: "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJrakNDQVRlZ0F3SUJBZ0lJUFByaFlQaG4yb3d3Q2dZSUtvWkl6ajBFQXdJd0l6RWhNQjhHQTFVRUF3d1kKYXpOekxXTnNhV1Z1ZEMxallVQXhOekkzTXpJMU56TTBNQjRYRFRJME1Ea3lOakEwTkRJeE5Gb1hEVEkxTURreQpOakEwTkRJeE5Gb3dNREVYTUJVR0ExVUVDaE1PYzNsemRHVnRPbTFoYzNSbGNuTXhGVEFUQmdOVkJBTVRESE41CmMzUmxiVHBoWkcxcGJqQlpNQk1HQnlxR1NNNDlBZ0VHQ0NxR1NNNDlBd0VIQTBJQUJMWEk4d01lcHRna04zd2oKYkVhMGh1dDdVVkY3eW12aGxmc2NNT0lKQ1BYOWRNOWVFSnlrYmMySlFEMkszSjJ0YlZIVjgzblgyWElOVkFvYQpBL1dKSWdpalNEQkdNQTRHQTFVZER3RUIvd1FFQXdJRm9EQVRCZ05WSFNVRUREQUtCZ2dyQmdFRkJRY0RBakFmCkJnTlZIU01FR0RBV2dCVENRcTRBQk54WEJOZ3JrcFVIUFg4ZlN2TGZVREFLQmdncWhrak9QUVFEQWdOSkFEQkcKQWlFQWl5RG4ya3FGd1VvZHkxRzYvakE2YzdyeG8rQmZpbmQ2OUVuYitWbUNMU29DSVFEZ0czRmc1Mm5taUJQUwpNaDRnN1U0STBYcmJjeXd0MVZobkhmM0MxNGtwWXc9PQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCi0tLS0tQkVHSU4gQ0VSVElGSUNBVEUtLS0tLQpNSUlCZURDQ0FSMmdBd0lCQWdJQkFEQUtCZ2dxaGtqT1BRUURBakFqTVNFd0h3WURWUVFEREJock0zTXRZMnhwClpXNTBMV05oUURFM01qY3pNalUzTXpRd0hoY05NalF3T1RJMk1EUTBNakUwV2hjTk16UXdPVEkwTURRME1qRTAKV2pBak1TRXdId1lEVlFRRERCaHJNM010WTJ4cFpXNTBMV05oUURFM01qY3pNalUzTXpRd1dUQVRCZ2NxaGtqTwpQUUlCQmdncWhrak9QUU1CQndOQ0FBVEczN0EvZXR5MUFaMGpieDhMMmJ1OTFMWmZOems1Q1h3dUErVmxrWEl4CmVlL1AvMmIzQzhFMDNWaEpUMEZEOUhFb0pYeHZuWWZtN1JVRVhOby9LekNrbzBJd1FEQU9CZ05WSFE4QkFmOEUKQkFNQ0FxUXdEd1lEVlIwVEFRSC9CQVV3QXdFQi96QWRCZ05WSFE0RUZnUVV3a0t1QUFUY1Z3VFlLNUtWQnoxLwpIMHJ5MzFBd0NnWUlLb1pJemowRUF3SURTUUF3UmdJaEFLemlDV0dnSkZySzdyaExGbmtOVTh0MDVsUmlaNHIwCk5FMG5uYW0xTk01MEFpRUE4b0NBUXNyVnhFNU1INjZSVS92eldDb01iTFVNazhYUGpTMjI1blNqcWU0PQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg==",
					ClientKeyData:         "LS0tLS1CRUdJTiBFQyBQUklWQVRFIEtFWS0tLS0tCk1IY0NBUUVFSUQ4Z3RQNGFWMjlwcTlWekRhbmtLcFVIT2h6aGU0WXRKM3A3WHgxNTNCMVpvQW9HQ0NxR1NNNDkKQXdFSG9VUURRZ0FFdGNqekF4Nm0yQ1EzZkNOc1JyU0c2M3RSVVh2S2ErR1YreHd3NGdrSTlmMTB6MTRRbktSdAp6WWxBUFlyY25hMXRVZFh6ZWRmWmNnMVVDaG9EOVlraUNBPT0KLS0tLS1FTkQgRUMgUFJJVkFURSBLRVktLS0tLQo=",
				},
			},
		},
	}

	data, err := yaml.Marshal(&kubeConfig)
	if err != nil {
		log.Printf("generateKubeConfigContent: 序列化 kubeconfig 失败: %v\n", err)
		return ""
	}

	return string(data)
}
