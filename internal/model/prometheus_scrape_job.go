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

package model

// MonitorScrapeJob 监控采集任务的配置
type MonitorScrapeJob struct {
	Model
	Name                     string `json:"name" binding:"required,min=1,max=50" gorm:"size:100;comment:采集任务名称"`
	UserID                   int    `json:"user_id" gorm:"index;not null;comment:任务关联的用户ID"`
	Enable                   int8   `json:"enable" gorm:"type:tinyint(1);default:2;not null;comment:是否启用采集任务: 1-启用, 2-禁用"`
	ServiceDiscoveryType     string `json:"service_discovery_type" gorm:"size:50;not null;default:'http';comment:服务发现类型(k8s/http)"`
	MetricsPath              string `json:"metrics_path" gorm:"size:255;not null;default:'/metrics';comment:监控采集的路径"`
	Scheme                   string `json:"scheme" gorm:"size:10;not null;default:'http';comment:监控采集的协议方案(http/https)"`
	ScrapeInterval           int    `json:"scrape_interval" gorm:"default:30;not null;comment:采集的时间间隔(秒)"`
	ScrapeTimeout            int    `json:"scrape_timeout" gorm:"default:10;not null;comment:采集的超时时间(秒)"`
	PoolID                   int    `json:"pool_id" gorm:"index;not null;comment:关联的采集池ID"`
	RelabelConfigsYamlString string `json:"relabel_configs_yaml_string" gorm:"type:text;comment:relabel配置的YAML字符串"`
	RefreshInterval          int    `json:"refresh_interval" gorm:"default:300;not null;comment:刷新目标的时间间隔(秒)"`
	Port                     int    `json:"port" gorm:"default:9090;not null;comment:采集端口号"`
	IpAddress                string `json:"ip_address" gorm:"size:255;comment:IP地址"`
	KubeConfigFilePath       string `json:"kube_config_file_path" gorm:"size:255;comment:K8s配置文件路径"`
	TlsCaFilePath            string `json:"tls_ca_file_path" gorm:"size:255;comment:TLS CA证书文件路径"`
	TlsCaContent             string `json:"tls_ca_content" gorm:"type:text;comment:TLS CA证书内容"`
	BearerToken              string `json:"bearer_token" gorm:"type:text;comment:鉴权Token内容"`
	BearerTokenFile          string `json:"bearer_token_file" gorm:"size:255;comment:鉴权Token文件路径"`
	KubernetesSdRole         string `json:"kubernetes_sd_role" gorm:"size:50;default:'pod';comment:K8s服务发现角色"`
	CreateUserName           string `json:"create_user_name" gorm:"type:varchar(50);comment:创建用户名称"`
}

func (m *MonitorScrapeJob) TableName() string {
	return "cl_monitor_scrape_jobs"
}

type DeleteMonitorScrapeJobReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}

type GetMonitorScrapeJobListReq struct {
	ListReq
	PoolID int   `json:"pool_id" form:"pool_id" binding:"omitempty"`
	Enable *int8 `json:"enable" form:"enable" binding:"omitempty"`
}

type GetMonitorScrapeJobDetailReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}

type CreateMonitorScrapeJobReq struct {
	Name                     string `json:"name" binding:"required,min=1,max=50"`
	UserID                   int    `json:"user_id" binding:"required"`
	Enable                   int8   `json:"enable"`
	ServiceDiscoveryType     string `json:"service_discovery_type"`
	MetricsPath              string `json:"metrics_path"`
	Scheme                   string `json:"scheme"`
	ScrapeInterval           int    `json:"scrape_interval"`
	ScrapeTimeout            int    `json:"scrape_timeout"`
	PoolID                   int    `json:"pool_id" binding:"required"`
	RelabelConfigsYamlString string `json:"relabel_configs_yaml_string"`
	RefreshInterval          int    `json:"refresh_interval"`
	Port                     int    `json:"port"`
	IpAddress                string `json:"ip_address"`
	KubeConfigFilePath       string `json:"kube_config_file_path"`
	TlsCaFilePath            string `json:"tls_ca_file_path"`
	TlsCaContent             string `json:"tls_ca_content"`
	BearerToken              string `json:"bearer_token"`
	BearerTokenFile          string `json:"bearer_token_file"`
	KubernetesSdRole         string `json:"kubernetes_sd_role"`
	CreateUserName           string `json:"create_user_name"`
}

type UpdateMonitorScrapeJobReq struct {
	ID                       int    `json:"id" form:"id" binding:"required"`
	Name                     string `json:"name" binding:"required,min=1,max=50"`
	Enable                   int8   `json:"enable"`
	ServiceDiscoveryType     string `json:"service_discovery_type"`
	MetricsPath              string `json:"metrics_path"`
	Scheme                   string `json:"scheme"`
	ScrapeInterval           int    `json:"scrape_interval"`
	ScrapeTimeout            int    `json:"scrape_timeout"`
	PoolID                   int    `json:"pool_id"`
	RelabelConfigsYamlString string `json:"relabel_configs_yaml_string"`
	RefreshInterval          int    `json:"refresh_interval"`
	Port                     int    `json:"port"`
	IpAddress                string `json:"ip_address"`
	KubeConfigFilePath       string `json:"kube_config_file_path"`
	TlsCaFilePath            string `json:"tls_ca_file_path"`
	TlsCaContent             string `json:"tls_ca_content"`
	BearerToken              string `json:"bearer_token"`
	BearerTokenFile          string `json:"bearer_token_file"`
	KubernetesSdRole         string `json:"kubernetes_sd_role"`
}
