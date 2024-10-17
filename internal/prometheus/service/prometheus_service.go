package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/cache"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao"
	userDao "github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils/prometheus"
	"github.com/prometheus/alertmanager/types"
	promModel "github.com/prometheus/common/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"sync"
	"time"
)

type PrometheusService interface {
	GetMonitorScrapePoolList(ctx context.Context, search *string) ([]*model.MonitorScrapePool, error)
	CreateMonitorScrapePool(ctx context.Context, monitorScrapePool *model.MonitorScrapePool) error
	UpdateMonitorScrapePool(ctx context.Context, monitorScrapePool *model.MonitorScrapePool) error
	DeleteMonitorScrapePool(ctx context.Context, id int) error

	GetMonitorScrapeJobList(ctx context.Context, search *string) ([]*model.MonitorScrapeJob, error)
	CreateMonitorScrapeJob(ctx context.Context, monitorScrapeJob *model.MonitorScrapeJob) error
	UpdateMonitorScrapeJob(ctx context.Context, monitorScrapeJob *model.MonitorScrapeJob) error
	DeleteMonitorScrapeJob(ctx context.Context, id int) error

	GetMonitorPrometheusYaml(ctx context.Context, ip string) string
	GetMonitorPrometheusAlertRuleYaml(ctx context.Context, ip string) string
	GetMonitorPrometheusRecordYaml(ctx context.Context, ip string) string
	GetMonitorAlertManagerYaml(ctx context.Context, ip string) string

	GetMonitorOnDutyGroupList(ctx context.Context, searchName *string) ([]*model.MonitorOnDutyGroup, error)
	CreateMonitorOnDutyGroup(ctx context.Context, monitorOnDutyGroup *model.MonitorOnDutyGroup) error
	CreateMonitorOnDutyGroupChange(ctx context.Context, monitorOnDutyChange *model.MonitorOnDutyChange) error
	UpdateMonitorOnDutyGroup(ctx context.Context, monitorOnDutyGroup *model.MonitorOnDutyGroup) error
	DeleteMonitorOnDutyGroup(ctx context.Context, id int) error
	GetMonitorOnDutyGroup(ctx context.Context, id int) (*model.MonitorOnDutyGroup, error)
	GetMonitorOnDutyGroupFuturePlan(ctx context.Context, id int, startTime string, endTime string) (model.OnDutyPlanResp, error)

	GetMonitorAlertManagerPoolList(ctx context.Context, searchName *string) ([]*model.MonitorAlertManagerPool, error)
	CreateMonitorAlertManagerPool(ctx context.Context, monitorAlertManagerPool *model.MonitorAlertManagerPool) error
	UpdateMonitorAlertManagerPool(ctx context.Context, monitorAlertManagerPool *model.MonitorAlertManagerPool) error
	DeleteMonitorAlertManagerPool(ctx context.Context, id int) error

	GetMonitorSendGroupList(ctx context.Context, searchName *string) ([]*model.MonitorSendGroup, error)
	CreateMonitorSendGroup(ctx context.Context, monitorSendGroup *model.MonitorSendGroup) error
	UpdateMonitorSendGroup(ctx context.Context, monitorSendGroup *model.MonitorSendGroup) error
	DeleteMonitorSendGroup(ctx context.Context, id int) error
	GetMonitorAlertRuleList(ctx context.Context, searchName *string) ([]*model.MonitorAlertRule, error)
	PromqlExprCheck(ctx context.Context, expr string) (bool, error)
	CreateMonitorAlertRule(ctx context.Context, monitorAlertRule *model.MonitorAlertRule) error
	UpdateMonitorAlertRule(ctx context.Context, monitorAlertRule *model.MonitorAlertRule) error
	EnableSwitchMonitorAlertRule(ctx context.Context, id int) error
	BatchEnableSwitchMonitorAlertRule(ctx context.Context, ids []int) error
	DeleteMonitorAlertRule(ctx context.Context, id int) error
	BatchDeleteMonitorAlertRule(ctx context.Context, ids []int) error

	GetMonitorAlertEventList(ctx context.Context, searchName *string) ([]*model.MonitorAlertEvent, error)
	EventAlertSilence(ctx context.Context, id int, event *model.AlertEventSilenceRequest, userId int) error
	EventAlertClaim(ctx context.Context, id int, userId int) error
	BatchEventAlertSilence(ctx context.Context, ids *model.BatchEventAlertSilenceRequest, userId int) error
	GetMonitorRecordRuleList(ctx context.Context, searchName *string) ([]*model.MonitorRecordRule, error)
	CreateMonitorRecordRule(ctx context.Context, monitorRecordRule *model.MonitorRecordRule) error
	UpdateMonitorRecordRule(ctx context.Context, monitorRecordRule *model.MonitorRecordRule) error
	DeleteMonitorRecordRule(ctx context.Context, id int) error
	BatchDeleteMonitorRecordRule(ctx context.Context, ids []int) error
	EnableSwitchMonitorRecordRule(ctx context.Context, id int) error
	BatchEnableSwitchMonitorRecordRule(ctx context.Context, ids []int) error
}

type prometheusService struct {
	dao     dao.PrometheusDao
	cache   cache.MonitorCache
	userDao userDao.UserDAO
	l       *zap.Logger
}

func NewPrometheusService(dao dao.PrometheusDao, cache cache.MonitorCache, l *zap.Logger, userDao userDao.UserDAO) PrometheusService {
	return &prometheusService{
		dao:     dao,
		userDao: userDao,
		l:       l,
		cache:   cache,
	}
}

// GetMonitorScrapePoolList 获取监控抓取池列表，可选根据名称过滤
func (p *prometheusService) GetMonitorScrapePoolList(ctx context.Context, search *string) ([]*model.MonitorScrapePool, error) {
	return pkg.HandleList(ctx, search,
		p.dao.SearchMonitorScrapePoolsByName, // 搜索函数
		p.dao.GetAllMonitorScrapePool) // 获取所有函数
}

// CreateMonitorScrapePool 创建新的监控抓取池
func (p *prometheusService) CreateMonitorScrapePool(ctx context.Context, monitorScrapePool *model.MonitorScrapePool) error {
	// 检查抓取池 IP 是否已存在
	exists, err := p.dao.CheckMonitorScrapePoolExists(ctx, monitorScrapePool)
	if err != nil {
		p.l.Error("创建抓取池失败：检查抓取池是否存在时出错", zap.Error(err))
		return err
	}
	if exists {
		return errors.New("抓取池 IP 已存在")
	}

	// 创建抓取池
	if err := p.dao.CreateMonitorScrapePool(ctx, monitorScrapePool); err != nil {
		p.l.Error("创建抓取池失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := p.cache.MonitorCacheManager(ctx); err != nil {
		p.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	p.l.Info("创建抓取池成功", zap.Int("id", monitorScrapePool.ID))
	return nil
}

// UpdateMonitorScrapePool 更新现有的监控抓取池
func (p *prometheusService) UpdateMonitorScrapePool(ctx context.Context, monitorScrapePool *model.MonitorScrapePool) error {
	// 确保要更新的抓取池存在
	pools, err := p.dao.GetAllMonitorScrapePool(ctx)
	if err != nil {
		p.l.Error("更新抓取池失败：获取抓取池时出错", zap.Error(err))
		return err
	}

	newPools := make([]*model.MonitorScrapePool, 0)

	for _, pool := range pools {
		if pool.ID == monitorScrapePool.ID {
			continue
		}

		if pool.Name == monitorScrapePool.Name {
			return errors.New("抓取池名称已存在")
		}

		newPools = append(newPools, pool)
	}

	// 检查新的抓取池 IP 是否已存在
	exists := pkg.CheckPoolIpExists(monitorScrapePool, newPools)
	if exists {
		return errors.New("抓取池 IP 已存在")
	}

	// 更新抓取池
	if err := p.dao.UpdateMonitorScrapePool(ctx, monitorScrapePool); err != nil {
		p.l.Error("更新抓取池失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := p.cache.MonitorCacheManager(ctx); err != nil {
		p.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	p.l.Info("更新抓取池成功", zap.Int("id", monitorScrapePool.ID))
	return nil
}

// DeleteMonitorScrapePool 删除指定的监控抓取池
func (p *prometheusService) DeleteMonitorScrapePool(ctx context.Context, id int) error {
	// 检查抓取池是否有相关的抓取作业
	jobs, err := p.dao.GetMonitorScrapeJobsByPoolId(ctx, id)
	if err != nil {
		p.l.Error("删除抓取池失败：获取抓取作业时出错", zap.Error(err))
		return err
	}

	if len(jobs) > 0 {
		return errors.New("抓取池存在相关抓取作业，无法删除")
	}

	// 删除抓取池
	if err := p.dao.DeleteMonitorScrapePool(ctx, id); err != nil {
		p.l.Error("删除抓取池失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := p.cache.MonitorCacheManager(ctx); err != nil {
		p.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	p.l.Info("删除抓取池成功", zap.Int("id", id))
	return nil
}

// GetMonitorScrapeJobList 获取监控抓取作业列表，可选根据名称过滤
func (p *prometheusService) GetMonitorScrapeJobList(ctx context.Context, search *string) ([]*model.MonitorScrapeJob, error) {
	return pkg.HandleList(ctx, search,
		p.dao.SearchMonitorScrapeJobsByName, // 搜索函数
		p.dao.GetAllMonitorScrapeJobs) // 获取所有函数
}

// CreateMonitorScrapeJob 创建新的监控抓取作业
func (p *prometheusService) CreateMonitorScrapeJob(ctx context.Context, monitorScrapeJob *model.MonitorScrapeJob) error {
	// 检查抓取作业是否已存在
	exists, err := p.dao.CheckMonitorScrapeJobExists(ctx, monitorScrapeJob.Name)
	if err != nil {
		p.l.Error("创建抓取作业失败：检查抓取作业是否存在时出错", zap.Error(err))
		return err
	}

	if exists {
		return errors.New("抓取作业已存在")
	}

	// 创建抓取作业
	if err := p.dao.CreateMonitorScrapeJob(ctx, monitorScrapeJob); err != nil {
		p.l.Error("创建抓取作业失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := p.cache.MonitorCacheManager(ctx); err != nil {
		p.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	p.l.Info("创建抓取作业成功", zap.Int("id", monitorScrapeJob.ID))
	return nil
}

// UpdateMonitorScrapeJob 更新现有的监控抓取作业
func (p *prometheusService) UpdateMonitorScrapeJob(ctx context.Context, monitorScrapeJob *model.MonitorScrapeJob) error {
	// 检查新的抓取作业名称是否已存在
	exists, err := p.dao.CheckMonitorScrapeJobExists(ctx, monitorScrapeJob.Name)
	if err != nil {
		p.l.Error("更新抓取作业失败：检查抓取作业是否存在时出错", zap.Error(err))
		return err
	}

	if exists {
		return errors.New("抓取作业名称已存在")
	}

	// 更新抓取作业
	if err := p.dao.UpdateMonitorScrapeJob(ctx, monitorScrapeJob); err != nil {
		p.l.Error("更新抓取作业失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := p.cache.MonitorCacheManager(ctx); err != nil {
		p.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	p.l.Info("更新抓取作业成功", zap.Int("id", monitorScrapeJob.ID))
	return nil
}

// DeleteMonitorScrapeJob 删除指定的监控抓取作业
func (p *prometheusService) DeleteMonitorScrapeJob(ctx context.Context, id int) error {
	// 删除抓取作业
	if err := p.dao.DeleteMonitorScrapeJob(ctx, id); err != nil {
		p.l.Error("删除抓取作业失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := p.cache.MonitorCacheManager(ctx); err != nil {
		p.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	p.l.Info("删除抓取作业成功", zap.Int("id", id))
	return nil
}

// GetMonitorPrometheusYaml 获取 Prometheus 主配置 YAML
func (p *prometheusService) GetMonitorPrometheusYaml(ctx context.Context, ip string) string {
	return p.cache.GetPrometheusMainConfigByIP(ip)
}

// GetMonitorPrometheusAlertRuleYaml 获取 Prometheus 告警规则 YAML
func (p *prometheusService) GetMonitorPrometheusAlertRuleYaml(ctx context.Context, ip string) string {
	return p.cache.GetPrometheusAlertRuleConfigYamlByIp(ip)
}

// GetMonitorPrometheusRecordYaml 获取 Prometheus 记录规则 YAML
func (p *prometheusService) GetMonitorPrometheusRecordYaml(ctx context.Context, ip string) string {
	return p.cache.GetPrometheusRecordRuleConfigYamlByIp(ip)
}

// GetMonitorAlertManagerYaml 获取 AlertManager 主配置 YAML
func (p *prometheusService) GetMonitorAlertManagerYaml(ctx context.Context, ip string) string {
	return p.cache.GetAlertManagerMainConfigYamlByIP(ip)
}

// GetMonitorOnDutyGroupList 获取值班组列表，可选根据名称过滤
func (p *prometheusService) GetMonitorOnDutyGroupList(ctx context.Context, searchName *string) ([]*model.MonitorOnDutyGroup, error) {
	return pkg.HandleList(ctx, searchName,
		p.dao.SearchMonitorOnDutyGroupByName, // 搜索函数
		p.dao.GetAllMonitorOndutyGroup) // 获取所有函数
}

// CreateMonitorOnDutyGroup 创建新的值班组
func (p *prometheusService) CreateMonitorOnDutyGroup(ctx context.Context, monitorOnDutyGroup *model.MonitorOnDutyGroup) error {
	// 检查值班组是否已存在
	exists, err := p.dao.CheckMonitorOnDutyGroupExists(ctx, monitorOnDutyGroup)
	if err != nil {
		p.l.Error("创建值班组失败：检查值班组是否存在时出错", zap.Error(err))
		return err
	}
	if exists {
		return errors.New("值班组已存在")
	}

	users, err := p.userDao.GetUserByUsernames(ctx, monitorOnDutyGroup.UserNames)
	if err != nil {
		p.l.Error("创建值班组失败：获取值班组成员信息时出错", zap.Error(err))
		return err
	}

	monitorOnDutyGroup.Members = users
	// 创建值班组
	if err := p.dao.CreateMonitorOnDutyGroup(ctx, monitorOnDutyGroup); err != nil {
		p.l.Error("创建值班组失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := p.cache.MonitorCacheManager(ctx); err != nil {
		p.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	p.l.Info("创建值班组成功", zap.Int("id", monitorOnDutyGroup.ID))
	return nil
}

// CreateMonitorOnDutyGroupChange 创建值班组变更记录
func (p *prometheusService) CreateMonitorOnDutyGroupChange(ctx context.Context, monitorOnDutyChange *model.MonitorOnDutyChange) error {
	// 验证值班组 ID 是否有效
	if monitorOnDutyChange.OnDutyGroupID == 0 {
		p.l.Error("创建值班组变更失败：值班组 ID 为空")
		return errors.New("值班组 ID 为空")
	}

	// 确保值班组存在
	_, err := p.dao.GetMonitorOnDutyGroupById(ctx, monitorOnDutyChange.OnDutyGroupID)
	if err != nil {
		p.l.Error("创建值班组变更失败：获取值班组时出错", zap.Error(err))
		return err
	}

	// 获取原始用户信息
	originUser, err := p.userDao.GetUserByID(ctx, monitorOnDutyChange.OriginUserID)
	if err != nil {
		p.l.Error("创建值班组变更失败：获取原始用户时出错", zap.Error(err))
		return err
	}

	// 获取目标用户信息
	targetUser, err := p.userDao.GetUserByID(ctx, monitorOnDutyChange.OnDutyUserID)
	if err != nil {
		p.l.Error("创建值班组变更失败：获取目标用户时出错", zap.Error(err))
		return err
	}

	// 更新用户 ID
	monitorOnDutyChange.OriginUserID = originUser.ID
	monitorOnDutyChange.OnDutyUserID = targetUser.ID

	// 创建值班组变更记录
	if err := p.dao.CreateMonitorOnDutyGroupChange(ctx, monitorOnDutyChange); err != nil {
		p.l.Error("创建值班组变更失败", zap.Error(err))
		return err
	}

	p.l.Info("创建值班组变更记录成功", zap.Int("onDutyGroupID", monitorOnDutyChange.OnDutyGroupID))
	return nil
}

// UpdateMonitorOnDutyGroup 更新现有的值班组
func (p *prometheusService) UpdateMonitorOnDutyGroup(ctx context.Context, monitorOnDutyGroup *model.MonitorOnDutyGroup) error {
	// 获取成员用户信息
	users, err := p.userDao.GetUserByUsernames(ctx, monitorOnDutyGroup.UserNames)
	if err != nil {
		p.l.Error("更新值班组失败：获取用户时出错", zap.Error(err))
		return err
	}

	// 更新成员
	monitorOnDutyGroup.Members = users

	// 更新值班组
	if err := p.dao.UpdateMonitorOnDutyGroup(ctx, monitorOnDutyGroup); err != nil {
		p.l.Error("更新值班组失败", zap.Error(err))
		return err
	}

	p.l.Info("更新值班组成功", zap.Int("id", monitorOnDutyGroup.ID))
	return nil
}

// DeleteMonitorOnDutyGroup 删除指定的值班组
func (p *prometheusService) DeleteMonitorOnDutyGroup(ctx context.Context, id int) error {
	// 检查值班组是否有关联的发送组
	sendGroups, err := p.dao.GetMonitorSendGroupByOnDutyGroupId(ctx, id)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		p.l.Error("删除值班组失败：获取关联发送组时出错", zap.Error(err))
		return err
	}
	if len(sendGroups) > 0 {
		return errors.New("值班组存在关联发送组，无法删除")
	}

	// 删除值班组
	if err := p.dao.DeleteMonitorOnDutyGroup(ctx, id); err != nil {
		p.l.Error("删除值班组失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := p.cache.MonitorCacheManager(ctx); err != nil {
		p.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	p.l.Info("删除值班组成功", zap.Int("id", id))
	return nil
}

// GetMonitorOnDutyGroup 获取指定的值班组
func (p *prometheusService) GetMonitorOnDutyGroup(ctx context.Context, id int) (*model.MonitorOnDutyGroup, error) {
	group, err := p.dao.GetMonitorOnDutyGroupById(ctx, id)
	if err != nil {
		p.l.Error("获取值班组失败", zap.Int("id", id), zap.Error(err))
		return nil, err
	}
	if group == nil {
		return nil, errors.New("值班组不存在")
	}
	return group, nil
}

// GetMonitorOnDutyGroupFuturePlan 获取指定值班组在指定时间范围内的值班计划变更
func (p *prometheusService) GetMonitorOnDutyGroupFuturePlan(ctx context.Context, id int, startTimeStr string, endTimeStr string) (model.OnDutyPlanResp, error) {
	// 解析开始和结束时间
	startTime, err := time.Parse("2006-01-02", startTimeStr)
	if err != nil {
		p.l.Error("开始时间格式错误", zap.String("startTime", startTimeStr), zap.Error(err))
		return model.OnDutyPlanResp{}, errors.New("开始时间格式错误")
	}

	endTime, err := time.Parse("2006-01-02", endTimeStr)
	if err != nil {
		p.l.Error("结束时间格式错误", zap.String("endTime", endTimeStr), zap.Error(err))
		return model.OnDutyPlanResp{}, errors.New("结束时间格式错误")
	}

	// 验证时间范围
	if endTime.Before(startTime) {
		errMsg := "结束时间不能早于开始时间"
		p.l.Error(errMsg, zap.String("startTime", startTimeStr), zap.String("endTime", endTimeStr))
		return model.OnDutyPlanResp{}, errors.New(errMsg)
	}

	// 获取值班组信息
	group, err := p.dao.GetMonitorOnDutyGroupById(ctx, id)
	if err != nil {
		p.l.Error("获取值班组失败", zap.Int("id", id), zap.Error(err))
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
	histories, err := p.dao.GetMonitorOnDutyHistoryByGroupIdAndTimeRange(ctx, id, startTimeStr, endTimeStr)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		p.l.Error("获取值班计划变更失败", zap.Int("id", id), zap.Error(err))
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
			user, err = p.userDao.GetUserByID(ctx, history.OnDutyUserID)
			if err != nil {
				p.l.Warn("获取用户信息失败", zap.Int("userID", history.OnDutyUserID), zap.Error(err))
				continue
			}
			if history.OriginUserID > 0 {
				originUser, err := p.userDao.GetUserByID(ctx, history.OriginUserID)
				if err == nil {
					originUserName = originUser.RealName
				} else {
					p.l.Warn("获取原始用户信息失败", zap.Int("originUserID", history.OriginUserID), zap.Error(err))
				}
			}
		} else {
			// 没有变更记录，按照排班规则计算值班人
			user = p.calculateOnDutyUser(ctx, group, currentDate, today)
			if user == nil {
				p.l.Warn("无法计算值班人", zap.String("date", currentDate))
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

	p.l.Info("获取值班计划成功", zap.Int("groupID", id), zap.String("startTime", startTimeStr), zap.String("endTime", endTimeStr))
	return planResp, nil
}

// calculateOnDutyUser 根据排班规则计算指定日期的值班人
func (p *prometheusService) calculateOnDutyUser(ctx context.Context, group *model.MonitorOnDutyGroup, dateStr string, todayStr string) *model.User {
	// 解析目标日期
	targetDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		p.l.Error("日期解析失败", zap.String("date", dateStr), zap.Error(err))
		return nil
	}

	// 解析今天的日期
	today, err := time.Parse("2006-01-02", todayStr)
	if err != nil {
		p.l.Error("解析今天日期失败", zap.String("today", todayStr), zap.Error(err))
		return nil
	}

	// 计算从今天开始的天数差
	daysDiff := int(targetDate.Sub(today).Hours()) / 24

	// 获取当前值班人的索引
	currentUserIndex := p.getCurrentUserIndex(ctx, group, todayStr)
	if currentUserIndex == -1 {
		p.l.Warn("未找到当前值班人", zap.String("today", todayStr))
		return nil
	}

	// 获取总成员数和每班轮值天数
	totalMembers := len(group.Members)
	shiftDays := group.ShiftDays

	// 检查轮班天数是否为零，避免除零错误
	if shiftDays == 0 || totalMembers == 0 {
		p.l.Error("轮班天数或成员数量无效", zap.Int("shiftDays", shiftDays), zap.Int("totalMembers", totalMembers))
		return nil
	}

	// 总的轮班周期长度
	totalShiftLength := totalMembers * shiftDays

	// 自定义取模函数，确保结果为非负数
	mod := func(a, b int) int {
		if b == 0 {
			p.l.Error("除零错误", zap.Int("b", b))
			return -1
		}
		return (a%b + b) % b
	}

	// 计算目标日期的值班人索引
	indexInShift := mod(currentUserIndex*shiftDays+daysDiff, totalShiftLength)
	if indexInShift == -1 {
		p.l.Error("取模运算出错，可能是因为轮班天数或成员数量无效", zap.Int("currentUserIndex", currentUserIndex), zap.Int("shiftDays", shiftDays), zap.Int("totalMembers", totalMembers))
		return nil
	}

	// 确定值班人的索引
	userIndex := indexInShift / shiftDays

	// 确认索引范围有效，返回对应的成员
	if userIndex >= 0 && userIndex < totalMembers {
		return group.Members[userIndex]
	}

	// 如果索引超出范围
	p.l.Error("计算出的值班人索引无效", zap.Int("userIndex", userIndex), zap.Int("totalMembers", totalMembers))

	return nil
}

// getCurrentUserIndex 获取当前值班人在成员列表中的索引
func (p *prometheusService) getCurrentUserIndex(ctx context.Context, group *model.MonitorOnDutyGroup, todayStr string) int {
	// 尝试从历史记录中获取今天的值班人
	todayHistory, err := p.dao.GetMonitorOnDutyHistoryByGroupIdAndDay(ctx, group.ID, todayStr)
	if err == nil && todayHistory.OnDutyUserID > 0 {
		for index, member := range group.Members {
			if member.ID == todayHistory.OnDutyUserID {
				return index
			}
		}
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		// 如果查询发生了其他错误，记录日志
		p.l.Error("获取今天的值班历史记录失败", zap.Error(err))
	}

	// 如果没有历史记录，默认第一个成员为当前值班人
	return 0
}

// GetMonitorAlertManagerPoolList 获取 AlertManager 集群池列表，可选根据名称过滤
func (p *prometheusService) GetMonitorAlertManagerPoolList(ctx context.Context, searchName *string) ([]*model.MonitorAlertManagerPool, error) {
	return pkg.HandleList(ctx, searchName,
		p.dao.SearchMonitorAlertManagerPoolByName, // 搜索函数
		p.dao.GetAllAlertManagerPools) // 获取所有函数
}

// CreateMonitorAlertManagerPool 创建新的 AlertManager 集群池
func (p *prometheusService) CreateMonitorAlertManagerPool(ctx context.Context, monitorAlertManagerPool *model.MonitorAlertManagerPool) error {
	// 检查 AlertManager IP 是否已存在
	exists, err := p.dao.CheckMonitorAlertManagerPoolExists(ctx, monitorAlertManagerPool)
	if err != nil {
		p.l.Error("创建 AlertManager 集群池失败：检查是否存在时出错", zap.Error(err))
		return err
	}
	if exists {
		return errors.New("AlertManager 集群池 IP 已存在")
	}

	// 创建 AlertManager 集群池
	if err := p.dao.CreateMonitorAlertManagerPool(ctx, monitorAlertManagerPool); err != nil {
		p.l.Error("创建 AlertManager 集群池失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := p.cache.MonitorCacheManager(ctx); err != nil {
		p.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	p.l.Info("创建 AlertManager 集群池成功", zap.Int("id", monitorAlertManagerPool.ID))
	return nil
}

// UpdateMonitorAlertManagerPool 更新现有的 AlertManager 集群池
func (p *prometheusService) UpdateMonitorAlertManagerPool(ctx context.Context, monitorAlertManagerPool *model.MonitorAlertManagerPool) error {
	alerts, err := p.dao.GetAllAlertManagerPools(ctx)
	if err != nil {
		p.l.Error("更新 AlertManager 集群池失败：获取集群池时出错", zap.Error(err))
		return err
	}

	// 检查新的 AlertManager IP 是否已存在
	exists := pkg.CheckAlertIpExists(monitorAlertManagerPool, alerts)
	if exists {
		return errors.New("AlertManager 集群池 IP 已存在")
	}

	// 更新 AlertManager 集群池
	if err := p.dao.UpdateMonitorAlertManagerPool(ctx, monitorAlertManagerPool); err != nil {
		p.l.Error("更新 AlertManager 集群池失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := p.cache.MonitorCacheManager(ctx); err != nil {
		p.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	p.l.Info("更新 AlertManager 集群池成功", zap.Int("id", monitorAlertManagerPool.ID))
	return nil
}

// DeleteMonitorAlertManagerPool 删除指定的 AlertManager 集群池
func (p *prometheusService) DeleteMonitorAlertManagerPool(ctx context.Context, id int) error {
	// 检查 AlertManager 集群池是否有关联的发送组
	sendGroups, err := p.dao.GetMonitorSendGroupByPoolId(ctx, id)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		p.l.Error("删除 AlertManager 集群池失败：获取关联发送组时出错", zap.Error(err))
		return err
	}

	if len(sendGroups) > 0 {
		return errors.New("AlertManager 集群池存在关联发送组，无法删除")
	}

	// 删除 AlertManager 集群池
	if err := p.dao.DeleteMonitorAlertManagerPool(ctx, id); err != nil {
		p.l.Error("删除 AlertManager 集群池失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := p.cache.MonitorCacheManager(ctx); err != nil {
		p.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	p.l.Info("删除 AlertManager 集群池成功", zap.Int("id", id))
	return nil
}

// GetMonitorSendGroupList 获取发送组列表，支持按名称搜索
func (p *prometheusService) GetMonitorSendGroupList(ctx context.Context, searchName *string) ([]*model.MonitorSendGroup, error) {
	return pkg.HandleList(ctx, searchName,
		p.dao.SearchMonitorSendGroupByName,
		p.dao.GetMonitorSendGroupList)
}

// CreateMonitorSendGroup 创建发送组
func (p *prometheusService) CreateMonitorSendGroup(ctx context.Context, monitorSendGroup *model.MonitorSendGroup) error {
	// 检查发送组是否已存在
	exists, err := p.dao.CheckMonitorSendGroupExists(ctx, monitorSendGroup)
	if err != nil {
		p.l.Error("创建发送组失败：检查发送组是否存在时出错", zap.Error(err))
		return err
	}

	if exists {
		return errors.New("发送组已存在")
	}

	// 创建发送组
	if err := p.dao.CreateMonitorSendGroup(ctx, monitorSendGroup); err != nil {
		p.l.Error("创建发送组失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := p.cache.MonitorCacheManager(ctx); err != nil {
		p.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	return nil
}

// UpdateMonitorSendGroup 更新发送组
func (p *prometheusService) UpdateMonitorSendGroup(ctx context.Context, monitorSendGroup *model.MonitorSendGroup) error {
	// 检查发送组名称是否重复
	exists, err := p.dao.CheckMonitorSendGroupNameExists(ctx, monitorSendGroup)
	if err != nil {
		p.l.Error("更新发送组失败：检查发送组名称时出错", zap.Error(err))
		return err
	}

	if exists {
		return errors.New("发送组名称已存在")
	}

	// 更新发送组
	if err := p.dao.UpdateMonitorSendGroup(ctx, monitorSendGroup); err != nil {
		p.l.Error("更新发送组失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := p.cache.MonitorCacheManager(ctx); err != nil {
		p.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	return nil
}

// DeleteMonitorSendGroup 删除发送组
func (p *prometheusService) DeleteMonitorSendGroup(ctx context.Context, id int) error {
	// 检查发送组是否有关联的资源
	associatedResources, err := p.dao.GetAssociatedResourcesBySendGroupId(ctx, id)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		p.l.Error("删除发送组失败：获取关联资源时出错", zap.Error(err))
		return err
	}

	if len(associatedResources) > 0 {
		return errors.New("发送组存在关联资源，无法删除")
	}

	// 删除发送组
	if err := p.dao.DeleteMonitorSendGroup(ctx, id); err != nil {
		p.l.Error("删除发送组失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := p.cache.MonitorCacheManager(ctx); err != nil {
		p.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	return nil
}

// GetMonitorAlertRuleList 获取告警规则列表，支持按名称搜索
func (p *prometheusService) GetMonitorAlertRuleList(ctx context.Context, searchName *string) ([]*model.MonitorAlertRule, error) {
	return pkg.HandleList(ctx, searchName,
		p.dao.SearchMonitorAlertRuleByName,
		p.dao.GetMonitorAlertRuleList)
}

// PromqlExprCheck 检查 PromQL 表达式是否有效
func (p *prometheusService) PromqlExprCheck(ctx context.Context, expr string) (bool, error) {
	return pkg.PromqlExprCheck(expr)
}

// CreateMonitorAlertRule 创建告警规则
func (p *prometheusService) CreateMonitorAlertRule(ctx context.Context, monitorAlertRule *model.MonitorAlertRule) error {
	// 检查告警规则是否已存在
	exists, err := p.dao.CheckMonitorAlertRuleExists(ctx, monitorAlertRule)
	if err != nil {
		p.l.Error("创建告警规则失败：检查告警规则是否存在时出错", zap.Error(err))
		return err
	}

	if exists {
		return errors.New("告警规则已存在")
	}

	// 创建告警规则
	if err := p.dao.CreateMonitorAlertRule(ctx, monitorAlertRule); err != nil {
		p.l.Error("创建告警规则失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := p.cache.MonitorCacheManager(ctx); err != nil {
		p.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	return nil
}

// UpdateMonitorAlertRule 更新告警规则
func (p *prometheusService) UpdateMonitorAlertRule(ctx context.Context, monitorAlertRule *model.MonitorAlertRule) error {
	// 检查告警规则名称是否重复
	exists, err := p.dao.CheckMonitorAlertRuleNameExists(ctx, monitorAlertRule)
	if err != nil {
		p.l.Error("更新告警规则失败：检查告警规则名称时出错", zap.Error(err))
		return err
	}

	if exists {
		return errors.New("告警规则名称已存在")
	}

	// 更新告警规则
	if err := p.dao.UpdateMonitorAlertRule(ctx, monitorAlertRule); err != nil {
		p.l.Error("更新告警规则失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := p.cache.MonitorCacheManager(ctx); err != nil {
		p.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	return nil
}

// EnableSwitchMonitorAlertRule 切换告警规则的启用状态
func (p *prometheusService) EnableSwitchMonitorAlertRule(ctx context.Context, id int) error {
	if err := p.dao.EnableSwitchMonitorAlertRule(ctx, id); err != nil {
		p.l.Error("切换告警规则状态失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := p.cache.MonitorCacheManager(ctx); err != nil {
		p.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	return nil
}

// BatchEnableSwitchMonitorAlertRule 批量切换告警规则的启用状态
func (p *prometheusService) BatchEnableSwitchMonitorAlertRule(ctx context.Context, ids []int) error {
	// 批量切换告警规则状态
	if err := p.dao.BatchEnableSwitchMonitorAlertRule(ctx, ids); err != nil {
		p.l.Error("批量切换告警规则状态失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := p.cache.MonitorCacheManager(ctx); err != nil {
		p.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	return nil
}

// DeleteMonitorAlertRule 删除告警规则
func (p *prometheusService) DeleteMonitorAlertRule(ctx context.Context, id int) error {
	// 删除告警规则
	if err := p.dao.DeleteMonitorAlertRule(ctx, id); err != nil {
		p.l.Error("删除告警规则失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := p.cache.MonitorCacheManager(ctx); err != nil {
		p.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	return nil
}

// BatchDeleteMonitorAlertRule 批量删除告警规则
func (p *prometheusService) BatchDeleteMonitorAlertRule(ctx context.Context, ids []int) error {
	for _, id := range ids {
		if err := p.DeleteMonitorAlertRule(ctx, id); err != nil {
			// 记录错误但继续删除其他规则
			p.l.Error("批量删除告警规则失败", zap.Int("id", id), zap.Error(err))
			return fmt.Errorf("删除告警规则 ID %d 失败: %v", id, err)
		}
	}

	return nil
}

// GetMonitorAlertEventList 获取告警事件列表，支持按名称搜索
func (p *prometheusService) GetMonitorAlertEventList(ctx context.Context, searchName *string) ([]*model.MonitorAlertEvent, error) {
	return pkg.HandleList(ctx, searchName,
		p.dao.SearchMonitorAlertEventByName,
		p.dao.GetMonitorAlertEventList)
}

// EventAlertSilence 设置单个告警事件为静默状态
func (p *prometheusService) EventAlertSilence(ctx context.Context, id int, event *model.AlertEventSilenceRequest, userId int) error {
	// 验证 ID 是否有效
	if id <= 0 {
		p.l.Error("设置静默失败：无效的 ID", zap.Int("id", id))
		return fmt.Errorf("无效的 ID: %d", id)
	}

	// 获取 AlertEvent
	alertEvent, err := p.dao.GetAlertEventByID(ctx, id)
	if err != nil {
		p.l.Error("设置静默失败：无法获取 AlertEvent", zap.Error(err), zap.Int("id", id))
		return err
	}

	// 获取用户信息
	user, err := p.userDao.GetUserByID(ctx, userId)
	if err != nil {
		p.l.Error("设置静默失败：无效的 userId", zap.Int("userId", userId))
		return fmt.Errorf("无效的 userId: %d", userId)
	}

	// 解析持续时间
	duration, err := promModel.ParseDuration(event.Time)
	if err != nil {
		p.l.Error("设置静默失败：解析持续时间错误", zap.Error(err))
		return fmt.Errorf("无效的持续时间: %v", err)
	}

	// 构建匹配器
	matchers, err := pkg.BuildMatchers(alertEvent, p.l, event.UseName)
	if err != nil {
		p.l.Error("设置静默失败：构建匹配器错误", zap.Error(err))
		return err
	}

	// 创建 Silence 对象
	silence := types.Silence{
		Matchers:  matchers,
		StartsAt:  time.Now(),
		EndsAt:    time.Now().Add(time.Duration(duration)),
		CreatedBy: user.RealName,
		Comment:   fmt.Sprintf("eventId: %v 操作人: %v 静默时间: %v", alertEvent.ID, user.RealName, duration),
	}

	// 序列化 Silence 对象为 JSON
	silenceData, err := json.Marshal(silence)
	if err != nil {
		p.l.Error("设置静默失败：序列化 Silence 对象失败", zap.Error(err))
		return fmt.Errorf("序列化失败: %v", err)
	}

	// 获取 AlertManager 地址
	alertPool, err := p.dao.GetAlertPoolByID(ctx, alertEvent.SendGroup.PoolID)
	if err != nil {
		p.l.Error("设置静默失败：无法获取 AlertPool", zap.Error(err))
		return err
	}

	if len(alertPool.AlertManagerInstances) == 0 {
		p.l.Error("设置静默失败：AlertManager 实例为空", zap.Int("poolID", alertPool.ID))
		return fmt.Errorf("AlertManager 实例为空")
	}

	alertAddr := fmt.Sprintf("http://%v:9093", alertPool.AlertManagerInstances[0])
	alertUrl := fmt.Sprintf("%s/api/v1/silences", alertAddr)

	// 发送 Silence 请求到 AlertManager
	silenceID, err := pkg.SendSilenceRequest(ctx, p.l, alertUrl, silenceData)
	if err != nil {
		p.l.Error("设置静默失败：发送 Silence 请求失败", zap.Error(err))
		return fmt.Errorf("发送 Silence 请求失败: %v", err)
	}

	// 更新 AlertEvent 状态为已静默
	alertEvent.Status = "已屏蔽"
	alertEvent.SilenceID = silenceID
	if err := p.dao.UpdateAlertEvent(ctx, alertEvent); err != nil {
		p.l.Error("设置静默失败：更新 AlertEvent 失败", zap.Error(err), zap.Int("id", id))
		return fmt.Errorf("更新 AlertEvent 失败: %v", err)
	}

	p.l.Info("设置静默成功", zap.Int("id", id), zap.String("silenceID", silenceID))
	return nil
}

// EventAlertClaim 认领告警事件
func (p *prometheusService) EventAlertClaim(ctx context.Context, id int, userId int) error {
	// 获取告警事件
	event, err := p.dao.GetMonitorAlertEventById(ctx, id)
	if err != nil {
		p.l.Error("认领告警事件失败：获取告警事件时出错", zap.Error(err))
		return err
	}

	// 更新认领用户
	event.RenLingUserID = userId

	// 更新数据库
	if err := p.dao.EventAlertClaim(ctx, event); err != nil {
		p.l.Error("认领告警事件失败：更新告警事件时出错", zap.Error(err))
		return err
	}

	p.l.Info("认领告警事件成功", zap.Int("id", id), zap.Int("userId", userId))
	return nil
}

// BatchEventAlertSilence 批量设置告警事件为静默状态
func (p *prometheusService) BatchEventAlertSilence(ctx context.Context, request *model.BatchEventAlertSilenceRequest, userId int) error {
	// 输入验证
	if request == nil || len(request.IDs) == 0 {
		p.l.Error("批量设置静默失败：未提供事件ID")
		return fmt.Errorf("未提供事件ID")
	}

	// 获取用户信息
	user, err := p.userDao.GetUserByID(ctx, userId)
	if err != nil {
		p.l.Error("批量设置静默失败：无效的 userId", zap.Int("userId", userId), zap.Error(err))
		return fmt.Errorf("无效的 userId: %d", userId)
	}

	// 解析持续时间
	duration, err := promModel.ParseDuration(request.Time)
	if err != nil {
		p.l.Error("批量设置静默失败：解析持续时间错误", zap.Error(err))
		return fmt.Errorf("无效的持续时间: %v", err)
	}

	// 初始化等待组和错误收集
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs []error

	// 定义信号量以限制并发数量（例如，最多 10 个并发 goroutine）
	sem := make(chan struct{}, 10)

	for _, id := range request.IDs {
		if id <= 0 {
			p.l.Error("批量设置静默跳过：无效的 ID", zap.Int("id", id))
			mu.Lock()
			errs = append(errs, fmt.Errorf("无效的 ID: %d", id))
			mu.Unlock()
			continue
		}

		wg.Add(1)
		sem <- struct{}{} // 获取信号量
		go func(eventID int) {
			defer wg.Done()
			defer func() { <-sem }() // 释放信号量

			// 获取 AlertEvent
			alertEvent, err := p.dao.GetAlertEventByID(ctx, eventID)
			if err != nil {
				p.l.Error("批量设置静默失败：无法获取 AlertEvent", zap.Error(err), zap.Int("id", eventID))
				mu.Lock()
				errs = append(errs, fmt.Errorf("事件 ID %d: %v", eventID, err))
				mu.Unlock()
				return
			}

			// 构建匹配器
			matchers, err := pkg.BuildMatchers(alertEvent, p.l, request.UseName)
			if err != nil {
				p.l.Error("批量设置静默失败：构建匹配器错误", zap.Error(err), zap.Int("id", eventID))
				mu.Lock()
				errs = append(errs, fmt.Errorf("事件 ID %d: %v", eventID, err))
				mu.Unlock()
				return
			}

			// 创建 Silence 对象
			silence := types.Silence{
				Matchers:  matchers,
				StartsAt:  time.Now(),
				EndsAt:    time.Now().Add(time.Duration(duration)),
				CreatedBy: user.RealName,
				Comment:   fmt.Sprintf("eventId: %v 操作人: %v 静默时间: %v", alertEvent.ID, user.RealName, duration),
			}

			// 序列化 Silence 对象为 JSON
			silenceData, err := json.Marshal(silence)
			if err != nil {
				p.l.Error("批量设置静默失败：序列化 Silence 对象失败", zap.Error(err), zap.Int("id", eventID))
				mu.Lock()
				errs = append(errs, fmt.Errorf("事件 ID %d: %v", eventID, err))
				mu.Unlock()
				return
			}

			// 获取 AlertManager 地址
			alertPool, err := p.dao.GetAlertPoolByID(ctx, alertEvent.SendGroup.PoolID)
			if err != nil {
				p.l.Error("批量设置静默失败：无法获取 AlertPool", zap.Error(err), zap.Int("id", eventID))
				mu.Lock()
				errs = append(errs, fmt.Errorf("事件 ID %d: %v", eventID, err))
				mu.Unlock()
				return
			}

			if len(alertPool.AlertManagerInstances) == 0 {
				p.l.Error("批量设置静默失败：AlertManager 实例为空", zap.Int("poolID", alertPool.ID), zap.Int("id", eventID))
				mu.Lock()
				errs = append(errs, fmt.Errorf("事件 ID %d: AlertManager 实例为空", eventID))
				mu.Unlock()
				return
			}

			alertAddr := fmt.Sprintf("http://%v:9093", alertPool.AlertManagerInstances[0])
			alertUrl := fmt.Sprintf("%s/api/v1/silences", alertAddr)

			// 发送 Silence 请求到 AlertManager
			silenceID, err := pkg.SendSilenceRequest(ctx, p.l, alertUrl, silenceData)
			if err != nil {
				p.l.Error("批量设置静默失败：发送 Silence 请求失败", zap.Error(err), zap.Int("id", eventID))
				mu.Lock()
				errs = append(errs, fmt.Errorf("事件 ID %d: %v", eventID, err))
				mu.Unlock()
				return
			}

			// 更新 AlertEvent 状态为已静默
			alertEvent.Status = "已屏蔽"
			alertEvent.SilenceID = silenceID
			if err := p.dao.UpdateAlertEvent(ctx, alertEvent); err != nil {
				p.l.Error("批量设置静默失败：更新 AlertEvent 失败", zap.Error(err), zap.Int("id", eventID))
				mu.Lock()
				errs = append(errs, fmt.Errorf("事件 ID %d: %v", eventID, err))
				mu.Unlock()
				return
			}

			p.l.Info("批量设置静默成功", zap.Int("id", eventID), zap.String("silenceID", silenceID))
		}(id)
	}

	// 等待所有 goroutine 完成
	wg.Wait()

	// 关闭信号量通道
	close(sem)

	if len(errs) > 0 {
		// 聚合错误
		errMsg := "批量设置静默过程中遇到以下错误："
		for _, e := range errs {
			errMsg += "\n" + e.Error()
		}
		p.l.Error(errMsg)
		return fmt.Errorf(errMsg)
	}

	p.l.Info("批量设置静默成功处理所有事件")
	return nil
}

// GetMonitorRecordRuleList 获取记录规则列表，支持按名称搜索
func (p *prometheusService) GetMonitorRecordRuleList(ctx context.Context, searchName *string) ([]*model.MonitorRecordRule, error) {
	return pkg.HandleList(ctx, searchName,
		p.dao.SearchMonitorRecordRuleByName,
		p.dao.GetMonitorRecordRuleList)
}

// CreateMonitorRecordRule 创建记录规则
func (p *prometheusService) CreateMonitorRecordRule(ctx context.Context, monitorRecordRule *model.MonitorRecordRule) error {
	// 检查记录规则是否已存在
	exists, err := p.dao.CheckMonitorRecordRuleExists(ctx, monitorRecordRule)
	if err != nil {
		p.l.Error("创建记录规则失败：检查记录规则是否存在时出错", zap.Error(err))
		return err
	}

	if exists {
		return errors.New("记录规则已存在")
	}

	// 创建记录规则
	if err := p.dao.CreateMonitorRecordRule(ctx, monitorRecordRule); err != nil {
		p.l.Error("创建记录规则失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := p.cache.MonitorCacheManager(ctx); err != nil {
		p.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	return nil
}

// UpdateMonitorRecordRule 更新记录规则
func (p *prometheusService) UpdateMonitorRecordRule(ctx context.Context, monitorRecordRule *model.MonitorRecordRule) error {
	// 更新记录规则
	if err := p.dao.UpdateMonitorRecordRule(ctx, monitorRecordRule); err != nil {
		p.l.Error("更新记录规则失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := p.cache.MonitorCacheManager(ctx); err != nil {
		p.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	return nil
}

// DeleteMonitorRecordRule 删除记录规则
func (p *prometheusService) DeleteMonitorRecordRule(ctx context.Context, id int) error {
	// 删除记录规则
	if err := p.dao.DeleteMonitorRecordRule(ctx, id); err != nil {
		p.l.Error("删除记录规则失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := p.cache.MonitorCacheManager(ctx); err != nil {
		p.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	return nil
}

// BatchDeleteMonitorRecordRule 批量删除记录规则
func (p *prometheusService) BatchDeleteMonitorRecordRule(ctx context.Context, ids []int) error {
	for _, id := range ids {
		if err := p.DeleteMonitorRecordRule(ctx, id); err != nil {
			// 记录错误但继续删除其他规则
			p.l.Error("批量删除记录规则失败", zap.Int("id", id), zap.Error(err))
			return fmt.Errorf("删除记录规则 ID %d 失败: %v", id, err)
		}
	}

	return nil
}

// EnableSwitchMonitorRecordRule 切换记录规则的启用状态
func (p *prometheusService) EnableSwitchMonitorRecordRule(ctx context.Context, id int) error {
	if err := p.dao.EnableSwitchMonitorRecordRule(ctx, id); err != nil {
		p.l.Error("切换记录规则状态失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := p.cache.MonitorCacheManager(ctx); err != nil {
		p.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	return nil
}

// BatchEnableSwitchMonitorRecordRule 批量切换记录规则的启用状态
func (p *prometheusService) BatchEnableSwitchMonitorRecordRule(ctx context.Context, ids []int) error {
	for _, id := range ids {
		if err := p.EnableSwitchMonitorRecordRule(ctx, id); err != nil {
			p.l.Error("批量切换记录规则状态失败", zap.Int("id", id), zap.Error(err))
			return fmt.Errorf("切换记录规则 ID %d 状态失败: %v", id, err)
		}
	}
	return nil
}
