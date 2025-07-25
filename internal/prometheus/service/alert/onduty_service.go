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

type alertManagerOnDutyService struct {
	dao     alert.AlertManagerOnDutyDAO
	sendDao alert.AlertManagerSendDAO
	cache   cache.MonitorCache
	userDao userDao.UserDAO
	l       *zap.Logger
}

func NewAlertManagerOnDutyService(dao alert.AlertManagerOnDutyDAO, sendDao alert.AlertManagerSendDAO, cache cache.MonitorCache, l *zap.Logger, userDao userDao.UserDAO) AlertManagerOnDutyService {
	return &alertManagerOnDutyService{
		dao:     dao,
		userDao: userDao,
		sendDao: sendDao,
		l:       l,
		cache:   cache,
	}
}

func (s *alertManagerOnDutyService) GetMonitorOnDutyGroupList(ctx context.Context, req *model.GetMonitorOnDutyGroupListReq) (model.ListResp[*model.MonitorOnDutyGroup], error) {
	// 从数据库获取值班组列表
	groups, total, err := s.dao.GetMonitorOnDutyList(ctx, req)
	if err != nil {
		s.l.Error("获取值班组列表失败", zap.Error(err))
		return model.ListResp[*model.MonitorOnDutyGroup]{}, err
	}

	// 返回值班组列表和总数
	return model.ListResp[*model.MonitorOnDutyGroup]{
		Items: groups,
		Total: total,
	}, nil
}

func (s *alertManagerOnDutyService) CreateMonitorOnDutyGroup(ctx context.Context, req *model.CreateMonitorOnDutyGroupReq) error {
	// 检查值班组名称是否已存在
	if exists, err := s.dao.CheckMonitorOnDutyGroupExists(ctx, &model.MonitorOnDutyGroup{Name: req.Name}); err != nil {
		s.l.Error("检查值班组是否存在失败", zap.Error(err))
		return err
	} else if exists {
		return ErrGroupExists
	}

	// 验证轮班天数是否有效
	if req.ShiftDays <= 0 {
		return ErrInvalidShiftDays
	}

	// 获取并验证所有成员是否存在
	users, err := s.userDao.GetUserByIDs(ctx, req.UserIDs)
	if err != nil {
		s.l.Error("获取成员信息失败", zap.Error(err))
		return err
	}
	if len(users) != len(req.UserIDs) {
		return ErrMembersNotFound
	}

	// 创建值班组对象
	group := &model.MonitorOnDutyGroup{
		Name:           req.Name,
		UserID:         req.UserID,
		ShiftDays:      req.ShiftDays,
		CreateUserName: req.CreateUserName,
		Description:    req.Description,
		Users:          s.convertUsersToOnDutyUsers(users),
	}

	// 保存值班组到数据库
	if err := s.dao.CreateMonitorOnDutyGroup(ctx, group); err != nil {
		s.l.Error("创建值班组失败", zap.Error(err))
		return err
	}

	return nil
}

func (s *alertManagerOnDutyService) CreateMonitorOnDutyGroupChange(ctx context.Context, req *model.CreateMonitorOnDutyGroupChangeReq) error {
	// 验证值班组是否存在
	if _, err := s.dao.GetMonitorOnDutyGroupByID(ctx, req.OnDutyGroupID); err != nil {
		s.l.Error("值班组不存在", zap.Int("groupID", req.OnDutyGroupID), zap.Error(err))
		return ErrGroupNotFound
	}

	// 创建值班变更记录
	change := &model.MonitorOnDutyChange{
		OnDutyGroupID:  req.OnDutyGroupID,
		UserID:         req.UserID,
		Date:           req.Date,
		OriginUserID:   req.OriginUserID,
		OnDutyUserID:   req.OnDutyUserID,
		CreateUserName: req.CreateUserName,
		Reason:         req.Reason,
	}

	// 保存值班变更记录到数据库
	if err := s.dao.CreateMonitorOnDutyGroupChange(ctx, change); err != nil {
		s.l.Error("创建值班变更失败", zap.Error(err))
		return err
	}

	return nil
}

func (s *alertManagerOnDutyService) UpdateMonitorOnDutyGroup(ctx context.Context, req *model.UpdateMonitorOnDutyGroupReq) error {
	// 获取要更新的值班组
	group, err := s.dao.GetMonitorOnDutyGroupByID(ctx, req.ID)
	if err != nil {
		s.l.Error("获取值班组失败", zap.Int("id", req.ID), zap.Error(err))
		return ErrGroupNotFound
	}

	// 如果名称变更，检查新名称是否已存在
	if group.Name != req.Name {
		if exists, err := s.dao.CheckMonitorOnDutyGroupExists(ctx, &model.MonitorOnDutyGroup{Name: req.Name}); err != nil {
			s.l.Error("检查值班组名称是否存在失败", zap.Error(err))
			return err
		} else if exists {
			return ErrGroupExists
		}
	}

	// 验证轮班天数是否有效
	if req.ShiftDays <= 0 {
		return ErrInvalidShiftDays
	}

	// 获取并验证所有成员是否存在
	users, err := s.userDao.GetUserByIDs(ctx, req.UserIDs)
	if err != nil {
		s.l.Error("获取成员信息失败", zap.Error(err))
		return err
	}
	if len(users) != len(req.UserIDs) {
		return ErrMembersNotFound
	}

	// 更新值班组信息
	group.Name = req.Name
	group.ShiftDays = req.ShiftDays
	group.Description = req.Description
	group.Users = s.convertUsersToOnDutyUsers(users)
	if req.Enable != nil {
		group.Enable = *req.Enable
	}

	// 保存更新后的值班组到数据库
	if err := s.dao.UpdateMonitorOnDutyGroup(ctx, group); err != nil {
		s.l.Error("更新值班组失败", zap.Error(err))
		return err
	}

	return nil
}

func (s *alertManagerOnDutyService) DeleteMonitorOnDutyGroup(ctx context.Context, req *model.DeleteMonitorOnDutyGroupReq) error {
	// 检查值班组是否关联了发送组，如果有关联则不允许删除
	_, count, err := s.sendDao.GetMonitorSendGroupByOnDutyGroupID(ctx, req.ID)
	if err != nil {
		s.l.Error("检查关联发送组失败", zap.Error(err))
		return err
	}
	if count > 0 {
		return ErrGroupHasSendGroup
	}

	// 删除值班组
	if err := s.dao.DeleteMonitorOnDutyGroup(ctx, req.ID); err != nil {
		s.l.Error("删除值班组失败", zap.Int("id", req.ID), zap.Error(err))
		return err
	}

	return nil
}

func (s *alertManagerOnDutyService) GetMonitorOnDutyGroup(ctx context.Context, req *model.GetMonitorOnDutyGroupReq) (*model.MonitorOnDutyGroup, error) {
	// 获取指定ID的值班组
	group, err := s.dao.GetMonitorOnDutyGroupByID(ctx, req.ID)
	if err != nil {
		s.l.Error("获取值班组失败", zap.Int("id", req.ID), zap.Error(err))
		return nil, ErrGroupNotFound
	}

	// 获取今日值班人员信息
	group.TodayDutyUser = s.getTodayDutyUser(ctx, group)
	return group, nil
}

func (s *alertManagerOnDutyService) GetMonitorOnDutyGroupFuturePlan(ctx context.Context, req *model.GetMonitorOnDutyGroupFuturePlanReq) ([]*model.MonitorOnDutyOne, error) {
	// 解析并验证时间范围
	startTime, endTime, err := s.parseTimeRange(req.StartTime, req.EndTime)
	if err != nil {
		return nil, err
	}

	// 检查时间范围是否超过一年
	if endTime.Sub(startTime) > 365*24*time.Hour {
		return nil, ErrTimeRangeTooLong
	}

	// 获取值班组信息
	group, err := s.dao.GetMonitorOnDutyGroupByID(ctx, req.ID)
	if err != nil {
		s.l.Error("获取值班组失败", zap.Int("id", req.ID), zap.Error(err))
		return nil, ErrGroupNotFound
	}

	// 获取指定时间范围内的值班变更记录
	changes, err := s.dao.GetMonitorOnDutyChangesByGroupAndTimeRange(ctx, req.ID, req.StartTime, req.EndTime)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		s.l.Error("获取值班变更失败", zap.Error(err))
		return nil, err
	}

	// 生成值班计划
	return s.generateDutyPlan(ctx, group, startTime, endTime, changes), nil
}

func (s *alertManagerOnDutyService) GetMonitorOnDutyHistory(ctx context.Context, req *model.GetMonitorOnDutyHistoryReq) (model.ListResp[*model.MonitorOnDutyHistory], error) {
	// 获取值班历史记录
	histories, total, err := s.dao.GetMonitorOnDutyHistoryList(ctx, req)
	if err != nil {
		s.l.Error("获取值班历史失败", zap.Error(err))
		return model.ListResp[*model.MonitorOnDutyHistory]{}, err
	}

	// 返回值班历史记录和总数
	return model.ListResp[*model.MonitorOnDutyHistory]{
		Items: histories,
		Total: total,
	}, nil
}

// 私有辅助方法

// convertUsersToOnDutyUsers 将用户对象转换为值班用户对象
func (s *alertManagerOnDutyService) convertUsersToOnDutyUsers(users []*model.User) []*model.User {
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

// parseTimeRange 解析并验证时间范围字符串
func (s *alertManagerOnDutyService) parseTimeRange(startStr, endStr string) (time.Time, time.Time, error) {
	start, err := time.Parse("2006-01-02", startStr)
	if err != nil {
		return time.Time{}, time.Time{}, ErrInvalidTimeRange
	}

	end, err := time.Parse("2006-01-02", endStr)
	if err != nil {
		return time.Time{}, time.Time{}, ErrInvalidTimeRange
	}

	// 确保结束时间不早于开始时间
	if end.Before(start) {
		return time.Time{}, time.Time{}, ErrInvalidTimeRange
	}

	return start, end, nil
}

// generateDutyPlan 生成指定时间范围内的值班计划
func (s *alertManagerOnDutyService) generateDutyPlan(ctx context.Context, group *model.MonitorOnDutyGroup, start, end time.Time, changes []*model.MonitorOnDutyChange) []*model.MonitorOnDutyOne {
	// 将值班变更记录转换为以日期为键的映射，便于快速查找
	changeMap := make(map[string]*model.MonitorOnDutyChange)
	for _, change := range changes {
		changeMap[change.Date] = change
	}

	// 获取指定时间范围内的值班历史记录
	historyRecords, err := s.dao.GetMonitorOnDutyHistoryByGroupIDAndTimeRange(ctx, group.ID, start.Format("2006-01-02"), end.Format("2006-01-02"))
	if err != nil {
		s.l.Error("获取值班历史记录失败", zap.Error(err), zap.Int("groupID", group.ID))
	}

	// 将历史记录转换为以日期为键的映射
	historyMap := make(map[string]*model.MonitorOnDutyHistory)
	for _, history := range historyRecords {
		historyMap[history.DateString] = history
	}

	var result []*model.MonitorOnDutyOne
	today := time.Now().Format("2006-01-02")

	// 遍历时间范围内的每一天，生成值班计划
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("2006-01-02")
		dutyOne := &model.MonitorOnDutyOne{Date: dateStr}

		// 如果存在值班变更记录，使用变更后的值班人员
		if change, exists := changeMap[dateStr]; exists {
			dutyOne.User = s.findUserByID(group.Users, change.OnDutyUserID)
			if change.OriginUserID > 0 {
				if originUser := s.findUserByID(group.Users, change.OriginUserID); originUser != nil {
					dutyOne.OriginUser = originUser.RealName
				}
			}
		} else if history, exists := historyMap[dateStr]; exists {
			// 如果存在历史记录，使用历史记录中的值班人员
			dutyUser := s.findUserByID(group.Users, history.OnDutyUserID)
			if dutyUser == nil {
				// 如果在当前组成员中找不到，直接从数据库查询
				dutyUser, err = s.userDao.GetUserByID(ctx, history.OnDutyUserID)
				if err != nil {
					s.l.Error("获取值班用户失败", zap.Error(err), zap.Int("userID", history.OnDutyUserID))
				}
			}
			dutyOne.User = dutyUser

			if history.OriginUserID > 0 {
				originUser := s.findUserByID(group.Users, history.OriginUserID)
				if originUser != nil {
					dutyOne.OriginUser = originUser.RealName
				} else {
					// 如果找不到原始用户，尝试从所有用户中查找
					user, err := s.userDao.GetUserByID(ctx, history.OriginUserID)
					if err == nil && user != nil {
						dutyOne.OriginUser = user.RealName
					} else {
						dutyOne.OriginUser = "未知用户"
					}
				}
			}
		} else {
			// 否则根据轮班规则计算值班人员
			dutyOne.User = s.calculateDutyUser(ctx, group, dateStr, today)
		}

		result = append(result, dutyOne)
	}

	return result
}

// calculateDutyUser 根据轮班规则计算指定日期的值班人员
func (s *alertManagerOnDutyService) calculateDutyUser(ctx context.Context, group *model.MonitorOnDutyGroup, dateStr, todayStr string) *model.User {
	// 验证参数有效性
	if group.ShiftDays <= 0 || len(group.Users) == 0 {
		return nil
	}

	// 计算目标日期与今天的天数差
	targetDate, _ := time.Parse("2006-01-02", dateStr)
	today, _ := time.Parse("2006-01-02", todayStr)
	daysDiff := int(targetDate.Sub(today).Hours()) / 24

	// 获取今天值班人员在组内的索引
	currentIndex := s.getCurrentUserIndex(ctx, group, todayStr)

	// 计算轮班周期总天数
	totalDays := len(group.Users) * group.ShiftDays

	// 计算目标日期的轮班索引
	shiftIndex := (currentIndex*group.ShiftDays + daysDiff) % totalDays
	if shiftIndex < 0 {
		shiftIndex += totalDays
	}

	// 根据轮班索引确定值班人员
	userIndex := shiftIndex / group.ShiftDays
	if userIndex >= 0 && userIndex < len(group.Users) {
		return group.Users[userIndex]
	}

	return nil
}

// getCurrentUserIndex 获取今天值班人员在组内的索引
func (s *alertManagerOnDutyService) getCurrentUserIndex(ctx context.Context, group *model.MonitorOnDutyGroup, todayStr string) int {
	// 尝试从历史记录中获取今天的值班人员
	if history, err := s.dao.GetMonitorOnDutyHistoryByGroupIDAndDay(ctx, group.ID, todayStr); err == nil && history != nil {
		for i, user := range group.Users {
			if user.ID == history.OnDutyUserID {
				return i
			}
		}
	}
	// 如果没有历史记录，默认从第一个成员开始
	return 0
}

// getTodayDutyUser 获取今天的值班人员
func (s *alertManagerOnDutyService) getTodayDutyUser(ctx context.Context, group *model.MonitorOnDutyGroup) *model.User {
	today := time.Now().Format("2006-01-02")
	return s.calculateDutyUser(ctx, group, today, today)
}

// findUserByID 根据ID在用户列表中查找用户
func (s *alertManagerOnDutyService) findUserByID(users []*model.User, id int) *model.User {
	for _, user := range users {
		if user.ID == id {
			return user
		}
	}

	// 如果在当前用户列表中找不到，从数据库查询
	user, err := s.userDao.GetUserByID(context.Background(), id)
	if err == nil && user != nil {
		return user
	}
	return nil
}
