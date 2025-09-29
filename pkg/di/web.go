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
	aiopsApi "github.com/GoSimplicity/AI-CloudOps/internal/aiops/api"
	cronApi "github.com/GoSimplicity/AI-CloudOps/internal/cron/api"
	k8sApi "github.com/GoSimplicity/AI-CloudOps/internal/k8s/api"
	notAuthHandler "github.com/GoSimplicity/AI-CloudOps/internal/not_auth/api"
	prometheusApi "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/api"
	systemApi "github.com/GoSimplicity/AI-CloudOps/internal/system/api"
	resourceApi "github.com/GoSimplicity/AI-CloudOps/internal/tree/api"
	userApi "github.com/GoSimplicity/AI-CloudOps/internal/user/api"
	workorderApi "github.com/GoSimplicity/AI-CloudOps/internal/workorder/api"
	"github.com/gin-gonic/gin"
)

// InitGinServer 初始化web服务
func InitGinServer(
	m []gin.HandlerFunc,
	aiopsHdl *aiopsApi.AIOpsHandler,
	userHdl *userApi.UserHandler,
	authApiHdl *systemApi.ApiHandler,
	authRoleHdl *systemApi.RoleHandler,
	systemHdl *systemApi.SystemHandler,
	notAuthHdl *notAuthHandler.NotAuthHandler,
	k8sClusterHdl *k8sApi.K8sClusterHandler,
	k8sDeploymentHdl *k8sApi.K8sDeploymentHandler,
	k8sNamespaceHdl *k8sApi.K8sNamespaceHandler,
	k8sNodeHdl *k8sApi.K8sNodeHandler,
	k8sSvcHdl *k8sApi.K8sSvcHandler,
	k8sYamlTaskHdl *k8sApi.K8sYamlTaskHandler,
	k8sYamlTemplateHdl *k8sApi.K8sYamlTemplateHandler,
	k8sDaemonSetHdl *k8sApi.K8sDaemonSetHandler,
	k8sEventHdl *k8sApi.K8sEventHandler,
	k8sStatefulSetHdl *k8sApi.K8sStatefulSetHandler,
	k8sServiceAccountHdl *k8sApi.K8sServiceAccountHandler,
	roleHdl *k8sApi.K8sRoleHandler,
	clusterRoleHdl *k8sApi.K8sClusterRoleHandler,
	roleBindingHdl *k8sApi.K8sRoleBindingHandler,
	clusterRoleBindingHdl *k8sApi.K8sClusterRoleBindingHandler,
	k8sConfigMapHdl *k8sApi.K8sConfigMapHandler,
	k8sSecretHdl *k8sApi.K8sSecretHandler,
	alertEventHdl *prometheusApi.AlertEventHandler,
	alertPoolHdl *prometheusApi.AlertPoolHandler,
	alertRuleHdl *prometheusApi.AlertRuleHandler,
	monitorConfigHdl *prometheusApi.MonitorConfigHandler,
	onDutyGroupHdl *prometheusApi.OnDutyGroupHandler,
	recordRuleHdl *prometheusApi.RecordRuleHandler,
	scrapePoolHdl *prometheusApi.ScrapePoolHandler,
	scrapeJobHdl *prometheusApi.ScrapeJobHandler,
	sendGroupHdl *prometheusApi.SendGroupHandler,
	auditHdl *systemApi.AuditHandler,
	formDesignHdl *workorderApi.FormDesignHandler,
	processHdl *workorderApi.WorkorderProcessHandler,
	templateHdl *workorderApi.TemplateHandler,
	instanceHdl *workorderApi.InstanceHandler,
	instanceFlowHdl *workorderApi.InstanceFlowHandler,
	instanceCommentHdl *workorderApi.InstanceCommentHandler,
	categoryHdl *workorderApi.CategoryGroupHandler,
	instanceTimeLineHdl *workorderApi.InstanceTimeLineHandler,
	treeNodeHdl *resourceApi.TreeNodeHandler,
	treeLocalHdl *resourceApi.TreeLocalHandler,
	notificationHdl *workorderApi.NotificationHandler,
	ingressHdl *k8sApi.K8sIngressHandler,
	k8sPodHdl *k8sApi.K8sPodHandler,
	k8sPVHdl *k8sApi.K8sPVHandler,
	k8sPVCHdl *k8sApi.K8sPVCHandler,
	cronJobHdl *cronApi.CronJobHandler,
) *gin.Engine {
	server := gin.Default()
	server.Use(m...)
	aiopsHdl.RegisterRoutes(server)
	userHdl.RegisterRoutes(server)
	authApiHdl.RegisterRouters(server)
	authRoleHdl.RegisterRouters(server)
	systemHdl.RegisterRouters(server)
	auditHdl.RegisterRouters(server)
	notAuthHdl.RegisterRouters(server)
	alertEventHdl.RegisterRouters(server)
	alertPoolHdl.RegisterRouters(server)
	alertRuleHdl.RegisterRouters(server)
	monitorConfigHdl.RegisterRouters(server)
	onDutyGroupHdl.RegisterRouters(server)
	recordRuleHdl.RegisterRouters(server)
	scrapePoolHdl.RegisterRouters(server)
	scrapeJobHdl.RegisterRouters(server)
	sendGroupHdl.RegisterRouters(server)
	k8sClusterHdl.RegisterRouters(server)
	k8sDeploymentHdl.RegisterRouters(server)
	k8sNamespaceHdl.RegisterRouters(server)
	k8sNodeHdl.RegisterRouters(server)
	k8sSvcHdl.RegisterRouters(server)
	k8sYamlTaskHdl.RegisterRouters(server)
	k8sYamlTemplateHdl.RegisterRouters(server)
	k8sDaemonSetHdl.RegisterRouters(server)
	k8sEventHdl.RegisterRouters(server)
	k8sStatefulSetHdl.RegisterRouters(server)
	k8sServiceAccountHdl.RegisterRouters(server)
	roleHdl.RegisterRouters(server)
	clusterRoleHdl.RegisterRouters(server)
	roleBindingHdl.RegisterRouters(server)
	clusterRoleBindingHdl.RegisterRouters(server)
	k8sConfigMapHdl.RegisterRouters(server)
	k8sSecretHdl.RegisterRouters(server)
	formDesignHdl.RegisterRouters(server)
	processHdl.RegisterRouters(server)
	templateHdl.RegisterRouters(server)
	instanceHdl.RegisterRouters(server)
	instanceFlowHdl.RegisterRouters(server)
	instanceCommentHdl.RegisterRouters(server)
	instanceTimeLineHdl.RegisterRouters(server)
	categoryHdl.RegisterRouters(server)
	treeNodeHdl.RegisterRouters(server)
	treeLocalHdl.RegisterRouters(server)
	notificationHdl.RegisterRouters(server)
	ingressHdl.RegisterRouters(server)
	k8sPodHdl.RegisterRouters(server)
	cronJobHdl.RegisterRouters(server)
	k8sPVHdl.RegisterRouters(server)
	k8sPVCHdl.RegisterRouters(server)
	return server
}
