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
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/webhook/dao"
	"github.com/GoSimplicity/AI-CloudOps/pkg/base"
	"github.com/GoSimplicity/AI-CloudOps/pkg/envmode"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/alertmanager/notify/webhook"
	"github.com/prometheus/alertmanager/pkg/labels"
	"github.com/prometheus/alertmanager/template"
	"github.com/prometheus/alertmanager/types"
	"go.uber.org/zap"
)

// WebHookHandler 负责处理Webhook相关的HTTP请求
type WebHookHandler struct {
	l                       *zap.Logger
	dao                     dao.WebhookDao
	alertQueue              chan template.Alert // 告警队列，用于异步处理
	workerWG                sync.WaitGroup      // 工作组用于等待所有工作者完成
	alertManagerAPI         string
	requestTimeoutSeconds   int
	actionSigningKey        string
	alertmanagerBearerToken string
	quitChan                chan struct{} // 用于优雅地关闭工作者的通道
}

// NewWebHookHandler 创建处理器并保存已验证配置的静默请求参数快照。
func NewWebHookHandler(cfg *config.WebhookConfig, l *zap.Logger, dao dao.WebhookDao, alertQueue chan template.Alert) (*WebHookHandler, error) {
	if cfg == nil {
		return nil, fmt.Errorf("Webhook配置不能为空")
	}
	if cfg.ImFeishu.RequestTimeoutSeconds < 1 || cfg.ImFeishu.RequestTimeoutSeconds > 600 {
		return nil, fmt.Errorf("webhook.im_feishu.request_timeout_seconds 必须在1到600之间")
	}
	if err := requireRuntimeSecrets(cfg); err != nil {
		return nil, err
	}

	handler := &WebHookHandler{
		l:                       l,
		dao:                     dao,
		alertQueue:              alertQueue,
		quitChan:                make(chan struct{}),
		alertManagerAPI:         cfg.AlertManagerAPI,
		requestTimeoutSeconds:   cfg.ImFeishu.RequestTimeoutSeconds,
		actionSigningKey:        strings.TrimSpace(cfg.ActionSigningKey),
		alertmanagerBearerToken: strings.TrimSpace(cfg.AlertmanagerBearerToken),
	}
	if l != nil && handler.actionSigningKey == "" {
		l.Warn("WEBHOOK_ACTION_SIGNING_KEY 未配置，飞书卡片不带认领和静默按钮")
	}
	if l != nil && handler.alertmanagerBearerToken == "" {
		l.Warn("WEBHOOK_ALERTMANAGER_BEARER_TOKEN 未配置，/api/v1/alerts/receive 不校验 Authorization")
	}
	if l != nil && strings.TrimSpace(cfg.BackendDomain) == "" {
		l.Warn("WEBHOOK_BACKEND_DOMAIN 未配置，飞书卡片操作链接没有对外地址")
	}
	return handler, nil
}

// requireRuntimeSecrets 按 envmode.IsProduction 判断。
// 只有 development/dev 允许缺省并在上面打警告；生产别名、未设置和无法识别的 ENV（含 test）拒绝启动。
// 卡片对外地址在同一套环境下还拒绝回环主机、未指定地址和无法解析的 URL。
func requireRuntimeSecrets(cfg *config.WebhookConfig) error {
	if !envmode.IsProduction() {
		return nil
	}
	var missing []string
	if strings.TrimSpace(cfg.AlertmanagerBearerToken) == "" {
		missing = append(missing, "WEBHOOK_ALERTMANAGER_BEARER_TOKEN")
	}
	if strings.TrimSpace(cfg.ActionSigningKey) == "" {
		missing = append(missing, "WEBHOOK_ACTION_SIGNING_KEY")
	}
	domain := strings.TrimSpace(cfg.BackendDomain)
	if domain == "" {
		missing = append(missing, "WEBHOOK_BACKEND_DOMAIN")
	}
	if len(missing) > 0 {
		return fmt.Errorf("生产或无法识别的环境缺少 Webhook 配置: %s", strings.Join(missing, "、"))
	}
	return validateProductionBackendDomain(domain)
}

// validateProductionBackendDomain 拒绝无法解析，或主机为回环、未指定地址的卡片对外地址。
// 回环包括 localhost（忽略大小写，以及 *.localhost）、127.0.0.0/8 和 ::1。
// 未指定地址是 0.0.0.0 和 ::（含 [::]）。错误不含配置值。
func validateProductionBackendDomain(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil || !httpURLWithHost(parsed) {
		return fmt.Errorf("生产或无法识别的环境拒绝使用 WEBHOOK_BACKEND_DOMAIN: 无法解析为带主机名的 http 或 https 地址")
	}
	if reason := nonPublicHostReason(parsed.Hostname()); reason != "" {
		return fmt.Errorf("生产或无法识别的环境拒绝使用 WEBHOOK_BACKEND_DOMAIN: 主机是%s", reason)
	}
	return nil
}

func httpURLWithHost(parsed *url.URL) bool {
	if parsed == nil {
		return false
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return false
	}
	return strings.TrimRight(parsed.Hostname(), ".") != ""
}

// nonPublicHostReason 识别不能当作卡片对外主机的地址。普通域名返回空串。
func nonPublicHostReason(host string) string {
	name := strings.TrimRight(strings.TrimSpace(host), ".")
	folded := strings.ToLower(name)
	if folded == "localhost" || strings.HasSuffix(folded, ".localhost") {
		return "回环地址"
	}
	ipText := name
	if zone := strings.IndexByte(ipText, '%'); zone >= 0 {
		ipText = ipText[:zone]
	}
	ip := net.ParseIP(ipText)
	if ip == nil {
		return ""
	}
	if ip.IsLoopback() {
		return "回环地址"
	}
	if ip.IsUnspecified() {
		return "未指定地址"
	}
	return ""
}

// RegisterRouters 注册Webhook相关的HTTP路由
func (w *WebHookHandler) RegisterRouters(server *gin.Engine) {
	alertGroup := server.Group("/api/v1/alerts")
	{
		alertGroup.POST("/receive", w.MonitorAlertReceive)
		alertGroup.GET("/silence", w.MonitorAlertSilence)
		alertGroup.POST("/silence", w.MonitorAlertSilence)
		alertGroup.GET("/unsilence", w.MonitorAlertUnSilence)
		alertGroup.POST("/unsilence", w.MonitorAlertUnSilence)
		alertGroup.GET("/renling", w.MonitorAlertClaim)
		alertGroup.POST("/renling", w.MonitorAlertClaim)
	}
}

// MonitorAlertReceive 处理来自Alertmanager的告警接收请求
func (w *WebHookHandler) MonitorAlertReceive(ctx *gin.Context) {
	if !w.authorizeReceive(ctx) {
		return
	}
	var msg webhook.Message

	if err := ctx.ShouldBindJSON(&msg); err != nil {
		w.l.Error("解析告警JSON失败", zap.String("operation", "alert_receive_decode"))
		base.ErrorWithMessage(ctx, "无效的JSON数据")
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
			base.ErrorWithMessage(ctx, "告警队列已满,请稍后重试")
			return
		}
	}

	base.SuccessWithMessage(ctx, "告警接收成功,正在处理中")
}

func (w *WebHookHandler) MonitorAlertSilence(ctx *gin.Context) {
	act, ok := w.beginSignedAction(ctx, "silence")
	if !ok {
		return
	}
	fingerprint := act.fingerprint
	hour := act.hour

	hourInt, err := strconv.Atoi(hour)
	if err != nil || hourInt <= 0 {
		base.ErrorWithMessage(ctx, "无效的静默时长")
		return
	}

	event, err := w.dao.GetMonitorAlertEventByFingerprintId(ctx, fingerprint)
	if err != nil {
		w.l.Error("获取告警事件失败", zap.String("operation", "alert_event_lookup"))
		base.ErrorWithMessage(ctx, "获取告警事件失败")
		return
	}
	if event == nil {
		base.ErrorWithMessage(ctx, "未找到对应的告警事件")
		return
	}
	expectedState := dao.AlertEventSilenceState{Status: event.Status, SilenceID: event.SilenceID}

	labelsM, err := parseSilenceLabels(event.Labels)
	if err != nil {
		w.l.Warn("无效的标签格式", zap.String("operation", "silence_labels"))
		base.ErrorWithMessage(ctx, "无效的标签格式")
		return
	}
	event.LabelsMap = labelsM

	matchers := make(labels.Matchers, 0, len(labelsM))
	for k, v := range labelsM {
		matchers = append(matchers, &labels.Matcher{
			Type:  labels.MatchEqual,
			Name:  k,
			Value: v,
		})
	}

	silence := types.Silence{
		Matchers:  matchers,
		StartsAt:  time.Now(),
		EndsAt:    time.Now().Add(time.Duration(hourInt) * time.Hour),
		CreatedBy: "系统管理员",
		Comment:   fmt.Sprintf("手动静默处理,持续%d小时", hourInt),
	}

	jsonData, err := json.Marshal(silence)
	if err != nil {
		w.l.Error("序列化静默请求失败", zap.String("operation", "request_encode"))
		base.ErrorWithMessage(ctx, "创建静默请求失败")
		return
	}

	url := fmt.Sprintf("%s/api/v2/silences", w.alertManagerAPI)
	resp, err := base.PostWithJsonString(w.l, "AlertSilence",
		w.requestTimeoutSeconds,
		url, string(jsonData), nil, nil)

	if err != nil {
		w.l.Error("调用静默接口失败", zap.String("operation", "silence_upstream"))
		base.ErrorWithMessage(ctx, "上游静默结果未确认，请人工核对，勿重复提交")
		return
	}

	// 解析响应获取silenceID
	var silenceResp struct {
		SilenceID string `json:"silenceID"`
	}
	if err := json.Unmarshal([]byte(resp), &silenceResp); err != nil {
		w.l.Error("解析静默响应失败", zap.String("operation", "response_decode"))
		base.ErrorWithMessage(ctx, "上游静默结果未确认，请人工核对，勿重复提交")
		return
	}

	if !validSilenceID(silenceResp.SilenceID) {
		w.l.Error("上游静默标识无效", zap.String("operation", "silence_id_validate"))
		base.ErrorWithMessage(ctx, "上游静默结果未确认，请人工核对，勿重复提交")
		return
	}
	// 更新告警事件的静默状态和silenceID
	desiredState := dao.AlertEventSilenceState{Status: model.MonitorAlertEventStatusSilenced, SilenceID: silenceResp.SilenceID}
	if err := w.dao.UpdateMonitorAlertEventSilence(ctx, event.ID, expectedState, desiredState); err != nil {
		w.l.Error("更新告警事件状态失败", zap.String("operation", "silence_local_save"), zap.Int("event_id", event.ID))
		base.ErrorWithMessage(ctx, "上游静默已受理，本地状态保存未确认，请人工核对，勿重复提交")
		return
	}

	w.l.Info("静默设置成功", zap.Int("event_id", event.ID), zap.Int("hour", hourInt))

	base.SuccessWithMessage(ctx, "静默设置成功")
}

func (w *WebHookHandler) MonitorAlertUnSilence(ctx *gin.Context) {
	act, ok := w.beginSignedAction(ctx, "unsilence")
	if !ok {
		return
	}
	fingerprint := act.fingerprint

	event, err := w.dao.GetMonitorAlertEventByFingerprintId(ctx, fingerprint)
	if err != nil {
		w.l.Error("获取告警事件失败", zap.String("operation", "alert_event_lookup"))
		base.ErrorWithMessage(ctx, "获取告警事件失败")
		return
	}

	if event == nil {
		base.ErrorWithMessage(ctx, "未找到对应的告警事件")
		return
	}

	if event.Status != model.MonitorAlertEventStatusSilenced {
		base.ErrorWithMessage(ctx, "该告警未处于静默状态")
		return
	}

	if !validSilenceID(event.SilenceID) {
		w.l.Error("本地静默标识无效", zap.String("operation", "silence_id_validate"), zap.Int("event_id", event.ID))
		base.ErrorWithMessage(ctx, "本地静默标识无效，请人工核对")
		return
	}
	// 与控制台相同：现有记录证明不了排他归属。不删除远端静默，也不改本地关联。
	w.l.Warn("取消静默被拒绝", zap.Int("event_id", event.ID))
	base.ErrorWithMessage(ctx, "未能取消，静默关联已保留")
}

// MonitorAlertClaim 认领告警。卡片链接没有登录用户，只把现有事件状态改为已认领并记日志，不写认领人，也不建新表。
func (w *WebHookHandler) MonitorAlertClaim(ctx *gin.Context) {
	act, ok := w.beginSignedAction(ctx, "renling")
	if !ok {
		return
	}
	event, err := w.dao.GetMonitorAlertEventByFingerprintId(ctx, act.fingerprint)
	if err != nil {
		w.l.Error("获取告警事件失败", zap.String("operation", "alert_event_lookup"))
		base.ErrorWithMessage(ctx, "获取告警事件失败")
		return
	}
	if event == nil {
		base.ErrorWithMessage(ctx, "未找到对应的告警事件")
		return
	}
	switch event.Status {
	case model.MonitorAlertEventStatusClaimed:
		base.SuccessWithMessage(ctx, "告警已认领")
		return
	case model.MonitorAlertEventStatusFiring, model.MonitorAlertEventStatusUpgraded:
	default:
		base.ErrorWithMessage(ctx, "当前状态不能认领")
		return
	}
	expected := event.Status
	if err := w.dao.ClaimMonitorAlertEvent(ctx, event.ID, expected); err != nil {
		w.l.Error("更新告警事件状态失败", zap.String("operation", "alert_claim_save"), zap.Int("event_id", event.ID))
		base.ErrorWithMessage(ctx, "告警状态已变化，认领未生效")
		return
	}
	w.l.Info("告警已从卡片认领", zap.String("operation", "alert_claim"), zap.Int("event_id", event.ID))
	base.SuccessWithMessage(ctx, "认领成功")
}

// AlertManager IDs occupy one bounded URL segment and fit the persisted column.
// Accept existing non-UUID test/legacy IDs without allowing path or control data.
func validSilenceID(id string) bool {
	if len(id) == 0 || len(id) > 100 {
		return false
	}
	for _, character := range id {
		if !((character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') || (character >= '0' && character <= '9') || character == '-' || character == '_') {
			return false
		}
	}
	return true
}
