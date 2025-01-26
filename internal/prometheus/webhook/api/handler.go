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

package api

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/webhook/dao"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/alertmanager/notify/webhook"
	"github.com/prometheus/alertmanager/pkg/labels"
	"github.com/prometheus/alertmanager/template"
	"github.com/prometheus/alertmanager/types"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// WebHookHandler 负责处理Webhook相关的HTTP请求
type WebHookHandler struct {
	l          *zap.Logger
	dao        dao.WebhookDao
	alertQueue chan template.Alert // 告警队列，用于异步处理
	workerWG   sync.WaitGroup      // 工作组用于等待所有工作者完成
	quitChan   chan struct{}       // 用于优雅地关闭工作者的通道
}

// NewWebHookHandler 创建一个新的WebHookHandler实例，并启动告警处理工作者
func NewWebHookHandler(l *zap.Logger, dao dao.WebhookDao, alertQueue chan template.Alert) *WebHookHandler {
	handler := &WebHookHandler{
		l:          l,
		dao:        dao,
		alertQueue: alertQueue,
		quitChan:   make(chan struct{}),
	}
	return handler
}

// RegisterRouters 注册Webhook相关的HTTP路由
func (w *WebHookHandler) RegisterRouters(server *gin.Engine) {
	alertGroup := server.Group("/api/v1/alerts")
	{
		alertGroup.POST("/receive", w.MonitorAlertReceive)    // 处理告警接收请求
		alertGroup.GET("/silence", w.MonitorAlertSilence)     // 处理静默告警请求
		alertGroup.GET("/unsilence", w.MonitorAlertUnSilence) // 处理取消静默告警请求
	}
}

// MonitorAlertReceive 处理来自Alertmanager的告警接收请求
func (w *WebHookHandler) MonitorAlertReceive(ctx *gin.Context) {
	var msg webhook.Message

	if err := ctx.ShouldBindJSON(&msg); err != nil {
		w.l.Error("解析告警JSON失败", zap.Error(err))
		utils.ErrorWithMessage(ctx, "无效的JSON数据")
		return
	}

	w.l.Info("收到告警消息",
		zap.String("状态", msg.Status),
		zap.Int("告警数量", len(msg.Alerts)),
	)

	for _, alert := range msg.Alerts {
		select {
		case w.alertQueue <- alert:
			w.l.Debug("告警已加入队列",
				zap.String("告警名称", alert.Labels["alertname"]),
				zap.String("告警级别", alert.Labels["severity"]))
		default:
			w.l.Warn("告警队列已满",
				zap.String("告警名称", alert.Labels["alertname"]))
			utils.ErrorWithMessage(ctx, "告警队列已满,请稍后重试")
			return
		}
	}

	utils.SuccessWithMessage(ctx, "告警接收成功,正在处理中")
}

// MonitorAlertSilence 处理静默告警的请求
func (w *WebHookHandler) MonitorAlertSilence(ctx *gin.Context) {
	fingerprint := ctx.DefaultQuery("fingerprint", "")
	hour := ctx.DefaultQuery("hour", "")

	// 参数校验
	if fingerprint == "" {
		utils.ErrorWithMessage(ctx, "缺少必要的fingerprint参数")
		return
	}

	hourInt, err := strconv.Atoi(hour)
	if err != nil || hourInt <= 0 {
		utils.ErrorWithMessage(ctx, "无效的静默时长")
		return
	}

	// 获取告警事件
	event, err := w.dao.GetMonitorAlertEventByFingerprintId(ctx, fingerprint)
	if err != nil {
		w.l.Error("获取告警事件失败", zap.Error(err))
		utils.ErrorWithMessage(ctx, "获取告警事件失败")
		return
	}
	if event == nil {
		utils.ErrorWithMessage(ctx, "未找到对应的告警事件")
		return
	}

	// 解析标签
	labelsM := make(map[string]string)
	for _, label := range event.Labels {
		parts := strings.Split(label, "=")
		if len(parts) != 2 {
			w.l.Warn("无效的标签格式", zap.String("label", label))
			continue
		}
		labelsM[parts[0]] = parts[1]
	}
	event.LabelsMap = labelsM

	// 构建匹配器
	matchers := make(labels.Matchers, 0, len(labelsM))
	for k, v := range labelsM {
		matchers = append(matchers, &labels.Matcher{
			Type:  labels.MatchEqual,
			Name:  k,
			Value: v,
		})
	}

	// 创建静默请求
	silence := types.Silence{
		Matchers:  matchers,
		StartsAt:  time.Now(),
		EndsAt:    time.Now().Add(time.Duration(hourInt) * time.Hour),
		CreatedBy: "系统管理员",
		Comment:   fmt.Sprintf("手动静默处理,持续%d小时", hourInt),
	}

	jsonData, err := json.Marshal(silence)
	if err != nil {
		w.l.Error("序列化静默请求失败", zap.Error(err))
		utils.ErrorWithMessage(ctx, "创建静默请求失败")
		return
	}

	url := fmt.Sprintf("%s/api/v2/silences", viper.GetString("webhook.alert_manager_api"))
	resp, err := utils.PostWithJsonString(w.l, "AlertSilence",
		viper.GetInt("webhook.im_feishu.request_timeout_seconds"),
		url, string(jsonData), nil, nil)

	if err != nil {
		w.l.Error("调用静默接口失败",
			zap.Error(err),
			zap.String("url", url),
			zap.String("request", string(jsonData)))
		utils.ErrorWithMessage(ctx, "设置静默失败")
		return
	}

	// 解析响应获取silenceID
	var silenceResp struct {
		SilenceID string `json:"silenceID"`
	}
	if err := json.Unmarshal([]byte(resp), &silenceResp); err != nil {
		w.l.Error("解析静默响应失败", zap.Error(err))
		utils.ErrorWithMessage(ctx, "设置静默失败")
		return
	}

	// 更新告警事件的静默状态和silenceID
	event.Status = "silenced"
	event.SilenceID = silenceResp.SilenceID
	if err := w.dao.UpdateMonitorAlertEvent(ctx, event); err != nil {
		w.l.Error("更新告警事件状态失败", zap.Error(err))
	}

	w.l.Info("静默设置成功",
		zap.String("fingerprint", fingerprint),
		zap.Int("hour", hourInt),
		zap.String("silenceID", silenceResp.SilenceID))

	utils.SuccessWithMessage(ctx, "静默设置成功")
}

// MonitorAlertUnSilence 处理取消静默告警的请求
func (w *WebHookHandler) MonitorAlertUnSilence(ctx *gin.Context) {
	fingerprint := ctx.Query("fingerprint")
	if fingerprint == "" {
		utils.ErrorWithMessage(ctx, "缺少必要的fingerprint参数")
		return
	}

	// 获取告警事件
	event, err := w.dao.GetMonitorAlertEventByFingerprintId(ctx, fingerprint)
	if err != nil {
		w.l.Error("获取告警事件失败", zap.Error(err))
		utils.ErrorWithMessage(ctx, "获取告警事件失败")
		return
	}

	if event == nil {
		utils.ErrorWithMessage(ctx, "未找到对应的告警事件")
		return
	}

	if event.Status != "silenced" {
		utils.ErrorWithMessage(ctx, "该告警未处于静默状态")
		return
	}

	// 构建取消静默的URL
	silenceURL := fmt.Sprintf("%s/api/v2/silence/%v",
		viper.GetString("webhook.alert_manager_api"),
		event.SilenceID)

	// 调用AlertManager API取消静默
	_, err = utils.DeleteWithId(w.l, "MonitorAlertUnSilence",
		viper.GetInt("webhook.im_feishu.request_timeout_seconds"),
		silenceURL, nil, nil)

	if err != nil {
		w.l.Error("取消静默失败",
			zap.Error(err),
			zap.String("url", silenceURL))
		utils.ErrorWithMessage(ctx, "取消静默失败")
		return
	}

	// 更新告警事件状态
	event.Status = "firing"
	event.SilenceID = ""
	if err := w.dao.UpdateMonitorAlertEvent(ctx, event); err != nil {
		w.l.Error("更新告警事件状态失败", zap.Error(err))
	}

	w.l.Info("取消静默成功",
		zap.String("fingerprint", fingerprint))

	utils.SuccessWithMessage(ctx, "取消静默成功")
}
