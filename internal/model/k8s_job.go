package model

import (
	batchv1 "k8s.io/api/batch/v1"
	"time"
)

// K8sJobRequest Job 相关请求结构
type K8sJobRequest struct {
	ClusterID int          `json:"cluster_id" binding:"required"` // 集群名称，必填
	Namespace string       `json:"namespace" binding:"required"`  // 命名空间，必填
	JobNames  []string     `json:"job_names"`                     // Job 名称，可选
	JobYaml   *batchv1.Job `json:"job_yaml"`                      // Job 对象, 可选
}

// K8sJobStatus Job 状态响应
type K8sJobStatus struct {
	Name                  string     `json:"name"`                    // Job 名称
	Namespace             string     `json:"namespace"`               // 命名空间
	Phase                 string     `json:"phase"`                   // Job 阶段 (Pending, Running, Succeeded, Failed)
	Active                int32      `json:"active"`                  // 活跃的Pod数
	Succeeded             int32      `json:"succeeded"`               // 成功的Pod数
	Failed                int32      `json:"failed"`                  // 失败的Pod数
	Completions           *int32     `json:"completions"`             // 期望完成数
	Parallelism           *int32     `json:"parallelism"`             // 并行度
	BackoffLimit          *int32     `json:"backoff_limit"`           // 重试限制
	ActiveDeadlineSeconds *int64     `json:"active_deadline_seconds"` // 活跃截止时间（秒）
	StartTime             *time.Time `json:"start_time"`              // 开始时间
	CompletionTime        *time.Time `json:"completion_time"`         // 完成时间
	CreationTimestamp     time.Time  `json:"creation_timestamp"`      // 创建时间
}

// K8sJobHistory Job 执行历史
type K8sJobHistory struct {
	Name              string     `json:"name"`               // Job 名称
	Namespace         string     `json:"namespace"`          // 命名空间
	Status            string     `json:"status"`             // 状态 (Pending, Running, Succeeded, Failed)
	Active            int32      `json:"active"`             // 活跃的Pod数
	Succeeded         int32      `json:"succeeded"`          // 成功的Pod数
	Failed            int32      `json:"failed"`             // 失败的Pod数
	StartTime         *time.Time `json:"start_time"`         // 开始时间
	CompletionTime    *time.Time `json:"completion_time"`    // 完成时间
	Duration          string     `json:"duration"`           // 执行时长
	CreationTimestamp time.Time  `json:"creation_timestamp"` // 创建时间
}
