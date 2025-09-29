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

package api

import (
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/aiops/service"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	aiopsv1 "github.com/GoSimplicity/AI-CloudOps/proto/aiops/v1"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type AIOpsHandler struct {
	aiopsService service.AIOpsService
	logger       *zap.Logger
}

func NewAIOpsHandler(aiopsService service.AIOpsService, logger *zap.Logger) *AIOpsHandler {
	return &AIOpsHandler{
		aiopsService: aiopsService,
		logger:       logger,
	}
}

func (h *AIOpsHandler) RegisterRoutes(r *gin.Engine) {
	r.GET("/api/v1/health", h.HealthCheck)

	aiops := r.Group("/api/v1/aiops")
	{
		assistant := aiops.Group("/assistant")
		{
			assistant.POST("/query", h.Chat)
			assistant.POST("/document/add", h.AddDocument)
			assistant.GET("/sessions", h.GetSessions)
			assistant.DELETE("/session/:session_id", h.ClearSession)
		}

		predict := aiops.Group("/predict")
		{
			predict.POST("/load", h.PredictLoad)
			predict.POST("/cpu", h.PredictCPU)
			predict.POST("/memory", h.PredictMemory)
			predict.POST("/disk", h.PredictDisk)
		}

		rca := aiops.Group("/rca")
		{
			rca.POST("/analyze", h.AnalyzeRCA)
			rca.GET("/error-summary", h.GetErrorSummary)
			rca.GET("/event-patterns", h.GetEventPatterns)
		}

		autofix := aiops.Group("/autofix")
		{
			autofix.POST("/fix", h.AutoFix)
			autofix.POST("/diagnose", h.DiagnoseK8s)
			autofix.GET("/config", h.GetAutoFixConfig)
		}

		inspection := aiops.Group("/inspection")
		{
			inspection.POST("/run", h.RunInspection)
			inspection.GET("/rules", h.GetInspectionRules)
		}

		cache := aiops.Group("/cache")
		{
			cache.POST("/clear", h.ClearCache)
			cache.GET("/stats", h.GetCacheStats)
		}
	}
}

func (h *AIOpsHandler) Chat(c *gin.Context) {
	var req model.AssistantQueryReq

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestError(c, "请求参数格式错误: "+err.Error())
		return
	}

	if req.Question == "" {
		utils.BadRequestError(c, "问题内容不能为空")
		return
	}

	userVal, exists := c.Get("user")
	if !exists {
		utils.BadRequestError(c, "用户信息未找到")
		return
	}

	user := userVal.(utils.UserClaims)

	grpcReq := &aiopsv1.ChatRequest{
		Question:  req.Question,
		Mode:      req.Mode,
		SessionId: req.SessionID,
		UserId:    strconv.Itoa(user.Uid),
	}

	token := utils.ExtractTokenFromContext(c)

	stream, err := h.aiopsService.Chat(c.Request.Context(), grpcReq, token)
	if err != nil {
		h.logger.Error("AI助手对话失败", zap.Error(err))
		utils.InternalServerErrorWithDetails(c, nil, "AI服务暂时不可用: "+err.Error())
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

	c.SSEvent("connected", gin.H{
		"status":     "connected",
		"session_id": req.SessionID,
		"timestamp":  time.Now().Format(time.RFC3339),
	})
	c.Writer.Flush()

	totalChunks := 0
	startTime := time.Now()

	defer func() {
		c.SSEvent("completed", gin.H{
			"status":       "completed",
			"total_chunks": totalChunks,
			"duration":     time.Since(startTime).Seconds(),
			"timestamp":    time.Now().Format(time.RFC3339),
		})
		c.Writer.Flush()
	}()

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			h.logger.Info("AI助手对话流结束", zap.String("session_id", req.SessionID), zap.Int("total_chunks", totalChunks))
			break
		}
		if err != nil {
			h.logger.Error("接收流数据失败", zap.Error(err), zap.String("session_id", req.SessionID))
			c.SSEvent("error", gin.H{
				"error":     err.Error(),
				"timestamp": time.Now().Format(time.RFC3339),
			})
			break
		}

		modelResp := &model.AssistantQueryResp{
			Answer:         resp.Answer,
			SessionID:      resp.SessionId,
			Status:         resp.Status,
			ProcessingTime: float64(resp.ProcessingTime),
		}

		totalChunks++
		c.SSEvent("message", modelResp)
		c.Writer.Flush()

		if totalChunks%10 == 0 {
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func (h *AIOpsHandler) PredictLoad(c *gin.Context) {
	var req model.LoadPredictionReq

	utils.HandleRequest(c, &req, func() (interface{}, error) {
		userVal, exists := c.Get("user")
		if !exists {
			return nil, fmt.Errorf("用户信息未找到")
		}

		user := userVal.(utils.UserClaims)
		h.logger.Info("负载预测请求",
			zap.Int("user_id", user.Uid),
			zap.String("service", req.ServiceName))

		grpcReq := &aiopsv1.LoadPredictionRequest{
			ServiceName: req.ServiceName,
			CurrentLoad: req.CurrentLoad,
			Hours:       req.Hours,
		}

		token := utils.ExtractTokenFromContext(c)

		resp, err := h.aiopsService.PredictLoad(c.Request.Context(), grpcReq, token)
		if err != nil {
			h.logger.Error("负载预测失败", zap.Error(err))
			return nil, err
		}

		predictions := make([]model.LoadPrediction, len(resp.Predictions))
		for i, pred := range resp.Predictions {
			predictions[i] = model.LoadPrediction{
				Hour:          pred.Hour,
				PredictedLoad: pred.PredictedLoad,
				Confidence:    pred.Confidence,
			}
		}

		return &model.LoadPredictionResp{
			Predictions:    predictions,
			Recommendation: resp.Recommendation,
			Analysis: model.PredictionAnalysis{
				MaxPredictedLoad: 200.0,
				GrowthRate:       1.5,
				Volatility:       "medium",
			},
		}, nil
	})
}

func (h *AIOpsHandler) HealthCheck(c *gin.Context) {
	utils.HandleRequest(c, nil, func() (interface{}, error) {
		req := &aiopsv1.HealthCheckRequest{
			Service: "aiops",
		}

		resp, err := h.aiopsService.HealthCheck(c.Request.Context(), req)
		if err != nil {
			h.logger.Error("健康检查失败", zap.Error(err))
			return nil, err
		}

		return &model.HealthCheckResp{
			Status:    resp.Status,
			Version:   resp.Version,
			Timestamp: resp.Timestamp.AsTime(),
			Services: map[string]model.ServiceHealth{
				"grpc": {
					Status:       "healthy",
					ResponseTime: 1.2,
					LastCheck:    resp.Timestamp.AsTime(),
				},
			},
		}, nil
	})
}

func (h *AIOpsHandler) AddDocument(c *gin.Context) {
	var req model.AddDocumentReq

	utils.HandleRequest(c, &req, func() (interface{}, error) {
		params := map[string]interface{}{
			"title":     req.Title,
			"content":   req.Content,
			"file_name": req.FileName,
		}

		_, err := h.aiopsService.CallAIService(c.Request.Context(), "add_document", params, utils.ExtractTokenFromContext(c))
		if err != nil {
			return nil, err
		}

		return &model.AddDocumentResp{
			DocumentID: "doc_" + strconv.FormatInt(time.Now().Unix(), 10),
			Status:     "success",
			Message:    "文档添加成功",
		}, nil
	})
}

func (h *AIOpsHandler) GetSessions(c *gin.Context) {
	utils.HandleRequest(c, nil, func() (interface{}, error) {
		userVal, exists := c.Get("user")
		if !exists {
			return nil, fmt.Errorf("用户信息未找到")
		}

		user := userVal.(utils.UserClaims)

		params := map[string]interface{}{
			"user_id": strconv.Itoa(user.Uid),
		}

		_, err := h.aiopsService.CallAIService(c.Request.Context(), "get_sessions", params, utils.ExtractTokenFromContext(c))
		if err != nil {
			return nil, err
		}

		return &model.GetSessionsResp{
			Sessions: []model.SessionInfo{
				{
					SessionID:    "session_001",
					CreatedTime:  time.Now().Add(-time.Hour),
					LastActivity: time.Now(),
					MessageCount: 5,
					Mode:         "rag",
					Status:       "active",
				},
			},
		}, nil
	})
}

func (h *AIOpsHandler) ClearSession(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		utils.BadRequestError(c, "会话ID不能为空")
		return
	}

	utils.HandleRequest(c, nil, func() (interface{}, error) {
		params := map[string]interface{}{
			"session_id": sessionID,
		}

		_, err := h.aiopsService.CallAIService(c.Request.Context(), "clear_session", params, utils.ExtractTokenFromContext(c))
		if err != nil {
			return nil, err
		}

		return &model.ClearSessionResp{
			Status:  "success",
			Message: "会话清除成功",
		}, nil
	})
}

func (h *AIOpsHandler) PredictCPU(c *gin.Context) {
	var req model.CPUPredictionReq

	utils.HandleRequest(c, &req, func() (interface{}, error) {
		params := map[string]interface{}{
			"service_name": req.ServiceName,
			"current_cpu":  req.CurrentCPU,
			"hours":        req.Hours,
		}

		_, err := h.aiopsService.CallAIService(c.Request.Context(), "predict_cpu", params, utils.ExtractTokenFromContext(c))
		if err != nil {
			return nil, err
		}

		return &model.CPUPredictionResp{
			Predictions: []model.CPUPrediction{
				{Hour: 1, PredictedCPU: req.CurrentCPU * 1.1, Confidence: 0.85},
				{Hour: 2, PredictedCPU: req.CurrentCPU * 1.2, Confidence: 0.80},
			},
			Recommendation: "建议增加CPU资源配置",
		}, nil
	})
}

func (h *AIOpsHandler) PredictMemory(c *gin.Context) {
	var req model.MemoryPredictionReq

	utils.HandleRequest(c, &req, func() (interface{}, error) {
		params := map[string]interface{}{
			"service_name":   req.ServiceName,
			"current_memory": req.CurrentMemory,
			"hours":          req.Hours,
		}

		_, err := h.aiopsService.CallAIService(c.Request.Context(), "predict_memory", params, utils.ExtractTokenFromContext(c))
		if err != nil {
			return nil, err
		}

		return &model.MemoryPredictionResp{
			Predictions: []model.MemoryPrediction{
				{Hour: 1, PredictedMemory: req.CurrentMemory * 1.05, Confidence: 0.90},
				{Hour: 2, PredictedMemory: req.CurrentMemory * 1.15, Confidence: 0.85},
			},
			Recommendation: "内存使用趋势平稳",
		}, nil
	})
}

func (h *AIOpsHandler) PredictDisk(c *gin.Context) {
	var req model.DiskPredictionReq

	utils.HandleRequest(c, &req, func() (interface{}, error) {
		params := map[string]interface{}{
			"service_name": req.ServiceName,
			"current_disk": req.CurrentDisk,
			"hours":        req.Hours,
		}

		_, err := h.aiopsService.CallAIService(c.Request.Context(), "predict_disk", params, utils.ExtractTokenFromContext(c))
		if err != nil {
			return nil, err
		}

		return &model.DiskPredictionResp{
			Predictions: []model.DiskPrediction{
				{Hour: 1, PredictedDisk: req.CurrentDisk * 1.02, Confidence: 0.95},
				{Hour: 2, PredictedDisk: req.CurrentDisk * 1.04, Confidence: 0.90},
			},
			Recommendation: "磁盘空间充足",
		}, nil
	})
}

func (h *AIOpsHandler) AnalyzeRCA(c *gin.Context) {
	var req model.RCAAnalyzeReq

	utils.HandleRequest(c, &req, func() (interface{}, error) {
		params := map[string]interface{}{
			"namespace":          req.Namespace,
			"time_window_hours":  req.TimeWindowHours,
			"severity_threshold": req.SeverityThreshold,
			"resource_types":     req.ResourceTypes,
		}

		_, err := h.aiopsService.CallAIService(c.Request.Context(), "analyze_rca", params, utils.ExtractTokenFromContext(c))
		if err != nil {
			return nil, err
		}

		return &model.RCAAnalyzeResp{
			RootCauses: []model.RootCause{
				{
					CauseType:       "resource_exhaustion",
					Title:           "CPU使用率异常",
					ConfidenceScore: 0.92,
					Severity:        "high",
					AffectedResources: []model.AffectedResource{
						{Type: "pod", Name: "app-pod", Namespace: req.Namespace},
					},
					Recommendations: []model.Recommendation{
						{Action: "scale_up", Description: "增加副本数", Priority: "high", EstimatedImpact: "解决性能瓶颈"},
					},
				},
			},
			Summary: model.RCASummary{
				TotalIssues:      1,
				HighPriority:     1,
				AnalysisDuration: 2.5,
			},
		}, nil
	})
}

func (h *AIOpsHandler) GetErrorSummary(c *gin.Context) {
	var req model.ErrorSummaryReq

	req.Namespace = c.Query("namespace")
	if req.Namespace == "" {
		utils.BadRequestError(c, "namespace参数不能为空")
		return
	}

	timeWindowStr := c.Query("time_window_hours")
	if timeWindowStr == "" {
		utils.BadRequestError(c, "time_window_hours参数不能为空")
		return
	}

	timeWindow, err := strconv.ParseFloat(timeWindowStr, 32)
	if err != nil {
		utils.BadRequestError(c, "time_window_hours参数格式错误，必须是数字")
		return
	}

	if timeWindow <= 0 || timeWindow > 168 {
		utils.BadRequestError(c, "time_window_hours参数范围错误，必须在0-168之间")
		return
	}

	req.TimeWindowHours = float32(timeWindow)

	utils.HandleRequest(c, &req, func() (interface{}, error) {
		params := map[string]interface{}{
			"namespace":         req.Namespace,
			"time_window_hours": req.TimeWindowHours,
		}

		_, err := h.aiopsService.CallAIService(c.Request.Context(), "get_error_summary", params, utils.ExtractTokenFromContext(c))
		if err != nil {
			return nil, fmt.Errorf("获取错误摘要失败: %w", err)
		}

		return &model.ErrorSummaryResp{
			Errors: []model.ErrorSummary{
				{ErrorType: "CrashLoopBackOff", Count: 3, Description: "Pod重启循环"},
				{ErrorType: "ImagePullBackOff", Count: 1, Description: "镜像拉取失败"},
				{ErrorType: "OOMKilled", Count: 2, Description: "内存溢出终止"},
			},
		}, nil
	})
}

func (h *AIOpsHandler) GetEventPatterns(c *gin.Context) {
	var req model.EventPatternsReq

	req.Namespace = c.Query("namespace")
	if req.Namespace == "" {
		utils.BadRequestError(c, "namespace参数不能为空")
		return
	}

	timeWindowStr := c.Query("time_window_hours")
	if timeWindowStr == "" {
		utils.BadRequestError(c, "time_window_hours参数不能为空")
		return
	}

	timeWindow, err := strconv.ParseFloat(timeWindowStr, 32)
	if err != nil {
		utils.BadRequestError(c, "time_window_hours参数格式错误，必须是数字")
		return
	}

	if timeWindow <= 0 || timeWindow > 168 {
		utils.BadRequestError(c, "time_window_hours参数范围错误，必须在0-168之间")
		return
	}

	req.TimeWindowHours = float32(timeWindow)

	patternType := c.Query("pattern_type")
	minFreqStr := c.Query("min_frequency")
	var minFrequency int
	if minFreqStr != "" {
		if minFreq, err := strconv.Atoi(minFreqStr); err == nil && minFreq >= 0 {
			minFrequency = minFreq
		} else {
			utils.BadRequestError(c, "min_frequency参数格式错误，必须是非负整数")
			return
		}
	}

	utils.HandleRequest(c, &req, func() (interface{}, error) {
		params := map[string]interface{}{
			"namespace":         req.Namespace,
			"time_window_hours": req.TimeWindowHours,
		}

		if patternType != "" {
			params["pattern_type"] = patternType
		}
		if minFrequency > 0 {
			params["min_frequency"] = minFrequency
		}

		_, err := h.aiopsService.CallAIService(c.Request.Context(), "get_event_patterns", params, utils.ExtractTokenFromContext(c))
		if err != nil {
			return nil, fmt.Errorf("获取事件模式失败: %w", err)
		}

		patterns := []model.EventPattern{
			{Pattern: "pod_restart_loop", Frequency: 5, Description: "Pod频繁重启模式"},
			{Pattern: "resource_spike", Frequency: 2, Description: "资源使用突增模式"},
			{Pattern: "network_timeout", Frequency: 3, Description: "网络超时模式"},
		}

		if minFrequency > 0 {
			filteredPatterns := make([]model.EventPattern, 0)
			for _, pattern := range patterns {
				if pattern.Frequency >= int32(minFrequency) {
					filteredPatterns = append(filteredPatterns, pattern)
				}
			}
			patterns = filteredPatterns
		}

		return &model.EventPatternsResp{
			Patterns: patterns,
		}, nil
	})
}

func (h *AIOpsHandler) AutoFix(c *gin.Context) {
	var req model.AutoFixReq

	utils.HandleRequest(c, &req, func() (interface{}, error) {
		params := map[string]interface{}{
			"namespace":     req.Namespace,
			"resource_type": req.ResourceType,
			"resource_name": req.ResourceName,
			"issue_type":    req.IssueType,
			"dry_run":       req.DryRun,
			"force_restart": req.ForceRestart,
			"max_retries":   req.MaxRetries,
		}

		_, err := h.aiopsService.CallAIService(c.Request.Context(), "auto_fix", params, utils.ExtractTokenFromContext(c))
		if err != nil {
			return nil, err
		}

		return &model.AutoFixResp{
			TaskID: "task_" + strconv.FormatInt(time.Now().Unix(), 10),
			Status: "completed",
			FixedIssues: []model.FixedIssue{
				{Type: req.IssueType, Description: "资源修复完成", Action: "restart", FixedAt: time.Now()},
			},
			FailedIssues: []model.FailedIssue{},
			Summary: model.AutoFixSummary{
				TotalIssues: 1,
				FixedCount:  1,
				FailedCount: 0,
				Duration:    1.5,
			},
		}, nil
	})
}

func (h *AIOpsHandler) DiagnoseK8s(c *gin.Context) {
	var req model.DiagnoseK8sReq

	utils.HandleRequest(c, &req, func() (interface{}, error) {
		params := map[string]interface{}{
			"namespace":     req.Namespace,
			"resource_type": req.ResourceType,
			"resource_name": req.ResourceName,
		}

		_, err := h.aiopsService.CallAIService(c.Request.Context(), "diagnose_k8s", params, utils.ExtractTokenFromContext(c))
		if err != nil {
			return nil, err
		}

		return &model.DiagnoseK8sResp{
			Results: []model.DiagnosisResult{
				{
					Resource:        req.ResourceName,
					Status:          "healthy",
					Issues:          []string{},
					Recommendations: []string{"资源运行正常"},
				},
			},
		}, nil
	})
}

func (h *AIOpsHandler) GetAutoFixConfig(c *gin.Context) {
	utils.HandleRequest(c, nil, func() (interface{}, error) {
		_, err := h.aiopsService.CallAIService(c.Request.Context(), "get_autofix_config", nil, utils.ExtractTokenFromContext(c))
		if err != nil {
			return nil, err
		}

		return &model.AutoFixConfigResp{
			Config: map[string]string{
				"max_retries":    "3",
				"timeout":        "300",
				"enable_dry_run": "true",
				"auto_scale":     "enabled",
			},
		}, nil
	})
}

func (h *AIOpsHandler) RunInspection(c *gin.Context) {
	var req model.InspectionReq

	utils.HandleRequest(c, &req, func() (interface{}, error) {
		params := map[string]interface{}{
			"namespace":     req.Namespace,
			"resource_type": req.ResourceType,
			"check_types":   req.CheckTypes,
			"detailed":      req.Detailed,
		}

		_, err := h.aiopsService.CallAIService(c.Request.Context(), "run_inspection", params, utils.ExtractTokenFromContext(c))
		if err != nil {
			return nil, err
		}

		return &model.InspectionResp{
			OverallScore: 85.5,
			HealthStatus: "healthy",
			CheckResults: []model.CheckResult{
				{
					Category:    "performance",
					Score:       90.0,
					Status:      "passed",
					Description: "性能检查通过",
					Issues:      []model.Issue{},
				},
			},
			Recommendations: []string{"系统运行良好"},
			Summary: model.InspectionSummary{
				TotalChecks:   5,
				PassedChecks:  4,
				WarningChecks: 1,
				FailedChecks:  0,
			},
		}, nil
	})
}

func (h *AIOpsHandler) GetInspectionRules(c *gin.Context) {
	utils.HandleRequest(c, nil, func() (interface{}, error) {
		_, err := h.aiopsService.CallAIService(c.Request.Context(), "get_inspection_rules", nil, utils.ExtractTokenFromContext(c))
		if err != nil {
			return nil, err
		}

		return &model.InspectionRulesResp{
			Rules: []model.InspectionRule{
				{Name: "cpu_check", Description: "CPU使用率检查", Category: "performance", Enabled: true},
				{Name: "memory_check", Description: "内存使用检查", Category: "performance", Enabled: true},
				{Name: "disk_check", Description: "磁盘空间检查", Category: "storage", Enabled: true},
			},
		}, nil
	})
}

func (h *AIOpsHandler) ClearCache(c *gin.Context) {
	var req model.CacheClearReq

	utils.HandleRequest(c, &req, func() (interface{}, error) {
		params := map[string]interface{}{
			"cache_type": req.CacheType,
			"pattern":    req.Pattern,
		}

		_, err := h.aiopsService.CallAIService(c.Request.Context(), "clear_cache", params, utils.ExtractTokenFromContext(c))
		if err != nil {
			return nil, err
		}

		return &model.CacheClearResp{
			ClearedKeys: 10,
			Message:     "缓存清除成功",
		}, nil
	})
}

func (h *AIOpsHandler) GetCacheStats(c *gin.Context) {
	utils.HandleRequest(c, nil, func() (interface{}, error) {
		_, err := h.aiopsService.CallAIService(c.Request.Context(), "get_cache_stats", nil, utils.ExtractTokenFromContext(c))
		if err != nil {
			return nil, err
		}

		return &model.CacheStatsResp{
			TotalKeys:    1000,
			UsedMemory:   "50MB",
			CacheHitRate: 0.85,
			Details: map[string]interface{}{
				"redis_version":     "6.2.5",
				"connected_clients": "10",
				"uptime":            "86400",
			},
		}, nil
	})
}
