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
	k8sApi "github.com/GoSimplicity/AI-CloudOps/internal/k8s/api"
	notAuthHandler "github.com/GoSimplicity/AI-CloudOps/internal/not_auth/api"
	prometheusApi "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/api"
	systemApi "github.com/GoSimplicity/AI-CloudOps/internal/system/api"
	treeApi "github.com/GoSimplicity/AI-CloudOps/internal/tree/api"
	userApi "github.com/GoSimplicity/AI-CloudOps/internal/user/api"
	"github.com/gin-gonic/gin"
)

// InitGinServer 初始化web服务
func InitGinServer(
	m []gin.HandlerFunc,
	userHdl *userApi.UserHandler,
	authApiHdl *systemApi.ApiHandler,
	authMenuHdl *systemApi.MenuHandler,
	authRoleHdl *systemApi.RoleHandler,
	authPermissionHdl *systemApi.PermissionHandler,
	treeNodeHdl *treeApi.TreeNodeHandler,
	treeAliResourceHdl *treeApi.AliResourceHandler,
	treeEcsResourceHdl *treeApi.EcsResourceHandler,
	treeEcsHdl *treeApi.EcsHandler,
	treeElbHdl *treeApi.ElbHandler,
	treeRdsHdl *treeApi.RdsHandler,
	notAuthHdl *notAuthHandler.NotAuthHandler,
	k8sClusterHdl *k8sApi.K8sClusterHandler,
	k8sConfigMapHdl *k8sApi.K8sConfigMapHandler,
	k8sDeploymentHdl *k8sApi.K8sDeploymentHandler,
	k8sNamespaceHdl *k8sApi.K8sNamespaceHandler,
	k8sNodeHdl *k8sApi.K8sNodeHandler,
	k8sPodHdl *k8sApi.K8sPodHandler,
	k8sSvcHdl *k8sApi.K8sSvcHandler,
	k8sTaintHdl *k8sApi.K8sTaintHandler,
	k8sYamlTaskHdl *k8sApi.K8sYamlTaskHandler,
	k8sYamlTemplateHdl *k8sApi.K8sYamlTemplateHandler,
	k8sAppHdl *k8sApi.K8sAppHandler,
	alertEventHdl *prometheusApi.AlertEventHandler,
	alertPoolHdl *prometheusApi.AlertPoolHandler,
	alertRuleHdl *prometheusApi.AlertRuleHandler,
	configYamlHdl *prometheusApi.ConfigYamlHandler,
	onDutyGroupHdl *prometheusApi.OnDutyGroupHandler,
	recordRuleHdl *prometheusApi.RecordRuleHandler,
	scrapePoolHdl *prometheusApi.ScrapePoolHandler,
	scrapeJobHdl *prometheusApi.ScrapeJobHandler,
	sendGroupHdl *prometheusApi.SendGroupHandler,
	auditHdl *systemApi.AuditHandler,
) *gin.Engine {
	server := gin.Default()
	server.Use(m...)
	userHdl.RegisterRoutes(server)
	authMenuHdl.RegisterRouters(server)
	authApiHdl.RegisterRouters(server)
	authRoleHdl.RegisterRouters(server)
	authPermissionHdl.RegisterRouters(server)
	auditHdl.RegisterRouters(server)
	treeEcsHdl.RegisterRouters(server)
	treeEcsResourceHdl.RegisterRouters(server)
	treeAliResourceHdl.RegisterRouters(server)
	treeElbHdl.RegisterRouters(server)
	treeRdsHdl.RegisterRouters(server)
	treeNodeHdl.RegisterRouters(server)
	notAuthHdl.RegisterRouters(server)
	alertEventHdl.RegisterRouters(server)
	alertPoolHdl.RegisterRouters(server)
	alertRuleHdl.RegisterRouters(server)
	configYamlHdl.RegisterRouters(server)
	onDutyGroupHdl.RegisterRouters(server)
	recordRuleHdl.RegisterRouters(server)
	scrapePoolHdl.RegisterRouters(server)
	scrapeJobHdl.RegisterRouters(server)
	sendGroupHdl.RegisterRouters(server)
	k8sClusterHdl.RegisterRouters(server)
	k8sAppHdl.RegisterRouters(server)
	k8sConfigMapHdl.RegisterRouters(server)
	k8sDeploymentHdl.RegisterRouters(server)
	k8sNamespaceHdl.RegisterRouters(server)
	k8sNodeHdl.RegisterRouters(server)
	k8sPodHdl.RegisterRouters(server)
	k8sSvcHdl.RegisterRouters(server)
	k8sTaintHdl.RegisterRouters(server)
	k8sYamlTaskHdl.RegisterRouters(server)
	k8sYamlTemplateHdl.RegisterRouters(server)

	return server
}
