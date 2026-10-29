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
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"gorm.io/gorm"
)

func InitTables(db *gorm.DB) error {
	// 检查数据库连接是否为nil
	if db == nil {
		return fmt.Errorf("数据库连接为空，跳过表初始化")
	}

	if err := db.AutoMigrate(
		// auth
		&model.User{},
		&model.Department{},
		&model.Api{},
		&model.AuditLog{},
		&model.Role{},
		&model.RoleApi{},
		&model.UserRole{},

		// tree
		&model.TreeNode{},
		&model.TreeLocalResource{},
		&model.TreeCloudResource{},
		&model.CloudAccount{},
		&model.CloudAccountRegion{},
		&model.K8sCluster{},
		&model.K8sYamlTemplate{},
		&model.K8sYamlTask{},
		&model.CloudResourceSyncHistory{},
		&model.CloudResourceChangeLog{},

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
		&model.MonitorConfig{},

		// 工单系统
		&model.WorkorderFormDesign{},
		&model.WorkorderInstance{},
		&model.WorkorderInstanceFlow{},
		&model.WorkorderInstanceComment{},
		&model.WorkorderProcess{},
		&model.WorkorderTemplate{},
		&model.WorkorderCategory{},
		&model.WorkorderNotification{},
		&model.WorkorderNotificationLog{},
		&model.WorkorderNotificationQueue{},
		&model.WorkorderNotificationEvent{},
		&model.WorkorderNotificationDelivery{},
		&model.WorkorderNotificationResolution{},
		&model.WorkorderInstanceTimeline{},

		&model.CronJob{},
	); err != nil {
		return err
	}
	if err := scrubCloudCredentialAudit(db); err != nil {
		return err
	}
	// 旧版本局部更新曾写入非法0，统一为非默认2，不猜测或恢复历史true意图
	return db.Unscoped().Model(&model.WorkorderNotification{}).Where("is_default = ?", 0).
		UpdateColumn("is_default", model.IsDefaultNo).Error
}

// scrubCloudCredentialAudit 清理旧版本审计副本，保留事件、操作者和原始时间
func scrubCloudCredentialAudit(db *gorm.DB) error {
	return db.Unscoped().Model(&model.CloudResourceChangeLog{}).
		Where("LOWER(TRIM(field_name)) IN ?", []string{"key", "password"}).
		Where("old_value <> ? OR new_value <> ?", model.CredentialAuditRedaction, model.CredentialAuditRedaction).
		UpdateColumns(map[string]any{"old_value": model.CredentialAuditRedaction, "new_value": model.CredentialAuditRedaction}).Error
}
