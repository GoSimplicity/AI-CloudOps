package api

import (
	"encoding/json"
	"fmt"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/webhook/dao"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils/apiresponse"
	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils/apiresponse"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/alertmanager/notify/webhook"
	"github.com/prometheus/alertmanager/pkg/labels"
	"github.com/prometheus/alertmanager/template"
	"github.com/prometheus/alertmanager/types"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"strconv"
	"strings"
	"sync"
	"time"
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
	return &WebHookHandler{
		l:          l,
		dao:        dao,
		alertQueue: alertQueue,
		quitChan:   make(chan struct{}),
	}
}

// RegisterRouters 注册Webhook相关的HTTP路由
func (w *WebHookHandler) RegisterRouters(server *gin.Engine) {
	alertGroup := server.Group("/api/v1/alerts")
	{
		alertGroup.POST("/receive", w.MonitorAlertReceive)     // 处理告警接收请求
		alertGroup.POST("/silence", w.MonitorAlertSilence)     // 处理静默告警请求
		alertGroup.POST("/unsilence", w.MonitorAlertUnSilence) // 处理取消静默告警请求
	}
}

// MonitorAlertReceive 处理来自Alertmanager的告警接收请求
func (w *WebHookHandler) MonitorAlertReceive(ctx *gin.Context) {
	var msg webhook.Message

	// 绑定并解析JSON请求体
	if err := ctx.ShouldBindJSON(&msg); err != nil {
		// 错误处理时应返回有用的提示信息，避免使用ctx.String
		w.l.Error("解析Alertmanager传来的告警JSON错误", zap.Error(err))
		apiresponse.ErrorWithMessage(ctx, "Invalid JSON payload")
		return
	}

	// 日志记录接收到的告警信息
	w.l.Info("接收Alertmanager的告警",
		zap.String("status", msg.Status),
		zap.Int("alert_count", len(msg.Alerts)),
	)

	// 异步处理每个告警，避免阻塞主线程
	for _, alert := range msg.Alerts {
		select {
		case w.alertQueue <- alert:
			w.l.Debug("将告警加入队列", zap.String("alertname", alert.Labels["alertname"]))
		default:
			// 告警队列已满，记录警告并返回
			w.l.Warn("告警队列已满，无法处理新的告警", zap.String("alertname", alert.Labels["alertname"]))
			apiresponse.ErrorWithMessage(ctx, "Alert queue is full")
			return
		}
	}

	apiresponse.SuccessWithMessage(ctx, "Alerts received and are being processed")
}

// MonitorAlertSilence 处理静默告警的请求
func (w *WebHookHandler) MonitorAlertSilence(ctx *gin.Context) {
	fingerprint := ctx.DefaultQuery("fingerprint", "")
	hour := ctx.DefaultQuery("hour", "")

	// 检查并转换参数
	hourInt, err := strconv.Atoi(hour)
	if err != nil || hourInt <= 0 {
		apiresponse.ErrorWithMessage(ctx, "Invalid hour parameter")
		return
	}

	// 从数据库获取告警事件
	event, err := w.dao.GetMonitorAlertEventByFingerprintId(ctx, fingerprint)
	if err != nil || event == nil {
		apiresponse.ErrorWithMessage(ctx, "Event not found")
		return
	}

	// 解析标签信息
	labelsM := map[string]string{}
	for _, label := range event.Labels {
		kvs := strings.Split(label, "=")
		if len(kvs) == 2 {
			labelsM[kvs[0]] = kvs[1]
		}
	}

	event.LabelsMatcher = labelsM
	// 创建告警匹配器
	matchers := labels.Matchers{}
	for k, v := range event.LabelsMatcher {
		matchers = append(matchers, &labels.Matcher{Type: labels.MatchEqual, Name: k, Value: v})
	}

	// 构建静默信息
	silence := types.Silence{
		Matchers:  matchers,
		StartsAt:  time.Now(),
		EndsAt:    time.Now().Add(time.Duration(hourInt) * time.Hour),
		CreatedBy: "admin",
		Comment:   "admin处理静默请求",
	}

	// 发送静默请求到Alertmanager
	jsonStr, _ := json.Marshal(silence)
	url := fmt.Sprintf("%s/api/v1/silences", viper.GetString("webhook.alert_manager_api"))
	_, err = pkg.PostWithJsonString(w.l, "AlertSilence", viper.GetInt("webhook.im_feishu.request_timeout_seconds"), url, string(jsonStr), nil, nil)

	if err != nil {
		w.l.Error("调用Alertmanager静默接口失败", zap.Error(err))
		apiresponse.ErrorWithMessage(ctx, "Failed to silence alert")
		return
	}

	apiresponse.Success(ctx)
}

// MonitorAlertUnSilence 处理取消静默告警的请求
func (w *WebHookHandler) MonitorAlertUnSilence(ctx *gin.Context) {
	fingerprint := ctx.Query("fingerprint")
	if fingerprint == "" {
		apiresponse.ErrorWithMessage(ctx, "Missing 'fingerprint' query parameter")
		return
	}

	// 查询事件及静默ID
	event, err := w.dao.GetMonitorAlertEventByFingerprintId(ctx, fingerprint)
	if err != nil || event == nil || event.SilenceID == "" {
		apiresponse.ErrorWithMessage(ctx, "No silence found for the event")
		return
	}
	// 取消静默
	silenceURL := fmt.Sprintf("%s/api/v1/silence/%s", viper.GetString("webhook.alert_manager_api"), event.SilenceID)
	_, err = apiresponse.DeleteWithId(w.l, "MonitorAlertUnSilence", viper.GetInt("webhook.im_feishu.request_timeout_seconds"), silenceURL, nil, nil)

	if err != nil {
		w.l.Error("取消告警静默失败", zap.Error(err))
		apiresponse.ErrorWithMessage(ctx, "Failed to unsilence alert")
		return
	}

	apiresponse.Success(ctx)
}
