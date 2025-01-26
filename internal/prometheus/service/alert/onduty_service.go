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
	ErrGroupIDEmpty      = errors.New("值班组 ID 为空")
	ErrGroupNotFound     = errors.New("值班组不存在")
	ErrGroupHasSendGroup = errors.New("值班组存在关联发送组，无法删除")
)

type AlertManagerOnDutyService interface {
	GetMonitorOnDutyGroupList(ctx context.Context, listReq *model.ListReq) ([]*model.MonitorOnDutyGroup, error)
	CreateMonitorOnDutyGroup(ctx context.Context, monitorOnDutyGroup *model.MonitorOnDutyGroup) error
	CreateMonitorOnDutyGroupChange(ctx context.Context, monitorOnDutyChange *model.MonitorOnDutyChange) error
	UpdateMonitorOnDutyGroup(ctx context.Context, monitorOnDutyGroup *model.MonitorOnDutyGroup) error
	DeleteMonitorOnDutyGroup(ctx context.Context, id int) error
	GetMonitorOnDutyGroup(ctx context.Context, id int) (*model.MonitorOnDutyGroup, error)
	GetMonitorOnDutyGroupFuturePlan(ctx context.Context, id int, startTimeStr string, endTimeStr string) (model.OnDutyPlanResp, error)
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

// GetMonitorOnDutyGroupList 获取值班组列表
func (a *alertManagerOnDutyService) GetMonitorOnDutyGroupList(ctx context.Context, listReq *model.ListReq) ([]*model.MonitorOnDutyGroup, error) {
	var groups []*model.MonitorOnDutyGroup

	if listReq.Search != "" {
		groups, err := a.dao.SearchMonitorOnDutyGroupByName(ctx, listReq.Search)
		if err != nil {
			a.l.Error("搜索值班组失败", zap.String("search", listReq.Search), zap.Error(err))
			return nil, err
		}

		groups = groups
	}

	offset := (listReq.Page - 1) * listReq.Size
	limit := listReq.Size

	groups, err := a.dao.GetMonitorOnDutyList(ctx, offset, limit)
	if err != nil {
		a.l.Error("获取值班组列表失败", zap.Error(err))
		return nil, err
	}

	for _, group := range groups {
		userNames := make([]string, len(group.Members))
		for i, member := range group.Members {
			userNames[i] = member.Username
		}
		group.UserNames = userNames
	}

	return groups, nil
}

// CreateMonitorOnDutyGroup 创建值班组
func (a *alertManagerOnDutyService) CreateMonitorOnDutyGroup(ctx context.Context, group *model.MonitorOnDutyGroup) error {
	exists, err := a.dao.CheckMonitorOnDutyGroupExists(ctx, group)
	if err != nil {
		a.l.Error("检查值班组是否存在失败", zap.Error(err))
		return err
	}

	if exists {
		return ErrGroupExists
	}

	users, err := a.userDao.GetUserByUsernames(ctx, group.UserNames)
	if err != nil {
		a.l.Error("获取值班组成员信息失败", zap.Error(err))
		return err
	}

	group.Members = users
	if err := a.dao.CreateMonitorOnDutyGroup(ctx, group); err != nil {
		a.l.Error("创建值班组失败", zap.Error(err))
		return err
	}

	if err := a.cache.MonitorCacheManager(ctx); err != nil {
		a.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	return nil
}

// CreateMonitorOnDutyGroupChange 创建值班组变更
func (a *alertManagerOnDutyService) CreateMonitorOnDutyGroupChange(ctx context.Context, change *model.MonitorOnDutyChange) error {
	if change.OnDutyGroupID == 0 {
		return ErrGroupIDEmpty
	}

	if _, err := a.dao.GetMonitorOnDutyGroupById(ctx, change.OnDutyGroupID); err != nil {
		a.l.Error("获取值班组失败", zap.Error(err))
		return err
	}

	if err := a.dao.CreateMonitorOnDutyGroupChange(ctx, change); err != nil {
		a.l.Error("创建值班组变更失败", zap.Error(err))
		return err
	}

	return nil
}

// UpdateMonitorOnDutyGroup 更新值班组
func (a *alertManagerOnDutyService) UpdateMonitorOnDutyGroup(ctx context.Context, group *model.MonitorOnDutyGroup) error {
	// 检查值班组是否存在
	exists, err := a.dao.CheckMonitorOnDutyGroupExists(ctx, group)
	if err != nil {
		a.l.Error("检查值班组是否存在失败", zap.Error(err))
		return err
	}

	if !exists {
		return errors.New("值班组不存在")
	}

	users, err := a.userDao.GetUserByUsernames(ctx, group.UserNames)
	if err != nil {
		a.l.Error("获取用户信息失败", zap.Error(err))
		return err
	}

	group.Members = users
	if err := a.dao.UpdateMonitorOnDutyGroup(ctx, group); err != nil {
		a.l.Error("更新值班组失败", zap.Error(err))
		return err
	}

	return nil
}

// DeleteMonitorOnDutyGroup 删除值班组
func (a *alertManagerOnDutyService) DeleteMonitorOnDutyGroup(ctx context.Context, id int) error {
	sendGroups, err := a.sendDao.GetMonitorSendGroupByOnDutyGroupId(ctx, id)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		a.l.Error("获取关联发送组失败", zap.Error(err))
		return err
	}

	if len(sendGroups) > 0 {
		return ErrGroupHasSendGroup
	}

	if err := a.dao.DeleteMonitorOnDutyGroup(ctx, id); err != nil {
		a.l.Error("删除值班组失败", zap.Error(err))
		return err
	}

	return nil
}

// GetMonitorOnDutyGroup 获取值班组
func (a *alertManagerOnDutyService) GetMonitorOnDutyGroup(ctx context.Context, id int) (*model.MonitorOnDutyGroup, error) {
	group, err := a.dao.GetMonitorOnDutyGroupById(ctx, id)
	if err != nil {
		a.l.Error("获取值班组失败", zap.Int("id", id), zap.Error(err))
		return nil, err
	}

	if group == nil {
		return nil, ErrGroupNotFound
	}

	return group, nil
}

// GetMonitorOnDutyGroupFuturePlan 获取值班组未来排班计划
func (a *alertManagerOnDutyService) GetMonitorOnDutyGroupFuturePlan(ctx context.Context, id int, startTimeStr string, endTimeStr string) (model.OnDutyPlanResp, error) {
	// 解析开始时间字符串
	startTime, err := time.Parse("2006-01-02", startTimeStr)
	if err != nil {
		a.l.Error("开始时间格式错误", zap.String("startTime", startTimeStr), zap.Error(err))
		return model.OnDutyPlanResp{}, errors.New("开始时间格式错误")
	}

	// 解析结束时间字符串
	endTime, err := time.Parse("2006-01-02", endTimeStr)
	if err != nil {
		a.l.Error("结束时间格式错误", zap.String("endTime", endTimeStr), zap.Error(err))
		return model.OnDutyPlanResp{}, errors.New("结束时间格式错误")
	}

	// 验证时间范围的合法性
	if endTime.Before(startTime) {
		errMsg := "结束时间不能早于开始时间"
		a.l.Error(errMsg, zap.String("startTime", startTimeStr), zap.String("endTime", endTimeStr))
		return model.OnDutyPlanResp{}, errors.New(errMsg)
	}

	// 根据ID获取值班组信息
	group, err := a.dao.GetMonitorOnDutyGroupById(ctx, id)
	if err != nil {
		a.l.Error("获取值班组失败", zap.Int("id", id), zap.Error(err))
		return model.OnDutyPlanResp{}, err
	}

	// 初始化返回结果结构体
	planResp := model.OnDutyPlanResp{
		Details:       make([]model.OnDutyOne, 0), // 值班详情列表,预分配内存以提高性能
		Map:           make(map[string]string),    // 日期到值班人真实姓名的映射
		UserNameMap:   make(map[string]string),    // 日期到值班人用户名的映射
		OriginUserMap: make(map[string]string),    // 日期到原始值班人的映射
	}

	// 获取指定时间范围内的值班计划变更记录
	changes, err := a.dao.GetMonitorOnDutyChangesByGroupAndTimeRange(ctx, id, startTimeStr, endTimeStr)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		a.l.Error("获取值班计划变更失败", zap.Int("id", id), zap.Error(err))
		return model.OnDutyPlanResp{}, err
	}

	// 构建日期到变更记录的映射,用于快速查找
	changeMap := make(map[string]*model.MonitorOnDutyChange, len(changes))
	for _, change := range changes {
		changeMap[change.Date] = change
	}

	// 获取当前日期,用于值班人计算
	today := time.Now().Format("2006-01-02")

	// 计算需要处理的总天数
	totalDays := int(endTime.Sub(startTime).Hours()/24) + 1

	// 遍历每一天,生成值班计划
	for i := 0; i < totalDays; i++ {
		currentDate := startTime.AddDate(0, 0, i).Format("2006-01-02")

		var user *model.User
		var originUserName string

		// 处理值班变更的情况
		if history, exists := changeMap[currentDate]; exists {
			// 获取变更后的值班人信息
			user, err = a.userDao.GetUserByID(ctx, history.OnDutyUserID)
			if err != nil {
				a.l.Warn("获取变更后用户信息失败", zap.Int("userID", history.OnDutyUserID), zap.Error(err))
				continue
			}

			// 如果存在原始值班人,获取原始值班人信息
			if history.OriginUserID > 0 {
				originUser, err := a.userDao.GetUserByID(ctx, history.OriginUserID)
				if err == nil {
					originUserName = originUser.RealName
				} else {
					a.l.Warn("获取原始值班人信息失败", zap.Int("originUserID", history.OriginUserID), zap.Error(err))
				}
			}
		} else {
			// 没有变更记录时,根据排班规则计算值班人
			user = CalculateOnDutyUser(ctx, a.l, a.dao, group, currentDate, today)
			if user == nil {
				a.l.Warn("计算值班人失败", zap.String("date", currentDate))
				continue
			}
		}

		// 构建单日值班信息
		onDutyOne := model.OnDutyOne{
			Date:       currentDate,    // 日期
			User:       user,           // 值班人
			OriginUser: originUserName, // 原始值班人
		}

		// 将信息添加到返回结果中
		planResp.Details = append(planResp.Details, onDutyOne) // 添加到值班详情列表
		planResp.Map[currentDate] = user.RealName              // 添加到日期到值班人真实姓名的映射
		planResp.UserNameMap[currentDate] = user.Username      // 添加到日期到值班人用户名的映射
		planResp.OriginUserMap[currentDate] = originUserName   // 添加到日期到原始值班人的映射
	}

	return planResp, nil
}

// CalculateOnDutyUser 根据排班规则计算指定日期的值班人
func CalculateOnDutyUser(ctx context.Context, l *zap.Logger, dao alert.AlertManagerOnDutyDAO, group *model.MonitorOnDutyGroup, dateStr string, todayStr string) *model.User {
	// 将目标日期字符串解析为时间对象
	// "2006-01-02" 是 Go 的时间格式模板
	targetDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		l.Error("日期解析失败", zap.String("date", dateStr), zap.Error(err))
		return nil
	}

	// 将今天的日期字符串解析为时间对象
	today, err := time.Parse("2006-01-02", todayStr)
	if err != nil {
		l.Error("解析今天日期失败", zap.String("today", todayStr), zap.Error(err))
		return nil
	}

	// 计算目标日期与今天相差的天数
	// 先计算小时数再除以24得到天数
	daysDiff := int(targetDate.Sub(today).Hours()) / 24

	// 获取今天值班人在成员列表中的索引位置
	currentUserIndex := GetCurrentUserIndex(ctx, l, dao, group, todayStr)
	if currentUserIndex == -1 {
		l.Warn("未找到当前值班人", zap.String("today", todayStr))
		return nil
	}

	// 获取值班组的基本信息
	totalMembers := len(group.Members) // 总成员数
	shiftDays := group.ShiftDays       // 每人值班天数

	// 参数有效性检查
	if shiftDays == 0 || totalMembers == 0 {
		l.Error("轮班天数或成员数量无效", zap.Int("shiftDays", shiftDays), zap.Int("totalMembers", totalMembers))
		return nil
	}

	// 计算一个完整轮班周期的总天数
	// 例如：3个人每人值班2天，则总周期为6天
	totalShiftLength := totalMembers * shiftDays

	// 定义一个安全的取模函数，确保结果为非负数
	mod := func(a, b int) int {
		if b == 0 {
			l.Error("除零错误", zap.Int("b", b))
			return -1
		}
		return (a%b + b) % b
	}

	// 计算目标日期在轮班周期中的位置
	// currentUserIndex*shiftDays：当前值班人的起始位置
	// +daysDiff：加上相差的天数
	// 对 totalShiftLength 取模：确保结果在一个轮班周期内
	indexInShift := mod(currentUserIndex*shiftDays+daysDiff, totalShiftLength)
	if indexInShift == -1 {
		l.Error("取模运算出错，可能是因为轮班天数或成员数量无效",
			zap.Int("currentUserIndex", currentUserIndex),
			zap.Int("shiftDays", shiftDays),
			zap.Int("totalMembers", totalMembers))
		return nil
	}

	// 根据位置计算值班人索引
	// 例如：如果 indexInShift=7，shiftDays=3，则 userIndex=2（第3个人）
	userIndex := indexInShift / shiftDays

	// 检查计算出的索引是否有效
	if userIndex >= 0 && userIndex < totalMembers {
		return group.Members[userIndex] // 返回对应的值班人
	}

	l.Error("计算出的值班人索引无效", zap.Int("userIndex", userIndex), zap.Int("totalMembers", totalMembers))
	return nil
}

func GetCurrentUserIndex(ctx context.Context, l *zap.Logger, dao alert.AlertManagerOnDutyDAO, group *model.MonitorOnDutyGroup, todayStr string) int {
	// 尝试从历史记录中获取今天的值班人
	todayHistory, err := dao.GetMonitorOnDutyHistoryByGroupIdAndDay(ctx, group.ID, todayStr)
	if err == nil && todayHistory != nil && todayHistory.OnDutyUserID > 0 {
		for index, member := range group.Members {
			if member.ID == todayHistory.OnDutyUserID {
				return index
			}
		}
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		// 如果查询发生了其他错误，记录日志
		l.Error("获取今天的值班历史记录失败", zap.Error(err))
	}

	// 如果没有历史记录，默认第一个成员为当前值班人
	return 0
}
