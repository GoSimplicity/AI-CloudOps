//go:build wireinject

package di

import (
	cron "github.com/GoSimplicity/AI-CloudOps/internal/cron"
	k8sHandler "github.com/GoSimplicity/AI-CloudOps/internal/k8s/api"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	k8sDao "github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao"
	k8sAdminService "github.com/GoSimplicity/AI-CloudOps/internal/k8s/service/admin"
	notAuthHandler "github.com/GoSimplicity/AI-CloudOps/internal/not_auth/api"
	notAuthService "github.com/GoSimplicity/AI-CloudOps/internal/not_auth/service"
	promHandler "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/api"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/cache"
	alertCache "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/cache/alert_cache"
	promCache "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/cache/prom_cache"
	recordCache "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/cache/record_cache"
	ruleCache "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/cache/rule_cache"
	alertEventDao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert/event"
	alertOnDutyDao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert/onduty"
	alertPoolDao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert/pool"
	alertRecordDao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert/record"
	alertRuleDao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert/rule"
	alertSendDao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert/send"
	scrapeJobDao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/scrape/job"
	scrapePoolDao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/scrape/pool"
	alertEventService "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/service/alert/event"
	alertOnDutyService "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/service/alert/onduty"
	alertPoolService "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/service/alert/pool"
	alertRecordService "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/service/alert/record"
	alertRuleService "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/service/alert/rule"
	alertSendService "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/service/alert/send"
	scrapeJobService "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/service/scrape/job"
	scrapePoolService "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/service/scrape/pool"
	yamlService "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/service/yaml"
	authHandler "github.com/GoSimplicity/AI-CloudOps/internal/system/api"
	apiDao "github.com/GoSimplicity/AI-CloudOps/internal/system/dao/api"
	authDao "github.com/GoSimplicity/AI-CloudOps/internal/system/dao/casbin"
	menuDao "github.com/GoSimplicity/AI-CloudOps/internal/system/dao/menu"
	roleDao "github.com/GoSimplicity/AI-CloudOps/internal/system/dao/role"
	apiService "github.com/GoSimplicity/AI-CloudOps/internal/system/service/api"
	menuService "github.com/GoSimplicity/AI-CloudOps/internal/system/service/menu"
	roleService "github.com/GoSimplicity/AI-CloudOps/internal/system/service/role"
	treeHandler "github.com/GoSimplicity/AI-CloudOps/internal/tree/api"
	aliDao "github.com/GoSimplicity/AI-CloudOps/internal/tree/dao/ali_resource"
	ecsDao "github.com/GoSimplicity/AI-CloudOps/internal/tree/dao/ecs"
	elbDao "github.com/GoSimplicity/AI-CloudOps/internal/tree/dao/elb"
	rdsDao "github.com/GoSimplicity/AI-CloudOps/internal/tree/dao/rds"
	nodeDao "github.com/GoSimplicity/AI-CloudOps/internal/tree/dao/tree_node"
	treeService "github.com/GoSimplicity/AI-CloudOps/internal/tree/service"
	userHandler "github.com/GoSimplicity/AI-CloudOps/internal/user/api"
	userDao "github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
	userService "github.com/GoSimplicity/AI-CloudOps/internal/user/service"
	ijwt "github.com/GoSimplicity/AI-CloudOps/pkg/utils/jwt"
	"github.com/google/wire"
	_ "github.com/google/wire"
)

func InitWebServer() *Cmd {
	wire.Build(
		InitMiddlewares,
		ijwt.NewJWTHandler,
		InitGinServer,
		InitLogger,
		InitRedis,
		InitDB,
		InitCasbin,
		InitAndRefreshK8sClient,
		client.NewK8sClient,
		cache.NewMonitorCache,
		alertCache.NewAlertConfigCache,
		ruleCache.NewRuleConfigCache,
		recordCache.NewRecordConfig,
		promCache.NewPromConfigCache,
		cron.NewCronManager,
		userHandler.NewUserHandler,
		authHandler.NewAuthHandler,
		notAuthHandler.NewNotAuthHandler,
		treeHandler.NewTreeHandler,
		k8sHandler.NewK8sPodHandler,
		k8sHandler.NewK8sAppHandler,
		k8sHandler.NewK8sNodeHandler,
		k8sHandler.NewK8sConfigMapHandler,
		k8sHandler.NewK8sClusterHandler,
		k8sHandler.NewK8sDeploymentHandler,
		k8sHandler.NewK8sNamespaceHandler,
		k8sHandler.NewK8sSvcHandler,
		k8sHandler.NewK8sTaintHandler,
		k8sHandler.NewK8sYamlTaskHandler,
		k8sHandler.NewK8sYamlTemplateHandler,
		k8sAdminService.NewClusterService,
		k8sAdminService.NewConfigMapService,
		k8sAdminService.NewDeploymentService,
		k8sAdminService.NewNamespaceService,
		k8sAdminService.NewPodService,
		k8sAdminService.NewSvcService,
		k8sAdminService.NewNodeService,
		k8sAdminService.NewTaintService,
		k8sAdminService.NewYamlTaskService,
		k8sAdminService.NewYamlTemplateService,
		userService.NewUserService,
		treeService.NewTreeService,
		apiService.NewApiService,
		roleService.NewRoleService,
		menuService.NewMenuService,
		promHandler.NewPrometheusHandler,
		alertEventService.NewAlertManagerEventService,
		alertOnDutyService.NewAlertManagerOnDutyService,
		alertPoolService.NewAlertManagerPoolService,
		alertRecordService.NewAlertManagerRecordService,
		alertRuleService.NewAlertManagerRuleService,
		alertSendService.NewAlertManagerSendService,
		scrapeJobService.NewPrometheusScrapeService,
		scrapePoolService.NewPrometheusPoolService,
		treeService.NewAliResourceService,
		alertEventDao.NewAlertManagerEventDAO,
		alertOnDutyDao.NewAlertManagerOnDutyDAO,
		alertPoolDao.NewAlertManagerPoolDAO,
		alertRecordDao.NewAlertManagerRecordDAO,
		alertRuleDao.NewAlertManagerRuleDAO,
		alertSendDao.NewAlertManagerSendDAO,
		scrapeJobDao.NewScrapeJobDAO,
		scrapePoolDao.NewScrapePoolDAO,
		aliDao.NewAliResourceDAO,
		yamlService.NewPrometheusConfigService,
		notAuthService.NewNotAuthService,
		userDao.NewUserDAO,
		apiDao.NewApiDAO,
		roleDao.NewRoleDAO,
		menuDao.NewMenuDAO,
		authDao.NewCasbinDAO,
		ecsDao.NewTreeEcsDAO,
		rdsDao.NewTreeRdsDAO,
		elbDao.NewTreeElbDAO,
		k8sDao.NewK8sDAO,
		nodeDao.NewTreeNodeDAO,
		wire.Struct(new(Cmd), "*"),
	)
	return new(Cmd)
}
