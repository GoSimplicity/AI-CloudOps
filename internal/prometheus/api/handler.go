package api

import (
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	alertEventService "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/service/alert/event"
	alertOnDutyService "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/service/alert/onduty"
	alertPoolService "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/service/alert/pool"
	alertRecordService "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/service/alert/record"
	alertRuleService "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/service/alert/rule"
	alertSendService "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/service/alert/send"
	scrapeJobService "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/service/scrape/job"
	scrapePoolService "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/service/scrape/pool"
	yamlService "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/service/yaml"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils/apiresponse"
	ijwt "github.com/GoSimplicity/AI-CloudOps/pkg/utils/jwt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"net/http"
	"strconv"
)

type PrometheusHandler struct {
	alertEventService  alertEventService.AlertManagerEventService
	alertOnDutyService alertOnDutyService.AlertManagerOnDutyService
	alertPoolService   alertPoolService.AlertManagerPoolService
	alertRecordService alertRecordService.AlertManagerRecordService
	alertRuleService   alertRuleService.AlertManagerRuleService
	alertSendService   alertSendService.AlertManagerSendService
	scrapeJobService   scrapeJobService.ScrapeJobService
	scrapePoolService  scrapePoolService.ScrapePoolService
	yamlService        yamlService.ConfigYamlService
	l                  *zap.Logger
}

func NewPrometheusHandler(
	l *zap.Logger,
	alertEventService alertEventService.AlertManagerEventService,
	alertOnDutyService alertOnDutyService.AlertManagerOnDutyService,
	alertPoolService alertPoolService.AlertManagerPoolService,
	alertRecordService alertRecordService.AlertManagerRecordService,
	alertRuleService alertRuleService.AlertManagerRuleService,
	alertSendService alertSendService.AlertManagerSendService,
	scrapeJobService scrapeJobService.ScrapeJobService,
	scrapePoolService scrapePoolService.ScrapePoolService,
	yamlService yamlService.ConfigYamlService,
) *PrometheusHandler {
	return &PrometheusHandler{
		l:                  l,
		alertEventService:  alertEventService,
		alertOnDutyService: alertOnDutyService,
		alertPoolService:   alertPoolService,
		alertRecordService: alertRecordService,
		alertRuleService:   alertRuleService,
		alertSendService:   alertSendService,
		scrapeJobService:   scrapeJobService,
		scrapePoolService:  scrapePoolService,
		yamlService:        yamlService,
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

		// AlertManager 集群相关路由
		alertManagerPools := monitorGroup.Group("/alertManager_pools")
		{
			alertManagerPools.GET("/", p.GetMonitorAlertManagerPoolList)       // 获取 AlertManager 集群池列表
			alertManagerPools.POST("/create", p.CreateMonitorAlertManagerPool) // 创建新的 AlertManager 集群池
			alertManagerPools.POST("/update", p.UpdateMonitorAlertManagerPool) // 更新现有的 AlertManager 集群池
			alertManagerPools.DELETE("/:id", p.DeleteMonitorAlertManagerPool)  // 删除指定的 AlertManager 集群池
		}

		// 告警规则相关路由
		alertRules := monitorGroup.Group("/alert_rules")
		{
			alertRules.GET("/", p.GetMonitorAlertRuleList)                        // 获取告警规则列表
			alertRules.POST("/promql_check", p.PromqlExprCheck)                   // 检查 PromQL 表达式的合法性
			alertRules.POST("/create", p.CreateMonitorAlertRule)                  // 创建新的告警规则
			alertRules.POST("/update", p.UpdateMonitorAlertRule)                  // 更新现有的告警规则
			alertRules.POST("/enable", p.EnableSwitchMonitorAlertRule)            // 切换告警规则的启用状态
			alertRules.POST("/batch_enable", p.BatchEnableSwitchMonitorAlertRule) // 批量切换告警规则的启用状态
			alertRules.DELETE("/:id", p.DeleteMonitorAlertRule)                   // 删除指定的告警规则
			alertRules.DELETE("/", p.BatchDeleteMonitorAlertRule)                 // 批量删除告警规则
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
			recordRules.POST("/update", p.UpdateMonitorRecordRule)            // 更新现有的预聚合规则
			recordRules.DELETE("/:id", p.DeleteMonitorRecordRule)             // 删除指定的预聚合规则
			recordRules.DELETE("/", p.BatchDeleteMonitorRecordRule)           // 批量删除预聚合规则
			recordRules.POST("/:id/enable", p.EnableSwitchMonitorRecordRule)  // 切换预聚合规则的启用状态
			recordRules.POST("/enable", p.BatchEnableSwitchMonitorRecordRule) // 批量切换预聚合规则的启用状态
		}

		// 值班组相关路由
		onDutyGroups := monitorGroup.Group("/onDuty_groups")
		{
			onDutyGroups.GET("/list", p.GetMonitorOnDutyGroupList)               // 获取值班组列表
			onDutyGroups.POST("/create", p.CreateMonitorOnDutyGroup)             // 创建新的值班组
			onDutyGroups.POST("/changes", p.CreateMonitorOnDutyGroupChange)      // 创建值班组的换班记录
			onDutyGroups.POST("/update", p.UpdateMonitorOnDutyGroup)             // 更新值班组信息
			onDutyGroups.DELETE("/:id", p.DeleteMonitorOnDutyGroup)              // 删除指定的值班组
			onDutyGroups.GET("/:id", p.GetMonitorOnDutyGroup)                    // 获取指定的值班组信息
			onDutyGroups.POST("/future_plan", p.GetMonitorOnDutyGroupFuturePlan) // 获取指定值班组的未来值班计划
		}

		// 发送组相关路由
		sendGroups := monitorGroup.Group("/send_groups")
		{
			sendGroups.GET("/list", p.GetMonitorSendGroupList)   // 获取发送组列表
			sendGroups.POST("/create", p.CreateMonitorSendGroup) // 创建新的发送组
			sendGroups.POST("/update", p.UpdateMonitorSendGroup) // 更新现有的发送组
			sendGroups.DELETE("/:id", p.DeleteMonitorSendGroup)  // 删除指定的发送组
		}

	}
}

// GetMonitorScrapePoolList 获取监控采集池列表
func (p *PrometheusHandler) GetMonitorScrapePoolList(ctx *gin.Context) {
	search := ctx.Query("search")

	list, err := p.scrapePoolService.GetMonitorScrapePoolList(ctx, &search)
	if err != nil {
		apiresponse.ErrorWithDetails(ctx, err, "获取监控采集池列表失败")
		return
	}

	apiresponse.SuccessWithData(ctx, list)
}

// CreateMonitorScrapePool 创建监控采集池
func (p *PrometheusHandler) CreateMonitorScrapePool(ctx *gin.Context) {
	var monitorScrapePool model.MonitorScrapePool

	uc := ctx.MustGet("user").(ijwt.UserClaims)
	if err := ctx.ShouldBind(&monitorScrapePool); err != nil {
		apiresponse.ErrorWithDetails(ctx, err, "参数错误")
		return
	}

	monitorScrapePool.UserID = uc.Uid
	if err := p.scrapePoolService.CreateMonitorScrapePool(ctx, &monitorScrapePool); err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// UpdateMonitorScrapePool 更新监控采集池
func (p *PrometheusHandler) UpdateMonitorScrapePool(ctx *gin.Context) {
	var monitorScrapePool model.MonitorScrapePool

	if err := ctx.ShouldBind(&monitorScrapePool); err != nil {
		apiresponse.ErrorWithDetails(ctx, err, "参数错误")
		return
	}

	if err := p.scrapePoolService.UpdateMonitorScrapePool(ctx, &monitorScrapePool); err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// DeleteMonitorScrapePool 删除监控采集池
func (p *PrometheusHandler) DeleteMonitorScrapePool(ctx *gin.Context) {
	id := ctx.Param("id")
	atom, err := strconv.Atoi(id)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, "参数错误")
		return
	}

	if err := p.scrapePoolService.DeleteMonitorScrapePool(ctx, atom); err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// GetMonitorScrapeJobList 获取监控采集 Job 列表
func (p *PrometheusHandler) GetMonitorScrapeJobList(ctx *gin.Context) {
	search := ctx.Query("search")
	list, err := p.scrapeJobService.GetMonitorScrapeJobList(ctx, &search)
	if err != nil {
		apiresponse.ErrorWithDetails(ctx, err, "获取监控采集 Job 列表失败")
		return
	}

	apiresponse.SuccessWithData(ctx, list)
}

// CreateMonitorScrapeJob 创建监控采集 Job
func (p *PrometheusHandler) CreateMonitorScrapeJob(ctx *gin.Context) {
	var monitorScrapeJob model.MonitorScrapeJob

	uc := ctx.MustGet("user").(ijwt.UserClaims)
	if err := ctx.ShouldBind(&monitorScrapeJob); err != nil {
		apiresponse.ErrorWithDetails(ctx, err, "参数错误")
		return
	}

	monitorScrapeJob.UserID = uc.Uid

	if err := p.scrapeJobService.CreateMonitorScrapeJob(ctx, &monitorScrapeJob); err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// UpdateMonitorScrapeJob 更新监控采集 Job
func (p *PrometheusHandler) UpdateMonitorScrapeJob(ctx *gin.Context) {
	var monitorScrapeJob model.MonitorScrapeJob

	if err := ctx.ShouldBind(&monitorScrapeJob); err != nil {
		apiresponse.ErrorWithDetails(ctx, err, "参数错误")
		return
	}

	if err := p.scrapeJobService.UpdateMonitorScrapeJob(ctx, &monitorScrapeJob); err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// DeleteMonitorScrapeJob 删除监控采集 Job
func (p *PrometheusHandler) DeleteMonitorScrapeJob(ctx *gin.Context) {
	id := ctx.Param("id")

	intId, err := strconv.Atoi(id)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, "参数错误")
		return
	}

	if err := p.scrapeJobService.DeleteMonitorScrapeJob(ctx, intId); err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// GetMonitorPrometheusYaml 获取单个 Prometheus 配置文件
func (p *PrometheusHandler) GetMonitorPrometheusYaml(ctx *gin.Context) {
	ip := ctx.Query("ip")

	yaml := p.yamlService.GetMonitorPrometheusYaml(ctx, ip)
	if yaml == "" {
		apiresponse.ErrorWithMessage(ctx, "获取 Prometheus 配置文件失败")
		return
	}

	ctx.String(http.StatusOK, yaml)
}

// GetMonitorPrometheusAlertRuleYaml 获取单个 Prometheus 告警配置规则文件
func (p *PrometheusHandler) GetMonitorPrometheusAlertRuleYaml(ctx *gin.Context) {
	ip := ctx.Query("ip")

	yaml := p.yamlService.GetMonitorPrometheusAlertRuleYaml(ctx, ip)
	if yaml == "" {
		apiresponse.ErrorWithMessage(ctx, "获取 Prometheus 告警配置文件失败")
		return
	}

	ctx.String(http.StatusOK, yaml)
}

// GetMonitorPrometheusRecordYaml 获取单个 Prometheus 记录配置文件
func (p *PrometheusHandler) GetMonitorPrometheusRecordYaml(ctx *gin.Context) {
	ip := ctx.Query("ip")

	yaml := p.yamlService.GetMonitorPrometheusRecordYaml(ctx, ip)
	if yaml == "" {
		apiresponse.ErrorWithMessage(ctx, "获取 Prometheus 记录配置文件失败")
		return
	}
	ctx.String(http.StatusOK, yaml)
}

// GetMonitorAlertManagerYaml 获取单个 AlertManager 配置文件
func (p *PrometheusHandler) GetMonitorAlertManagerYaml(ctx *gin.Context) {
	ip := ctx.Query("ip")

	yaml := p.yamlService.GetMonitorAlertManagerYaml(ctx, ip)
	if yaml == "" {
		apiresponse.ErrorWithMessage(ctx, "获取 AlertManager 配置文件失败")
		return
	}

	ctx.String(http.StatusOK, yaml)
}

// GetMonitorOnDutyGroupList 获取值班组列表
func (p *PrometheusHandler) GetMonitorOnDutyGroupList(ctx *gin.Context) {
	searchName := ctx.Query("name")

	list, err := p.alertOnDutyService.GetMonitorOnDutyGroupList(ctx, &searchName)
	if err != nil {
		apiresponse.ErrorWithDetails(ctx, err, "获取值班组列表失败")
		return
	}

	apiresponse.SuccessWithData(ctx, list)
}

// CreateMonitorOnDutyGroup 创建新的值班组
func (p *PrometheusHandler) CreateMonitorOnDutyGroup(ctx *gin.Context) {
	var onDutyGroup model.MonitorOnDutyGroup

	uc := ctx.MustGet("user").(ijwt.UserClaims)
	if err := ctx.ShouldBind(&onDutyGroup); err != nil {
		apiresponse.ErrorWithDetails(ctx, err, "参数错误")
		return
	}

	onDutyGroup.UserID = uc.Uid

	if err := p.alertOnDutyService.CreateMonitorOnDutyGroup(ctx, &onDutyGroup); err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// CreateMonitorOnDutyGroupChange 创建值班组的换班记录
func (p *PrometheusHandler) CreateMonitorOnDutyGroupChange(ctx *gin.Context) {
	var onDutyGroupChange model.MonitorOnDutyChange

	uc := ctx.MustGet("user").(ijwt.UserClaims)

	if err := ctx.ShouldBind(&onDutyGroupChange); err != nil {
		apiresponse.ErrorWithDetails(ctx, err, "参数错误")
		return
	}

	onDutyGroupChange.UserID = uc.Uid

	if err := p.alertOnDutyService.CreateMonitorOnDutyGroupChange(ctx, &onDutyGroupChange); err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// UpdateMonitorOnDutyGroup 更新值班组信息
func (p *PrometheusHandler) UpdateMonitorOnDutyGroup(ctx *gin.Context) {
	var onDutyGroup model.MonitorOnDutyGroup

	if err := ctx.ShouldBind(&onDutyGroup); err != nil {
		apiresponse.ErrorWithDetails(ctx, err, "参数错误")
		return
	}

	if err := p.alertOnDutyService.UpdateMonitorOnDutyGroup(ctx, &onDutyGroup); err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// DeleteMonitorOnDutyGroup 删除指定的值班组
func (p *PrometheusHandler) DeleteMonitorOnDutyGroup(ctx *gin.Context) {
	id := ctx.Param("id")
	intId, err := strconv.Atoi(id)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, "参数错误")
		return
	}

	if err := p.alertOnDutyService.DeleteMonitorOnDutyGroup(ctx, intId); err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// GetMonitorOnDutyGroup 获取指定的值班组信息
func (p *PrometheusHandler) GetMonitorOnDutyGroup(ctx *gin.Context) {
	id := ctx.Param("id")
	intId, err := strconv.Atoi(id)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, "参数错误")
		return
	}

	group, err := p.alertOnDutyService.GetMonitorOnDutyGroup(ctx, intId)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.SuccessWithData(ctx, group)
}

// GetMonitorOnDutyGroupFuturePlan 获取指定值班组的未来值班计划
func (p *PrometheusHandler) GetMonitorOnDutyGroupFuturePlan(ctx *gin.Context) {
	var req struct {
		Id        int    `json:"id"`
		StartTime string `json:"startTime"`
		EndTime   string `json:"endTime"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	plans, err := p.alertOnDutyService.GetMonitorOnDutyGroupFuturePlan(ctx, req.Id, req.StartTime, req.EndTime)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.SuccessWithData(ctx, plans)
}

// GetMonitorAlertManagerPoolList 获取 AlertManager 集群池列表
func (p *PrometheusHandler) GetMonitorAlertManagerPoolList(ctx *gin.Context) {
	searchName := ctx.Query("name")

	alerts, err := p.alertPoolService.GetMonitorAlertManagerPoolList(ctx, &searchName)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.SuccessWithData(ctx, alerts)
}

// CreateMonitorAlertManagerPool 创建新的 AlertManager 集群池
func (p *PrometheusHandler) CreateMonitorAlertManagerPool(ctx *gin.Context) {
	var alertManagerPool model.MonitorAlertManagerPool

	uc := ctx.MustGet("user").(ijwt.UserClaims)

	if err := ctx.ShouldBind(&alertManagerPool); err != nil {
		apiresponse.ErrorWithDetails(ctx, err, "参数错误")
		return
	}

	alertManagerPool.UserID = uc.Uid

	if err := p.alertPoolService.CreateMonitorAlertManagerPool(ctx, &alertManagerPool); err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// UpdateMonitorAlertManagerPool 更新现有的 AlertManager 集群池
func (p *PrometheusHandler) UpdateMonitorAlertManagerPool(ctx *gin.Context) {
	var alertManagerPool model.MonitorAlertManagerPool

	if err := ctx.ShouldBind(&alertManagerPool); err != nil {
		apiresponse.ErrorWithDetails(ctx, err, "参数错误")
		return
	}

	if err := p.alertPoolService.UpdateMonitorAlertManagerPool(ctx, &alertManagerPool); err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// DeleteMonitorAlertManagerPool 删除指定的 AlertManager 集群池
func (p *PrometheusHandler) DeleteMonitorAlertManagerPool(ctx *gin.Context) {
	id := ctx.Param("id")

	intId, err := strconv.Atoi(id)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, "参数错误")
		return
	}

	if err := p.alertPoolService.DeleteMonitorAlertManagerPool(ctx, intId); err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// GetMonitorSendGroupList 获取发送组列表
func (p *PrometheusHandler) GetMonitorSendGroupList(ctx *gin.Context) {
	searchName := ctx.Query("name")

	list, err := p.alertSendService.GetMonitorSendGroupList(ctx, &searchName)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.SuccessWithData(ctx, list)
}

// CreateMonitorSendGroup 创建新的发送组
func (p *PrometheusHandler) CreateMonitorSendGroup(ctx *gin.Context) {
	var sendGroup model.MonitorSendGroup

	uc := ctx.MustGet("user").(ijwt.UserClaims)

	if err := ctx.ShouldBind(&sendGroup); err != nil {
		apiresponse.ErrorWithDetails(ctx, err, "参数错误")
		return
	}

	sendGroup.UserID = uc.Uid

	if err := p.alertSendService.CreateMonitorSendGroup(ctx, &sendGroup); err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// UpdateMonitorSendGroup 更新现有的发送组
func (p *PrometheusHandler) UpdateMonitorSendGroup(ctx *gin.Context) {
	var sendGroup model.MonitorSendGroup

	if err := ctx.ShouldBind(&sendGroup); err != nil {
		apiresponse.ErrorWithDetails(ctx, err, "参数错误")
		return
	}

	if err := p.alertSendService.UpdateMonitorSendGroup(ctx, &sendGroup); err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// DeleteMonitorSendGroup 删除指定的发送组
func (p *PrometheusHandler) DeleteMonitorSendGroup(ctx *gin.Context) {
	id := ctx.Param("id")

	intId, err := strconv.Atoi(id)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, "参数错误")
		return
	}

	if err := p.alertSendService.DeleteMonitorSendGroup(ctx, intId); err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// GetMonitorAlertRuleList 获取告警规则列表
func (p *PrometheusHandler) GetMonitorAlertRuleList(ctx *gin.Context) {
	searchName := ctx.Query("name")

	list, err := p.alertRuleService.GetMonitorAlertRuleList(ctx, &searchName)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.SuccessWithData(ctx, list)
}

// PromqlExprCheck 检查 PromQL 表达式的合法性
func (p *PrometheusHandler) PromqlExprCheck(ctx *gin.Context) {
	var promql model.PromqlExprCheckReq

	if err := ctx.ShouldBind(&promql); err != nil {
		apiresponse.ErrorWithDetails(ctx, err, "参数错误")
		return
	}

	exist, err := p.alertRuleService.PromqlExprCheck(ctx, promql.PromqlExpr)
	if !exist || err != nil {
		apiresponse.ErrorWithMessage(ctx, "PromQL 表达式不合法")
		return
	}

	apiresponse.Success(ctx)
}

// CreateMonitorAlertRule 创建新的告警规则
func (p *PrometheusHandler) CreateMonitorAlertRule(ctx *gin.Context) {
	var alertRule model.MonitorAlertRule

	uc := ctx.MustGet("user").(ijwt.UserClaims)

	if err := ctx.ShouldBind(&alertRule); err != nil {
		apiresponse.ErrorWithDetails(ctx, err, "参数错误")
		return
	}

	alertRule.UserID = uc.Uid

	if err := p.alertRuleService.CreateMonitorAlertRule(ctx, &alertRule); err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// UpdateMonitorAlertRule 更新现有的告警规则
func (p *PrometheusHandler) UpdateMonitorAlertRule(ctx *gin.Context) {
	var alertRule model.MonitorAlertRule

	if err := ctx.ShouldBind(&alertRule); err != nil {
		apiresponse.ErrorWithDetails(ctx, err, "参数错误")
		return
	}

	if err := p.alertRuleService.UpdateMonitorAlertRule(ctx, &alertRule); err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// EnableSwitchMonitorAlertRule 切换告警规则的启用状态
func (p *PrometheusHandler) EnableSwitchMonitorAlertRule(ctx *gin.Context) {
	var req model.IdRequest

	if err := ctx.ShouldBind(&req); err != nil {
		apiresponse.ErrorWithDetails(ctx, err, "参数错误")
		return
	}

	if err := p.alertRuleService.EnableSwitchMonitorAlertRule(ctx, req.ID); err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// BatchEnableSwitchMonitorAlertRule 批量切换告警规则的启用状态
func (p *PrometheusHandler) BatchEnableSwitchMonitorAlertRule(ctx *gin.Context) {
	var req model.BatchRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiresponse.ErrorWithDetails(ctx, err, "参数错误")
		return
	}

	if err := p.alertRuleService.BatchEnableSwitchMonitorAlertRule(ctx, req.IDs); err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// DeleteMonitorAlertRule 删除指定的告警规则
func (p *PrometheusHandler) DeleteMonitorAlertRule(ctx *gin.Context) {
	id := ctx.Param("id")

	intId, err := strconv.Atoi(id)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, "参数错误")
		return
	}

	if err := p.alertRuleService.DeleteMonitorAlertRule(ctx, intId); err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// BatchDeleteMonitorAlertRule 批量删除告警规则
func (p *PrometheusHandler) BatchDeleteMonitorAlertRule(ctx *gin.Context) {
	var req model.BatchRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiresponse.ErrorWithDetails(ctx, err, "参数错误")
		return
	}

	if err := p.alertRuleService.BatchDeleteMonitorAlertRule(ctx, req.IDs); err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// GetMonitorAlertEventList 获取告警事件列表
func (p *PrometheusHandler) GetMonitorAlertEventList(ctx *gin.Context) {
	searchName := ctx.Query("name")

	list, err := p.alertEventService.GetMonitorAlertEventList(ctx, &searchName)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.SuccessWithData(ctx, list)
}

// EventAlertSilence 将指定告警事件设置为静默状态
func (p *PrometheusHandler) EventAlertSilence(ctx *gin.Context) {
	var silence model.AlertEventSilenceRequest

	uc := ctx.MustGet("user").(ijwt.UserClaims)

	id := ctx.Param("id")
	intId, err := strconv.Atoi(id)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, "参数错误")
		return
	}

	if err := ctx.ShouldBind(&silence); err != nil {
		apiresponse.ErrorWithDetails(ctx, err, "参数错误")
		return
	}

	if err := p.alertEventService.EventAlertSilence(ctx, intId, &silence, uc.Uid); err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// EventAlertClaim 认领指定的告警事件
func (p *PrometheusHandler) EventAlertClaim(ctx *gin.Context) {
	uc := ctx.MustGet("user").(ijwt.UserClaims)

	id := ctx.Param("id")
	intId, err := strconv.Atoi(id)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, "参数错误")
		return
	}

	if err := p.alertEventService.EventAlertClaim(ctx, intId, uc.Uid); err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// EventAlertUnSilence 取消指定告警事件的静默状态
func (p *PrometheusHandler) EventAlertUnSilence(ctx *gin.Context) {
	uc := ctx.MustGet("user").(ijwt.UserClaims)

	id := ctx.Param("id")
	intId, err := strconv.Atoi(id)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, "参数错误")
		return
	}

	if err := p.alertEventService.EventAlertClaim(ctx, intId, uc.Uid); err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// BatchEventAlertSilence 批量设置告警事件为静默状态
func (p *PrometheusHandler) BatchEventAlertSilence(ctx *gin.Context) {
	var req model.BatchEventAlertSilenceRequest

	uc := ctx.MustGet("user").(ijwt.UserClaims)

	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiresponse.ErrorWithDetails(ctx, err, "参数错误")
		return
	}

	if err := p.alertEventService.BatchEventAlertSilence(ctx, &req, uc.Uid); err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// GetMonitorRecordRuleList 获取预聚合规则列表
func (p *PrometheusHandler) GetMonitorRecordRuleList(ctx *gin.Context) {
	searchName := ctx.Query("name")

	list, err := p.alertRecordService.GetMonitorRecordRuleList(ctx, &searchName)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.SuccessWithData(ctx, list)
}

// CreateMonitorRecordRule 创建新的预聚合规则
func (p *PrometheusHandler) CreateMonitorRecordRule(ctx *gin.Context) {
	var recordRule model.MonitorRecordRule

	uc := ctx.MustGet("user").(ijwt.UserClaims)
	if err := ctx.ShouldBind(&recordRule); err != nil {
		apiresponse.ErrorWithDetails(ctx, err, "参数错误")
		return
	}

	recordRule.UserID = uc.Uid

	if err := p.alertRecordService.CreateMonitorRecordRule(ctx, &recordRule); err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// UpdateMonitorRecordRule 更新现有的预聚合规则
func (p *PrometheusHandler) UpdateMonitorRecordRule(ctx *gin.Context) {
	var recordRule model.MonitorRecordRule

	if err := ctx.ShouldBind(&recordRule); err != nil {
		apiresponse.ErrorWithDetails(ctx, err, "参数错误")
		return
	}

	if err := p.alertRecordService.UpdateMonitorRecordRule(ctx, &recordRule); err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// DeleteMonitorRecordRule 删除指定的预聚合规则
func (p *PrometheusHandler) DeleteMonitorRecordRule(ctx *gin.Context) {
	id := ctx.Param("id")

	intId, err := strconv.Atoi(id)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, "参数错误")
		return
	}

	if err := p.alertRecordService.DeleteMonitorRecordRule(ctx, intId); err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// BatchDeleteMonitorRecordRule 批量删除预聚合规则
func (p *PrometheusHandler) BatchDeleteMonitorRecordRule(ctx *gin.Context) {
	var req model.BatchRequest

	if err := ctx.ShouldBind(&req); err != nil {
		apiresponse.ErrorWithDetails(ctx, err, "参数错误")
		return
	}

	if err := p.alertRecordService.BatchDeleteMonitorRecordRule(ctx, req.IDs); err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// EnableSwitchMonitorRecordRule 切换预聚合规则的启用状态
func (p *PrometheusHandler) EnableSwitchMonitorRecordRule(ctx *gin.Context) {
	id := ctx.Param("id")

	intId, err := strconv.Atoi(id)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, "参数错误")
		return
	}

	if err := p.alertRecordService.EnableSwitchMonitorRecordRule(ctx, intId); err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// BatchEnableSwitchMonitorRecordRule 批量切换预聚合规则的启用状态
func (p *PrometheusHandler) BatchEnableSwitchMonitorRecordRule(ctx *gin.Context) {
	var req model.BatchRequest

	if err := ctx.ShouldBind(&req); err != nil {
		apiresponse.ErrorWithDetails(ctx, err, "参数错误")
		return
	}

	if err := p.alertRecordService.BatchEnableSwitchMonitorRecordRule(ctx, req.IDs); err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}
