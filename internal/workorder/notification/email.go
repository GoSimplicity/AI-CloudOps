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

package notification

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gopkg.in/gomail.v2"
)

type EmailChannel struct {
	config EmailConfig
	logger *zap.Logger
}

func NewEmailChannel(config EmailConfig, logger *zap.Logger) *EmailChannel {
	return &EmailChannel{
		config: config,
		logger: logger,
	}
}

func (e *EmailChannel) GetName() string {
	return model.NotificationChannelEmail
}

// Send 发送邮件
func (e *EmailChannel) Send(ctx context.Context, request *SendRequest) (*SendResponse, error) {
	startTime := time.Now()

	// 验证邮箱
	if !validateEmailAddress(request.RecipientAddr) {
		return &SendResponse{
			Success:      false,
			MessageID:    request.MessageID,
			Status:       "failed",
			ErrorMessage: "invalid email address",
			SendTime:     startTime,
		}, fmt.Errorf("invalid email address: %s", request.RecipientAddr)
	}

	// 创建邮件
	m := gomail.NewMessage()
	m.SetHeader("From", fmt.Sprintf("%s <%s>", e.config.GetFromName(), e.config.GetUsername()))
	m.SetHeader("To", request.RecipientAddr)

	// 设置主题
	subject := request.Subject
	if subject == "" {
		subject = "工单通知"
	}
	m.SetHeader("Subject", subject)

	// 设置内容
	content := e.buildEmailContent(request)
	m.SetBody("text/html", content)

	// 附件
	for _, attachment := range request.Attachments {
		m.Attach(attachment.Name, gomail.SetCopyFunc(func(w io.Writer) error {
			_, err := w.Write(attachment.Content)
			return err
		}))
	}

	// SMTP连接
	d := gomail.NewDialer(e.config.GetSMTPHost(), e.config.GetSMTPPort(), e.config.GetUsername(), e.config.GetPassword())

	// 配置TLS
	if e.config.GetUseTLS() {
		d.TLSConfig = &tls.Config{
			ServerName:         e.config.GetSMTPHost(),
			InsecureSkipVerify: false,
		}
	}

	// 根据SMTP服务器类型设置StartTLS
	if strings.Contains(e.config.GetSMTPHost(), "qq.com") && e.config.GetSMTPPort() == 587 {
		d.SSL = false
	}

	// 发送
	if err := d.DialAndSend(m); err != nil {
		// 解析错误信息，提供更详细的错误说明
		errorMsg := e.parseEmailError(err)

		e.logger.Error("发送邮件失败",
			zap.String("recipient", request.RecipientAddr),
			zap.String("subject", subject),
			zap.String("smtp_host", e.config.GetSMTPHost()),
			zap.Int("smtp_port", e.config.GetSMTPPort()),
			zap.Bool("use_tls", e.config.GetUseTLS()),
			zap.String("error_detail", errorMsg),
			zap.Error(err))

		return &SendResponse{
			Success:      false,
			MessageID:    request.MessageID,
			Status:       "failed",
			ErrorMessage: errorMsg,
			SendTime:     startTime,
		}, fmt.Errorf("%s: %v", errorMsg, err)
	}

	e.logger.Info("邮件发送成功",
		zap.String("recipient", request.RecipientAddr),
		zap.String("subject", subject),
		zap.Duration("duration", time.Since(startTime)))

	return &SendResponse{
		Success:   true,
		MessageID: request.MessageID,
		Status:    "sent",
		SendTime:  startTime,
		ResponseData: map[string]interface{}{
			"smtp_host": e.config.GetSMTPHost(),
			"duration":  time.Since(startTime).String(),
		},
	}, nil
}

// parseEmailError 解析邮件发送错误，提供更详细的错误说明
func (e *EmailChannel) parseEmailError(err error) string {
	errStr := err.Error()

	// QQ邮箱特定错误
	if strings.Contains(errStr, "535") && strings.Contains(e.config.GetSMTPHost(), "qq.com") {
		return "QQ邮箱认证失败：请检查是否使用了授权码而非密码。请前往QQ邮箱设置->账户->POP3/IMAP/SMTP/Exchange/CardDAV/CalDAV服务，生成授权码"
	}

	// 163邮箱特定错误
	if strings.Contains(errStr, "535") && strings.Contains(e.config.GetSMTPHost(), "163.com") {
		return "163邮箱认证失败：请检查是否使用了授权码而非密码。请前往163邮箱设置->POP3/SMTP/IMAP，开启服务并获取授权码"
	}

	// 通用错误处理
	switch {
	case strings.Contains(errStr, "535"):
		return "SMTP认证失败：账号或密码错误，或需要使用授权码"
	case strings.Contains(errStr, "550"):
		return "发送失败：收件人地址无效或被拒绝"
	case strings.Contains(errStr, "554"):
		return "发送失败：邮件内容被拒绝，可能被识别为垃圾邮件"
	case strings.Contains(errStr, "connection refused"):
		return "连接失败：无法连接到SMTP服务器，请检查服务器地址和端口"
	case strings.Contains(errStr, "timeout"):
		return "连接超时：SMTP服务器响应超时"
	case strings.Contains(errStr, "certificate"):
		return "证书错误：TLS证书验证失败"
	default:
		return "邮件发送失败"
	}
}

// Validate 验证
func (e *EmailChannel) Validate() error {
	// 验证必要配置
	if e.config.GetSMTPHost() == "" {
		return fmt.Errorf("SMTP host is required")
	}
	if e.config.GetSMTPPort() == 0 {
		return fmt.Errorf("SMTP port is required")
	}
	if e.config.GetUsername() == "" {
		return fmt.Errorf("SMTP username is required")
	}
	if e.config.GetPassword() == "" {
		return fmt.Errorf("SMTP password is required")
	}

	// 验证端口合法性
	validPorts := []int{25, 465, 587, 2525}
	isValidPort := false
	for _, port := range validPorts {
		if e.config.GetSMTPPort() == port {
			isValidPort = true
			break
		}
	}
	if !isValidPort {
		e.logger.Warn("使用非标准SMTP端口", zap.Int("port", e.config.GetSMTPPort()))
	}

	return e.config.Validate()
}

// IsEnabled 是否启用
func (e *EmailChannel) IsEnabled() bool {
	return e.config.IsEnabled()
}

// GetMaxRetries 最大重试次数
func (e *EmailChannel) GetMaxRetries() int {
	return e.config.GetMaxRetries()
}

// GetRetryInterval 重试间隔
func (e *EmailChannel) GetRetryInterval() time.Duration {
	return e.config.GetRetryInterval()
}

// buildEmailContent 构建内容
func (e *EmailChannel) buildEmailContent(request *SendRequest) string {
	template := `<!DOCTYPE html>
 <html lang="zh-CN">
 <head>
     <meta charset="UTF-8">
     <meta name="viewport" content="width=device-width, initial-scale=1.0">
     <meta http-equiv="X-UA-Compatible" content="IE=edge">
     <title>工单通知</title>
     <style>
         * { box-sizing: border-box; }
         body {
             margin: 0;
             padding: 0;
             font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Roboto', 'Helvetica Neue', Arial, 'Microsoft YaHei', sans-serif;
             line-height: 1.6;
             color: #2d3748;
             background-color: #f7fafc;
         }
         .email-wrapper {
             width: 100%%;
             background-color: #f7fafc;
             padding: 20px 0;
         }
         .email-container {
             max-width: 600px;
             margin: 0 auto;
             background-color: #ffffff;
             border-radius: 12px;
             box-shadow: 0 4px 20px rgba(0, 0, 0, 0.08);
             overflow: hidden;
             border: 1px solid #e2e8f0;
         }
         .header {
             background: linear-gradient(135deg, #3182ce 0%%, #2b77cb 50%%, #2c5aa0 100%%);
             color: white;
             padding: 32px 24px;
             text-align: center;
             position: relative;
         }
         .header::before {
             content: '';
             position: absolute;
             top: 0;
             left: 0;
             right: 0;
             bottom: 0;
             background: url('data:image/svg+xml,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><circle cx="20" cy="20" r="2" fill="rgba(255,255,255,0.1)"/><circle cx="80" cy="40" r="1.5" fill="rgba(255,255,255,0.1)"/><circle cx="40" cy="80" r="1" fill="rgba(255,255,255,0.1)"/></svg>');
         }
         .header-content {
             position: relative;
             z-index: 1;
         }
         .header h1 {
             margin: 0;
             font-size: 26px;
             font-weight: 600;
             margin-bottom: 8px;
         }
         .header-subtitle {
             font-size: 14px;
             opacity: 0.9;
             margin: 0;
         }
         .content {
             padding: 32px 24px;
         }
         .greeting {
             font-size: 16px;
             margin-bottom: 24px;
             color: #4a5568;
         }
         .info-section {
             margin-bottom: 28px;
         }
         .info-grid {
             display: table;
             width: 100%%;
             border-collapse: separate;
             border-spacing: 0;
         }
         .info-row {
             display: table-row;
         }
         .info-label, .info-value {
             display: table-cell;
             padding: 12px 16px;
             border-bottom: 1px solid #f1f5f9;
             vertical-align: top;
         }
         .info-label {
             font-weight: 600;
             color: #2d3748;
             background-color: #f8fafc;
             width: 120px;
             border-right: 1px solid #f1f5f9;
         }
         .info-value {
             color: #4a5568;
             background-color: #ffffff;
         }
         .info-grid .info-row:last-child .info-label,
         .info-grid .info-row:last-child .info-value {
             border-bottom: none;
         }
         .priority-badge {
             display: inline-block;
             padding: 4px 12px;
             border-radius: 20px;
             font-size: 12px;
             font-weight: 600;
             text-transform: uppercase;
             letter-spacing: 0.5px;
         }
         .priority-high {
             background-color: #fed7d7;
             color: #c53030;
         }
         .priority-medium {
             background-color: #feebc8;
             color: #d69e2e;
         }
         .priority-low {
             background-color: #c6f6d5;
             color: #38a169;
         }
         .status-badge {
             display: inline-block;
             padding: 4px 12px;
             border-radius: 20px;
             font-size: 12px;
             font-weight: 600;
             background-color: #e6fffa;
             color: #319795;
         }
         .message-section {
             margin: 28px 0;
         }
         .message-title {
             font-size: 16px;
             font-weight: 600;
             color: #2d3748;
             margin-bottom: 12px;
             display: flex;
             align-items: center;
         }
         .message-content {
             background: linear-gradient(135deg, #f8fafc 0%%, #edf2f7 100%%);
             border: 1px solid #e2e8f0;
             border-radius: 8px;
             padding: 20px;
             white-space: pre-wrap;
             word-wrap: break-word;
             font-size: 14px;
             line-height: 1.7;
             color: #4a5568;
             border-left: 4px solid #3182ce;
         }
         .action-section {
             margin-top: 32px;
             padding: 20px;
             background-color: #f8fafc;
             border-radius: 8px;
             border: 1px solid #e2e8f0;
             text-align: center;
         }
         .action-title {
             font-size: 14px;
             color: #4a5568;
             margin-bottom: 16px;
         }
         .btn {
             display: inline-block;
             padding: 12px 24px;
             background-color: #3182ce;
             color: white;
             text-decoration: none;
             border-radius: 6px;
             font-weight: 600;
             font-size: 14px;
             margin: 0 8px;
             transition: background-color 0.2s;
         }
         .btn:hover {
             background-color: #2c5aa0;
         }
         .btn-secondary {
             background-color: #718096;
         }
         .btn-secondary:hover {
             background-color: #4a5568;
         }
         .footer {
             background-color: #2d3748;
             color: #a0aec0;
             padding: 24px;
             text-align: center;
             font-size: 12px;
         }
         .footer-content {
             margin-bottom: 16px;
         }
         .footer-links {
             margin-bottom: 12px;
         }
         .footer-links a {
             color: #63b3ed;
             text-decoration: none;
             margin: 0 12px;
         }
         .footer-links a:hover {
             text-decoration: underline;
         }
         .divider {
             height: 1px;
             background: linear-gradient(90deg, transparent 0%%, #e2e8f0 50%%, transparent 100%%);
             margin: 24px 0;
         }
         @media only screen and (max-width: 600px) {
             .email-wrapper { padding: 10px 0; }
             .email-container { margin: 0 10px; border-radius: 8px; }
             .header { padding: 24px 16px; }
             .content { padding: 24px 16px; }
             .info-label, .info-value { padding: 10px 12px; font-size: 14px; }
             .info-label { width: 100px; }
             .message-content { padding: 16px; }
             .btn { display: block; margin: 8px 0; width: 100%%; }
         }
     </style>
 </head>
 <body>
     <div class="email-wrapper">
         <div class="email-container">
             <div class="header">
                 <div class="header-content">
                     <h1>📋 AI-CloudOps</h1>
                     <p class="header-subtitle">智能运维管理平台 - 工单系统通知</p>
                 </div>
             </div>
             
             <div class="content">
                 <div class="greeting">
                     尊敬的 <strong>%s</strong>，您好！
                 </div>
                 
                 <div class="info-section">
                     <div class="info-grid">
                         <div class="info-row">
                             <div class="info-label">通知类型</div>
                             <div class="info-value">%s</div>
                         </div>
                         <div class="info-row">
                             <div class="info-label">接收邮箱</div>
                             <div class="info-value">%s</div>
                         </div>
                         %s
                         <div class="info-row">
                             <div class="info-label">优先级</div>
                             <div class="info-value"><span class="%s">%s</span></div>
                         </div>
                         <div class="info-row">
                             <div class="info-label">发送时间</div>
                             <div class="info-value">%s</div>
                         </div>
                     </div>
                 </div>
                 
                 <div class="divider"></div>
                 
                 <div class="message-section">
                     <div class="message-title">
                         📄 详细信息
                     </div>
                     <div class="message-content">%s</div>
                 </div>
                 
                 <div class="action-section">
                     <div class="action-title">您可以通过以下方式查看详情：</div>
                     <a href="#" class="btn">查看工单详情</a>
                     <a href="#" class="btn btn-secondary">访问系统</a>
                 </div>
             </div>
             
             <div class="footer">
                 <div class="footer-content">
                     <div class="footer-links">
                         <a href="#">帮助中心</a>
                         <a href="#">联系支持</a>
                         <a href="#">系统状态</a>
                     </div>
                     <p>此邮件由 AI-CloudOps 智能运维管理平台自动发送</p>
                     <p>如有疑问，请联系系统管理员或查看帮助文档</p>
                 </div>
             </div>
         </div>
     </div>
 </body>
 </html>`

	// 优先级显示
	var priorityClass, priorityText string
	switch request.Priority {
	case 1:
		priorityClass = "priority-high"
		priorityText = "高优先级"
	case 2:
		priorityClass = "priority-medium"
		priorityText = "中优先级"
	case 3:
		priorityClass = "priority-low"
		priorityText = "低优先级"
	default:
		priorityClass = "priority-medium"
		priorityText = "普通"
	}

	// 工单信息
	instanceInfo := ""
	if request.InstanceID != nil {
		instanceInfo = fmt.Sprintf(`
                         <div class="info-row">
                             <div class="info-label">工单编号</div>
                             <div class="info-value">#%d</div>
                         </div>`, *request.InstanceID)
	}

	// 事件显示名
	eventTypeDisplay := getEventTypeDisplay(request.EventType)

	// 处理收件人名称
	recipientName := request.RecipientName
	if recipientName == "" {
		recipientName = "用户"
	}

	// 处理内容，避免XSS攻击，并替换模板变量
	content := escapeHTML(request.Content)
	
	// 替换模板变量
	content = strings.ReplaceAll(content, "{instanceTitle}", getInstanceTitle(request))
	content = strings.ReplaceAll(content, "{currentTime}", time.Now().Format("2006年01月02日 15:04:05"))

	return fmt.Sprintf(template,
		recipientName,
		eventTypeDisplay,
		request.RecipientAddr,
		instanceInfo,
		priorityClass,
		priorityText,
		time.Now().Format("2006年01月02日 15:04:05"),
		content)
}

// getInstanceTitle 获取工单标题
func getInstanceTitle(request *SendRequest) string {
	if request.Subject != "" {
		return request.Subject
	}
	if request.InstanceID != nil {
		return fmt.Sprintf("工单 #%d", *request.InstanceID)
	}
	return "工单通知"
}

// escapeHTML 转义HTML特殊字符，防止XSS攻击
func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&#39;")
	return s
}

// getEventTypeDisplay 获取事件显示名
func getEventTypeDisplay(eventType string) string {
	switch eventType {
	case "workorder_created":
		return "🆕 工单创建通知"
	case "workorder_updated":
		return "🔄 工单更新通知"
	case "workorder_assigned":
		return "👤 工单分配通知"
	case "workorder_completed":
		return "✅工单完成通知"
	case "workorder_closed":
		return "🔒 工单关闭通知"
	case "workorder_reopened":
		return "🔓 工单重新打开通知"
	case "workorder_commented":
		return "💬 工单评论通知"
	case "workorder_escalated":
		return "⚡ 工单升级通知"
	case "workorder_due_soon":
		return "⏰ 工单即将到期通知"
	case "workorder_overdue":
		return "🚨 工单逾期通知"
	default:
		return "📢 系统通知"
	}
}

// validateEmailAddress 验证邮箱
func validateEmailAddress(email string) bool {
	// 基本格式检查
	if email == "" || len(email) > 254 {
		return false
	}

	// 邮箱验证正则（更严格的验证）
	pattern := `^[a-zA-Z0-9.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`
	re := regexp.MustCompile(pattern)
	return re.MatchString(email)
}
