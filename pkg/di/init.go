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
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"gorm.io/gorm"
)

func InitTables(db *gorm.DB) error {
	return db.AutoMigrate(
		// auth
		&model.User{},
		&model.Api{},
		&model.AuditLog{},
		&model.CasbinRule{},

		// tree
		&model.TreeNode{},
		&model.ResourceEcs{},
		&model.ResourceElb{},
		&model.ResourceRds{},
		&model.TerraformConfig{},

		// k8s

		&model.K8sCluster{},
		&model.K8sProject{},
		&model.K8sYamlTask{},
		&model.K8sYamlTemplate{},
		&model.K8sApp{},      // 注意，放在这里，确保它先于 K8sInstance 创建
		&model.K8sInstance{}, // 外键依赖 K8sApp，确保 K8sApp 先创建
		&model.K8sCronjob{},
		&model.K8sPod{},
		// prometheus
		&model.MonitorScrapePool{},
		&model.MonitorScrapeJob{},
		&model.MonitorAlertManagerPool{},
		&model.MonitorAlertRule{},
		&model.MonitorRecordRule{},
		&model.MonitorOnDutyHistory{},
		&model.MonitorOnDutyGroup{},
		&model.MonitorSendGroup{},
		&model.MonitorOnDutyChange{},
		&model.MonitorAlertEvent{},

		&model.FormDesign{},
		&model.Process{},
		&model.Template{},
	)
}
