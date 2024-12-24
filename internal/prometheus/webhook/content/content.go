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

package content

import (
	"context"
	"encoding/json"
	"fmt"
	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/spf13/viper"
	"net/http"
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/webhook/constant"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/webhook/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/webhook/robot"
	"github.com/prometheus/alertmanager/template"
	"go.uber.org/zap"
)

type WebhookContent interface {
	// GenerateFeishuCardContentOneAlert 生成单个告警的 Feishu 卡片内容并发送到群聊和私聊
	GenerateFeishuCardContentOneAlert(ctx context.Context, alert template.Alert, event *model.MonitorAlertEvent, rule *model.MonitorAlertRule, sendGroup *model.MonitorSendGroup) error
	// SentFeishuGroup 发送消息到 Feishu 群聊
	SentFeishuGroup(ctx context.Context, msg string, robotToken string) error
	// SentFeishuPrivate 发送消息到 Feishu 私聊
	SentFeishuPrivate(ctx context.Context, cardContent string, privateUserIds map[string]string) error
}

type webhookContent struct {
	l      *zap.Logger
	dao    dao.WebhookDao
	robot  robot.WebhookRobot
	client *http.Client
}

func NewWebhookContent(l *zap.Logger, dao dao.WebhookDao, robot robot.WebhookRobot) WebhookContent {
	return &webhookContent{
		l:     l,
		dao:   dao,
		robot: robot,
		client: &http.Client{
			Timeout: 10 * time.Second, // 设置默认超时时间
		},
	}
}

// GenerateFeishuCardContentOneAlert 生成单个告警的 Feishu 卡片内容并发送到群聊和私聊
func (wc *webhookContent) GenerateFeishuCardContentOneAlert(ctx context.Context, alert template.Alert, event *model.MonitorAlertEvent, rule *model.MonitorAlertRule, sendGroup *model.MonitorSendGroup) error {
	// 构建告警标题
	alertHeader := fmt.Sprintf("[触发次数:%v]告警标题:%s ；当前值 %s",
		event.EventTimes,
		alert.Labels["alertname"],
		alert.Annotations["description_value"],
	)

	// 获取告警严重性和绑定的服务节点
	severity := constant.AlertSeverity(alert.Labels["severity"])
	treeNode := alert.Labels["bind_tree_node"]

	// 根据严重性获取标题颜色
	alertHeaderColor, ok := constant.SeverityTitleColorMap[severity]
	if !ok {
		// 如果未定义的严重性，使用默认颜色
		alertHeaderColor = "red"
	}

	// 构建告警详细信息
	msgSeverity := fmt.Sprintf(`**🌡️告警级别：**\n%s`, severity)
	alertStatus := constant.AlertStatus(alert.Status)
	msgStatus := fmt.Sprintf(`**📝当前状态：**\n<font color='%s'>%s</font>`, constant.StatusColorMap[alertStatus], constant.StatusChineseMap[alertStatus])
	msgStreeNode := fmt.Sprintf(`**🏝️ 绑定的服务树：**\n<font color='green'>%s</font>`, treeNode)
	msgTime := fmt.Sprintf(`**🕐 触发时间：**\n%s`, alert.StartsAt.Format("2006-01-02 15:04:05"))

	// 构建 Grafana 和规则链接
	var msgGrafana, msgExpr string
	if rule != nil {
		msgGrafana = fmt.Sprintf(`**🗳查看grafana大盘图**\n[链接地址](%s)`, rule.GrafanaLink)
		msgExpr = fmt.Sprintf(`**🏹修改告警规则** \n[规则地址](%s) \n<font color='red'>%s</font>`,
			fmt.Sprintf("%s/%s?ruleid=%v",
				viper.GetString("webhook.front_domain"),
				"monitor/rule/detail",
				rule.ID),
			rule.Expr,
		)
	}

	// 私聊用户ID列表
	privateUserIds := map[string]string{}

	// 获取值班组信息
	msgOnduty := "值班组和值班人信息(出现这个说明值班信息获取有问题)"
	yuanshiRen := ""
	onDutyGroup, err := wc.dao.GetOnDutyGroupById(ctx, sendGroup.OnDutyGroupID)
	if err != nil {
		return fmt.Errorf("获取值班组失败: %w", err)
	}

	// 构建值班组详情页链接
	onDutyGroupUrl := fmt.Sprintf(constant.SendGroupURLTemplate,
		viper.GetString("webhook.front_domain"),
		"monitor/onduty/detail",
		onDutyGroup.ID,
	)

	// 填充当天的值班用户
	onDutyGroup, err = wc.dao.FillTodayOnDutyUser(ctx, onDutyGroup)
	if err != nil {
		wc.l.Error("填充当天值班用户失败", zap.Error(err), zap.Int("onDutyGroupId", onDutyGroup.ID))
		return fmt.Errorf("填充当天值班用户失败: %w", err)
	}

	if onDutyGroup.TodayDutyUser != nil {
		yuanshiRen = onDutyGroup.TodayDutyUser.RealName
		msgOnduty = fmt.Sprintf(`**👨‍💻 值班组 [%s](%s)：**\n当日值班人:%s\n user_id=%s<at id=%s></at>`,
			onDutyGroup.Name,
			onDutyGroupUrl,
			onDutyGroup.TodayDutyUser.RealName,
			onDutyGroup.TodayDutyUser.FeiShuUserId,
			onDutyGroup.TodayDutyUser.FeiShuUserId,
		)
		privateUserIds[onDutyGroup.TodayDutyUser.FeiShuUserId] = ""
	}

	// 告警升级状态
	msgUpgrade := `**🎛️ 升级状态：**\n未升级`

	// 判断是否需要升级告警
	if event.Status != "renlinged" && alert.Status == string(constant.AlertStatusFiring) && sendGroup.FirstUpgradeUsers != nil && len(sendGroup.FirstUpgradeUsers) > 0 {
		upgradeMinutes := sendGroup.UpgradeMinutes
		if upgradeMinutes == 0 {
			upgradeMinutes = viper.GetInt("webhook.default_upgrade_minutes")
		}
		if time.Since(alert.StartsAt) > time.Minute*time.Duration(upgradeMinutes) {
			var upgradeUserNames, upgradeUserAtIds strings.Builder
			for _, user := range sendGroup.FirstUpgradeUsers {
				privateUserIds[user.FeiShuUserId] = ""
				upgradeUserNames.WriteString(fmt.Sprintf(" %s", user.RealName))
				upgradeUserAtIds.WriteString(fmt.Sprintf(" <at id=%s></at> ", user.FeiShuUserId))
			}

			msgUpgrade = fmt.Sprintf(`**🎛️ 升级状态：**\n**<font color='red'>已升级</font>** [接收人变化]\n[由 %s] -->[%s] `,
				yuanshiRen,
				upgradeUserNames.String(),
			)

			// 更新值班组中的接收人
			msgOnduty = fmt.Sprintf(`**👨‍💻 值班组 [%s](%s)：**\n   告警升级接收人: %s`,
				onDutyGroup.Name,
				onDutyGroupUrl,
				upgradeUserAtIds.String(),
			)
			event.Status = "upgraded"
			if err := wc.dao.UpdateMonitorAlertEvent(ctx, event); err != nil {
				return fmt.Errorf("更新告警事件状态失败: %w", err)
			}
		}
	}

	// 判断是否被认领
	if event.RenLingUser != nil {
		msgOnduty = fmt.Sprintf(`**👨‍💻 值班组 [%s](%s)：**\n认领人:%s\n user_id=%s<at id=%s></at>`,
			onDutyGroup.Name,
			onDutyGroupUrl,
			event.RenLingUser.RealName,
			event.RenLingUser.FeiShuUserId,
			event.RenLingUser.FeiShuUserId,
		)
	}

	// 处理告警标签和注释
	labelMap := pkg.CloneMap(alert.Labels)
	delete(labelMap, "alertname")
	delete(labelMap, "severity")
	delete(labelMap, "bind_tree_node")
	delete(labelMap, "alert_rule_id")
	delete(labelMap, "alert_send_group")

	anno := pkg.CloneMap(alert.Annotations)
	delete(anno, "description_value")

	msgLabel := fmt.Sprintf(`**🛶标签信息：**\n%s`, pkg.FormatMap(labelMap))
	msgAnno := fmt.Sprintf(`**🚂注释信息：**\n%s`, pkg.FormatMap(anno))

	// 构建发送组信息
	sendGroupUrl := fmt.Sprintf(constant.SendGroupURLTemplate,
		viper.GetString("webhook.front_domain"),
		"monitor/sendgroup/detail",
		sendGroup.ID,
	)
	msgSendGroup := fmt.Sprintf(`**📝修改发送组：**\n[%s](%s)`,
		sendGroup.Name,
		sendGroupUrl,
	)
	BackendDomain := viper.GetString("webhook.backend_domain")
	// 构建各类操作的 URL
	buttonURLs := []string{
		fmt.Sprintf(constant.RenderingURLTemplate, BackendDomain, "renling", alert.Fingerprint),    // 认领告警
		fmt.Sprintf(constant.SilenceURLTemplate, BackendDomain, "silence", alert.Fingerprint, 1),   // 屏蔽1小时
		fmt.Sprintf(constant.SilenceURLTemplate, BackendDomain, "silence", alert.Fingerprint, 24),  // 屏蔽24小时
		fmt.Sprintf(constant.UnsilenceURLTemplate, BackendDomain, "unsilence", alert.Fingerprint),  // 取消屏蔽
		fmt.Sprintf(constant.SilenceURLTemplate, BackendDomain, "silence", alert.Fingerprint, 6),   // 屏蔽6小时
		fmt.Sprintf(constant.SilenceURLTemplate, BackendDomain, "silence", alert.Fingerprint, 168), // 屏蔽7天
	}

	// 使用 feiShuCardContent 模板构建 Feishu 卡片内容
	cardContent, err := wc.buildFeishuCardContent(
		alertHeaderColor, // header.template
		alertHeader,      // header.title.content
		msgLabel,         // 第一行标签信息
		msgAnno,          // 第一行 anno 信息
		msgSeverity,      // 第二行告警级别
		msgStatus,        // 第二行当前状态
		msgStreeNode,     // 绑定的服务树
		msgTime,          // 触发时间
		msgUpgrade,       // 升级状态
		msgOnduty,        // 值班组信息
		msgGrafana,       // 查看 Grafana 大盘图
		msgSendGroup,     // 修改发送组
		msgExpr,          // 修改告警规则
		buttonURLs[0],    // 认领告警 URL
		buttonURLs[1],    // 屏蔽1小时 URL
		buttonURLs[2],    // 屏蔽24小时 URL
		buttonURLs[3],    // 取消屏蔽 URL
		buttonURLs[4],    // 屏蔽6小时 URL
		buttonURLs[5],    // 屏蔽7天 URL
	)
	if err != nil {
		return fmt.Errorf("构建 Feishu 卡片内容失败: %w", err)
	}
	// 私聊发送
	if err := wc.SentFeishuPrivate(ctx, cardContent, privateUserIds); err != nil {
		wc.l.Error("发送 Feishu 私聊消息失败",
			zap.Error(err),
			zap.Any("privateUserIds", privateUserIds),
		)
		return fmt.Errorf("发送 Feishu 私聊消息失败: %w", err)
	}

	// 群聊发送
	msgQun := fmt.Sprintf(constant.CartDataGroup, cardContent)

	if err := wc.SentFeishuGroup(ctx, msgQun, sendGroup.FeiShuQunRobotToken); err != nil {
		wc.l.Error("发送 Feishu 群聊消息失败",
			zap.Error(err),
			zap.String("robotToken", sendGroup.FeiShuQunRobotToken),
		)
		return fmt.Errorf("发送 Feishu 群聊消息失败: %w", err)
	}

	return nil
}

// buildFeishuCardContent 构建 Feishu 卡片内容的 JSON 字符串
func (wc *webhookContent) buildFeishuCardContent(
	alertHeaderColor, alertHeader, msgLabel, msgAnno, msgSeverity, msgStatus,
	msgStreeNode, msgTime, msgUpgrade, msgOnduty, msgGrafana, msgSendGroup, msgExpr string,
	buttonURL1, buttonURL2, buttonURL3,
	buttonURL4, buttonURL5, buttonURL6 string,
) (string, error) {

	// 格式化 feiShuCardContent 模板
	cardContent := fmt.Sprintf(constant.CardContent,
		alertHeaderColor, // header.template
		alertHeader,      // header.title.content
		msgLabel,         // 第一行标签信息
		msgAnno,          // 第一行 anno 信息
		msgSeverity,      // 第二行告警级别
		msgStatus,        // 第二行当前状态
		msgStreeNode,     // 绑定的服务树
		msgTime,          // 触发时间
		msgUpgrade,       // 升级状态
		msgOnduty,        // 值班组信息
		msgGrafana,       // 查看 Grafana 大盘图
		msgSendGroup,     // 修改发送组
		msgExpr,          // 修改告警规则
		buttonURL1,       // 认领告警 URL
		buttonURL2,       // 屏蔽1小时 URL
		buttonURL3,       // 屏蔽24小时 URL
		buttonURL4,       // 取消屏蔽 URL
		buttonURL5,       // 屏蔽6小时 URL
		buttonURL6,       // 屏蔽7天 URL
	)

	// 验证生成的 JSON 是否有效
	var temp interface{}
	if err := json.Unmarshal([]byte(cardContent), &temp); err != nil {
		return "", fmt.Errorf("生成的 Feishu 卡片内容 JSON 无效: %w", err)
	}

	return cardContent, nil
}

// SentFeishuGroup 发送消息到 Feishu 群聊
func (wc *webhookContent) SentFeishuGroup(ctx context.Context, msg string, robotToken string) error {
	// 构建 Feishu 群聊机器人 API URL
	url := fmt.Sprintf("%s/%s", viper.GetString("webhook.im_feishu.group_message_api"), robotToken)

	// 发送 HTTP POST 请求
	response, err := pkg.PostWithJson(ctx, wc.client, wc.l, url, msg, nil, nil)
	if err != nil {
		wc.l.Error("发送飞书群聊卡片消息失败",
			zap.Error(err),
			zap.Any("结果", string(response)),
		)
		return fmt.Errorf("发送飞书群聊卡片消息失败: %w", err)
	}

	return nil
}

// FeishuPrivateCardMsg 私聊消息的结构体
type FeishuPrivateCardMsg struct {
	ReceiveId     string `json:"receive_id"`
	ReceiveIdType string `json:"receive_id_type"`
	MsgType       string `json:"msg_type"`
	Content       string `json:"content"`
}

// SentFeishuPrivate 发送消息到 Feishu 私聊
func (wc *webhookContent) SentFeishuPrivate(ctx context.Context, cardContent string, privateUserIds map[string]string) error {
	for userId := range privateUserIds {
		// 构建私聊消息结构体
		feishuPrivateCardMsg := FeishuPrivateCardMsg{
			ReceiveId:     userId,
			ReceiveIdType: "user_id",
			MsgType:       "interactive",
			Content:       cardContent,
		}

		// 序列化消息结构体为 JSON
		data, err := json.Marshal(feishuPrivateCardMsg)
		if err != nil {
			wc.l.Error("序列化 Feishu 私聊消息失败",
				zap.Error(err),
				zap.Any("userId", userId),
			)
			continue
		}

		// 构建 Feishu 私聊机器人 API URL
		url := "https://open.feishu.cn/open-apis/im/v1/messages"

		// 构建请求头
		headers := map[string]string{
			"Authorization": fmt.Sprintf("Bearer %s", wc.robot.GetPrivateRobotToken()),
			"Content-Type":  "application/json; charset=utf-8",
		}
		params := map[string]string{"receive_id_type": "user_id"}

		// 发送 HTTP POST 请求
		response, err := pkg.PostWithJson(ctx, wc.client, wc.l, url, string(data), params, headers)
		if err != nil {
			wc.l.Error("发送飞书私聊卡片消息失败",
				zap.Error(err),
				zap.Any("结果", string(response)),
				zap.Any("userId", userId),
			)
			continue
		}
	}

	return nil
}
