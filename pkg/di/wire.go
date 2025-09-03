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
	k8sDao "github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	k8sService "github.com/GoSimplicity/AI-CloudOps/internal/k8s/service"
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
	authHandler.NewSystemHandler,
	userHandler.NewUserHandler,
	notAuthHandler.NewNotAuthHandler,
	k8sHandler.NewK8sPodHandler,
	k8sHandler.NewK8sNodeHandler,
	k8sHandler.NewK8sClusterHandler,
	k8sHandler.NewK8sDeploymentHandler,
	k8sHandler.NewK8sNamespaceHandler,
	k8sHandler.NewK8sSvcHandler,
	k8sHandler.NewK8sYamlTaskHandler,
	k8sHandler.NewK8sYamlTemplateHandler,
	k8sHandler.NewK8sConfigMapHandler,
	k8sHandler.NewK8sSecretHandler,
	k8sHandler.NewK8sDaemonSetHandler,
	k8sHandler.NewK8sEventHandler,
	k8sHandler.NewK8sPVHandler,
	k8sHandler.NewK8sPVCHandler,
	k8sHandler.NewK8sIngressHandler,
	k8sHandler.NewK8sStatefulSetHandler,
	k8sHandler.NewK8sServiceAccountHandler,
	k8sHandler.NewK8sRoleHandler,
	k8sHandler.NewK8sClusterRoleHandler,
	k8sHandler.NewK8sRoleBindingHandler,
	k8sHandler.NewK8sClusterRoleBindingHandler,

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
	workorderHandler.NewInstanceTimeLineHandler,
	workorderHandler.NewTemplateHandler,
	workorderHandler.NewWorkorderProcessHandler,
	workorderHandler.NewCategoryGroupHandler,
	workorderHandler.NewNotificationHandler,
	treeHandler.NewTreeNodeHandler,
	treeHandler.NewTreeLocalHandler,
)

var ServiceSet = wire.NewSet(
	k8sService.NewClusterService,
	k8sService.NewDeploymentService,
	k8sService.NewNamespaceService,
	k8sService.NewPodService,
	k8sService.NewSvcService,
	k8sService.NewNodeService,
	k8sService.NewTaintService,
	k8sService.NewYamlTaskService,
	k8sService.NewYamlTemplateService,
	k8sService.NewConfigMapService,
	k8sService.NewSecretService,
	k8sService.NewDaemonSetService,
	k8sService.NewEventService,
	k8sService.NewPVService,
	k8sService.NewPVCService,
	k8sService.NewIngressService,
	k8sService.NewStatefulSetService,
	k8sService.NewServiceAccountService,
	k8sService.NewRoleService,
	k8sService.NewClusterRoleService,
	k8sService.NewRoleBindingService,
	k8sService.NewClusterRoleBindingService,
	k8sService.NewRBACService,
	userService.NewUserService,
	authService.NewApiService,
	authService.NewRoleService,
	authService.NewAuditService,
	authService.NewSystemService,
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
	workorderService.NewWorkorderInstanceTimeLineService,
	workorderService.NewWorkorderTemplateService,
	workorderService.NewWorkorderProcessService,
	workorderService.NewCategoryGroupService,
	workorderService.NewWorkorderNotificationService,
	treeService.NewTreeNodeService,
	treeService.NewTreeLocalService,
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
	k8sDao.NewYamlTaskDAO,
	k8sDao.NewYamlTemplateDAO,
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
	treeDao.NewTreeLocalDAO,
)

var SSHSet = wire.NewSet(
	ssh.NewSSH,
)

var UtilSet = wire.NewSet(
	ijwt.NewJWTHandler,
)

var ManagerSet = wire.NewSet(
	manager.NewClusterManager,
	manager.NewDeploymentManager,
	manager.NewNamespaceManager,
	manager.NewPodManager,
	manager.NewServiceManager,
	manager.NewNodeManager,
	manager.NewConfigMapManager,
	manager.NewSecretManager,
	manager.NewEventManager,
	manager.NewStatefulSetManager,
	manager.NewDaemonSetManager,
	manager.NewIngressManager,        // 网络入口管理器
	manager.NewPVManager,             // 持久卷管理器
	manager.NewPVCManager,            // 持久卷声明管理器
	manager.NewRBACManager,           // RBAC 权限管理器
	manager.NewServiceAccountManager, // ServiceAccount 管理器
	manager.NewTaintManager,          // 节点污点管理器
	manager.NewYamlManager,           // YAML 模板和任务管理器
)

var JobSet = wire.NewSet(
	startup.NewApplicationBootstrap,
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

var NotificationSet = wire.NewSet(
	InitAsynqClient,
	InitNotificationConfig,
	InitNotificationManager,
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
		ManagerSet,
		CacheSet,
		ClientSet,
		NotificationSet,
	)
	return &Cmd{}
}
