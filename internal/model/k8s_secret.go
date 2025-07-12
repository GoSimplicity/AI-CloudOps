package model

import (
	core "k8s.io/api/core/v1"
	"time"
)

// K8sSecretRequest Secret 相关请求结构
type K8sSecretRequest struct {
	ClusterID   int          `json:"cluster_id" binding:"required"` // 集群名称，必填
	Namespace   string       `json:"namespace" binding:"required"`  // 命名空间，必填
	SecretNames []string     `json:"secret_names"`                  // Secret 名称，可选
	SecretYaml  *core.Secret `json:"secret_yaml"`                   // Secret 对象, 可选
}

// K8sSecretStatus Secret 状态响应
type K8sSecretStatus struct {
	Name              string          `json:"name"`               // Secret 名称
	Namespace         string          `json:"namespace"`          // 命名空间
	Type              core.SecretType `json:"type"`               // Secret 类型
	DataKeys          []string        `json:"data_keys"`          // 数据键列表（不包含敏感值）
	DataSize          int             `json:"data_size"`          // 数据总大小
	Immutable         *bool           `json:"immutable"`          // 是否不可变
	CreationTimestamp time.Time       `json:"creation_timestamp"` // 创建时间
}

// K8sSecretEncryptionRequest Secret 加密请求
type K8sSecretEncryptionRequest struct {
	ClusterID  int               `json:"cluster_id" binding:"required"` // 集群ID
	Namespace  string            `json:"namespace" binding:"required"`  // 命名空间
	Name       string            `json:"name" binding:"required"`       // Secret 名称
	Type       core.SecretType   `json:"type"`                          // Secret 类型
	Data       map[string]string `json:"data"`                          // 明文数据
	StringData map[string]string `json:"string_data"`                   // 字符串数据
	Immutable  *bool             `json:"immutable"`                     // 是否不可变
}
