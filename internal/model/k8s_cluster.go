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

package model

type Env int8

const (
	EnvProd  Env = iota + 1 // 生产环境
	EnvDev                  // 开发环境
	EnvStage                // 预发环境
	EnvRc                   // 测试环境
	EnvPress                // 灰度环境
)

type ClusterStatus int8

const (
	StatusRunning ClusterStatus = iota + 1 // 运行中
	StatusStopped                          // 停止
	StatusError                            // 异常
)

// K8sCluster Kubernetes 集群的配置
type K8sCluster struct {
	Model
	Name                 string        `json:"name" binding:"required,min=1,max=200" gorm:"size:100;comment:集群名称"`        // 集群名称
	CpuRequest           string        `json:"cpu_request,omitempty" gorm:"comment:CPU 请求量 (m)"`                          // CPU 请求量
	CpuLimit             string        `json:"cpu_limit,omitempty" gorm:"comment:CPU 限制量 (m)"`                            // CPU 限制量
	MemoryRequest        string        `json:"memory_request,omitempty" gorm:"comment:内存请求量 (Mi)"`                        // 内存请求量
	MemoryLimit          string        `json:"memory_limit,omitempty" gorm:"comment:内存限制量 (Mi)"`                          // 内存限制量
	RestrictNamespace    StringList    `json:"restrict_namespace" gorm:"comment:资源限制命名空间"`                                // 资源限制命名空间
	Status               ClusterStatus `json:"status" gorm:"index;comment:集群状态 (1:Running, 2:Stopped, 3:Error)"`          // 集群状态
	Env                  Env           `json:"env,omitempty" gorm:"comment:集群环境 (1:Prod, 2:Dev, 3:Stage, 4:Rc, 5:Press)"` // 集群环境
	Version              string        `json:"version,omitempty" gorm:"comment:集群版本"`                                     // 集群版本
	ApiServerAddr        string        `json:"api_server_addr,omitempty" gorm:"comment:API Server 地址"`                    // API Server 地址
	KubeConfigContent    string        `json:"kube_config_content,omitempty" gorm:"type:text;comment:kubeConfig 内容"`      // kubeConfig 内容
	ActionTimeoutSeconds int           `json:"action_timeout_seconds,omitempty" gorm:"comment:操作超时时间（秒）"`                 // 操作超时时间（秒）
	CreateUserName       string        `json:"create_user_name,omitempty" gorm:"comment:创建者用户名"`                          // 创建者用户名
	CreateUserID         int           `json:"create_user_id,omitempty" gorm:"comment:创建者用户ID"`                           // 创建者用户ID
	Tags                 KeyValueList  `json:"tags,omitempty" gorm:"type:text;serializer:json;comment:标签"`                // 标签
}

func (k8sCluster *K8sCluster) TableName() string {
	return "cl_k8s_clusters"
}

// CreateClusterReq 创建集群请求
type CreateClusterReq struct {
	Name                 string       `json:"name" binding:"required,min=1,max=200"` // 集群名称
	CpuRequest           string       `json:"cpu_request,omitempty"`                 // CPU 请求量
	CpuLimit             string       `json:"cpu_limit,omitempty"`                   // CPU 限制量
	MemoryRequest        string       `json:"memory_request,omitempty"`              // 内存请求量
	MemoryLimit          string       `json:"memory_limit,omitempty"`                // 内存限制量
	RestrictNamespace    StringList   `json:"restrict_namespace"`                    // 资源限制命名空间
	Env                  Env          `json:"env,omitempty"`                         // 集群环境
	Version              string       `json:"version,omitempty"`                     // 集群版本
	ApiServerAddr        string       `json:"api_server_addr,omitempty"`             // API Server 地址
	KubeConfigContent    string       `json:"kube_config_content,omitempty"`         // kubeConfig 内容
	ActionTimeoutSeconds int          `json:"action_timeout_seconds,omitempty"`      // 操作超时时间（秒）
	CreateUserName       string       `json:"create_user_name,omitempty"`            // 创建者用户名
	CreateUserID         int          `json:"create_user_id,omitempty"`              // 创建者用户ID
	Tags                 KeyValueList `json:"tags,omitempty"`                        // 标签
}

// UpdateClusterReq 更新集群请求
type UpdateClusterReq struct {
	ID                   int          `json:"id" form:"id" uri:"id" binding:"required" comment:"集群ID"`
	Name                 string       `json:"name" binding:"required,min=1,max=200"` // 集群名称
	CpuRequest           string       `json:"cpu_request,omitempty"`                 // CPU 请求量
	CpuLimit             string       `json:"cpu_limit,omitempty"`                   // CPU 限制量
	MemoryRequest        string       `json:"memory_request,omitempty"`              // 内存请求量
	MemoryLimit          string       `json:"memory_limit,omitempty"`                // 内存限制量
	RestrictNamespace    StringList   `json:"restrict_namespace"`                    // 资源限制命名空间
	Env                  Env          `json:"env,omitempty"`                         // 集群环境
	Version              string       `json:"version,omitempty"`                     // 集群版本
	ApiServerAddr        string       `json:"api_server_addr,omitempty"`             // API Server 地址
	KubeConfigContent    string       `json:"kube_config_content,omitempty"`         // kubeConfig 内容
	ActionTimeoutSeconds int          `json:"action_timeout_seconds,omitempty"`      // 操作超时时间（秒）
	Tags                 KeyValueList `json:"tags,omitempty"`                        // 标签
}

// DeleteClusterReq 删除集群请求
type DeleteClusterReq struct {
	ID int `json:"id" form:"id" uri:"id" binding:"required" comment:"集群ID"`
}

// RefreshClusterReq 刷新集群请求
type RefreshClusterReq struct {
	ID int `json:"id" form:"id" uri:"id" binding:"required" comment:"集群ID"`
}

// GetClusterReq 获取单个集群请求
type GetClusterReq struct {
	ID int `json:"id" form:"id" uri:"id" binding:"required" comment:"集群ID"`
}

// ListClustersReq 获取集群列表请求
type ListClustersReq struct {
	ListReq
	Status string `json:"status" form:"status"`
	Env    string `json:"env" form:"env"`
}
