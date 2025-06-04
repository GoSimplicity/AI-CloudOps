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

package service

import (
	"context"
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/system/dao"
	"go.uber.org/zap"
)

type AuditService interface {
	CreateAuditLog(ctx context.Context, req *model.CreateAuditLogRequest) error
	CreateAuditLogAsync(ctx context.Context, req *model.CreateAuditLogRequest)
	BatchCreateAuditLogs(ctx context.Context, logs []model.AuditLog) error
	ListAuditLogs(ctx context.Context, req *model.ListAuditLogsRequest) (*model.ListResp[model.AuditLog], error)
	GetAuditLogDetail(ctx context.Context, id int) (*model.AuditLog, error)
	SearchAuditLogs(ctx context.Context, req *model.SearchAuditLogsRequest) (*model.ListResp[model.AuditLog], error)
	GetAuditStatistics(ctx context.Context) (*model.AuditStatistics, error)
	GetAuditTypes(ctx context.Context) ([]model.AuditTypeInfo, error)
	DeleteAuditLog(ctx context.Context, id int) error
	BatchDeleteAuditLogs(ctx context.Context, ids []int) error
	ArchiveAuditLogs(ctx context.Context, req *model.ArchiveAuditLogsRequest) error
	Close() error
}

type auditService struct {
	dao       dao.AuditDAO
	logger    *zap.Logger
	asyncChan chan *model.AuditLog
	done      chan struct{}
}

func NewAuditService(dao dao.AuditDAO, logger *zap.Logger) AuditService {
	s := &auditService{
		dao:       dao,
		logger:    logger,
		asyncChan: make(chan *model.AuditLog, 1000),
		done:      make(chan struct{}),
	}

	// 启动单个后台处理协程
	go s.processAsync()

	return s
}

// 同步创建审计日志
func (s *auditService) CreateAuditLog(ctx context.Context, req *model.CreateAuditLogRequest) error {
	auditLog := s.buildAuditLog(req)
	if err := s.dao.CreateAuditLog(ctx, auditLog); err != nil {
		s.logger.Error("创建审计日志失败", zap.Error(err))
		return fmt.Errorf("创建审计日志失败: %w", err)
	}
	return nil
}

// 异步创建审计日志
func (s *auditService) CreateAuditLogAsync(ctx context.Context, req *model.CreateAuditLogRequest) {
	auditLog := s.buildAuditLog(req)

	select {
	case s.asyncChan <- auditLog:
		// 成功入队
	default:
		// 队列满时记录并丢弃
		s.logger.Warn("审计日志队列已满",
			zap.String("操作", req.OperationType),
			zap.String("端点", req.Endpoint))
	}
}

// 简化的异步处理
func (s *auditService) processAsync() {
	for {
		select {
		case log := <-s.asyncChan:
			ctx := context.Background()
			if err := s.dao.CreateAuditLog(ctx, log); err != nil {
				s.logger.Error("异步创建审计日志失败", zap.Error(err))
			}
		case <-s.done:
			// 处理剩余日志
			for {
				select {
				case log := <-s.asyncChan:
					ctx := context.Background()
					if err := s.dao.CreateAuditLog(ctx, log); err != nil {
						s.logger.Error("关闭时处理审计日志失败", zap.Error(err))
					}
				default:
					return
				}
			}
		}
	}
}

// 构建审计日志对象
func (s *auditService) buildAuditLog(req *model.CreateAuditLogRequest) *model.AuditLog {
	return &model.AuditLog{
		UserID:        req.UserID,
		TraceID:       req.TraceID,
		IPAddress:     req.IPAddress,
		UserAgent:     req.UserAgent,
		HttpMethod:    req.HttpMethod,
		Endpoint:      req.Endpoint,
		OperationType: req.OperationType,
		TargetType:    req.TargetType,
		TargetID:      req.TargetID,
		StatusCode:    req.StatusCode,
		RequestBody:   req.RequestBody,
		ResponseBody:  req.ResponseBody,
		Duration:      req.Duration,
		ErrorMsg:      req.ErrorMsg,
	}
}

// 批量创建审计日志
func (s *auditService) BatchCreateAuditLogs(ctx context.Context, logs []model.AuditLog) error {
	if len(logs) == 0 {
		return nil
	}

	if err := s.dao.BatchCreateAuditLogs(ctx, logs); err != nil {
		s.logger.Error("批量创建审计日志失败", zap.Error(err), zap.Int("count", len(logs)))
		return fmt.Errorf("批量创建审计日志失败: %w", err)
	}
	return nil
}

// 获取审计日志列表
func (s *auditService) ListAuditLogs(ctx context.Context, req *model.ListAuditLogsRequest) (*model.ListResp[model.AuditLog], error) {
	// 参数校验
	req.Size = s.validatePageSize(req.Size, 10, 100)
	if req.Page <= 0 {
		req.Page = 1
	}

	total, logs, err := s.dao.ListAuditLogs(ctx, req)
	if err != nil {
		s.logger.Error("获取审计日志列表失败", zap.Error(err), zap.Any("request", req))
		return nil, fmt.Errorf("获取审计日志列表失败: %w", err)
	}

	return &model.ListResp[model.AuditLog]{
		Items: logs,
		Total: total,
	}, nil
}

// 获取审计日志详情
func (s *auditService) GetAuditLogDetail(ctx context.Context, id int) (*model.AuditLog, error) {
	log, err := s.dao.GetAuditLogByID(ctx, id)
	if err != nil {
		s.logger.Error("获取审计日志详情失败", zap.Error(err), zap.Int("id", id))
		return nil, fmt.Errorf("获取审计日志详情失败: %w", err)
	}
	return log, nil
}

// 搜索审计日志
func (s *auditService) SearchAuditLogs(ctx context.Context, req *model.SearchAuditLogsRequest) (*model.ListResp[model.AuditLog], error) {
	// 参数校验
	req.Size = s.validatePageSize(req.Size, 10, 100)
	if req.Page <= 0 {
		req.Page = 1
	}

	total, logs, err := s.dao.SearchAuditLogs(ctx, req)
	if err != nil {
		s.logger.Error("搜索审计日志失败", zap.Error(err), zap.Any("request", req))
		return nil, fmt.Errorf("搜索审计日志失败: %w", err)
	}

	return &model.ListResp[model.AuditLog]{
		Items: logs,
		Total: total,
	}, nil
}

// 获取审计统计信息
func (s *auditService) GetAuditStatistics(ctx context.Context) (*model.AuditStatistics, error) {
	stats, err := s.dao.GetAuditStatistics(ctx)
	if err != nil {
		s.logger.Error("获取审计统计信息失败", zap.Error(err))
		return nil, fmt.Errorf("获取审计统计信息失败: %w", err)
	}
	return stats, nil
}

// 获取审计类型列表
func (s *auditService) GetAuditTypes(ctx context.Context) ([]model.AuditTypeInfo, error) {
	auditTypes := []model.AuditTypeInfo{
		{Type: "CREATE", Description: "创建操作", Category: "数据操作"},
		{Type: "UPDATE", Description: "更新操作", Category: "数据操作"},
		{Type: "DELETE", Description: "删除操作", Category: "数据操作"},
		{Type: "VIEW", Description: "查看操作", Category: "数据访问"},
		{Type: "LOGIN", Description: "登录操作", Category: "身份认证"},
		{Type: "LOGOUT", Description: "登出操作", Category: "身份认证"},
		{Type: "EXPORT", Description: "导出操作", Category: "数据导出"},
		{Type: "IMPORT", Description: "导入操作", Category: "数据导入"},
		{Type: "CONFIG", Description: "配置操作", Category: "系统配置"},
		{Type: "DEPLOY", Description: "部署操作", Category: "运维操作"},
	}

	return auditTypes, nil
}

// 删除审计日志
func (s *auditService) DeleteAuditLog(ctx context.Context, id int) error {
	if err := s.dao.DeleteAuditLog(ctx, id); err != nil {
		s.logger.Error("删除审计日志失败", zap.Error(err), zap.Int("ID", id))
		return fmt.Errorf("删除审计日志失败: %w", err)
	}
	return nil
}

// 批量删除审计日志
func (s *auditService) BatchDeleteAuditLogs(ctx context.Context, ids []int) error {
	if len(ids) == 0 {
		return nil
	}

	// 限制批量删除数量
	if len(ids) > 1000 {
		s.logger.Warn("批量删除审计日志数量过多，已限制为最大值",
			zap.Int("原始数量", len(ids)), zap.Int("最大数量", 1000))
		ids = ids[:1000]
	}

	if err := s.dao.BatchDeleteAuditLogs(ctx, ids); err != nil {
		s.logger.Error("批量删除审计日志失败", zap.Error(err), zap.Ints("ID列表", ids))
		return fmt.Errorf("批量删除审计日志失败: %w", err)
	}
	return nil
}

// 归档审计日志
func (s *auditService) ArchiveAuditLogs(ctx context.Context, req *model.ArchiveAuditLogsRequest) error {
	if err := s.dao.ArchiveAuditLogs(ctx, req.StartTime, req.EndTime); err != nil {
		s.logger.Error("归档审计日志失败", zap.Error(err), zap.Any("请求", req))
		return fmt.Errorf("归档审计日志失败: %w", err)
	}
	return nil
}

// 关闭服务
func (s *auditService) Close() error {
	s.logger.Info("关闭审计服务...")
	close(s.done)
	s.logger.Info("审计服务关闭成功")
	return nil
}

// 工具方法：校验分页大小
func (s *auditService) validatePageSize(size, defaultSize, maxSize int) int {
	if size <= 0 {
		return defaultSize
	}
	if size > maxSize {
		return maxSize
	}
	return size
}
