//go:build wireinject

package di

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

import (
	cron "github.com/GoSimplicity/AI-CloudOps/internal/cron"
	k8sHandler "github.com/GoSimplicity/AI-CloudOps/internal/k8s/api"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	k8sDao "github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao/admin"
	k8sAdminService "github.com/GoSimplicity/AI-CloudOps/internal/k8s/service/admin"
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
		cache.NewAlertConfigCache,
		cache.NewRuleConfigCache,
		cache.NewRecordConfig,
		cache.NewPromConfigCache,
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
		treeService.NewAliResourceService,
		alertDao.NewAlertManagerEventDAO,
		alertDao.NewAlertManagerOnDutyDAO,
		alertDao.NewAlertManagerPoolDAO,
		alertDao.NewAlertManagerRecordDAO,
		alertDao.NewAlertManagerRuleDAO,
		alertDao.NewAlertManagerSendDAO,
		scrapeJobDao.NewScrapeJobDAO,
		scrapeJobDao.NewScrapePoolDAO,
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
		k8sDao.NewClusterDAO,
		k8sDao.NewYamlTemplateDAO,
		k8sDao.NewYamlTaskDAO,
		nodeDao.NewTreeNodeDAO,
		wire.Struct(new(Cmd), "*"),
	)
	return new(Cmd)
}
