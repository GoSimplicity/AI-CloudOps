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

package mock

import (
	"log"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"gorm.io/gorm"
)

type ApiMock struct {
	db *gorm.DB
}

func NewApiMock(db *gorm.DB) *ApiMock {
	return &ApiMock{
		db: db,
	}
}

func (m *ApiMock) InitApi() error {
	// 检查是否已经初始化过API
	var count int64
	m.db.Model(&model.Api{}).Count(&count)
	if count > 0 {
		log.Println("[API已经初始化过,跳过Mock]")
		return nil
	}

	log.Println("[API Mock开始]")

	apis := []model.Api{
		{ID: 1, Path: "/*", Method: 1, Name: "所有接口GET权限", Description: "所有接口GET权限", Version: "v1", Category: 1, IsPublic: 1},
		{ID: 2, Path: "/*", Method: 2, Name: "所有接口POST权限", Description: "所有接口POST权限", Version: "v1", Category: 1, IsPublic: 1},
		{ID: 3, Path: "/*", Method: 3, Name: "所有接口PUT权限", Description: "所有接口PUT权限", Version: "v1", Category: 1, IsPublic: 1},
		{ID: 4, Path: "/*", Method: 4, Name: "所有接口DELETE权限", Description: "所有接口DELETE权限", Version: "v1", Category: 1, IsPublic: 1},
		{ID: 5, Path: "/api/user/logout", Method: 2, Name: "用户登出", Description: "用户退出登录", Version: "v1", Category: 1, IsPublic: 1},
		{ID: 6, Path: "/api/user/codes", Method: 1, Name: "获取权限码", Description: "获取用户权限码", Version: "v1", Category: 1, IsPublic: 0},
		{ID: 7, Path: "/api/user/list", Method: 1, Name: "用户列表", Description: "获取用户列表", Version: "v1", Category: 1, IsPublic: 0},
		{ID: 8, Path: "/api/user/profile", Method: 1, Name: "获取用户信息", Description: "获取用户信息", Version: "v1", Category: 1, IsPublic: 0},
		{ID: 10, Path: "/api/user/change_password", Method: 2, Name: "修改密码", Description: "修改密码", Version: "v1", Category: 1, IsPublic: 0},
		{ID: 11, Path: "/api/user/:id", Method: 4, Name: "删除用户", Description: "删除用户", Version: "v1", Category: 1, IsPublic: 0},
		{ID: 12, Path: "/api/user/profile/update", Method: 2, Name: "更新用户信息", Description: "更新用户信息", Version: "v1", Category: 1, IsPublic: 0},
		{ID: 13, Path: "/api/menus/list", Method: 2, Name: "获取菜单列表", Description: "获取菜单列表", Version: "v1", Category: 1, IsPublic: 0},
		{ID: 14, Path: "/api/menus/create", Method: 2, Name: "创建菜单", Description: "创建菜单", Version: "v1", Category: 1, IsPublic: 0},
		{ID: 15, Path: "/api/menus/update", Method: 2, Name: "更新菜单", Description: "更新菜单", Version: "v1", Category: 1, IsPublic: 0},
		{ID: 16, Path: "/api/menus/:id", Method: 4, Name: "删除菜单", Description: "删除菜单", Version: "v1", Category: 1, IsPublic: 0},
		{ID: 17, Path: "/api/menus/update_related", Method: 2, Name: "更新用户菜单关联", Description: "更新用户菜单关联", Version: "v1", Category: 1, IsPublic: 0},
		{ID: 18, Path: "/api/apis/list", Method: 2, Name: "获取API列表", Description: "获取API列表", Version: "v1", Category: 1, IsPublic: 0},
		{ID: 19, Path: "/api/apis/create", Method: 2, Name: "创建API", Description: "创建API", Version: "v1", Category: 1, IsPublic: 0},
		{ID: 20, Path: "/api/apis/update", Method: 2, Name: "更新API", Description: "更新API", Version: "v1", Category: 1, IsPublic: 0},
		{ID: 21, Path: "/api/apis/:id", Method: 4, Name: "删除API", Description: "删除API", Version: "v1", Category: 1, IsPublic: 0},
		{ID: 22, Path: "/api/roles/list", Method: 2, Name: "获取角色列表", Description: "获取角色列表", Version: "v1", Category: 1, IsPublic: 0},
		{ID: 23, Path: "/api/roles/create", Method: 2, Name: "创建角色", Description: "创建角色", Version: "v1", Category: 1, IsPublic: 0},
		{ID: 24, Path: "/api/roles/update", Method: 2, Name: "更新角色", Description: "更新角色", Version: "v1", Category: 1, IsPublic: 0},
		{ID: 25, Path: "/api/roles/:id", Method: 4, Name: "删除角色", Description: "删除角色", Version: "v1", Category: 1, IsPublic: 0},
		{ID: 26, Path: "/api/roles/:id", Method: 1, Name: "获取角色详情", Description: "获取角色详情", Version: "v1", Category: 1, IsPublic: 0},
		{ID: 27, Path: "/api/roles/user/:id", Method: 1, Name: "获取用户角色", Description: "获取用户角色", Version: "v1", Category: 1, IsPublic: 0},
		{ID: 28, Path: "/api/permissions/user/assign", Method: 2, Name: "分配用户角色", Description: "分配用户角色", Version: "v1", Category: 1, IsPublic: 0},
		{ID: 29, Path: "/api/permissions/users/assign", Method: 2, Name: "批量分配用户角色", Description: "批量分配用户角色", Version: "v1", Category: 1, IsPublic: 0},
		{ID: 30, Path: "/api/tree/node/listTreeNode", Method: 1, Name: "获取资源树节点列表", Description: "获取资源树节点列表", Version: "v1", Category: 4, IsPublic: 0},
		{ID: 31, Path: "/api/tree/node/createTreeNode", Method: 2, Name: "创建资源树节点", Description: "创建资源树节点", Version: "v1", Category: 4, IsPublic: 0},
		{ID: 32, Path: "/api/tree/node/updateTreeNode", Method: 2, Name: "更新资源树节点", Description: "更新资源树节点", Version: "v1", Category: 4, IsPublic: 0},
		{ID: 33, Path: "/api/tree/node/deleteTreeNode/:id", Method: 4, Name: "删除资源树节点", Description: "删除资源树节点", Version: "v1", Category: 4, IsPublic: 0},
		{ID: 34, Path: "/api/tree/ecs/getEcsList", Method: 1, Name: "获取ECS资源列表", Description: "获取ECS资源列表", Version: "v1", Category: 4, IsPublic: 0},
		{ID: 35, Path: "/api/tree/ecs/resource/createEcsResource", Method: 2, Name: "创建ECS资源", Description: "创建ECS资源", Version: "v1", Category: 4, IsPublic: 0},
		{ID: 36, Path: "/api/tree/ecs/resource/deleteEcsResource/:id", Method: 4, Name: "删除ECS资源", Description: "删除ECS资源", Version: "v1", Category: 4, IsPublic: 0},
		{ID: 37, Path: "/api/tree/ecs/resource/updateEcsResource", Method: 2, Name: "更新ECS资源", Description: "更新ECS资源", Version: "v1", Category: 4, IsPublic: 0},
		{ID: 38, Path: "/api/tree/ecs/bindEcs", Method: 2, Name: "绑定ECS资源", Description: "绑定ECS资源", Version: "v1", Category: 4, IsPublic: 0},
		{ID: 39, Path: "/api/tree/ecs/unBindEcs", Method: 2, Name: "解绑ECS资源", Description: "解绑ECS资源", Version: "v1", Category: 4, IsPublic: 0},
		{ID: 40, Path: "/api/tree/ecs/ali/resource/createAliResource", Method: 2, Name: "创建阿里云ECS资源", Description: "创建阿里云ECS资源", Version: "v1", Category: 4, IsPublic: 0},
		{ID: 41, Path: "/api/tree/ecs/ali/resource/updateAliResource", Method: 2, Name: "更新阿里云ECS资源", Description: "更新阿里云ECS资源", Version: "v1", Category: 4, IsPublic: 0},
		{ID: 42, Path: "/api/tree/ecs/ali/resource/deleteAliResource/:id", Method: 4, Name: "删除阿里云ECS资源", Description: "删除阿里云ECS资源", Version: "v1", Category: 4, IsPublic: 0},
		{ID: 43, Path: "/api/tree/elb/getElbList", Method: 1, Name: "获取ELB资源列表", Description: "获取ELB资源列表", Version: "v1", Category: 4, IsPublic: 0},
		{ID: 44, Path: "/api/tree/rds/getRdsList", Method: 1, Name: "获取RDS资源列表", Description: "获取RDS资源列表", Version: "v1", Category: 4, IsPublic: 0},
		{ID: 45, Path: "/api/monitor/scrape_pools/list", Method: 1, Name: "获取监控采集池列表", Description: "获取监控采集池列表", Version: "v1", Category: 3, IsPublic: 0},
		{ID: 46, Path: "/api/monitor/scrape_pools/create", Method: 2, Name: "创建监控采集池", Description: "创建监控采集池", Version: "v1", Category: 3, IsPublic: 0},
		{ID: 47, Path: "/api/monitor/scrape_pools/:id", Method: 4, Name: "删除监控采集池", Description: "删除监控采集池", Version: "v1", Category: 3, IsPublic: 0},
		{ID: 48, Path: "/api/monitor/scrape_pools/update", Method: 2, Name: "更新监控采集池", Description: "更新监控采集池", Version: "v1", Category: 3, IsPublic: 0},
		{ID: 49, Path: "/api/monitor/scrape_jobs/list", Method: 1, Name: "获取监控采集任务列表", Description: "获取监控采集任务列表", Version: "v1", Category: 3, IsPublic: 0},
		{ID: 50, Path: "/api/monitor/scrape_jobs/create", Method: 2, Name: "创建监控采集任务", Description: "创建监控采集任务", Version: "v1", Category: 3, IsPublic: 0},
		{ID: 51, Path: "/api/monitor/scrape_jobs/:id", Method: 4, Name: "删除监控采集任务", Description: "删除监控采集任务", Version: "v1", Category: 3, IsPublic: 0},
		{ID: 52, Path: "/api/monitor/scrape_jobs/update", Method: 2, Name: "更新监控采集任务", Description: "更新监控采集任务", Version: "v1", Category: 3, IsPublic: 0},
		{ID: 53, Path: "/api/monitor/alertManager_pools/list", Method: 1, Name: "获取告警管理池列表", Description: "获取告警管理池列表", Version: "v1", Category: 3, IsPublic: 0},
		{ID: 54, Path: "/api/monitor/alertManager_pools/create", Method: 2, Name: "创建告警管理池", Description: "创建告警管理池", Version: "v1", Category: 3, IsPublic: 0},
		{ID: 55, Path: "/api/monitor/alertManager_pools/update", Method: 2, Name: "更新告警管理池", Description: "更新告警管理池", Version: "v1", Category: 3, IsPublic: 0},
		{ID: 56, Path: "/api/monitor/alertManager_pools/:id", Method: 4, Name: "删除告警管理池", Description: "删除告警管理池", Version: "v1", Category: 3, IsPublic: 0},
		{ID: 57, Path: "/api/monitor/alert_rules/list", Method: 1, Name: "获取告警规则列表", Description: "获取告警规则列表", Version: "v1", Category: 3, IsPublic: 0},
		{ID: 58, Path: "/api/monitor/alert_rules/create", Method: 2, Name: "创建告警规则", Description: "创建告警规则", Version: "v1", Category: 3, IsPublic: 0},
		{ID: 59, Path: "/api/monitor/alert_rules/update", Method: 2, Name: "更新告警规则", Description: "更新告警规则", Version: "v1", Category: 3, IsPublic: 0},
		{ID: 60, Path: "/api/monitor/alert_rules/:id", Method: 4, Name: "删除告警规则", Description: "删除告警规则", Version: "v1", Category: 3, IsPublic: 0},
		{ID: 61, Path: "/api/monitor/alert_rules/promql_check", Method: 2, Name: "验证PromQL表达式", Description: "验证PromQL表达式", Version: "v1", Category: 3, IsPublic: 0},
		{ID: 62, Path: "/api/monitor/alert_events/list", Method: 1, Name: "获取告警事件列表", Description: "获取告警事件列表", Version: "v1", Category: 3, IsPublic: 0},
		{ID: 63, Path: "/api/monitor/record_rules/list", Method: 1, Name: "获取记录规则列表", Description: "获取记录规则列表", Version: "v1", Category: 3, IsPublic: 0},
		{ID: 64, Path: "/api/monitor/record_rules/create", Method: 2, Name: "创建记录规则", Description: "创建记录规则", Version: "v1", Category: 3, IsPublic: 0},
		{ID: 65, Path: "/api/monitor/record_rules/update", Method: 2, Name: "更新记录规则", Description: "更新记录规则", Version: "v1", Category: 3, IsPublic: 0},
		{ID: 66, Path: "/api/monitor/record_rules/:id", Method: 4, Name: "删除记录规则", Description: "删除记录规则", Version: "v1", Category: 3, IsPublic: 0},
		{ID: 67, Path: "/api/monitor/onDuty_groups/list", Method: 1, Name: "获取值班组列表", Description: "获取值班组列表", Version: "v1", Category: 3, IsPublic: 0},
		{ID: 68, Path: "/api/monitor/onDuty_groups/:id", Method: 1, Name: "获取值班组详情", Description: "获取值班组详情", Version: "v1", Category: 3, IsPublic: 0},
		{ID: 69, Path: "/api/monitor/onDuty_groups/create", Method: 2, Name: "创建值班组", Description: "创建值班组", Version: "v1", Category: 3, IsPublic: 0},
		{ID: 70, Path: "/api/monitor/onDuty_groups/update", Method: 2, Name: "更新值班组", Description: "更新值班组", Version: "v1", Category: 3, IsPublic: 0},
		{ID: 71, Path: "/api/monitor/onDuty_groups/:id", Method: 4, Name: "删除值班组", Description: "删除值班组", Version: "v1", Category: 3, IsPublic: 0},
		{ID: 72, Path: "/api/monitor/onDuty_groups/future_plan", Method: 2, Name: "获取值班未来计划", Description: "获取值班未来计划", Version: "v1", Category: 3, IsPublic: 0},
		{ID: 73, Path: "/api/monitor/onDuty_groups/changes", Method: 2, Name: "创建值班变更", Description: "创建值班变更", Version: "v1", Category: 3, IsPublic: 0},
		{ID: 74, Path: "/api/monitor/send_groups/list", Method: 1, Name: "获取发送组列表", Description: "获取发送组列表", Version: "v1", Category: 3, IsPublic: 0},
		{ID: 75, Path: "/api/monitor/send_groups/create", Method: 2, Name: "创建发送组", Description: "创建发送组", Version: "v1", Category: 3, IsPublic: 0},
		{ID: 76, Path: "/api/monitor/send_groups/update", Method: 2, Name: "更新发送组", Description: "更新发送组", Version: "v1", Category: 3, IsPublic: 0},
		{ID: 77, Path: "/api/monitor/send_groups/:id", Method: 4, Name: "删除发送组", Description: "删除发送组", Version: "v1", Category: 3, IsPublic: 0},
		{ID: 78, Path: "/api/k8s/clusters/list", Method: 1, Name: "获取集群列表", Description: "获取所有集群列表", Version: "v1", Category: 2, IsPublic: 0},
		{ID: 79, Path: "/api/k8s/clusters/:id", Method: 1, Name: "获取集群详情", Description: "获取单个集群详情", Version: "v1", Category: 2, IsPublic: 0},
		{ID: 80, Path: "/api/k8s/clusters/create", Method: 2, Name: "创建集群", Description: "创建新的集群", Version: "v1", Category: 2, IsPublic: 0},
		{ID: 81, Path: "/api/k8s/clusters/update", Method: 2, Name: "更新集群", Description: "更新集群信息", Version: "v1", Category: 2, IsPublic: 0},
		{ID: 82, Path: "/api/k8s/clusters/delete/:id", Method: 4, Name: "删除集群", Description: "删除单个集群", Version: "v1", Category: 2, IsPublic: 0},
		{ID: 83, Path: "/api/k8s/clusters/batch_delete", Method: 4, Name: "批量删除集群", Description: "批量删除集群", Version: "v1", Category: 2, IsPublic: 0},
		{ID: 84, Path: "/api/k8s/nodes/list/:id", Method: 1, Name: "获取节点列表", Description: "获取集群节点列表", Version: "v1", Category: 2, IsPublic: 0},
		{ID: 85, Path: "/api/k8s/nodes/:path", Method: 1, Name: "获取节点详情", Description: "获取节点详细信息", Version: "v1", Category: 2, IsPublic: 0},
		{ID: 86, Path: "/api/k8s/nodes/labels/add", Method: 2, Name: "添加节点标签", Description: "添加节点标签", Version: "v1", Category: 2, IsPublic: 0},
		{ID: 87, Path: "/api/k8s/namespaces/list", Method: 1, Name: "获取命名空间列表", Description: "获取所有命名空间列表", Version: "v1", Category: 2, IsPublic: 0},
		{ID: 88, Path: "/api/k8s/namespaces/select/:id", Method: 1, Name: "获取集群命名空间", Description: "获取指定集群的命名空间", Version: "v1", Category: 2, IsPublic: 0},
		{ID: 89, Path: "/api/k8s/namespaces/create", Method: 2, Name: "创建命名空间", Description: "创建新的命名空间", Version: "v1", Category: 2, IsPublic: 0},
		{ID: 90, Path: "/api/k8s/namespaces/delete/:id", Method: 4, Name: "删除命名空间", Description: "删除命名空间", Version: "v1", Category: 2, IsPublic: 0},
		{ID: 91, Path: "/api/k8s/namespaces/:id", Method: 1, Name: "获取命名空间详情", Description: "获取命名空间详细信息", Version: "v1", Category: 2, IsPublic: 0},
		{ID: 92, Path: "/api/k8s/namespaces/update", Method: 2, Name: "更新命名空间", Description: "更新命名空间信息", Version: "v1", Category: 2, IsPublic: 0},
		{ID: 93, Path: "/api/k8s/pods/:id", Method: 1, Name: "获取Pod列表", Description: "获取命名空间下的Pod列表", Version: "v1", Category: 2, IsPublic: 0},
		{ID: 94, Path: "/api/k8s/pods/:id/:podName/containers", Method: 1, Name: "获取容器列表", Description: "获取Pod下的容器列表", Version: "v1", Category: 2, IsPublic: 0},
		{ID: 95, Path: "/api/k8s/pods/:id/:podName/:container/logs", Method: 1, Name: "获取容器日志", Description: "获取容器日志", Version: "v1", Category: 2, IsPublic: 0},
		{ID: 96, Path: "/api/k8s/pods/:id/:podName/yaml", Method: 1, Name: "获取Pod YAML", Description: "获取Pod的YAML配置", Version: "v1", Category: 2, IsPublic: 0},
		{ID: 97, Path: "/api/k8s/pods/delete/:id", Method: 4, Name: "删除Pod", Description: "删除Pod", Version: "v1", Category: 2, IsPublic: 0},
		{ID: 98, Path: "/api/k8s/services/:id", Method: 1, Name: "获取服务列表", Description: "获取服务列表", Version: "v1", Category: 2, IsPublic: 0},
		{ID: 99, Path: "/api/k8s/services/:id/:svcName/yaml", Method: 1, Name: "获取服务YAML", Description: "获取服务的YAML配置", Version: "v1", Category: 2, IsPublic: 0},
		{ID: 100, Path: "/api/k8s/services/update", Method: 2, Name: "更新服务", Description: "更新服务配置", Version: "v1", Category: 2, IsPublic: 0},
		{ID: 101, Path: "/api/k8s/services/delete/:id", Method: 4, Name: "删除服务", Description: "删除服务", Version: "v1", Category: 2, IsPublic: 0},
		{ID: 102, Path: "/api/k8s/deployments/:id", Method: 1, Name: "获取部署列表", Description: "获取部署列表", Version: "v1", Category: 2, IsPublic: 0},
		{ID: 103, Path: "/api/k8s/deployments/:id/yaml", Method: 1, Name: "获取部署YAML", Description: "获取部署的YAML配置", Version: "v1", Category: 2, IsPublic: 0},
		{ID: 104, Path: "/api/k8s/deployments/delete/:id", Method: 4, Name: "删除部署", Description: "删除部署", Version: "v1", Category: 2, IsPublic: 0},
		{ID: 105, Path: "/api/k8s/deployments/restart/:id", Method: 2, Name: "重启部署", Description: "重启部署", Version: "v1", Category: 2, IsPublic: 0},
		{ID: 106, Path: "/api/k8s/configmaps/:id", Method: 1, Name: "获取配置映射列表", Description: "获取配置映射列表", Version: "v1", Category: 2, IsPublic: 0},
		{ID: 107, Path: "/api/k8s/configmaps/:id/yaml", Method: 1, Name: "获取配置映射YAML", Description: "获取配置映射的YAML", Version: "v1", Category: 2, IsPublic: 0},
		{ID: 108, Path: "/api/k8s/configmaps/delete/:id", Method: 4, Name: "删除配置映射", Description: "删除配置映射", Version: "v1", Category: 2, IsPublic: 0},
		{ID: 109, Path: "/api/k8s/yaml_templates/list", Method: 1, Name: "获取YAML模板列表", Description: "获取YAML模板列表", Version: "v1", Category: 2, IsPublic: 0},
		{ID: 110, Path: "/api/k8s/yaml_templates/create", Method: 2, Name: "创建YAML模板", Description: "创建YAML模板", Version: "v1", Category: 2, IsPublic: 0},
		{ID: 111, Path: "/api/k8s/yaml_templates/update", Method: 2, Name: "更新YAML模板", Description: "更新YAML模板", Version: "v1", Category: 2, IsPublic: 0},
		{ID: 112, Path: "/api/k8s/yaml_templates/delete/:id", Method: 4, Name: "删除YAML模板", Description: "删除YAML模板", Version: "v1", Category: 2, IsPublic: 0},
		{ID: 113, Path: "/api/k8s/yaml_templates/:id/yaml", Method: 1, Name: "获取YAML模板详情", Description: "获取YAML模板详情", Version: "v1", Category: 2, IsPublic: 0},
		{ID: 114, Path: "/api/k8s/yaml_templates/check", Method: 2, Name: "检查YAML模板", Description: "检查YAML模板", Version: "v1", Category: 2, IsPublic: 0},
		{ID: 115, Path: "/api/k8s/yaml_tasks/list", Method: 1, Name: "获取YAML任务列表", Description: "获取YAML任务列表", Version: "v1", Category: 2, IsPublic: 0},
		{ID: 116, Path: "/api/k8s/yaml_tasks/delete/:id", Method: 4, Name: "删除YAML任务", Description: "删除YAML任务", Version: "v1", Category: 2, IsPublic: 0},
		{ID: 117, Path: "/api/k8s/yaml_tasks/create", Method: 2, Name: "创建YAML任务", Description: "创建YAML任务", Version: "v1", Category: 2, IsPublic: 0},
		{ID: 118, Path: "/api/k8s/yaml_tasks/update", Method: 2, Name: "更新YAML任务", Description: "更新YAML任务", Version: "v1", Category: 2, IsPublic: 0},
		{ID: 119, Path: "/api/k8s/yaml_tasks/apply/:id", Method: 2, Name: "应用YAML任务", Description: "应用YAML任务", Version: "v1", Category: 2, IsPublic: 0},
	}

	for _, api := range apis {
		if err := m.db.Create(&api).Error; err != nil {
			// 使用FirstOrCreate方法,如果记录存在则跳过,不存在则创建
			result := m.db.Where("id = ?", api.ID).FirstOrCreate(&api)
			if result.Error != nil {
				return result.Error
			}

			if result.RowsAffected == 1 {
				log.Printf("创建API [%s] 成功", api.Name)
			} else {
				log.Printf("API [%s] 已存在,跳过创建", api.Name)
			}
		}
	}

	log.Println("[API Mock结束]")

	return nil
}
