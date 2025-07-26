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

package alert

import (
	"context"
	"errors"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/cache"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert"
	userDao "github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// 业务错误定义
var (
	ErrGroupExists       = errors.New("值班组已存在")
	ErrGroupIDEmpty      = errors.New("值班组ID为空")
	ErrGroupNotFound     = errors.New("值班组不存在")
	ErrGroupHasSendGroup = errors.New("值班组存在关联发送组，无法删除")
	ErrInvalidTimeRange  = errors.New("时间范围无效")
	ErrInvalidShiftDays  = errors.New("轮班天数无效")
	ErrMembersNotFound   = errors.New("部分成员不存在")
	ErrTimeRangeTooLong  = errors.New("查询时间范围不能超过一年")
)

// 常量定义
const (
	MaxTimeRangeDays = 365
	DefaultPageSize  = 10
	DateFormat       = "2006-01-02"
)

type AlertManagerOnDutyService interface {
	GetMonitorOnDutyGroupList(ctx context.Context, req *model.GetMonitorOnDutyGroupListReq) (model.ListResp[*model.MonitorOnDutyGroup], error)
	CreateMonitorOnDutyGroup(ctx context.Context, req *model.CreateMonitorOnDutyGroupReq) error
	CreateMonitorOnDutyGroupChange(ctx context.Context, req *model.CreateMonitorOnDutyGroupChangeReq) error
	UpdateMonitorOnDutyGroup(ctx context.Context, req *model.UpdateMonitorOnDutyGroupReq) error
	DeleteMonitorOnDutyGroup(ctx context.Context, req *model.DeleteMonitorOnDutyGroupReq) error
	GetMonitorOnDutyGroup(ctx context.Context, req *model.GetMonitorOnDutyGroupReq) (*model.MonitorOnDutyGroup, error)
	GetMonitorOnDutyGroupFuturePlan(ctx context.Context, req *model.GetMonitorOnDutyGroupFuturePlanReq) ([]*model.MonitorOnDutyOne, error)
	GetMonitorOnDutyHistory(ctx context.Context, req *model.GetMonitorOnDutyHistoryReq) (model.ListResp[*model.MonitorOnDutyHistory], error)
}

type onDutyService struct {
	dao     alert.AlertManagerOnDutyDAO
	sendDao alert.AlertManagerSendDAO
	cache   cache.MonitorCache
	userDao userDao.UserDAO
	logger  *zap.Logger
}

func NewAlertManagerOnDutyService(dao alert.AlertManagerOnDutyDAO, sendDao alert.AlertManagerSendDAO, cache cache.MonitorCache, l *zap.Logger, userDao userDao.UserDAO) AlertManagerOnDutyService {
	return &onDutyService{
		dao:     dao,
		userDao: userDao,
		sendDao: sendDao,
		logger:  l,
		cache:   cache,
	}
}

// 值班组管理

func (s *onDutyService) GetMonitorOnDutyGroupList(ctx context.Context, req *model.GetMonitorOnDutyGroupListReq) (model.ListResp[*model.MonitorOnDutyGroup], error) {
	groups, total, err := s.dao.GetMonitorOnDutyList(ctx, req)
	if err != nil {
		s.logger.Error("获取值班组列表失败", zap.Error(err))
		return model.ListResp[*model.MonitorOnDutyGroup]{}, err
	}

	// 为每个值班组填充今日值班人信息
	for _, group := range groups {
		if len(group.Users) > 0 {
			group.TodayDutyUser = s.getTodayDutyUser(ctx, group)
		}
	}

	return model.ListResp[*model.MonitorOnDutyGroup]{
		Items: groups,
		Total: total,
	}, nil
}

func (s *onDutyService) CreateMonitorOnDutyGroup(ctx context.Context, req *model.CreateMonitorOnDutyGroupReq) error {
	// 参数验证
	if err := s.validateCreateGroupRequest(req); err != nil {
		return err
	}

	// 检查值班组名称是否已存在
	if exists, err := s.dao.CheckMonitorOnDutyGroupExists(ctx, &model.MonitorOnDutyGroup{Name: req.Name}); err != nil {
		s.logger.Error("检查值班组是否存在失败", zap.Error(err))
		return err
	} else if exists {
		return ErrGroupExists
	}

	// 获取并验证所有成员是否存在
	users, err := s.userDao.GetUserByIDs(ctx, req.UserIDs)
	if err != nil {
		s.logger.Error("获取成员信息失败", zap.Error(err))
		return err
	}
	if len(users) != len(req.UserIDs) {
		return ErrMembersNotFound
	}

	// 创建值班组对象
	group := s.buildGroupFromRequest(req, users)

	// 保存值班组到数据库
	if err := s.dao.CreateMonitorOnDutyGroup(ctx, group); err != nil {
		s.logger.Error("创建值班组失败", zap.Error(err))
		return err
	}

	return nil
}

func (s *onDutyService) CreateMonitorOnDutyGroupChange(ctx context.Context, req *model.CreateMonitorOnDutyGroupChangeReq) error {
	// 验证值班组是否存在
	if _, err := s.dao.GetMonitorOnDutyGroupByID(ctx, req.OnDutyGroupID); err != nil {
		s.logger.Error("值班组不存在", zap.Int("groupID", req.OnDutyGroupID), zap.Error(err))
		return ErrGroupNotFound
	}

	// 创建值班变更记录
	change := s.buildChangeFromRequest(req)

	// 保存值班变更记录到数据库
	if err := s.dao.CreateMonitorOnDutyGroupChange(ctx, change); err != nil {
		s.logger.Error("创建值班变更失败", zap.Error(err))
		return err
	}

	return nil
}

func (s *onDutyService) UpdateMonitorOnDutyGroup(ctx context.Context, req *model.UpdateMonitorOnDutyGroupReq) error {
	// 参数验证
	if err := s.validateUpdateGroupRequest(req); err != nil {
		return err
	}

	// 获取要更新的值班组
	group, err := s.dao.GetMonitorOnDutyGroupByID(ctx, req.ID)
	if err != nil {
		s.logger.Error("获取值班组失败", zap.Int("id", req.ID), zap.Error(err))
		return ErrGroupNotFound
	}

	// 如果名称变更，检查新名称是否已存在
	if group.Name != req.Name {
		if exists, err := s.dao.CheckMonitorOnDutyGroupExists(ctx, &model.MonitorOnDutyGroup{Name: req.Name}); err != nil {
			s.logger.Error("检查值班组名称是否存在失败", zap.Error(err))
			return err
		} else if exists {
			return ErrGroupExists
		}
	}

	// 获取并验证所有成员是否存在
	users, err := s.userDao.GetUserByIDs(ctx, req.UserIDs)
	if err != nil {
		s.logger.Error("获取成员信息失败", zap.Error(err))
		return err
	}
	if len(users) != len(req.UserIDs) {
		return ErrMembersNotFound
	}

	// 更新值班组信息
	s.updateGroupFromRequest(group, req, users)

	// 保存更新后的值班组到数据库
	if err := s.dao.UpdateMonitorOnDutyGroup(ctx, group); err != nil {
		s.logger.Error("更新值班组失败", zap.Error(err))
		return err
	}

	return nil
}

func (s *onDutyService) DeleteMonitorOnDutyGroup(ctx context.Context, req *model.DeleteMonitorOnDutyGroupReq) error {
	// 检查值班组是否关联了发送组，如果有关联则不允许删除
	_, count, err := s.sendDao.GetMonitorSendGroupByOnDutyGroupID(ctx, req.ID)
	if err != nil {
		s.logger.Error("检查关联发送组失败", zap.Error(err))
		return err
	}
	if count > 0 {
		return ErrGroupHasSendGroup
	}

	// 删除值班组
	if err := s.dao.DeleteMonitorOnDutyGroup(ctx, req.ID); err != nil {
		s.logger.Error("删除值班组失败", zap.Int("id", req.ID), zap.Error(err))
		return err
	}

	return nil
}

func (s *onDutyService) GetMonitorOnDutyGroup(ctx context.Context, req *model.GetMonitorOnDutyGroupReq) (*model.MonitorOnDutyGroup, error) {
	// 获取指定ID的值班组
	group, err := s.dao.GetMonitorOnDutyGroupByID(ctx, req.ID)
	if err != nil {
		s.logger.Error("获取值班组失败", zap.Int("id", req.ID), zap.Error(err))
		return nil, ErrGroupNotFound
	}

	// 获取今日值班人员信息
	group.TodayDutyUser = s.getTodayDutyUser(ctx, group)
	return group, nil
}

// 值班计划和历史

func (s *onDutyService) GetMonitorOnDutyGroupFuturePlan(ctx context.Context, req *model.GetMonitorOnDutyGroupFuturePlanReq) ([]*model.MonitorOnDutyOne, error) {
	// 解析并验证时间范围
	startTime, endTime, err := s.parseAndValidateTimeRange(req.StartTime, req.EndTime)
	if err != nil {
		return nil, err
	}

	// 获取值班组信息
	group, err := s.dao.GetMonitorOnDutyGroupByID(ctx, req.ID)
	if err != nil {
		s.logger.Error("获取值班组失败", zap.Int("id", req.ID), zap.Error(err))
		return nil, ErrGroupNotFound
	}

	// 获取指定时间范围内的值班变更记录
	changes, err := s.dao.GetMonitorOnDutyChangesByGroupAndTimeRange(ctx, req.ID, req.StartTime, req.EndTime)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		s.logger.Error("获取值班变更失败", zap.Error(err))
		return nil, err
	}

	// 生成值班计划
	return s.generateDutyPlan(ctx, group, startTime, endTime, changes), nil
}

func (s *onDutyService) GetMonitorOnDutyHistory(ctx context.Context, req *model.GetMonitorOnDutyHistoryReq) (model.ListResp[*model.MonitorOnDutyHistory], error) {
	// 获取值班历史记录
	histories, total, err := s.dao.GetMonitorOnDutyHistoryList(ctx, req)
	if err != nil {
		s.logger.Error("获取值班历史失败", zap.Error(err))
		return model.ListResp[*model.MonitorOnDutyHistory]{}, err
	}

	// 返回值班历史记录和总数
	return model.ListResp[*model.MonitorOnDutyHistory]{
		Items: histories,
		Total: total,
	}, nil
}

// 私有辅助方法

func (s *onDutyService) validateCreateGroupRequest(req *model.CreateMonitorOnDutyGroupReq) error {
	if req.ShiftDays <= 0 {
		return ErrInvalidShiftDays
	}
	if len(req.UserIDs) == 0 {
		return ErrMembersNotFound
	}
	return nil
}

func (s *onDutyService) validateUpdateGroupRequest(req *model.UpdateMonitorOnDutyGroupReq) error {
	if req.ShiftDays <= 0 {
		return ErrInvalidShiftDays
	}
	if len(req.UserIDs) == 0 {
		return ErrMembersNotFound
	}
	return nil
}

func (s *onDutyService) buildGroupFromRequest(req *model.CreateMonitorOnDutyGroupReq, users []*model.User) *model.MonitorOnDutyGroup {
	return &model.MonitorOnDutyGroup{
		Name:           req.Name,
		UserID:         req.UserID,
		ShiftDays:      req.ShiftDays,
		CreateUserName: req.CreateUserName,
		Description:    req.Description,
		Users:          s.convertUsersToOnDutyUsers(users),
	}
}

func (s *onDutyService) buildChangeFromRequest(req *model.CreateMonitorOnDutyGroupChangeReq) *model.MonitorOnDutyChange {
	return &model.MonitorOnDutyChange{
		OnDutyGroupID:  req.OnDutyGroupID,
		UserID:         req.UserID,
		Date:           req.Date,
		OriginUserID:   req.OriginUserID,
		OnDutyUserID:   req.OnDutyUserID,
		CreateUserName: req.CreateUserName,
		Reason:         req.Reason,
	}
}

func (s *onDutyService) updateGroupFromRequest(group *model.MonitorOnDutyGroup, req *model.UpdateMonitorOnDutyGroupReq, users []*model.User) {
	group.Name = req.Name
	group.ShiftDays = req.ShiftDays
	group.Description = req.Description
	group.Users = s.convertUsersToOnDutyUsers(users)
	if req.Enable != nil {
		group.Enable = *req.Enable
	}
}

func (s *onDutyService) convertUsersToOnDutyUsers(users []*model.User) []*model.User {
	onDutyUsers := make([]*model.User, len(users))
	for i, user := range users {
		onDutyUsers[i] = &model.User{
			Model:        model.Model{ID: user.ID},
			RealName:     user.RealName,
			Username:     user.Username,
			FeiShuUserId: user.FeiShuUserId,
		}
	}
	return onDutyUsers
}

func (s *onDutyService) parseAndValidateTimeRange(startStr, endStr string) (time.Time, time.Time, error) {
	start, err := time.Parse(DateFormat, startStr)
	if err != nil {
		return time.Time{}, time.Time{}, ErrInvalidTimeRange
	}

	end, err := time.Parse(DateFormat, endStr)
	if err != nil {
		return time.Time{}, time.Time{}, ErrInvalidTimeRange
	}

	if end.Before(start) {
		return time.Time{}, time.Time{}, ErrInvalidTimeRange
	}

	if end.Sub(start) > MaxTimeRangeDays*24*time.Hour {
		return time.Time{}, time.Time{}, ErrTimeRangeTooLong
	}

	return start, end, nil
}

func (s *onDutyService) generateDutyPlan(ctx context.Context, group *model.MonitorOnDutyGroup, start, end time.Time, changes []*model.MonitorOnDutyChange) []*model.MonitorOnDutyOne {
	changeMap := s.buildChangeMap(changes)
	historyRecords, err := s.dao.GetMonitorOnDutyHistoryByGroupIDAndTimeRange(ctx, group.ID, start.Format(DateFormat), end.Format(DateFormat))
	if err != nil {
		s.logger.Error("获取值班历史记录失败", zap.Error(err), zap.Int("groupID", group.ID))
	}

	historyMap := s.buildHistoryMap(historyRecords)
	var result []*model.MonitorOnDutyOne
	today := time.Now().Format(DateFormat)

	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format(DateFormat)
		dutyOne := &model.MonitorOnDutyOne{Date: dateStr}

		if change, exists := changeMap[dateStr]; exists {
			dutyOne.User = s.findUserByID(group.Users, change.OnDutyUserID)
			if change.OriginUserID > 0 {
				if originUser := s.findUserByID(group.Users, change.OriginUserID); originUser != nil {
					dutyOne.OriginUser = originUser.RealName
				}
			}
		} else if history, exists := historyMap[dateStr]; exists {
			dutyUser := s.findUserByID(group.Users, history.OnDutyUserID)
			if dutyUser == nil {
				dutyUser, err = s.userDao.GetUserByID(ctx, history.OnDutyUserID)
				if err != nil {
					s.logger.Error("获取值班用户失败", zap.Error(err), zap.Int("userID", history.OnDutyUserID))
				}
			}
			dutyOne.User = dutyUser

			if history.OriginUserID > 0 {
				originUser := s.findUserByID(group.Users, history.OriginUserID)
				if originUser != nil {
					dutyOne.OriginUser = originUser.RealName
				} else {
					user, err := s.userDao.GetUserByID(ctx, history.OriginUserID)
					if err == nil && user != nil {
						dutyOne.OriginUser = user.RealName
					} else {
						dutyOne.OriginUser = "未知用户"
					}
				}
			}
		} else {
			dutyOne.User = s.calculateDutyUser(ctx, group, dateStr, today)
		}

		result = append(result, dutyOne)
	}

	return result
}

func (s *onDutyService) buildChangeMap(changes []*model.MonitorOnDutyChange) map[string]*model.MonitorOnDutyChange {
	changeMap := make(map[string]*model.MonitorOnDutyChange)
	for _, change := range changes {
		changeMap[change.Date] = change
	}
	return changeMap
}

func (s *onDutyService) buildHistoryMap(histories []*model.MonitorOnDutyHistory) map[string]*model.MonitorOnDutyHistory {
	historyMap := make(map[string]*model.MonitorOnDutyHistory)
	for _, history := range histories {
		historyMap[history.DateString] = history
	}
	return historyMap
}

func (s *onDutyService) calculateDutyUser(ctx context.Context, group *model.MonitorOnDutyGroup, dateStr, todayStr string) *model.User {
	if group.ShiftDays <= 0 || len(group.Users) == 0 {
		return nil
	}

	targetDate, _ := time.Parse(DateFormat, dateStr)
	today, _ := time.Parse(DateFormat, todayStr)
	daysDiff := int(targetDate.Sub(today).Hours()) / 24

	currentIndex := s.getCurrentUserIndex(ctx, group, todayStr)
	totalDays := len(group.Users) * group.ShiftDays
	shiftIndex := (currentIndex*group.ShiftDays + daysDiff) % totalDays
	if shiftIndex < 0 {
		shiftIndex += totalDays
	}

	userIndex := shiftIndex / group.ShiftDays
	if userIndex >= 0 && userIndex < len(group.Users) {
		return group.Users[userIndex]
	}

	return nil
}

func (s *onDutyService) getCurrentUserIndex(ctx context.Context, group *model.MonitorOnDutyGroup, todayStr string) int {
	if history, err := s.dao.GetMonitorOnDutyHistoryByGroupIDAndDay(ctx, group.ID, todayStr); err == nil && history != nil {
		for i, user := range group.Users {
			if user.ID == history.OnDutyUserID {
				return i
			}
		}
	}
	return 0
}

func (s *onDutyService) getTodayDutyUser(ctx context.Context, group *model.MonitorOnDutyGroup) *model.User {
	today := time.Now().Format(DateFormat)
	
	// 首先尝试从历史记录中获取今日值班人
	if history, err := s.dao.GetMonitorOnDutyHistoryByGroupIDAndDay(ctx, group.ID, today); err == nil && history != nil {
		// 从值班组成员中查找
		for _, user := range group.Users {
			if user.ID == history.OnDutyUserID {
				return user
			}
		}
		// 如果在当前成员中找不到，从数据库查询
		if user, err := s.userDao.GetUserByID(ctx, history.OnDutyUserID); err == nil && user != nil {
			return user
		}
	}
	
	// 检查今日是否有换班记录
	if changes, err := s.dao.GetMonitorOnDutyChangesByGroupAndTimeRange(ctx, group.ID, today, today); err == nil && len(changes) > 0 {
		// 取最新的换班记录
		latestChange := changes[len(changes)-1]
		for _, user := range group.Users {
			if user.ID == latestChange.OnDutyUserID {
				return user
			}
		}
		// 如果在当前成员中找不到，从数据库查询
		if user, err := s.userDao.GetUserByID(ctx, latestChange.OnDutyUserID); err == nil && user != nil {
			return user
		}
	}
	
	// 如果没有历史记录和换班记录，根据轮班规则计算
	return s.calculateDutyUser(ctx, group, today, today)
}

func (s *onDutyService) findUserByID(users []*model.User, id int) *model.User {
	for _, user := range users {
		if user.ID == id {
			return user
		}
	}

	user, err := s.userDao.GetUserByID(context.Background(), id)
	if err == nil && user != nil {
		return user
	}
	return nil
}
