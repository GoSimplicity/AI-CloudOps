package api

import (
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils/apiresponse"
	ijwt "github.com/GoSimplicity/AI-CloudOps/pkg/utils/jwt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"strconv"
)

type PrometheusHandler struct {
	service service.PrometheusService
	l       *zap.Logger
}

func NewPrometheusHandler(service service.PrometheusService, l *zap.Logger) *PrometheusHandler {
	return &PrometheusHandler{
		service: service,
		l:       l,
	}
}

func (p *PrometheusHandler) RegisterRouters(server *gin.Engine) {
	monitorGroup := server.Group("/api/monitor")
	{
		// 采集池相关路由
		scrapePools := monitorGroup.Group("/scrape_pools")
		{
			scrapePools.GET("/", p.GetMonitorScrapePoolList)       // 获取监控采集池列表
			scrapePools.POST("/create", p.CreateMonitorScrapePool) // 创建监控采集池
			scrapePools.POST("/update", p.UpdateMonitorScrapePool) // 更新监控采集池
			scrapePools.DELETE("/:id", p.DeleteMonitorScrapePool)  // 删除监控采集池
		}

		// 采集 Job 相关路由
		scrapeJobs := monitorGroup.Group("/scrape_jobs")
		{
			scrapeJobs.GET("/", p.GetMonitorScrapeJobList)       // 获取监控采集 Job 列表
			scrapeJobs.POST("/create", p.CreateMonitorScrapeJob) // 创建监控采集 Job
			scrapeJobs.POST("/update", p.UpdateMonitorScrapeJob) // 更新监控采集 Job
			scrapeJobs.DELETE("/:id", p.DeleteMonitorScrapeJob)  // 删除监控采集 Job
		}

		// Prometheus 配置相关路由
		prometheusConfigs := monitorGroup.Group("/prometheus_configs")
		{
			prometheusConfigs.GET("/prometheus", p.GetMonitorPrometheusYaml)                // 获取单个 Prometheus 配置文件
			prometheusConfigs.GET("/prometheus_alert", p.GetMonitorPrometheusAlertRuleYaml) // 获取单个 Prometheus 告警配置文件
			prometheusConfigs.GET("/prometheus_record", p.GetMonitorPrometheusRecordYaml)   // 获取单个 Prometheus 记录配置文件
			prometheusConfigs.GET("/alertManager", p.GetMonitorAlertManagerYaml)            // 获取单个 AlertManager 配置文件
		}

		// 值班组相关路由
		onDutyGroups := monitorGroup.Group("/onDuty_groups")
		{
			onDutyGroups.GET("/", p.GetMonitorOnDutyGroupList)                      // 获取值班组列表
			onDutyGroups.POST("/create", p.CreateMonitorOnDutyGroup)                // 创建新的值班组
			onDutyGroups.POST("/changes", p.CreateMonitorOnDutyGroupChange)         // 创建值班组的换班记录
			onDutyGroups.POST("/update", p.UpdateMonitorOnDutyGroup)                // 更新值班组信息
			onDutyGroups.DELETE("/:id", p.DeleteMonitorOnDutyGroup)                 // 删除指定的值班组
			onDutyGroups.GET("/:id", p.GetMonitorOnDutyGroup)                       // 获取指定的值班组信息
			onDutyGroups.GET("/:id/future_plan", p.GetMonitorOnDutyGroupFuturePlan) // 获取指定值班组的未来值班计划
		}

		// AlertManager 集群相关路由
		alertManagerPools := monitorGroup.Group("/alertManager_pools")
		{
			alertManagerPools.GET("/", p.GetMonitorAlertManagerPoolList)       // 获取 AlertManager 集群池列表
			alertManagerPools.POST("/create", p.CreateMonitorAlertManagerPool) // 创建新的 AlertManager 集群池
			alertManagerPools.POST("/update", p.UpdateMonitorAlertManagerPool) // 更新现有的 AlertManager 集群池
			alertManagerPools.DELETE("/:id", p.DeleteMonitorAlertManagerPool)  // 删除指定的 AlertManager 集群池
		}

		// 发送组相关路由
		sendGroups := monitorGroup.Group("/send_groups")
		{
			sendGroups.GET("/", p.GetMonitorSendGroupList)       // 获取发送组列表
			sendGroups.POST("/create", p.CreateMonitorSendGroup) // 创建新的发送组
			sendGroups.POST("/update", p.UpdateMonitorSendGroup) // 更新现有的发送组
			sendGroups.DELETE("/:id", p.DeleteMonitorSendGroup)  // 删除指定的发送组
		}

		// 告警规则相关路由
		alertRules := monitorGroup.Group("/alert_rules")
		{
			alertRules.GET("/", p.GetMonitorAlertRuleList)                  // 获取告警规则列表
			alertRules.POST("/promql_check", p.PromqlExprCheck)             // 检查 PromQL 表达式的合法性
			alertRules.POST("/create", p.CreateMonitorAlertRule)            // 创建新的告警规则
			alertRules.POST("/update", p.UpdateMonitorAlertRule)            // 更新现有的告警规则
			alertRules.POST("/:id/enable", p.EnableSwitchMonitorAlertRule)  // 切换告警规则的启用状态
			alertRules.POST("/enable", p.BatchEnableSwitchMonitorAlertRule) // 批量切换告警规则的启用状态
			alertRules.DELETE("/:id", p.DeleteMonitorAlertRule)             // 删除指定的告警规则
			alertRules.DELETE("/", p.BatchDeleteMonitorAlertRule)           // 批量删除告警规则
		}

		// 告警事件相关路由
		alertEvents := monitorGroup.Group("/alert_events")
		{
			alertEvents.GET("/", p.GetMonitorAlertEventList)          // 获取告警事件列表
			alertEvents.POST("/:id/silence", p.EventAlertSilence)     // 将指定告警事件设置为静默状态
			alertEvents.POST("/:id/claim", p.EventAlertClaim)         // 认领指定的告警事件
			alertEvents.POST("/:id/unSilence", p.EventAlertUnSilence) // 取消指定告警事件的静默状态
			alertEvents.POST("/silence", p.BatchEventAlertSilence)    // 批量设置告警事件为静默状态
		}

		// 预聚合规则相关路由
		recordRules := monitorGroup.Group("/record_rules")
		{
			recordRules.GET("/", p.GetMonitorRecordRuleList)                  // 获取预聚合规则列表
			recordRules.POST("/create", p.CreateMonitorRecordRule)            // 创建新的预聚合规则
			recordRules.PUT("/update", p.UpdateMonitorRecordRule)             // 更新现有的预聚合规则
			recordRules.DELETE("/:id", p.DeleteMonitorRecordRule)             // 删除指定的预聚合规则
			recordRules.DELETE("/", p.BatchDeleteMonitorRecordRule)           // 批量删除预聚合规则
			recordRules.POST("/:id/enable", p.EnableSwitchMonitorRecordRule)  // 切换预聚合规则的启用状态
			recordRules.POST("/enable", p.BatchEnableSwitchMonitorRecordRule) // 批量切换预聚合规则的启用状态
		}
	}
}

// GetMonitorScrapePoolList 获取监控采集池列表
func (p *PrometheusHandler) GetMonitorScrapePoolList(ctx *gin.Context) {
	search := ctx.Query("search")

	list, err := p.service.GetMonitorScrapePoolList(ctx, &search)
	if err != nil {
		apiresponse.ErrorWithDetails(ctx, err, "获取监控采集池列表失败")
		return
	}

	apiresponse.SuccessWithData(ctx, list)
}

// CreateMonitorScrapePool 创建监控采集池
func (p *PrometheusHandler) CreateMonitorScrapePool(ctx *gin.Context) {
	var monitorScrapePool *model.MonitorScrapePool

	uc := ctx.MustGet("user").(ijwt.UserClaims)
	err := ctx.ShouldBind(&monitorScrapePool)
	if err != nil {
		apiresponse.ErrorWithDetails(ctx, err, "参数错误")
		return
	}

	monitorScrapePool.UserID = uc.Uid
	if err := p.service.CreateMonitorScrapePool(ctx, monitorScrapePool); err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// UpdateMonitorScrapePool 更新监控采集池
func (p *PrometheusHandler) UpdateMonitorScrapePool(ctx *gin.Context) {
	var monitorScrapePool *model.MonitorScrapePool

	err := ctx.ShouldBind(&monitorScrapePool)
	if err != nil {
		apiresponse.ErrorWithDetails(ctx, err, "参数错误")
		return
	}

	if err := p.service.UpdateMonitorScrapePool(ctx, monitorScrapePool); err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// DeleteMonitorScrapePool 删除监控采集池
func (p *PrometheusHandler) DeleteMonitorScrapePool(ctx *gin.Context) {
	id := ctx.Query("id")
	atom, err := strconv.Atoi(id)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, "参数错误")
		return
	}

	if err := p.service.DeleteMonitorScrapePool(ctx, atom); err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// GetMonitorScrapeJobList 获取监控采集 Job 列表
func (p *PrometheusHandler) GetMonitorScrapeJobList(ctx *gin.Context) {
	search := ctx.Query("search")
	list, err := p.service.GetMonitorScrapeJobList(ctx, &search)
	if err != nil {
		apiresponse.ErrorWithDetails(ctx, err, "获取监控采集 Job 列表失败")
		return
	}

	apiresponse.SuccessWithData(ctx, list)
}

// CreateMonitorScrapeJob 创建监控采集 Job
func (p *PrometheusHandler) CreateMonitorScrapeJob(ctx *gin.Context) {
	var monitorScrapeJob *model.MonitorScrapeJob

	uc := ctx.MustGet("user").(ijwt.UserClaims)
	err := ctx.ShouldBind(&monitorScrapeJob)
	if err != nil {
		apiresponse.ErrorWithDetails(ctx, err, "参数错误")
		return
	}

	monitorScrapeJob.UserID = uc.Uid

	if err := p.service.CreateMonitorScrapeJob(ctx, monitorScrapeJob); err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// UpdateMonitorScrapeJob 更新监控采集 Job
func (p *PrometheusHandler) UpdateMonitorScrapeJob(ctx *gin.Context) {
	var monitorScrapeJob *model.MonitorScrapeJob

	err := ctx.ShouldBind(&monitorScrapeJob)
	if err != nil {
		apiresponse.ErrorWithDetails(ctx, err, "参数错误")
		return
	}

	if err := p.service.UpdateMonitorScrapeJob(ctx, monitorScrapeJob); err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

func (p *PrometheusHandler) DeleteMonitorScrapeJob(ctx *gin.Context) {
	id := ctx.Query("id")
	atom, err := strconv.Atoi(id)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, "参数错误")
		return
	}

	if err := p.service.DeleteMonitorScrapeJob(ctx, atom); err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// GetMonitorPrometheusYaml 获取单个 Prometheus 配置文件
func (p *PrometheusHandler) GetMonitorPrometheusYaml(ctx *gin.Context) {
	ip := ctx.Query("ip")

	yaml := p.service.GetMonitorPrometheusYaml(ctx, ip)
	if yaml == "" {
		apiresponse.ErrorWithMessage(ctx, "获取 Prometheus 配置文件失败")
		return
	}

	apiresponse.SuccessWithData(ctx, yaml)
}

// GetMonitorPrometheusAlertRuleYaml 获取单个 Prometheus 告警配置规则文件
func (p *PrometheusHandler) GetMonitorPrometheusAlertRuleYaml(ctx *gin.Context) {
	ip := ctx.Query("ip")

	yaml := p.service.GetMonitorPrometheusAlertRuleYaml(ctx, ip)
	if yaml == "" {
		apiresponse.ErrorWithMessage(ctx, "获取 Prometheus 告警配置文件失败")
		return
	}

	apiresponse.SuccessWithData(ctx, yaml)
}

// GetMonitorPrometheusRecordYaml 获取单个 Prometheus 记录配置文件
func (p *PrometheusHandler) GetMonitorPrometheusRecordYaml(ctx *gin.Context) {
	ip := ctx.Query("ip")

	yaml := p.service.GetMonitorPrometheusRecordYaml(ctx, ip)
	if yaml == "" {
		apiresponse.ErrorWithMessage(ctx, "获取 Prometheus 记录配置文件失败")
		return
	}
	apiresponse.SuccessWithData(ctx, yaml)
	// TODO: 实现获取单个 Prometheus 记录配置文件的逻辑
}

// GetMonitorAlertManagerYaml 获取单个 AlertManager 配置文件
func (p *PrometheusHandler) GetMonitorAlertManagerYaml(ctx *gin.Context) {
	ip := ctx.Query("ip")

	yaml := p.service.GetMonitorAlertManagerYaml(ctx, ip)
	if yaml == "" {
		apiresponse.ErrorWithMessage(ctx, "获取 AlertManager 配置文件失败")
		return
	}

	apiresponse.SuccessWithData(ctx, yaml)
}

// GetMonitorOnDutyGroupList 获取值班组列表
func (p *PrometheusHandler) GetMonitorOnDutyGroupList(ctx *gin.Context) {
	// TODO: 实现获取值班组列表的逻辑
}

// CreateMonitorOnDutyGroup 创建新的值班组
func (p *PrometheusHandler) CreateMonitorOnDutyGroup(ctx *gin.Context) {
	// TODO: 实现创建新的值班组的逻辑
}

// CreateMonitorOnDutyGroupChange 创建值班组的换班记录
func (p *PrometheusHandler) CreateMonitorOnDutyGroupChange(ctx *gin.Context) {
	// TODO: 实现创建值班组的换班记录的逻辑
}

// UpdateMonitorOnDutyGroup 更新值班组信息
func (p *PrometheusHandler) UpdateMonitorOnDutyGroup(ctx *gin.Context) {
	// TODO: 实现更新值班组信息的逻辑
}

// DeleteMonitorOnDutyGroup 删除指定的值班组
func (p *PrometheusHandler) DeleteMonitorOnDutyGroup(ctx *gin.Context) {
	// TODO: 实现删除指定的值班组的逻辑
}

// GetMonitorOnDutyGroup 获取指定的值班组信息
func (p *PrometheusHandler) GetMonitorOnDutyGroup(ctx *gin.Context) {
	// TODO: 实现获取指定的值班组信息的逻辑
}

// GetMonitorOnDutyGroupFuturePlan 获取指定值班组的未来值班计划
func (p *PrometheusHandler) GetMonitorOnDutyGroupFuturePlan(ctx *gin.Context) {
	// TODO: 实现获取指定值班组的未来值班计划的逻辑
}

// GetMonitorAlertManagerPoolList 获取 AlertManager 集群池列表
func (p *PrometheusHandler) GetMonitorAlertManagerPoolList(ctx *gin.Context) {
	// TODO: 实现获取 AlertManager 集群池列表的逻辑
}

// CreateMonitorAlertManagerPool 创建新的 AlertManager 集群池
func (p *PrometheusHandler) CreateMonitorAlertManagerPool(ctx *gin.Context) {
	// TODO: 实现创建新的 AlertManager 集群池的逻辑
}

// UpdateMonitorAlertManagerPool 更新现有的 AlertManager 集群池
func (p *PrometheusHandler) UpdateMonitorAlertManagerPool(ctx *gin.Context) {
	// TODO: 实现更新现有的 AlertManager 集群池的逻辑
}

// DeleteMonitorAlertManagerPool 删除指定的 AlertManager 集群池
func (p *PrometheusHandler) DeleteMonitorAlertManagerPool(ctx *gin.Context) {
	// TODO: 实现删除指定的 AlertManager 集群池的逻辑
}

// GetMonitorSendGroupList 获取发送组列表
func (p *PrometheusHandler) GetMonitorSendGroupList(ctx *gin.Context) {
	// TODO: 实现获取发送组列表的逻辑
}

// CreateMonitorSendGroup 创建新的发送组
func (p *PrometheusHandler) CreateMonitorSendGroup(ctx *gin.Context) {
	// TODO: 实现创建新的发送组的逻辑
}

// UpdateMonitorSendGroup 更新现有的发送组
func (p *PrometheusHandler) UpdateMonitorSendGroup(ctx *gin.Context) {
	// TODO: 实现更新现有的发送组的逻辑
}

// DeleteMonitorSendGroup 删除指定的发送组
func (p *PrometheusHandler) DeleteMonitorSendGroup(ctx *gin.Context) {
	// TODO: 实现删除指定的发送组的逻辑
}

// GetMonitorAlertRuleList 获取告警规则列表
func (p *PrometheusHandler) GetMonitorAlertRuleList(ctx *gin.Context) {
	// TODO: 实现获取告警规则列表的逻辑
}

// PromqlExprCheck 检查 PromQL 表达式的合法性
func (p *PrometheusHandler) PromqlExprCheck(ctx *gin.Context) {
	// TODO: 实现检查 PromQL 表达式的合法性的逻辑
}

// CreateMonitorAlertRule 创建新的告警规则
func (p *PrometheusHandler) CreateMonitorAlertRule(ctx *gin.Context) {
	// TODO: 实现创建新的告警规则的逻辑
}

// UpdateMonitorAlertRule 更新现有的告警规则
func (p *PrometheusHandler) UpdateMonitorAlertRule(ctx *gin.Context) {
	// TODO: 实现更新现有的告警规则的逻辑
}

// EnableSwitchMonitorAlertRule 切换告警规则的启用状态
func (p *PrometheusHandler) EnableSwitchMonitorAlertRule(ctx *gin.Context) {
	// TODO: 实现切换告警规则的启用状态的逻辑
}

// BatchEnableSwitchMonitorAlertRule 批量切换告警规则的启用状态
func (p *PrometheusHandler) BatchEnableSwitchMonitorAlertRule(ctx *gin.Context) {
	// TODO: 实现批量切换告警规则的启用状态的逻辑
}

// DeleteMonitorAlertRule 删除指定的告警规则
func (p *PrometheusHandler) DeleteMonitorAlertRule(ctx *gin.Context) {
	// TODO: 实现删除指定的告警规则的逻辑
}

// BatchDeleteMonitorAlertRule 批量删除告警规则
func (p *PrometheusHandler) BatchDeleteMonitorAlertRule(ctx *gin.Context) {
	// TODO: 实现批量删除告警规则的逻辑
}

// GetMonitorAlertEventList 获取告警事件列表
func (p *PrometheusHandler) GetMonitorAlertEventList(ctx *gin.Context) {
	// TODO: 实现获取告警事件列表的逻辑
}

// EventAlertSilence 将指定告警事件设置为静默状态
func (p *PrometheusHandler) EventAlertSilence(ctx *gin.Context) {
	// TODO: 实现将指定告警事件设置为静默状态的逻辑
}

// EventAlertClaim 认领指定的告警事件
func (p *PrometheusHandler) EventAlertClaim(ctx *gin.Context) {
	// TODO: 实现认领指定的告警事件的逻辑
}

// EventAlertUnSilence 取消指定告警事件的静默状态
func (p *PrometheusHandler) EventAlertUnSilence(ctx *gin.Context) {
	// TODO: 实现取消指定告警事件的静默状态的逻辑
}

// BatchEventAlertSilence 批量设置告警事件为静默状态
func (p *PrometheusHandler) BatchEventAlertSilence(ctx *gin.Context) {
	// TODO: 实现批量设置告警事件为静默状态的逻辑
}

// GetMonitorRecordRuleList 获取预聚合规则列表
func (p *PrometheusHandler) GetMonitorRecordRuleList(ctx *gin.Context) {
	// TODO: 实现获取预聚合规则列表的逻辑
}

// CreateMonitorRecordRule 创建新的预聚合规则
func (p *PrometheusHandler) CreateMonitorRecordRule(ctx *gin.Context) {
	// TODO: 实现创建新的预聚合规则的逻辑
}

// UpdateMonitorRecordRule 更新现有的预聚合规则
func (p *PrometheusHandler) UpdateMonitorRecordRule(ctx *gin.Context) {
	// TODO: 实现更新现有的预聚合规则的逻辑
}

// DeleteMonitorRecordRule 删除指定的预聚合规则
func (p *PrometheusHandler) DeleteMonitorRecordRule(ctx *gin.Context) {
	// TODO: 实现删除指定的预聚合规则的逻辑
}

// BatchDeleteMonitorRecordRule 批量删除预聚合规则
func (p *PrometheusHandler) BatchDeleteMonitorRecordRule(ctx *gin.Context) {
	// TODO: 实现批量删除预聚合规则的逻辑
}

// EnableSwitchMonitorRecordRule 切换预聚合规则的启用状态
func (p *PrometheusHandler) EnableSwitchMonitorRecordRule(ctx *gin.Context) {
	// TODO: 实现切换预聚合规则的启用状态的逻辑
}

// BatchEnableSwitchMonitorRecordRule 批量切换预聚合规则的启用状态
func (p *PrometheusHandler) BatchEnableSwitchMonitorRecordRule(ctx *gin.Context) {
	// TODO: 实现批量切换预聚合规则的启用状态的逻辑
}
