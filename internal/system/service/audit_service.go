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
	"sync"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/system/dao"
	"go.uber.org/zap"
)

type AuditService interface {
	CreateAuditLog(ctx context.Context, req *model.CreateAuditLogRequest) error
	CreateAuditLogAsync(ctx context.Context, req *model.CreateAuditLogRequest) // 新增异步接口
	BatchCreateAuditLogs(ctx context.Context, logs []model.AuditLog) error
	ListAuditLogs(ctx context.Context, req *model.ListAuditLogsRequest) (*model.ListResp[model.AuditLog], error)
	GetAuditLogDetail(ctx context.Context, id int) (*model.AuditLog, error)
	SearchAuditLogs(ctx context.Context, req *model.SearchAuditLogsRequest) (*model.ListResp[model.AuditLog], error)
	GetAuditStatistics(ctx context.Context) (*model.AuditStatistics, error)
	GetAuditTypes(ctx context.Context) ([]model.AuditTypeInfo, error)
	ExportAuditLogs(ctx context.Context, req *model.ExportAuditLogsRequest) ([]byte, error)
	DeleteAuditLog(ctx context.Context, id int) error
	BatchDeleteAuditLogs(ctx context.Context, ids []int) error
	ArchiveAuditLogs(ctx context.Context, req *model.ArchiveAuditLogsRequest) error
	Close() error // 修改为返回error
}

type auditService struct {
	dao    dao.AuditDAO
	logger *zap.Logger

	// 性能优化相关字段
	logChannel    chan *model.AuditLog
	batchSize     int
	flushInterval time.Duration
	buffer        []*model.AuditLog
	bufferMutex   sync.Mutex
	done          chan struct{}
	wg            sync.WaitGroup
	closed        bool
	closeMutex    sync.RWMutex
}

func NewAuditService(dao dao.AuditDAO, logger *zap.Logger) AuditService {
	s := &auditService{
		dao:           dao,
		logger:        logger,
		logChannel:    make(chan *model.AuditLog, 10000), // 缓冲队列
		batchSize:     100,                               // 批量大小
		flushInterval: 5 * time.Second,                   // 刷新间隔
		buffer:        make([]*model.AuditLog, 0, 100),
		done:          make(chan struct{}),
		closed:        false,
	}

	// 启动后台批量处理协程
	s.wg.Add(1)
	go s.batchProcessor()

	return s
}

// 检查服务是否已关闭
func (s *auditService) isClosed() bool {
	s.closeMutex.RLock()
	defer s.closeMutex.RUnlock()
	return s.closed
}

// 同步创建审计日志
func (s *auditService) CreateAuditLog(ctx context.Context, req *model.CreateAuditLogRequest) error {
	if s.isClosed() {
		return fmt.Errorf("审计服务已关闭")
	}

	auditLog := s.buildAuditLog(req)

	if err := s.dao.CreateAuditLog(ctx, auditLog); err != nil {
		s.logger.Error("创建审计日志失败", zap.Error(err), zap.Any("请求", req))
		return fmt.Errorf("创建审计日志失败: %w", err)
	}

	return nil
}

// 异步创建审计日志 - 用于middleware高性能场景
func (s *auditService) CreateAuditLogAsync(ctx context.Context, req *model.CreateAuditLogRequest) {
	if s.isClosed() {
		s.logger.Warn("审计服务已关闭，丢弃日志",
			zap.String("操作", req.OperationType))
		return
	}

	auditLog := s.buildAuditLog(req)

	select {
	case s.logChannel <- auditLog:
		// 成功入队
	case <-ctx.Done():
		// 上下文取消
		s.logger.Debug("上下文取消，丢弃审计日志",
			zap.String("操作", req.OperationType))
		return
	default:
		// 队列满时直接丢弃并记录，避免创建goroutine
		s.logger.Warn("审计日志队列已满，丢弃日志",
			zap.String("操作", req.OperationType),
			zap.String("端点", req.Endpoint),
			zap.String("用户ID", fmt.Sprintf("%d", req.UserID)))
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

// 后台批量处理协程
func (s *auditService) batchProcessor() {
	defer s.wg.Done()
	ticker := time.NewTicker(s.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case log := <-s.logChannel:
			s.bufferMutex.Lock()
			s.buffer = append(s.buffer, log)
			shouldFlush := len(s.buffer) >= s.batchSize
			s.bufferMutex.Unlock()

			if shouldFlush {
				s.flushBufferWithContext(context.Background())
			}

		case <-ticker.C:
			s.flushBufferWithContext(context.Background())

		case <-s.done:
			// 处理剩余队列中的日志，设置超时避免死锁
			s.logger.Info("审计服务正在关闭，处理剩余日志")
			s.drainChannel()
			return
		}
	}
}

// 排空剩余的日志
func (s *auditService) drainChannel() {
	timeout := time.After(10 * time.Second) // 增加超时时间
	processed := 0

	for {
		select {
		case log := <-s.logChannel:
			s.bufferMutex.Lock()
			s.buffer = append(s.buffer, log)
			processed++
			// 达到批量大小或缓冲区快满时立即刷新
			if len(s.buffer) >= s.batchSize {
				s.bufferMutex.Unlock()
				s.flushBufferWithContext(context.Background())
			} else {
				s.bufferMutex.Unlock()
			}
		case <-timeout:
			s.logger.Warn("关闭超时，可能丢失一些日志",
				zap.Int("已处理", processed))
			s.flushBufferWithContext(context.Background())
			return
		default:
			// 队列为空，刷新缓冲区并退出
			s.flushBufferWithContext(context.Background())
			s.logger.Info("所有剩余日志已处理",
				zap.Int("已处理", processed))
			return
		}
	}
}

// 带上下文的刷新缓冲区
func (s *auditService) flushBufferWithContext(ctx context.Context) {
	s.bufferMutex.Lock()
	if len(s.buffer) == 0 {
		s.bufferMutex.Unlock()
		return
	}

	logs := make([]model.AuditLog, len(s.buffer))
	for i, log := range s.buffer {
		logs[i] = *log
	}
	count := len(s.buffer)
	s.buffer = s.buffer[:0] // 清空缓冲区，但保持容量
	s.bufferMutex.Unlock()

	// 使用传入的context，但设置最大超时
	flushCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// 最多重试3次
	maxRetries := 3
	var err error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		err = s.dao.BatchCreateAuditLogs(flushCtx, logs)
		if err == nil {
			break
		}
		s.logger.Warn("批量写入审计日志失败，正在重试", 
			zap.Error(err), 
			zap.Int("数量", count), 
			zap.Int("重试次数", attempt))
		
		// 如果不是最后一次尝试，则等待一段时间再重试
		if attempt < maxRetries {
			time.Sleep(time.Duration(attempt*500) * time.Millisecond)
		}
	}
	
	if err != nil {
		s.logger.Error("批量写入审计日志最终失败，放弃处理", 
			zap.Error(err), 
			zap.Int("数量", count), 
			zap.Int("尝试次数", maxRetries))
	} else {
		s.logger.Debug("成功刷新审计日志", zap.Int("数量", count))
	}
}

func (s *auditService) BatchCreateAuditLogs(ctx context.Context, logs []model.AuditLog) error {
	if s.isClosed() {
		return fmt.Errorf("审计服务已关闭")
	}

	if len(logs) == 0 {
		return nil
	}

	if err := s.dao.BatchCreateAuditLogs(ctx, logs); err != nil {
		s.logger.Error("批量创建审计日志失败", zap.Error(err), zap.Int("count", len(logs)))
		return fmt.Errorf("批量创建审计日志失败: %w", err)
	}

	return nil
}

func (s *auditService) ListAuditLogs(ctx context.Context, req *model.ListAuditLogsRequest) (*model.ListResp[model.AuditLog], error) {
	if s.isClosed() {
		return nil, fmt.Errorf("审计服务已关闭")
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

func (s *auditService) GetAuditLogDetail(ctx context.Context, id int) (*model.AuditLog, error) {
	if s.isClosed() {
		return nil, fmt.Errorf("审计服务已关闭")
	}

	log, err := s.dao.GetAuditLogByID(ctx, id)
	if err != nil {
		s.logger.Error("获取审计日志详情失败", zap.Error(err), zap.Int("id", id))
		return nil, fmt.Errorf("获取审计日志详情失败: %w", err)
	}

	return log, nil
}

func (s *auditService) SearchAuditLogs(ctx context.Context, req *model.SearchAuditLogsRequest) (*model.ListResp[model.AuditLog], error) {
	if s.isClosed() {
		return nil, fmt.Errorf("审计服务已关闭")
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

func (s *auditService) GetAuditStatistics(ctx context.Context) (*model.AuditStatistics, error) {
	if s.isClosed() {
		return nil, fmt.Errorf("审计服务已关闭")
	}

	stats, err := s.dao.GetAuditStatistics(ctx)
	if err != nil {
		s.logger.Error("获取审计统计信息失败", zap.Error(err))
		return nil, fmt.Errorf("获取审计统计信息失败: %w", err)
	}

	return stats, nil
}

func (s *auditService) GetAuditTypes(ctx context.Context) ([]model.AuditTypeInfo, error) {
	if s.isClosed() {
		return nil, fmt.Errorf("审计服务已关闭")
	}

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

func (s *auditService) ExportAuditLogs(ctx context.Context, req *model.ExportAuditLogsRequest) ([]byte, error) {
	if s.isClosed() {
		return nil, fmt.Errorf("审计服务已关闭")
	}

	data, err := s.dao.ExportAuditLogs(ctx, req)
	if err != nil {
		s.logger.Error("导出审计日志失败", zap.Error(err), zap.Any("request", req))
		return nil, fmt.Errorf("导出审计日志失败: %w", err)
	}

	return data, nil
}

func (s *auditService) DeleteAuditLog(ctx context.Context, id int) error {
	if s.isClosed() {
		return fmt.Errorf("审计服务已关闭")
	}

	if err := s.dao.DeleteAuditLog(ctx, id); err != nil {
		s.logger.Error("删除审计日志失败", zap.Error(err), zap.Int("ID", id))
		return fmt.Errorf("删除审计日志失败: %w", err)
	}

	return nil
}

func (s *auditService) BatchDeleteAuditLogs(ctx context.Context, ids []int) error {
	if s.isClosed() {
		return fmt.Errorf("审计服务已关闭")
	}

	if len(ids) == 0 {
		return nil
	}

	if err := s.dao.BatchDeleteAuditLogs(ctx, ids); err != nil {
		s.logger.Error("批量删除审计日志失败", zap.Error(err), zap.Ints("ID列表", ids))
		return fmt.Errorf("批量删除审计日志失败: %w", err)
	}

	return nil
}

func (s *auditService) ArchiveAuditLogs(ctx context.Context, req *model.ArchiveAuditLogsRequest) error {
	if s.isClosed() {
		return fmt.Errorf("审计服务已关闭")
	}

	if err := s.dao.ArchiveAuditLogs(ctx, req.StartTime, req.EndTime); err != nil {
		s.logger.Error("归档审计日志失败", zap.Error(err), zap.Any("请求", req))
		return fmt.Errorf("归档审计日志失败: %w", err)
	}

	return nil
}

// 关闭服务，确保所有日志都被处理
func (s *auditService) Close() error {
	s.closeMutex.Lock()
	if s.closed {
		s.closeMutex.Unlock()
		return nil // 已经关闭
	}
	s.closed = true
	s.closeMutex.Unlock()

	s.logger.Info("关闭审计服务...")

	// 发送关闭信号
	close(s.done)

	// 等待后台协程完成，设置超时避免无限等待
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		s.logger.Info("审计服务关闭成功")
		return nil
	case <-time.After(15 * time.Second): // 总超时时间
		s.logger.Error("超时等待审计服务关闭")
		return fmt.Errorf("超时等待审计服务关闭")
	}
}
