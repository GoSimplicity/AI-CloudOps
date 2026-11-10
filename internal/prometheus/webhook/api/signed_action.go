package api

import (
	"crypto/subtle"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/webhook/action"
	"github.com/GoSimplicity/AI-CloudOps/pkg/base"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type signedAction struct {
	name        string
	fingerprint string
	hour        string
	exp         string
	sig         string
}

func requestValue(ctx *gin.Context, key string) string {
	if ctx.Request != nil && ctx.Request.Method == http.MethodPost {
		if value := ctx.PostForm(key); value != "" {
			return value
		}
	}
	return ctx.Query(key)
}

// beginSignedAction 校验 HMAC。GET 只返回确认页，POST 才允许继续改状态。
// 缺少 fingerprint 时保持原来的 JSON 200，供发布栈健康检查使用，且不访问数据库。
func (w *WebHookHandler) beginSignedAction(ctx *gin.Context, name string) (signedAction, bool) {
	act := signedAction{
		name:        name,
		fingerprint: requestValue(ctx, "fingerprint"),
		exp:         requestValue(ctx, "exp"),
		sig:         requestValue(ctx, "sig"),
	}
	if name == "silence" {
		act.hour = requestValue(ctx, "hour")
	}
	if act.fingerprint == "" {
		base.ErrorWithMessage(ctx, "缺少必要的fingerprint参数")
		return act, false
	}
	extra := ""
	if name == "silence" {
		extra = act.hour
	}
	if err := action.Verify(w.actionSigningKey, name, act.fingerprint, extra, act.exp, act.sig, time.Now()); err != nil {
		w.l.Warn("告警操作签名被拒绝", zap.String("operation", "action_signature"), zap.String("action", name))
		if ctx.Request.Method == http.MethodGet {
			writeActionErrorPage(ctx, "链接无效或已过期")
		} else {
			base.Forbidden(ctx, map[string]interface{}{}, "链接无效或已过期")
		}
		return act, false
	}
	if ctx.Request.Method == http.MethodGet {
		writeConfirmPage(ctx, act)
		return act, false
	}
	if ctx.Request.Method != http.MethodPost {
		base.ErrorWithMessage(ctx, "请通过确认页提交")
		return act, false
	}
	return act, true
}

func (w *WebHookHandler) authorizeReceive(ctx *gin.Context) bool {
	want := w.alertmanagerBearerToken
	if want == "" {
		return true
	}
	got, ok := bearerToken(ctx.GetHeader("Authorization"))
	if !ok || subtle.ConstantTimeCompare([]byte(got), []byte(want)) != 1 {
		w.l.Warn("告警接收鉴权失败", zap.String("operation", "alert_receive_auth"))
		base.Unauthorized(ctx, http.StatusUnauthorized, map[string]interface{}{}, "未授权")
		return false
	}
	return true
}

func bearerToken(header string) (string, bool) {
	const prefix = "bearer "
	if len(header) < len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return "", false
	}
	token := header[len(prefix):]
	if token == "" || strings.ContainsAny(token, " \t") {
		return "", false
	}
	return token, true
}

// parseSilenceLabels 只按第一个 '=' 切分。任一标签没有 '=' 或键为空时整次静默失败，避免少匹配导致范围变大。
func parseSilenceLabels(raw []string) (map[string]string, error) {
	if len(raw) == 0 {
		return nil, errInvalidSilenceLabel
	}
	parsed := make(map[string]string, len(raw))
	for _, label := range raw {
		name, value, ok := strings.Cut(label, "=")
		if !ok || name == "" {
			return nil, errInvalidSilenceLabel
		}
		parsed[name] = value
	}
	return parsed, nil
}

var errInvalidSilenceLabel = errString("无效的标签格式")

type errString string

func (e errString) Error() string { return string(e) }

var actionPage = template.Must(template.New("action").Parse(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="referrer" content="no-referrer">
<title>{{.Title}}</title>
</head>
<body>
<p>{{.Message}}</p>
{{if .Confirm}}
<form method="post" action="{{.ActionPath}}">
<input type="hidden" name="fingerprint" value="{{.Fingerprint}}">
{{if .Hour}}<input type="hidden" name="hour" value="{{.Hour}}">{{end}}
<input type="hidden" name="exp" value="{{.Exp}}">
<input type="hidden" name="sig" value="{{.Sig}}">
<button type="submit">{{.Button}}</button>
</form>
{{end}}
</body>
</html>`))

type actionPageData struct {
	Title       string
	Message     string
	Button      string
	ActionPath  string
	Fingerprint string
	Hour        string
	Exp         string
	Sig         string
	Confirm     bool
}

func writeConfirmPage(ctx *gin.Context, act signedAction) {
	title, message, button := confirmCopy(act)
	writeActionPage(ctx, http.StatusOK, actionPageData{
		Title: title, Message: message, Button: button, Confirm: true,
		ActionPath: ctx.Request.URL.Path, Fingerprint: act.fingerprint, Hour: act.hour, Exp: act.exp, Sig: act.sig,
	})
}

func writeActionErrorPage(ctx *gin.Context, message string) {
	writeActionPage(ctx, http.StatusForbidden, actionPageData{Title: "无法完成操作", Message: message})
}

func writeActionPage(ctx *gin.Context, status int, data actionPageData) {
	ctx.Header("Content-Type", "text/html; charset=utf-8")
	ctx.Header("X-Content-Type-Options", "nosniff")
	ctx.Header("Referrer-Policy", "no-referrer")
	ctx.Header("Cache-Control", "no-store")
	ctx.Status(status)
	if err := actionPage.Execute(ctx.Writer, data); err != nil {
		ctx.Status(http.StatusInternalServerError)
	}
}

func confirmCopy(act signedAction) (title, message, button string) {
	switch act.name {
	case "silence":
		return "确认静默", "确认静默这条告警 " + act.hour + " 小时？打开链接不会静默，提交后才会调用 Alertmanager。", "确认静默"
	case "unsilence":
		return "确认取消静默", "确认取消静默？打开链接不会生效，提交后才会调用 Alertmanager。", "确认取消静默"
	case "renling":
		return "确认认领", "确认认领这条告警？打开链接不会生效。提交后只把现有告警状态标为已认领并记日志，不会写入认领人。", "确认认领"
	default:
		return "确认操作", "确认继续？", "确认"
	}
}
