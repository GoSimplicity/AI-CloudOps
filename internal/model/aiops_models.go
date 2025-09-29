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
 */

package model

import (
	"time"
)

// ===============================
// AI助手相关模型
// ===============================

// AssistantQueryReq AI助手查询请求
type AssistantQueryReq struct {
	Question  string `json:"question" binding:"required" validate:"min=1"`
	Mode      string `json:"mode" binding:"omitempty,oneof=rag mcp" validate:"oneof=rag mcp"`
	SessionID string `json:"session_id" binding:"omitempty"`
}

// AssistantQueryResp AI助手查询响应
type AssistantQueryResp struct {
	Answer            string           `json:"answer"`
	SessionID         string           `json:"session_id"`
	Status            string           `json:"status"`
	SourceDocuments   []SourceDocument `json:"source_documents,omitempty"`
	RelevanceScore    float64          `json:"relevance_score,omitempty"`
	RecallRate        float64          `json:"recall_rate,omitempty"`
	FollowUpQuestions []string         `json:"follow_up_questions,omitempty"`
	ProcessingTime    float64          `json:"processing_time"`
	ToolCalls         []ToolCall       `json:"tool_calls,omitempty"`
}

// SourceDocument 源文档信息
type SourceDocument struct {
	Title          string  `json:"title"`
	Content        string  `json:"content"`
	RelevanceScore float64 `json:"relevance_score"`
	Source         string  `json:"source,omitempty"`
}

// ToolCall 工具调用信息
type ToolCall struct {
	ToolName      string  `json:"tool_name"`
	ExecutionTime float64 `json:"execution_time"`
	Status        string  `json:"status"`
	Result        string  `json:"result,omitempty"`
}

// ===============================
// 负载预测相关模型
// ===============================

// LoadPredictionReq 负载预测请求
type LoadPredictionReq struct {
	ServiceName string  `json:"service_name" binding:"required"`
	CurrentLoad float32 `json:"current_load" binding:"required,min=0"`
	Hours       int32   `json:"hours" binding:"required,min=1,max=168"`
}

// LoadPredictionResp 负载预测响应
type LoadPredictionResp struct {
	Predictions    []LoadPrediction   `json:"predictions"`
	Recommendation string             `json:"recommendation"`
	Analysis       PredictionAnalysis `json:"analysis"`
}

// LoadPrediction 单个负载预测点
type LoadPrediction struct {
	Hour          int32   `json:"hour"`
	PredictedLoad float32 `json:"predicted_load"`
	Confidence    float32 `json:"confidence"`
}

// PredictionAnalysis 预测分析
type PredictionAnalysis struct {
	MaxPredictedLoad float32 `json:"max_predicted_load"`
	GrowthRate       float32 `json:"growth_rate"`
	Volatility       string  `json:"volatility"`
}

// ===============================
// 根因分析相关模型
// ===============================

// RCAAnalyzeReq 根因分析请求
type RCAAnalyzeReq struct {
	Namespace         string   `json:"namespace" binding:"required"`
	TimeWindowHours   float32  `json:"time_window_hours" binding:"required,min=0.1,max=24"`
	SeverityThreshold float32  `json:"severity_threshold" binding:"omitempty,min=0,max=1"`
	ResourceTypes     []string `json:"resource_types" binding:"omitempty"`
}

// RCAAnalyzeResp 根因分析响应
type RCAAnalyzeResp struct {
	RootCauses []RootCause `json:"root_causes"`
	Summary    RCASummary  `json:"summary"`
}

// RootCause 根本原因
type RootCause struct {
	CauseType          string               `json:"cause_type"`
	Title              string               `json:"title"`
	ConfidenceScore    float64              `json:"confidence_score"`
	Severity           string               `json:"severity"`
	AffectedResources  []AffectedResource   `json:"affected_resources"`
	Recommendations    []Recommendation     `json:"recommendations"`
	SupportingEvidence []SupportingEvidence `json:"supporting_evidence"`
}

// AffectedResource 受影响的资源
type AffectedResource struct {
	Type      string `json:"type"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

// Recommendation 建议
type Recommendation struct {
	Action          string `json:"action"`
	Description     string `json:"description"`
	Priority        string `json:"priority"`
	EstimatedImpact string `json:"estimated_impact"`
}

// SupportingEvidence 支撑证据
type SupportingEvidence struct {
	Type        string    `json:"type"`
	Description string    `json:"description"`
	Timestamp   time.Time `json:"timestamp"`
}

// RCASummary 根因分析摘要
type RCASummary struct {
	TotalIssues      int     `json:"total_issues"`
	HighPriority     int     `json:"high_priority"`
	AnalysisDuration float64 `json:"analysis_duration"`
}

// ===============================
// 自动修复相关模型
// ===============================

// AutoFixReq 自动修复请求
type AutoFixReq struct {
	Namespace    string `json:"namespace" binding:"required"`
	ResourceType string `json:"resource_type" binding:"required,oneof=pod deployment service"`
	ResourceName string `json:"resource_name" binding:"required"`
	IssueType    string `json:"issue_type" binding:"required"`
	DryRun       bool   `json:"dry_run" binding:"omitempty"`
	ForceRestart bool   `json:"force_restart" binding:"omitempty"`
	MaxRetries   int    `json:"max_retries" binding:"omitempty,min=1,max=10"`
}

// AutoFixResp 自动修复响应
type AutoFixResp struct {
	TaskID       string         `json:"task_id"`
	Status       string         `json:"status"`
	FixedIssues  []FixedIssue   `json:"fixed_issues"`
	FailedIssues []FailedIssue  `json:"failed_issues"`
	Summary      AutoFixSummary `json:"summary"`
}

// FixedIssue 已修复问题
type FixedIssue struct {
	Type        string    `json:"type"`
	Description string    `json:"description"`
	Action      string    `json:"action"`
	FixedAt     time.Time `json:"fixed_at"`
}

// FailedIssue 修复失败问题
type FailedIssue struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Reason      string `json:"reason"`
}

// AutoFixSummary 自动修复摘要
type AutoFixSummary struct {
	TotalIssues int     `json:"total_issues"`
	FixedCount  int     `json:"fixed_count"`
	FailedCount int     `json:"failed_count"`
	Duration    float64 `json:"duration"`
}

// ===============================
// 系统检查相关模型
// ===============================

// InspectionReq 系统检查请求
type InspectionReq struct {
	Namespace    string   `json:"namespace" binding:"omitempty"`
	ResourceType string   `json:"resource_type" binding:"omitempty"`
	CheckTypes   []string `json:"check_types" binding:"omitempty"`
	Detailed     bool     `json:"detailed" binding:"omitempty"`
}

// InspectionResp 系统检查响应
type InspectionResp struct {
	OverallScore    float64           `json:"overall_score"`
	HealthStatus    string            `json:"health_status"`
	CheckResults    []CheckResult     `json:"check_results"`
	Recommendations []string          `json:"recommendations"`
	Summary         InspectionSummary `json:"summary"`
}

// CheckResult 检查结果
type CheckResult struct {
	Category    string  `json:"category"`
	Score       float64 `json:"score"`
	Status      string  `json:"status"`
	Description string  `json:"description"`
	Issues      []Issue `json:"issues,omitempty"`
}

// Issue 问题项
type Issue struct {
	Type        string `json:"type"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Resource    string `json:"resource,omitempty"`
}

// InspectionSummary 检查摘要
type InspectionSummary struct {
	TotalChecks   int `json:"total_checks"`
	PassedChecks  int `json:"passed_checks"`
	WarningChecks int `json:"warning_checks"`
	FailedChecks  int `json:"failed_checks"`
}

// ===============================
// 缓存管理相关模型
// ===============================

// CacheClearReq 缓存清除请求
type CacheClearReq struct {
	CacheType string `json:"cache_type" binding:"omitempty,oneof=all knowledge vector session"`
	Pattern   string `json:"pattern" binding:"omitempty"`
}

// CacheClearResp 缓存清除响应
type CacheClearResp struct {
	ClearedKeys int    `json:"cleared_keys"`
	Message     string `json:"message"`
}

// CacheStatsResp 缓存统计响应
type CacheStatsResp struct {
	TotalKeys    int                    `json:"total_keys"`
	UsedMemory   string                 `json:"used_memory"`
	CacheHitRate float64                `json:"cache_hit_rate"`
	Details      map[string]interface{} `json:"details"`
}

// ===============================
// 预测扩展模型
// ===============================

// CPUPredictionReq CPU预测请求
type CPUPredictionReq struct {
	ServiceName string  `json:"service_name" binding:"required"`
	CurrentCPU  float32 `json:"current_cpu" binding:"required,min=0"`
	Hours       int32   `json:"hours" binding:"required,min=1,max=168"`
}

// CPUPredictionResp CPU预测响应
type CPUPredictionResp struct {
	Predictions    []CPUPrediction `json:"predictions"`
	Recommendation string          `json:"recommendation"`
}

// CPUPrediction CPU预测点
type CPUPrediction struct {
	Hour         int32   `json:"hour"`
	PredictedCPU float32 `json:"predicted_cpu"`
	Confidence   float32 `json:"confidence"`
}

// MemoryPredictionReq 内存预测请求
type MemoryPredictionReq struct {
	ServiceName   string  `json:"service_name" binding:"required"`
	CurrentMemory float32 `json:"current_memory" binding:"required,min=0"`
	Hours         int32   `json:"hours" binding:"required,min=1,max=168"`
}

// MemoryPredictionResp 内存预测响应
type MemoryPredictionResp struct {
	Predictions    []MemoryPrediction `json:"predictions"`
	Recommendation string             `json:"recommendation"`
}

// MemoryPrediction 内存预测点
type MemoryPrediction struct {
	Hour            int32   `json:"hour"`
	PredictedMemory float32 `json:"predicted_memory"`
	Confidence      float32 `json:"confidence"`
}

// DiskPredictionReq 磁盘预测请求
type DiskPredictionReq struct {
	ServiceName string  `json:"service_name" binding:"required"`
	CurrentDisk float32 `json:"current_disk" binding:"required,min=0"`
	Hours       int32   `json:"hours" binding:"required,min=1,max=168"`
}

// DiskPredictionResp 磁盘预测响应
type DiskPredictionResp struct {
	Predictions    []DiskPrediction `json:"predictions"`
	Recommendation string           `json:"recommendation"`
}

// DiskPrediction 磁盘预测点
type DiskPrediction struct {
	Hour          int32   `json:"hour"`
	PredictedDisk float32 `json:"predicted_disk"`
	Confidence    float32 `json:"confidence"`
}

// ===============================
// AI助手扩展模型
// ===============================

// AddDocumentReq 添加文档请求
type AddDocumentReq struct {
	Title    string `json:"title" binding:"required"`
	Content  string `json:"content" binding:"required"`
	FileName string `json:"file_name" binding:"required"`
}

// AddDocumentResp 添加文档响应
type AddDocumentResp struct {
	DocumentID string `json:"document_id"`
	Status     string `json:"status"`
	Message    string `json:"message"`
}

// GetSessionsResp 获取会话列表响应
type GetSessionsResp struct {
	Sessions []SessionInfo `json:"sessions"`
}

// SessionInfo 会话信息
type SessionInfo struct {
	SessionID    string    `json:"session_id"`
	CreatedTime  time.Time `json:"created_time"`
	LastActivity time.Time `json:"last_activity"`
	MessageCount int32     `json:"message_count"`
	Mode         string    `json:"mode"`
	Status       string    `json:"status"`
}

// ClearSessionResp 清除会话响应
type ClearSessionResp struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// ===============================
// 错误和事件模型
// ===============================

// ErrorSummaryReq 错误摘要请求
type ErrorSummaryReq struct {
	Namespace       string  `json:"namespace" binding:"required"`
	TimeWindowHours float32 `json:"time_window_hours" binding:"required,min=0.1,max=24"`
}

// ErrorSummaryResp 错误摘要响应
type ErrorSummaryResp struct {
	Errors []ErrorSummary `json:"errors"`
}

// ErrorSummary 错误摘要
type ErrorSummary struct {
	ErrorType   string `json:"error_type"`
	Count       int32  `json:"count"`
	Description string `json:"description"`
}

// EventPatternsReq 事件模式请求
type EventPatternsReq struct {
	Namespace       string  `json:"namespace" binding:"required"`
	TimeWindowHours float32 `json:"time_window_hours" binding:"required,min=0.1,max=24"`
}

// EventPatternsResp 事件模式响应
type EventPatternsResp struct {
	Patterns []EventPattern `json:"patterns"`
}

// EventPattern 事件模式
type EventPattern struct {
	Pattern     string `json:"pattern"`
	Frequency   int32  `json:"frequency"`
	Description string `json:"description"`
}

// ===============================
// 诊断和配置模型
// ===============================

// DiagnoseK8sReq K8s诊断请求
type DiagnoseK8sReq struct {
	Namespace    string `json:"namespace" binding:"required"`
	ResourceType string `json:"resource_type" binding:"required"`
	ResourceName string `json:"resource_name" binding:"required"`
}

// DiagnoseK8sResp K8s诊断响应
type DiagnoseK8sResp struct {
	Results []DiagnosisResult `json:"results"`
}

// DiagnosisResult 诊断结果
type DiagnosisResult struct {
	Resource        string   `json:"resource"`
	Status          string   `json:"status"`
	Issues          []string `json:"issues"`
	Recommendations []string `json:"recommendations"`
}

// AutoFixConfigResp 自动修复配置响应
type AutoFixConfigResp struct {
	Config map[string]string `json:"config"`
}

// InspectionRulesResp 检查规则响应
type InspectionRulesResp struct {
	Rules []InspectionRule `json:"rules"`
}

// InspectionRule 检查规则
type InspectionRule struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Enabled     bool   `json:"enabled"`
}

// ===============================
// 健康检查相关模型
// ===============================

// HealthCheckResp 健康检查响应
type HealthCheckResp struct {
	Status    string                   `json:"status"`
	Version   string                   `json:"version"`
	Timestamp time.Time                `json:"timestamp"`
	Services  map[string]ServiceHealth `json:"services,omitempty"`
}

// ServiceHealth 服务健康状态
type ServiceHealth struct {
	Status       string    `json:"status"`
	ResponseTime float64   `json:"response_time"`
	LastCheck    time.Time `json:"last_check"`
}
