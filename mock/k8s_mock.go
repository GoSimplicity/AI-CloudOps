/*
 * MIT License
 *
 * Copyright (c) 2024 Bamboo
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 *
 */

package mock

import (
	"log"
	"sync"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"gorm.io/gorm"
)

type K8sClientMock struct {
	sync.RWMutex
	db *gorm.DB
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

	// 检查是否已经初始化过集群
	var count int64
	m.db.Model(&model.K8sCluster{}).Count(&count)
	if count > 0 {
		log.Println("[K8s集群已经初始化过,跳过Mock]")
		return
	}

	log.Println("[K8s集群Mock开始]")

	clusters := []model.K8sCluster{
		{
			Name:          "Cluster-1",
			CreateUserID:  1,
			CpuRequest:    "100m",
			CpuLimit:      "200m",
			MemoryRequest: "256Mi",
			MemoryLimit:   "512Mi",
			Env:           model.EnvDev,
			Version:       "v1.32.0",
			ApiServerAddr: "https://kubernetes.example.invalid",
			KubeConfigContent: `apiVersion: v1
kind: Config
clusters:
- name: example
  cluster:
    server: https://kubernetes.example.invalid
contexts:
- name: example
  context:
    cluster: example
    user: example
current-context: example
users:
- name: example
  user: {}
`,
			ActionTimeoutSeconds: 30,
		},
	}

	// 插入mock数据
	for _, cluster := range clusters {
		if err := m.db.Create(&cluster).Error; err != nil {
			log.Printf("populateMockData: 插入 K8sCluster 失败: %v\n", err)
			continue
		}
		log.Printf("populateMockData: 初始化 Kubernetes 集群成功，ClusterID: %d\n", cluster.ID)
	}
}
