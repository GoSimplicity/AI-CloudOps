//go:build wireinject

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

package di

import (
	cron "github.com/GoSimplicity/AI-CloudOps/internal/cron"
	k8sHandler "github.com/GoSimplicity/AI-CloudOps/internal/k8s/api"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	k8sDao "github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao/admin"
	k8sAppDao "github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao/user"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	k8sAdminService "github.com/GoSimplicity/AI-CloudOps/internal/k8s/service/admin"
	k8sAppService "github.com/GoSimplicity/AI-CloudOps/internal/k8s/service/user"
	notAuthHandler "github.com/GoSimplicity/AI-CloudOps/internal/not_auth/api"
	notAuthService "github.com/GoSimplicity/AI-CloudOps/internal/not_auth/service"
	promHandler "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/api"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/cache"
	alertDao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert"
	configDao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
	scrapeJobDao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/scrape"
	alertService "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/service/alert"
	configService "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/service/config"
	scrapeJobService "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/service/scrape"
	"github.com/GoSimplicity/AI-CloudOps/internal/startup"
	authHandler "github.com/GoSimplicity/AI-CloudOps/internal/system/api"
	authDao "github.com/GoSimplicity/AI-CloudOps/internal/system/dao"
	authService "github.com/GoSimplicity/AI-CloudOps/internal/system/service"
	treeHandler "github.com/GoSimplicity/AI-CloudOps/internal/tree/api"
	treeDao "github.com/GoSimplicity/AI-CloudOps/internal/tree/dao"
	treeProvider "github.com/GoSimplicity/AI-CloudOps/internal/tree/provider"
	treeService "github.com/GoSimplicity/AI-CloudOps/internal/tree/service"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/ssh"
	userHandler "github.com/GoSimplicity/AI-CloudOps/internal/user/api"
	userDao "github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
	userService "github.com/GoSimplicity/AI-CloudOps/internal/user/service"
	workorderHandler "github.com/GoSimplicity/AI-CloudOps/internal/workorder/api"
	workorderDao "github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
	workorderService "github.com/GoSimplicity/AI-CloudOps/internal/workorder/service"
	ijwt "github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/wire"
	_ "github.com/google/wire"
)

type Cmd struct {
	Server    *gin.Engine
	Bootstrap startup.ApplicationBootstrap
	Cron      cron.CronManager
}

var HandlerSet = wire.NewSet(
	authHandler.NewRoleHandler,
	authHandler.NewApiHandler,
	authHandler.NewAuditHandler,
	userHandler.NewUserHandler,
	notAuthHandler.NewNotAuthHandler,
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
	k8sHandler.NewK8sResourceQuotaHandler,
	k8sHandler.NewK8sLimitRangeHandler,
	k8sHandler.NewK8sLabelHandler,
	k8sHandler.NewK8sNodeAffinityHandler,
	k8sHandler.NewK8sPodAffinityHandler,
	k8sHandler.NewK8sAffinityVisualizationHandler,
	k8sHandler.NewK8sRBACHandler,
	k8sHandler.NewK8sServiceAccountHandler,
	k8sHandler.NewK8sTolerationHandler,
	promHandler.NewAlertPoolHandler,
	promHandler.NewMonitorConfigHandler,
	promHandler.NewOnDutyGroupHandler,
	promHandler.NewRecordRuleHandler,
	promHandler.NewAlertRuleHandler,
	promHandler.NewSendGroupHandler,
	promHandler.NewScrapeJobHandler,
	promHandler.NewScrapePoolHandler,
	promHandler.NewAlertEventHandler,
	workorderHandler.NewFormDesignHandler,
	workorderHandler.NewInstanceHandler,
	workorderHandler.NewInstanceFlowHandler,
	workorderHandler.NewInstanceCommentHandler,
	workorderHandler.NewTemplateHandler,
	workorderHandler.NewWorkorderProcessHandler,
	workorderHandler.NewCategoryGroupHandler,
	workorderHandler.NewNotificationHandler,
	treeHandler.NewTreeNodeHandler,
	treeHandler.NewTreeCloudHandler,
	treeHandler.NewTreeEcsHandler,
	treeHandler.NewTreeLocalHandler,
	treeHandler.NewTreeVpcHandler,
	treeHandler.NewTreeSecurityGroupHandler,
	treeHandler.NewTreeRdsHandler,
	treeHandler.NewTreeElbHandler,
)

var ServiceSet = wire.NewSet(
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
	k8sAdminService.NewResourceQuotaService,
	k8sAdminService.NewLimitRangeService,
	k8sAdminService.NewLabelService,
	k8sAdminService.NewNodeAffinityService,
	k8sAdminService.NewPodAffinityService,
	k8sAdminService.NewAffinityVisualizationService,
	k8sAdminService.NewRBACService,
	k8sAdminService.NewServiceAccountService,
	k8sAdminService.NewTolerationService,
	k8sAppService.NewAppService,
	k8sAppService.NewInstanceService,
	k8sAppService.NewCronjobService,
	k8sAppService.NewProjectService,
	userService.NewUserService,
	authService.NewApiService,
	authService.NewRoleService,
	authService.NewAuditService,
	alertService.NewAlertManagerEventService,
	alertService.NewAlertManagerOnDutyService,
	alertService.NewAlertManagerPoolService,
	alertService.NewAlertManagerRecordService,
	alertService.NewAlertManagerRuleService,
	alertService.NewAlertManagerSendService,
	scrapeJobService.NewPrometheusScrapeService,
	scrapeJobService.NewPrometheusPoolService,
	configService.NewMonitorConfigService,
	notAuthService.NewNotAuthService,
	workorderService.NewFormDesignService,
	workorderService.NewInstanceService,
	workorderService.NewInstanceFlowService,
	workorderService.NewInstanceCommentService,
	workorderService.NewWorkorderTemplateService,
	workorderService.NewWorkorderProcessService,
	workorderService.NewCategoryGroupService,
	workorderService.NewWorkorderNotificationService,
	treeService.NewTreeNodeService,
	treeService.NewTreeCloudService,
	treeService.NewTreeEcsService,
	treeService.NewTreeLocalService,
	treeService.NewTreeVpcService,
	treeService.NewTreeElbService,
	treeService.NewTreeRdsService,
	treeService.NewTreeSecurityGroupService,
)

var DaoSet = wire.NewSet(
	alertDao.NewAlertManagerEventDAO,
	alertDao.NewAlertManagerOnDutyDAO,
	alertDao.NewAlertManagerPoolDAO,
	alertDao.NewAlertManagerRecordDAO,
	alertDao.NewAlertManagerRuleDAO,
	alertDao.NewAlertManagerSendDAO,
	scrapeJobDao.NewScrapeJobDAO,
	scrapeJobDao.NewScrapePoolDAO,
	configDao.NewMonitorConfigDAO,
	userDao.NewUserDAO,
	authDao.NewRoleDAO,
	authDao.NewApiDAO,
	authDao.NewAuditDAO,
	k8sDao.NewClusterDAO,
	k8sDao.NewYamlTemplateDAO,
	k8sDao.NewYamlTaskDAO,
	k8sAppDao.NewAppDAO,
	k8sAppDao.NewProjectDAO,
	k8sAppDao.NewCornJobDAO,
	workorderDao.NewWorkorderFormDesignDAO,
	workorderDao.NewTemplateDAO,
	workorderDao.NewWorkorderInstanceDAO,
	workorderDao.NewProcessDAO,
	workorderDao.NewWorkorderCategoryDAO,
	workorderDao.NewWorkorderInstanceCommentDAO,
	workorderDao.NewInstanceFlowDAO,
	workorderDao.NewInstanceTimeLineDAO,
	workorderDao.NewNotificationDAO,
	treeDao.NewTreeNodeDAO,
	treeDao.NewTreeCloudDAO,
	treeDao.NewTreeEcsDAO,
	treeDao.NewTreeLocalDAO,
	treeDao.NewTreeVpcDAO,
	treeDao.NewTreeElbDAO,
	treeDao.NewTreeRdsDAO,
	treeDao.NewTreeSecurityGroupDAO,
)

var SSHSet = wire.NewSet(
	ssh.NewSSH,
)

var UtilSet = wire.NewSet(
	ijwt.NewJWTHandler,
)

var JobSet = wire.NewSet(
	manager.NewClusterManager,
	startup.NewApplicationBootstrap,
)

var ProviderSet = wire.NewSet(
	treeProvider.NewAliyunProvider,
	treeProvider.NewProviderFactoryWithAliyun,
)

var CronSet = wire.NewSet(
	cron.NewCronManager,
)

var Injector = wire.NewSet(
	InitMiddlewares,
	InitGinServer,
	InitLogger,
	InitRedis,
	InitDB,
	CronSet,
	wire.Struct(new(Cmd), "*"),
)

var CacheSet = wire.NewSet(
	cache.NewMonitorCache,
	cache.NewAlertManagerConfigCache,
	cache.NewAlertRuleConfigCache,
	cache.NewRecordRuleConfigCache,
	cache.NewPrometheusConfigCache,
	cache.NewBatchConfigManager,
)

var ClientSet = wire.NewSet(
	client.NewK8sClient,
)

func ProvideCmd() *Cmd {
	wire.Build(
		Injector,
		HandlerSet,
		ServiceSet,
		DaoSet,
		SSHSet,
		UtilSet,
		JobSet,
		CacheSet,
		ClientSet,
		ProviderSet,
	)
	return &Cmd{}
}
