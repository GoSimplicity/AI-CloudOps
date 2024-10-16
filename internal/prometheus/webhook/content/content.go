package content

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/spf13/viper"
	"io"
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

// FeiShuCardDataGroup 用于群聊发送的 JSON 模板
const FeiShuCardDataGroup = `
{
  "msg_type": "interactive",
  "card": %s
}
`

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
		msgExpr = fmt.Sprintf(`**🏹修改告警规则**  [规则地址](%s)\n<font color='red'>%s</font>`,
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
			upgradeMinutes = constant.DefaultUpgradeMinutes
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
	labelMap := cloneMap(alert.Labels)
	delete(labelMap, "alertname")
	delete(labelMap, "severity")
	delete(labelMap, "bind_tree_node")
	delete(labelMap, "alert_rule_id")
	delete(labelMap, "alert_send_group")

	anno := cloneMap(alert.Annotations)
	delete(anno, "description_value")

	msgLabel := fmt.Sprintf(`**🛶标签信息：**\n%s`, formatMap(labelMap))
	msgAnno := fmt.Sprintf(`**🚂anno信息：**\n%s`, formatMap(anno))

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
		fmt.Sprintf(constant.SilenceByNameURLTemplate, BackendDomain, "silence", alert.Fingerprint, 1),
		fmt.Sprintf(constant.SilenceByNameURLTemplate, BackendDomain, "silence", alert.Fingerprint, 6),
		fmt.Sprintf(constant.SilenceByNameURLTemplate, BackendDomain, "silence", alert.Fingerprint, 24),
		fmt.Sprintf(constant.RenderingURLTemplate, BackendDomain, "renling", alert.Fingerprint),
		fmt.Sprintf(constant.SilenceURLTemplate, BackendDomain, "silence", alert.Fingerprint, 1),
		fmt.Sprintf(constant.SilenceURLTemplate, BackendDomain, "silence", alert.Fingerprint, 24),
		fmt.Sprintf(constant.UnsilenceURLTemplate, BackendDomain, "unsilence", alert.Fingerprint),
		fmt.Sprintf(constant.SilenceURLTemplate, BackendDomain, "silence", alert.Fingerprint, 6),
		fmt.Sprintf(constant.SilenceURLTemplate, BackendDomain, "silence", alert.Fingerprint, 168), // 7天
	}

	// 构建 Feishu 卡片内容
	cardContent, err := wc.buildFeishuCardContent(
		alertHeaderColor,
		alertHeader,
		msgLabel,
		msgAnno,
		msgSeverity,
		msgStatus,
		msgStreeNode,
		msgTime,
		msgUpgrade,
		msgOnduty,
		msgGrafana,
		msgSendGroup,
		msgExpr,
		buttonURLs...,
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
	msgQun := fmt.Sprintf(FeiShuCardDataGroup, cardContent)
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
func (wc *webhookContent) buildFeishuCardContent(alertHeaderColor, alertHeader, msgLabel, msgAnno, msgSeverity, msgStatus, msgStreeNode, msgTime, msgUpgrade, msgOnduty, msgGrafana, msgSendGroup, msgExpr string, buttonURLs ...string) (string, error) {
	if len(buttonURLs) < 9 {
		return "", errors.New("buttonURLs 数量不足")
	}

	// 构建 Feishu 卡片结构体
	card := map[string]interface{}{
		"header": map[string]interface{}{
			"template": alertHeaderColor,
			"title": map[string]interface{}{
				"content": alertHeader,
				"tag":     "plain_text",
			},
		},
		"elements": []map[string]interface{}{
			{
				"tag": "div",
				"fields": []map[string]interface{}{
					{
						"is_short": true,
						"text": map[string]interface{}{
							"tag":     "lark_md",
							"content": msgLabel,
						},
					},
					{
						"is_short": true,
						"text": map[string]interface{}{
							"tag":     "lark_md",
							"content": msgAnno,
						},
					},
				},
			},
			{
				"tag": "div",
				"fields": []map[string]interface{}{
					{
						"is_short": true,
						"text": map[string]interface{}{
							"tag":     "lark_md",
							"content": msgSeverity,
						},
					},
					{
						"is_short": true,
						"text": map[string]interface{}{
							"tag":     "lark_md",
							"content": msgStatus,
						},
					},
				},
			},
			{
				"tag":              "column_set",
				"flex_mode":        "none",
				"background_style": "default",
				"columns": []map[string]interface{}{
					{
						"tag":            "column",
						"width":          "weighted",
						"weight":         1,
						"vertical_align": "top",
						"elements": []map[string]interface{}{
							{
								"tag": "div",
								"text": map[string]interface{}{
									"content": msgStreeNode,
									"tag":     "lark_md",
								},
							},
						},
					},
					{
						"tag":            "column",
						"width":          "weighted",
						"weight":         1,
						"vertical_align": "top",
						"elements": []map[string]interface{}{
							{
								"tag": "div",
								"text": map[string]interface{}{
									"content": msgTime,
									"tag":     "lark_md",
								},
							},
						},
					},
				},
			},
			{
				"tag":              "column_set",
				"flex_mode":        "none",
				"background_style": "default",
				"columns": []map[string]interface{}{
					{
						"tag":            "column",
						"width":          "weighted",
						"weight":         1,
						"vertical_align": "top",
						"elements": []map[string]interface{}{
							{
								"tag": "div",
								"text": map[string]interface{}{
									"content": msgUpgrade,
									"tag":     "lark_md",
								},
							},
						},
					},
					{
						"tag":            "column",
						"width":          "weighted",
						"weight":         1,
						"vertical_align": "top",
						"elements": []map[string]interface{}{
							{
								"tag":     "markdown",
								"content": msgOnduty,
							},
						},
					},
				},
			},
			{
				"tag": "div",
				"fields": []map[string]interface{}{
					{
						"is_short": true,
						"text": map[string]interface{}{
							"tag":     "lark_md",
							"content": msgGrafana,
						},
					},
					{
						"is_short": true,
						"text": map[string]interface{}{
							"tag":     "lark_md",
							"content": msgSendGroup,
						},
					},
				},
			},
			{
				"tag": "hr",
			},
			{
				"tag":     "markdown",
				"content": msgExpr,
			},
			{
				"tag": "hr",
			},
			{
				"tag": "div",
				"text": map[string]interface{}{
					"tag":     "lark_md",
					"content": "🔴 告警屏蔽按钮 [下面是单一告警屏蔽👇][右侧是按告警名称屏蔽👉]",
				},
				"extra": map[string]interface{}{
					"tag": "overflow",
					"options": []map[string]interface{}{
						{
							"text": map[string]interface{}{
								"tag":     "plain_text",
								"content": "屏蔽1小时",
							},
							"value": "appStore",
							"url":   buttonURLs[0],
						},
						{
							"text": map[string]interface{}{
								"tag":     "plain_text",
								"content": "屏蔽6小时",
							},
							"value": "document",
							"url":   buttonURLs[1],
						},
						{
							"text": map[string]interface{}{
								"tag":     "plain_text",
								"content": "屏蔽24小时",
							},
							"value": "document",
							"url":   buttonURLs[2],
						},
					},
				},
			},
			{
				"tag": "action",
				"actions": []map[string]interface{}{
					{
						"tag": "button",
						"text": map[string]interface{}{
							"tag":     "plain_text",
							"content": "认领告警",
						},
						"type": "primary",
						"multi_url": map[string]interface{}{
							"url": buttonURLs[3],
						},
						"confirm": map[string]interface{}{
							"title": map[string]interface{}{
								"tag":     "plain_text",
								"content": "确定认领吗",
							},
							"text": map[string]interface{}{
								"tag":     "plain_text",
								"content": "",
							},
						},
					},
					{
						"tag": "button",
						"text": map[string]interface{}{
							"tag":     "plain_text",
							"content": "屏蔽1小时",
						},
						"type": "default",
						"multi_url": map[string]interface{}{
							"url": buttonURLs[4],
						},
						"confirm": map[string]interface{}{
							"title": map[string]interface{}{
								"tag":     "plain_text",
								"content": "确定屏蔽吗",
							},
							"text": map[string]interface{}{
								"tag":     "plain_text",
								"content": "",
							},
						},
					},
					{
						"tag": "button",
						"text": map[string]interface{}{
							"tag":     "plain_text",
							"content": "屏蔽24小时",
						},
						"type": "danger",
						"multi_url": map[string]interface{}{
							"url": buttonURLs[5],
						},
						"confirm": map[string]interface{}{
							"title": map[string]interface{}{
								"tag":     "plain_text",
								"content": "确定屏蔽吗",
							},
							"text": map[string]interface{}{
								"tag":     "plain_text",
								"content": "",
							},
						},
					},
				},
			},
			{
				"tag": "hr",
			},
			{
				"tag": "action",
				"actions": []map[string]interface{}{
					{
						"tag": "button",
						"text": map[string]interface{}{
							"tag":     "plain_text",
							"content": "取消屏蔽",
						},
						"type": "primary",
						"multi_url": map[string]interface{}{
							"url": buttonURLs[6],
						},
						"confirm": map[string]interface{}{
							"title": map[string]interface{}{
								"tag":     "plain_text",
								"content": "确定取消吗",
							},
							"text": map[string]interface{}{
								"tag":     "plain_text",
								"content": "",
							},
						},
					},
					{
						"tag": "button",
						"text": map[string]interface{}{
							"tag":     "plain_text",
							"content": "屏蔽6小时",
						},
						"type": "default",
						"multi_url": map[string]interface{}{
							"url": buttonURLs[7],
						},
						"confirm": map[string]interface{}{
							"title": map[string]interface{}{
								"tag":     "plain_text",
								"content": "确定屏蔽吗",
							},
							"text": map[string]interface{}{
								"tag":     "plain_text",
								"content": "",
							},
						},
					},
					{
						"tag": "button",
						"text": map[string]interface{}{
							"tag":     "plain_text",
							"content": "屏蔽7天",
						},
						"type": "danger",
						"multi_url": map[string]interface{}{
							"url": buttonURLs[8],
						},
						"confirm": map[string]interface{}{
							"title": map[string]interface{}{
								"tag":     "plain_text",
								"content": "确定屏蔽吗",
							},
							"text": map[string]interface{}{
								"tag":     "plain_text",
								"content": "",
							},
						},
					},
				},
			},
			{
				"tag": "hr",
			},
			{
				"tag": "div",
				"text": map[string]interface{}{
					"content": "🙋‍♂️ [我要反馈错误](https://open.feishu.cn/document/uAjLw4CM/ukTMukTMukTM/reference/im-v1/message-development-tutorial/introduction?from=mcb) | 📝 [录入报警处理过程](https://open.feishu.cn/document/uAjLw4CM/ukTMukTMukTM/reference/im-v1/message-development-tutorial/introduction?from=mcb)",
					"tag":     "lark_md",
				},
			},
		},
	}

	// 序列化 Feishu 卡片结构体为 JSON 字符串
	cardJSON, err := json.Marshal(card)
	if err != nil {
		return "", fmt.Errorf("序列化 Feishu 卡片内容失败: %w", err)
	}

	return string(cardJSON), nil
}

// SentFeishuGroup 发送消息到 Feishu 群聊
func (wc *webhookContent) SentFeishuGroup(ctx context.Context, msg string, robotToken string) error {
	// 构建 Feishu 群聊机器人 API URL
	url := fmt.Sprintf("%s/%s", viper.GetString("webhook.im_feishu.group_message_api"), robotToken)

	// 发送 HTTP POST 请求
	response, err := wc.postWithJson(ctx, url, msg, nil)
	if err != nil {
		wc.l.Error("发送飞书群聊卡片消息失败",
			zap.Error(err),
			zap.Any("结果", string(response)),
		)
		return fmt.Errorf("发送飞书群聊卡片消息失败: %w", err)
	}

	return nil
}

// FeishuPrivateCardMsg 定义私聊消息的结构体
type FeishuPrivateCardMsg struct {
	MsgType   string `json:"msg_type"`
	ReceiveId string `json:"receive_id"`
	Content   string `json:"content"`
}

// SentFeishuPrivate 发送消息到 Feishu 私聊
func (wc *webhookContent) SentFeishuPrivate(ctx context.Context, cardContent string, privateUserIds map[string]string) error {
	for userId := range privateUserIds {
		// 构建私聊消息结构体
		feishuPrivateCardMsg := FeishuPrivateCardMsg{
			MsgType:   "interactive",
			ReceiveId: userId,
			Content:   cardContent,
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
			"Content-Type":  "application/json",
		}

		// 发送 HTTP POST 请求
		response, err := wc.postWithJson(ctx, url, string(data), headers)
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

// postWithJson 发送带有JSON字符串的POST请求
func (wc *webhookContent) postWithJson(ctx context.Context, url string, jsonStr string, headers map[string]string) ([]byte, error) {
	// 创建 HTTP 请求
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer([]byte(jsonStr)))
	if err != nil {
		wc.l.Error("创建 HTTP 请求失败",
			zap.Error(err),
			zap.String("url", url),
		)
		return nil, err
	}

	// 设置请求头
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// 设置默认 Content-Type
	if _, exists := headers["Content-Type"]; !exists {
		req.Header.Set("Content-Type", "application/json")
	}

	// 发送请求
	resp, err := wc.client.Do(req)
	if err != nil {
		wc.l.Error("发送 HTTP 请求失败",
			zap.Error(err),
			zap.String("url", url),
		)
		return nil, err
	}
	defer resp.Body.Close()

	// 读取响应体
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		wc.l.Error("读取响应体失败",
			zap.Error(err),
			zap.String("url", url),
		)
		return nil, err
	}

	// 检查 HTTP 状态码
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		wc.l.Error("服务器返回非2xx状态码",
			zap.String("url", url),
			zap.Int("statusCode", resp.StatusCode),
			zap.String("responseBody", string(bodyBytes)),
		)
		return bodyBytes, fmt.Errorf("server returned HTTP status %s", resp.Status)
	}

	return bodyBytes, nil
}

// cloneMap 克隆一个字符串到字符串的映射
func cloneMap(original map[string]string) map[string]string {
	if original == nil {
		return nil
	}
	cloned := make(map[string]string, len(original))
	for k, v := range original {
		cloned[k] = v
	}
	return cloned
}

// formatMap 将 map[string]string 格式化为字符串，每个键值对占一行
func formatMap(m map[string]string) string {
	var builder strings.Builder
	for k, v := range m {
		builder.WriteString(fmt.Sprintf("%s=%s ", k, v))
	}
	return strings.TrimSpace(builder.String())
}
