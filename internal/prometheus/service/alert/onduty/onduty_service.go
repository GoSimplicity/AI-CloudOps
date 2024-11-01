package onduty

import (
	"context"
	"errors"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/cache"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert/onduty"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert/send"
	userDao "github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils/prometheus"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"time"
)

type AlertManagerOnDutyService interface {
	GetMonitorOnDutyGroupList(ctx context.Context, searchName *string) ([]*model.MonitorOnDutyGroup, error)
	CreateMonitorOnDutyGroup(ctx context.Context, monitorOnDutyGroup *model.MonitorOnDutyGroup) error
	CreateMonitorOnDutyGroupChange(ctx context.Context, monitorOnDutyChange *model.MonitorOnDutyChange) error
	UpdateMonitorOnDutyGroup(ctx context.Context, monitorOnDutyGroup *model.MonitorOnDutyGroup) error
	DeleteMonitorOnDutyGroup(ctx context.Context, id int) error
	GetMonitorOnDutyGroup(ctx context.Context, id int) (*model.MonitorOnDutyGroup, error)
	GetMonitorOnDutyGroupFuturePlan(ctx context.Context, id int, startTimeStr string, endTimeStr string) (model.OnDutyPlanResp, error)
}

type alertManagerOnDutyService struct {
	dao     onduty.AlertManagerOnDutyDAO
	sendDao send.AlertManagerSendDAO
	cache   cache.MonitorCache
	userDao userDao.UserDAO
	l       *zap.Logger
}

func NewAlertManagerOnDutyService(dao onduty.AlertManagerOnDutyDAO, sendDao send.AlertManagerSendDAO, cache cache.MonitorCache, l *zap.Logger, userDao userDao.UserDAO) AlertManagerOnDutyService {
	return &alertManagerOnDutyService{
		dao:     dao,
		userDao: userDao,
		sendDao: sendDao,
		l:       l,
		cache:   cache,
	}
}

func (a *alertManagerOnDutyService) GetMonitorOnDutyGroupList(ctx context.Context, searchName *string) ([]*model.MonitorOnDutyGroup, error) {
	// 调用 HandleList 获取值班组列表
	list, err := pkg.HandleList(ctx, searchName,
		a.dao.SearchMonitorOnDutyGroupByName, // 搜索函数
		a.dao.GetAllMonitorOnDutyGroup)
	if err != nil {
		a.l.Error("获取值班组列表失败", zap.Error(err))
		return nil, err
	}

	// 遍历每个值班组，构建 UserNames 列表
	for _, group := range list {
		// 预分配切片容量
		userNames := make([]string, 0, len(group.Members))

		// 直接赋值
		for _, member := range group.Members {
			userNames = append(userNames, member.Username)
		}

		group.UserNames = userNames
	}

	return list, nil
}

func (a *alertManagerOnDutyService) CreateMonitorOnDutyGroup(ctx context.Context, monitorOnDutyGroup *model.MonitorOnDutyGroup) error {
	// 检查值班组是否已存在
	exists, err := a.dao.CheckMonitorOnDutyGroupExists(ctx, monitorOnDutyGroup)
	if err != nil {
		a.l.Error("创建值班组失败：检查值班组是否存在时出错", zap.Error(err))
		return err
	}
	if exists {
		return errors.New("值班组已存在")
	}

	users, err := a.userDao.GetUserByUsernames(ctx, monitorOnDutyGroup.UserNames)
	if err != nil {
		a.l.Error("创建值班组失败：获取值班组成员信息时出错", zap.Error(err))
		return err
	}

	monitorOnDutyGroup.Members = users
	// 创建值班组
	if err := a.dao.CreateMonitorOnDutyGroup(ctx, monitorOnDutyGroup); err != nil {
		a.l.Error("创建值班组失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := a.cache.MonitorCacheManager(ctx); err != nil {
		a.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	a.l.Info("创建值班组成功", zap.Int("id", monitorOnDutyGroup.ID))

	return nil
}

func (a *alertManagerOnDutyService) CreateMonitorOnDutyGroupChange(ctx context.Context, monitorOnDutyChange *model.MonitorOnDutyChange) error {
	// 验证值班组 ID 是否有效
	if monitorOnDutyChange.OnDutyGroupID == 0 {
		a.l.Error("创建值班组变更失败：值班组 ID 为空")
		return errors.New("值班组 ID 为空")
	}

	// 确保值班组存在
	_, err := a.dao.GetMonitorOnDutyGroupById(ctx, monitorOnDutyChange.OnDutyGroupID)
	if err != nil {
		a.l.Error("创建值班组变更失败：获取值班组时出错", zap.Error(err))
		return err
	}

	// 获取原始用户信息
	originUser, err := a.userDao.GetUserByID(ctx, monitorOnDutyChange.OriginUserID)
	if err != nil {
		a.l.Error("创建值班组变更失败：获取原始用户时出错", zap.Error(err))
		return err
	}

	// 获取目标用户信息
	targetUser, err := a.userDao.GetUserByID(ctx, monitorOnDutyChange.OnDutyUserID)
	if err != nil {
		a.l.Error("创建值班组变更失败：获取目标用户时出错", zap.Error(err))
		return err
	}

	// 更新用户 ID
	monitorOnDutyChange.OriginUserID = originUser.ID
	monitorOnDutyChange.OnDutyUserID = targetUser.ID

	// 创建值班组变更记录
	if err := a.dao.CreateMonitorOnDutyGroupChange(ctx, monitorOnDutyChange); err != nil {
		a.l.Error("创建值班组变更失败", zap.Error(err))
		return err
	}

	a.l.Info("创建值班组变更记录成功", zap.Int("onDutyGroupID", monitorOnDutyChange.OnDutyGroupID))
	return nil
}

func (a *alertManagerOnDutyService) UpdateMonitorOnDutyGroup(ctx context.Context, monitorOnDutyGroup *model.MonitorOnDutyGroup) error {
	// 获取成员用户信息
	users, err := a.userDao.GetUserByUsernames(ctx, monitorOnDutyGroup.UserNames)
	if err != nil {
		a.l.Error("更新值班组失败：获取用户时出错", zap.Error(err))
		return err
	}

	// 更新成员
	monitorOnDutyGroup.Members = users

	// 更新值班组
	if err := a.dao.UpdateMonitorOnDutyGroup(ctx, monitorOnDutyGroup); err != nil {
		a.l.Error("更新值班组失败", zap.Error(err))
		return err
	}

	a.l.Info("更新值班组成功", zap.Int("id", monitorOnDutyGroup.ID))
	return nil
}

func (a *alertManagerOnDutyService) DeleteMonitorOnDutyGroup(ctx context.Context, id int) error {
	// 检查值班组是否有关联的发送组
	sendGroups, err := a.sendDao.GetMonitorSendGroupByOnDutyGroupId(ctx, id)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		a.l.Error("删除值班组失败：获取关联发送组时出错", zap.Error(err))
		return err
	}
	if len(sendGroups) > 0 {
		return errors.New("值班组存在关联发送组，无法删除")
	}

	// 删除值班组
	if err := a.dao.DeleteMonitorOnDutyGroup(ctx, id); err != nil {
		a.l.Error("删除值班组失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := a.cache.MonitorCacheManager(ctx); err != nil {
		a.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	a.l.Info("删除值班组成功", zap.Int("id", id))
	return nil
}

func (a *alertManagerOnDutyService) GetMonitorOnDutyGroup(ctx context.Context, id int) (*model.MonitorOnDutyGroup, error) {
	group, err := a.dao.GetMonitorOnDutyGroupById(ctx, id)
	if err != nil {
		a.l.Error("获取值班组失败", zap.Int("id", id), zap.Error(err))
		return nil, err
	}
	if group == nil {
		return nil, errors.New("值班组不存在")
	}
	return group, nil
}

func (a *alertManagerOnDutyService) GetMonitorOnDutyGroupFuturePlan(ctx context.Context, id int, startTimeStr string, endTimeStr string) (model.OnDutyPlanResp, error) {
	// 解析开始和结束时间
	startTime, err := time.Parse("2006-01-02", startTimeStr)
	if err != nil {
		a.l.Error("开始时间格式错误", zap.String("startTime", startTimeStr), zap.Error(err))
		return model.OnDutyPlanResp{}, errors.New("开始时间格式错误")
	}

	endTime, err := time.Parse("2006-01-02", endTimeStr)
	if err != nil {
		a.l.Error("结束时间格式错误", zap.String("endTime", endTimeStr), zap.Error(err))
		return model.OnDutyPlanResp{}, errors.New("结束时间格式错误")
	}

	// 验证时间范围
	if endTime.Before(startTime) {
		errMsg := "结束时间不能早于开始时间"
		a.l.Error(errMsg, zap.String("startTime", startTimeStr), zap.String("endTime", endTimeStr))
		return model.OnDutyPlanResp{}, errors.New(errMsg)
	}

	// 获取值班组信息
	group, err := a.dao.GetMonitorOnDutyGroupById(ctx, id)
	if err != nil {
		a.l.Error("获取值班组失败", zap.Int("id", id), zap.Error(err))
		return model.OnDutyPlanResp{}, err
	}

	// 初始化返回结果
	planResp := model.OnDutyPlanResp{
		Details:       []model.OnDutyOne{},
		Map:           make(map[string]string),
		UserNameMap:   make(map[string]string),
		OriginUserMap: make(map[string]string),
	}

	// 获取指定时间范围内的值班计划变更
	histories, err := a.dao.GetMonitorOnDutyHistoryByGroupIdAndTimeRange(ctx, id, startTimeStr, endTimeStr)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		a.l.Error("获取值班计划变更失败", zap.Int("id", id), zap.Error(err))
		return model.OnDutyPlanResp{}, err
	}

	// 构建变更记录的映射，方便后续查询
	historyMap := make(map[string]*model.MonitorOnDutyHistory)
	for _, history := range histories {
		historyMap[history.DateString] = history
	}

	// 获取今天的日期
	today := time.Now().Format("2006-01-02")

	// 生成从开始日期到结束日期的所有日期列表
	totalDays := int(endTime.Sub(startTime).Hours()/24) + 1
	for i := 0; i < totalDays; i++ {
		currentDate := startTime.AddDate(0, 0, i).Format("2006-01-02") // 计算开始日期和结束日期之间的每一天

		var user *model.User
		var originUserName string

		// 如果有历史变更记录，使用变更后的用户
		if history, exists := historyMap[currentDate]; exists {
			user, err = a.userDao.GetUserByID(ctx, history.OnDutyUserID)
			if err != nil {
				a.l.Warn("获取用户信息失败", zap.Int("userID", history.OnDutyUserID), zap.Error(err))
				continue
			}
			if history.OriginUserID > 0 {
				originUser, err := a.userDao.GetUserByID(ctx, history.OriginUserID)
				if err == nil {
					originUserName = originUser.RealName
				} else {
					a.l.Warn("获取原始用户信息失败", zap.Int("originUserID", history.OriginUserID), zap.Error(err))
				}
			}
		} else {
			// 没有变更记录，按照排班规则计算值班人
			user = pkg.CalculateOnDutyUser(ctx, a.l, a.dao, group, currentDate, today)
			if user == nil {
				a.l.Warn("无法计算值班人", zap.String("date", currentDate))
				continue
			}
		}

		// 构建值班信息
		onDutyOne := model.OnDutyOne{
			Date:       currentDate,
			User:       user,
			OriginUser: originUserName,
		}

		// 添加到返回结果
		planResp.Details = append(planResp.Details, onDutyOne)
		planResp.Map[currentDate] = user.RealName
		planResp.UserNameMap[currentDate] = user.Username
		planResp.OriginUserMap[currentDate] = originUserName
	}

	a.l.Info("获取值班计划成功", zap.Int("groupID", id), zap.String("startTime", startTimeStr), zap.String("endTime", endTimeStr))
	return planResp, nil
}
