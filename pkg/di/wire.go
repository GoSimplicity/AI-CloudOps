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
	"github.com/GoSimplicity/AI-CloudOps/internal/job"
	k8sHandler "github.com/GoSimplicity/AI-CloudOps/internal/k8s/api"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	k8sDao "github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao/admin"
	k8sAppDao "github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao/user"
	k8sAdminService "github.com/GoSimplicity/AI-CloudOps/internal/k8s/service/admin"
	k8sAppService "github.com/GoSimplicity/AI-CloudOps/internal/k8s/service/user"
	notAuthHandler "github.com/GoSimplicity/AI-CloudOps/internal/not_auth/api"
	notAuthService "github.com/GoSimplicity/AI-CloudOps/internal/not_auth/service"
	promHandler "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/api"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/cache"
	alertDao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert"
	scrapeJobDao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/scrape"
	alertService "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/service/alert"
	scrapeJobService "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/service/scrape"
	yamlService "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/service/yaml"
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
	ijwt "github.com/GoSimplicity/AI-CloudOps/pkg/utils"
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
		InitAsynqClient,
		InitAsynqServer,
		InitScheduler,
		job.NewTimedScheduler,
		job.NewTimedTask,
		client.NewK8sClient,
		cache.NewMonitorCache,
		cache.NewAlertConfigCache,
		cache.NewRuleConfigCache,
		cache.NewRecordConfig,
		cache.NewPromConfigCache,
		cron.NewCronManager,
		authHandler.NewRoleHandler,
		authHandler.NewApiHandler,
		authHandler.NewAuditHandler,
		userHandler.NewUserHandler,
		notAuthHandler.NewNotAuthHandler,
		treeHandler.NewEcsHandler,
		treeHandler.NewRdsHandler,
		treeHandler.NewElbHandler,
		treeHandler.NewEcsResourceHandler,
		treeHandler.NewTreeNodeHandler,
		treeHandler.NewAliResourceHandler,
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
		k8sAppService.NewAppService,
		k8sAppService.NewInstanceService,
		k8sAppService.NewCronjobService,
		k8sAppService.NewProjectService,
		userService.NewUserService,
		treeService.NewTreeNodeService,
		treeService.NewEcsService,
		treeService.NewElbService,
		treeService.NewRdsService,
		treeService.NewEcsResourceService,
		treeService.NewAliResourceService,
		authService.NewApiService,
		authService.NewRoleService,
		authService.NewAuditService,
		promHandler.NewAlertPoolHandler,
		promHandler.NewConfigYamlHandler,
		promHandler.NewOnDutyGroupHandler,
		promHandler.NewRecordRuleHandler,
		promHandler.NewAlertRuleHandler,
		promHandler.NewSendGroupHandler,
		promHandler.NewScrapeJobHandler,
		promHandler.NewScrapePoolHandler,
		promHandler.NewAlertEventHandler,
		alertService.NewAlertManagerEventService,
		alertService.NewAlertManagerOnDutyService,
		alertService.NewAlertManagerPoolService,
		alertService.NewAlertManagerRecordService,
		alertService.NewAlertManagerRuleService,
		alertService.NewAlertManagerSendService,
		scrapeJobService.NewPrometheusScrapeService,
		scrapeJobService.NewPrometheusPoolService,
		alertDao.NewAlertManagerEventDAO,
		alertDao.NewAlertManagerOnDutyDAO,
		alertDao.NewAlertManagerPoolDAO,
		alertDao.NewAlertManagerRecordDAO,
		alertDao.NewAlertManagerRuleDAO,
		alertDao.NewAlertManagerSendDAO,
		scrapeJobDao.NewScrapeJobDAO,
		scrapeJobDao.NewScrapePoolDAO,
		yamlService.NewPrometheusConfigService,
		notAuthService.NewNotAuthService,
		userDao.NewUserDAO,
		authDao.NewRoleDAO,
		authDao.NewApiDAO,
		authDao.NewAuditDAO,
		treeDao.NewAliResourceDAO,
		treeDao.NewEcsResourceDAO,
		treeDao.NewTreeEcsDAO,
		treeDao.NewTreeRdsDAO,
		treeDao.NewTreeElbDAO,
		treeDao.NewTreeNodeDAO,
		k8sDao.NewClusterDAO,
		k8sDao.NewYamlTemplateDAO,
		k8sDao.NewYamlTaskDAO,
		k8sAppDao.NewAppDAO,
		k8sAppDao.NewInstanceDAO,
		k8sAppDao.NewProjectDAO,
		k8sAppDao.NewCornJobDAO,
		job.NewCreateK8sClusterTask,
		job.NewUpdateK8sClusterTask,
		job.NewRoutes,
		ssh.NewSSH,
		wire.Struct(new(Cmd), "*"),
	)
	return new(Cmd)
}
