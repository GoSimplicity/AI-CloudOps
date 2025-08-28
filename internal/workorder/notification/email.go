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

// Send 发送邮件通知到指定收件人
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

	// 智能检测并设置SMTP配置
	smtpHost, smtpPort, useTLS := e.detectSMTPConfig(e.config.GetUsername())

	// SMTP连接
	d := gomail.NewDialer(smtpHost, smtpPort, e.config.GetUsername(), e.config.GetPassword())

	// 配置TLS
	if useTLS {
		d.TLSConfig = &tls.Config{
			ServerName:         smtpHost,
			InsecureSkipVerify: false,
		}
	}

	// 根据SMTP服务器类型设置StartTLS
	if strings.Contains(smtpHost, "qq.com") && smtpPort == 587 {
		d.SSL = false
	}

	// 发送
	if err := d.DialAndSend(m); err != nil {
		// 解析错误信息，提供更详细的错误说明
		errorMsg := e.parseEmailError(err, smtpHost)

		e.logger.Error("发送邮件失败",
			zap.String("recipient", request.RecipientAddr),
			zap.String("subject", subject),
			zap.String("smtp_host", smtpHost),
			zap.Int("smtp_port", smtpPort),
			zap.Bool("use_tls", useTLS),
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

// parseEmailError 解析邮件错误并返回友好提示信息
func (e *EmailChannel) parseEmailError(err error, smtpHost string) string {
	errStr := err.Error()

	// QQ邮箱特定错误
	if strings.Contains(errStr, "535") && strings.Contains(smtpHost, "qq.com") {
		return "QQ邮箱认证失败：请检查是否使用了授权码而非密码。请前往QQ邮箱设置->账户->POP3/IMAP/SMTP/Exchange/CardDAV/CalDAV服务，生成授权码"
	}

	// 163邮箱特定错误
	if strings.Contains(errStr, "535") && strings.Contains(smtpHost, "163.com") {
		return "163邮箱认证失败：请检查是否使用了授权码而非密码。请前往163邮箱设置->POP3/SMTP/IMAP，开启服务并获取授权码"
	}

	// 126邮箱特定错误
	if strings.Contains(errStr, "535") && strings.Contains(smtpHost, "126.com") {
		return "126邮箱认证失败：请检查是否使用了授权码而非密码。请前往126邮箱设置开启SMTP服务并获取授权码"
	}

	// Gmail特定错误
	if strings.Contains(errStr, "535") && strings.Contains(smtpHost, "gmail.com") {
		return "Gmail认证失败：请检查是否启用了两步验证并使用应用密码。请前往Google账户设置->安全->两步验证->应用密码生成专用密码"
	}

	// Outlook特定错误
	if strings.Contains(errStr, "535") && strings.Contains(smtpHost, "outlook.com") {
		return "Outlook认证失败：请检查账号密码是否正确，或考虑使用应用密码"
	}

	// 通用错误处理
	switch {
	case strings.Contains(errStr, "535"):
		return "SMTP认证失败：账号或密码错误，或需要使用授权码/应用密码"
	case strings.Contains(errStr, "550"):
		return "发送失败：收件人地址无效或被拒绝，请检查收件人邮箱地址是否正确"
	case strings.Contains(errStr, "551"):
		return "发送失败：用户不在本地，邮箱地址可能不存在"
	case strings.Contains(errStr, "552"):
		return "发送失败：邮箱存储空间已满"
	case strings.Contains(errStr, "553"):
		return "发送失败：邮箱地址格式错误"
	case strings.Contains(errStr, "554"):
		return "发送失败：邮件内容被拒绝，可能被识别为垃圾邮件或包含敏感内容"
	case strings.Contains(errStr, "connection refused"):
		return "连接失败：无法连接到SMTP服务器，请检查服务器地址和端口是否正确"
	case strings.Contains(errStr, "timeout"):
		return "连接超时：SMTP服务器响应超时，请检查网络连接或稍后重试"
	case strings.Contains(errStr, "certificate"):
		return "证书错误：TLS证书验证失败，请检查SMTP服务器证书"
	case strings.Contains(errStr, "no such host"):
		return "域名解析失败：无法解析SMTP服务器域名，请检查服务器地址"
	case strings.Contains(errStr, "network is unreachable"):
		return "网络不可达：请检查网络连接"
	case strings.Contains(errStr, "authentication failed"):
		return "认证失败：用户名或密码错误"
	default:
		return fmt.Sprintf("邮件发送失败：%s", errStr)
	}
}

// Validate 验证邮件通道配置有效性
func (e *EmailChannel) Validate() error {
	// 验证通道是否启用
	if !e.config.IsEnabled() {
		return nil // 如果未启用，跳过验证
	}

	// 验证用户名（发件人邮箱）
	if e.config.GetUsername() == "" {
		return fmt.Errorf("SMTP username (sender email) is required")
	}

	// 验证邮箱格式
	if !validateEmailAddress(e.config.GetUsername()) {
		return fmt.Errorf("invalid sender email format: %s", e.config.GetUsername())
	}
	// 验证密码
	if e.config.GetPassword() == "" {
		return fmt.Errorf("SMTP password is required")
	}

	// 检测SMTP配置
	detectedHost, detectedPort, detectedTLS := e.detectSMTPConfig(e.config.GetUsername())

	// 如果配置了自定义SMTP主机，验证其有效性
	configuredHost := e.config.GetSMTPHost()
	if configuredHost != "" && configuredHost != "smtp.gmail.com" {
		// 验证主机名格式
		if !strings.Contains(configuredHost, ".") {
			return fmt.Errorf("invalid SMTP host format: %s", configuredHost)
		}

		// 验证端口
		configuredPort := e.config.GetSMTPPort()
		if configuredPort <= 0 || configuredPort > 65535 {
			return fmt.Errorf("invalid SMTP port: %d", configuredPort)
		}

		// 验证端口合法性
		validPorts := []int{25, 465, 587, 993, 995, 2525}
		isValidPort := false
		for _, port := range validPorts {
			if configuredPort == port {
				isValidPort = true
				break
			}
		}
		if !isValidPort {
			e.logger.Warn("使用非标准SMTP端口",
				zap.Int("port", configuredPort),
				zap.String("host", configuredHost))
		}
	} else {
		// 使用检测到的配置，验证其有效性
		if detectedHost == "" {
			return fmt.Errorf("无法为邮箱 %s 检测到合适的SMTP配置", e.config.GetUsername())
		}

		e.logger.Info("使用自动检测的SMTP配置",
			zap.String("email", e.config.GetUsername()),
			zap.String("detected_host", detectedHost),
			zap.Int("detected_port", detectedPort),
			zap.Bool("detected_tls", detectedTLS))
	}

	// 验证重试配置
	if e.config.GetMaxRetries() < 0 || e.config.GetMaxRetries() > 10 {
		e.logger.Warn("重试次数设置异常", zap.Int("max_retries", e.config.GetMaxRetries()))
	}

	// 验证超时配置
	timeout := e.config.GetTimeout()
	if timeout < 5*time.Second || timeout > 120*time.Second {
		e.logger.Warn("超时时间设置异常", zap.Duration("timeout", timeout))
	}

	// 验证重试间隔
	retryInterval := e.config.GetRetryInterval()
	if retryInterval < 1*time.Second || retryInterval > 30*time.Minute {
		e.logger.Warn("重试间隔设置异常", zap.Duration("retry_interval", retryInterval))
	}

	// 验证发件人名称
	fromName := e.config.GetFromName()
	if fromName == "" {
		e.logger.Warn("未设置发件人名称，将使用默认名称")
	} else if len(fromName) > 100 {
		return fmt.Errorf("发件人名称过长，最大支持100字符")
	}

	return e.config.Validate()
}

// IsEnabled 检查通道是否启用
func (e *EmailChannel) IsEnabled() bool {
	return e.config.IsEnabled()
}

// GetMaxRetries 获取最大重试次数
func (e *EmailChannel) GetMaxRetries() int {
	return e.config.GetMaxRetries()
}

// GetRetryInterval 获取重试间隔
func (e *EmailChannel) GetRetryInterval() time.Duration {
	return e.config.GetRetryInterval()
}

// buildEmailContent 构建邮件内容
func (e *EmailChannel) buildEmailContent(request *SendRequest) string {
	template := `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta http-equiv="X-UA-Compatible" content="IE=edge">
    <title>AI-CloudOps 工单通知</title>
    <style>
        * { 
            box-sizing: border-box; 
            margin: 0; 
            padding: 0; 
        }
        
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'SF Pro Text', 'Helvetica Neue', 'PingFang SC', 'Microsoft YaHei', sans-serif;
            line-height: 1.6;
            color: #1a1a1a;
            background-color: #f5f5f5;
            font-size: 16px;
            -webkit-font-smoothing: antialiased;
            -moz-osx-font-smoothing: grayscale;
        }
        
        .email-wrapper {
            width: 100%%;
            background-color: #f5f5f5;
            padding: 40px 20px;
            min-height: 100vh;
        }
        
        .email-container {
            max-width: 640px;
            margin: 0 auto;
            background-color: #ffffff;
            border-radius: 8px;
            box-shadow: 0 4px 20px rgba(0, 0, 0, 0.08);
            overflow: hidden;
            border: 1px solid #e5e5e5;
        }
        
        /* 头部区域 - 简洁商务风格 */
        .header {
            background: linear-gradient(135deg, #2563eb 0%%, #1d4ed8 100%%);
            color: white;
            padding: 32px 24px;
            text-align: left;
            border-bottom: 1px solid rgba(255, 255, 255, 0.1);
        }
        
        .header-content {
            display: flex;
            align-items: center;
            justify-content: space-between;
        }
        
        .logo-section {
            display: flex;
            align-items: center;
        }
        
        .logo-icon {
            width: 40px;
            height: 40px;
            background: rgba(255, 255, 255, 0.15);
            border-radius: 8px;
            display: flex;
            align-items: center;
            justify-content: center;
            margin-right: 12px;
            font-size: 20px;
        }
        
        .logo-text {
            font-size: 20px;
            font-weight: 600;
            letter-spacing: -0.5px;
        }
        
        .notification-type {
            background: rgba(255, 255, 255, 0.15);
            padding: 6px 12px;
            border-radius: 20px;
            font-size: 12px;
            font-weight: 500;
            text-transform: uppercase;
            letter-spacing: 0.5px;
        }
        
        /* 内容区域 */
        .content {
            padding: 32px 24px;
        }
        
        .greeting {
            font-size: 16px;
            color: #4a4a4a;
            margin-bottom: 24px;
            line-height: 1.5;
        }
        
        /* 工单信息卡片 */
        .workorder-card {
            background: #fafafa;
            border: 1px solid #e5e5e5;
            border-radius: 8px;
            padding: 24px;
            margin-bottom: 24px;
        }
        
        .workorder-header {
            display: flex;
            align-items: center;
            justify-content: space-between;
            margin-bottom: 20px;
            padding-bottom: 16px;
            border-bottom: 1px solid #e5e5e5;
        }
        
        .workorder-title {
            font-size: 14px;
            font-weight: 600;
            color: #1a1a1a;
            text-transform: uppercase;
            letter-spacing: 0.5px;
        }
        
        .priority-badge {
            padding: 4px 8px;
            border-radius: 4px;
            font-size: 11px;
            font-weight: 600;
            text-transform: uppercase;
            letter-spacing: 0.5px;
        }
        
        .priority-high {
            background: #fee2e2;
            color: #dc2626;
            border: 1px solid #fecaca;
        }
        
        .priority-medium {
            background: #fef3c7;
            color: #d97706;
            border: 1px solid #fed7aa;
        }
        
        .priority-low {
            background: #d1fae5;
            color: #059669;
            border: 1px solid #a7f3d0;
        }
        
        .info-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 16px;
        }
        
        .info-item {
            display: flex;
            flex-direction: column;
        }
        
        .info-label {
            font-size: 12px;
            color: #8a8a8a;
            font-weight: 500;
            margin-bottom: 4px;
            text-transform: uppercase;
            letter-spacing: 0.5px;
        }
        
        .info-value {
            font-size: 14px;
            color: #1a1a1a;
            font-weight: 500;
            word-break: break-all;
        }
        
        /* 消息内容区域 */
        .message-section {
            margin: 24px 0;
        }
        
        .message-card {
            background: white;
            border: 1px solid #e5e5e5;
            border-left: 4px solid #2563eb;
            border-radius: 8px;
            padding: 20px;
        }
        
        .message-header {
            font-size: 14px;
            font-weight: 600;
            color: #1a1a1a;
            margin-bottom: 12px;
            text-transform: uppercase;
            letter-spacing: 0.5px;
        }
        
        .message-content {
            font-size: 14px;
            line-height: 1.6;
            color: #4a4a4a;
            white-space: pre-wrap;
            word-wrap: break-word;
        }
        
        /* 操作按钮区域 */
        .action-section {
            margin-top: 32px;
            padding: 24px;
            background: #fafafa;
            border: 1px solid #e5e5e5;
            border-radius: 8px;
            text-align: center;
        }
        
        .action-title {
            font-size: 14px;
            color: #4a4a4a;
            margin-bottom: 16px;
            font-weight: 500;
        }
        
        .btn {
            display: inline-block;
            padding: 12px 24px;
            margin: 4px 8px;
            border-radius: 6px;
            font-weight: 500;
            font-size: 14px;
            text-decoration: none;
            transition: all 0.2s ease;
            border: 1px solid;
            min-width: 140px;
        }
        
        .btn-primary {
            background: #2563eb;
            color: white;
            border-color: #2563eb;
        }
        
        .btn-primary:hover {
            background: #1d4ed8;
            border-color: #1d4ed8;
        }
        
        .btn-secondary {
            background: white;
            color: #4a4a4a;
            border-color: #d1d5db;
        }
        
        .btn-secondary:hover {
            background: #f9fafb;
            border-color: #9ca3af;
        }
        
        /* 页脚区域 */
        .footer {
            background: #1a1a1a;
            color: #a3a3a3;
            padding: 24px;
            text-align: center;
        }
        
        .footer-brand {
            font-size: 16px;
            font-weight: 600;
            color: white;
            margin-bottom: 12px;
        }
        
        .footer-links {
            margin-bottom: 16px;
        }
        
        .footer-links a {
            color: #3b82f6;
            text-decoration: none;
            margin: 0 12px;
            font-size: 13px;
            font-weight: 500;
        }
        
        .footer-links a:hover {
            color: #60a5fa;
            text-decoration: underline;
        }
        
        .footer-text {
            font-size: 12px;
            line-height: 1.5;
            margin-bottom: 6px;
            color: #737373;
        }
        
        .footer-text:last-child {
            margin-bottom: 0;
        }
        
        /* 分隔线 */
        .divider {
            height: 1px;
            background: #e5e5e5;
            margin: 24px 0;
        }
        
        /* 响应式设计 */
        @media only screen and (max-width: 600px) {
            .email-wrapper { 
                padding: 20px 10px; 
            }
            
            .email-container { 
                margin: 0; 
                border-radius: 6px; 
            }
            
            .header { 
                padding: 24px 16px; 
            }
            
            .content { 
                padding: 24px 16px; 
            }
            
            .workorder-card {
                padding: 16px;
            }
            
            .info-grid { 
                grid-template-columns: 1fr; 
                gap: 12px;
            }
            
            .header-content {
                flex-direction: column;
                align-items: flex-start;
                gap: 12px;
            }
            
            .btn { 
                display: block; 
                margin: 8px 0; 
                width: 100%%; 
            }
            
            .action-section { 
                padding: 20px 16px; 
            }
        }
    </style>
</head>
<body>
    <div class="email-wrapper">
        <div class="email-container">
            <div class="header">
                <div class="header-content">
                    <div class="logo-section">
                        <div class="logo-icon">⚡</div>
                        <div class="logo-text">AI-CloudOps</div>
                    </div>
                    <div class="notification-type">%s</div>
                </div>
            </div>
            
            <div class="content">
                <div class="greeting">
                    尊敬的 <strong>%s</strong>，您好！<br>
                    您收到一条新的工单通知，请及时查看处理。
                </div>
                
                <div class="workorder-card">
                    <div class="workorder-header">
                        <div class="workorder-title">工单信息</div>
                        <div class="priority-badge %s">%s</div>
                    </div>
                    <div class="info-grid">
                        <div class="info-item">
                            <div class="info-label">工单编号</div>
                            <div class="info-value">%s</div>
                        </div>
                        <div class="info-item">
                            <div class="info-label">接收邮箱</div>
                            <div class="info-value">%s</div>
                        </div>
                        <div class="info-item">
                            <div class="info-label">通知时间</div>
                            <div class="info-value">%s</div>
                        </div>
                    </div>
                </div>
                
                <div class="divider"></div>
                
                <div class="message-section">
                    <div class="message-card">
                        <div class="message-header">通知内容</div>
                        <div class="message-content">%s</div>
                    </div>
                </div>
                
                <div class="action-section">
                    <div class="action-title">请登录系统查看详细信息或进行相关操作</div>
                    <a href="#" class="btn btn-primary">立即查看</a>
                    <a href="#" class="btn btn-secondary">管理平台</a>
                </div>
            </div>
            
            <div class="footer">
                <div class="footer-brand">AI-CloudOps 智能运维管理平台</div>
                <div class="footer-links">
                    <a href="#">帮助中心</a>
                    <a href="#">技术支持</a>
                    <a href="#">系统状态</a>
                </div>
                <div class="footer-text">此邮件由AI-CloudOps系统自动发送，请勿直接回复</div>
                <div class="footer-text">如有疑问请联系技术支持 | 服务热线：400-000-0000</div>
                <div class="footer-text">Copyright © 2024 AI-CloudOps. All rights reserved.</div>
            </div>
        </div>
    </div>
</body>
</html>`

	// 优先级显示配置
	var priorityClass, priorityText string
	switch request.Priority {
	case 1:
		priorityClass = "priority-high"
		priorityText = "高优先级 HIGH"
	case 2:
		priorityClass = "priority-medium"
		priorityText = "中优先级 MEDIUM"
	case 3:
		priorityClass = "priority-low"
		priorityText = "低优先级 LOW"
	default:
		priorityClass = "priority-medium"
		priorityText = "普通 NORMAL"
	}

	// 获取工单编号
	workorderNumber := "系统通知"
	if request.InstanceID != nil {
		workorderNumber = fmt.Sprintf("#%d", *request.InstanceID)
	}

	// 事件类型显示
	eventTypeDisplay := fmt.Sprintf("%s %s", GetEventTypeIcon(request.EventType), GetEventTypeText(request.EventType))

	// 处理收件人名称
	recipientName := request.RecipientName
	if recipientName == "" {
		recipientName = "尊敬的用户"
	}

	// 先对内容进行模板渲染，然后进行HTML转义以防止XSS攻击
	renderedContent, err := RenderTemplate(request.Content, request)
	if err != nil {
		renderedContent = request.Content // 渲染失败时使用原始内容
	}
	content := escapeHTML(renderedContent)

	return fmt.Sprintf(template,
		eventTypeDisplay,                      // 通知类型徽章
		recipientName,                         // 收件人名称
		priorityClass,                         // 优先级CSS类
		priorityText,                          // 优先级文本
		workorderNumber,                       // 工单编号
		request.RecipientAddr,                 // 邮箱地址
		time.Now().Format("2006-01-02 15:04"), // 发送时间
		content)                               // 消息内容
}

// escapeHTML 转义HTML特殊字符防止XSS攻击
func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&#39;")
	return s
}

// detectSMTPConfig 根据邮箱地址自动检测SMTP配置
func (e *EmailChannel) detectSMTPConfig(username string) (string, int, bool) {
	// 如果配置中明确设置了SMTP主机，优先使用配置
	if e.config.GetSMTPHost() != "" && e.config.GetSMTPHost() != "smtp.gmail.com" {
		return e.config.GetSMTPHost(), e.config.GetSMTPPort(), e.config.GetUseTLS()
	}

	// 从用户名中提取域名
	if username == "" || !strings.Contains(username, "@") {
		// 如果没有有效的用户名，使用配置默认值
		return e.config.GetSMTPHost(), e.config.GetSMTPPort(), e.config.GetUseTLS()
	}

	domain := strings.ToLower(strings.Split(username, "@")[1])

	// 根据域名选择合适的SMTP配置
	switch domain {
	case "qq.com":
		return "smtp.qq.com", 587, true
	case "163.com":
		return "smtp.163.com", 465, true
	case "126.com":
		return "smtp.126.com", 465, true
	case "gmail.com":
		return "smtp.gmail.com", 587, true
	case "outlook.com", "hotmail.com", "live.com":
		return "smtp-mail.outlook.com", 587, true
	case "yahoo.com":
		return "smtp.mail.yahoo.com", 587, true
	case "sina.com":
		return "smtp.sina.com", 465, true
	case "sohu.com":
		return "smtp.sohu.com", 465, true
	case "foxmail.com":
		return "smtp.qq.com", 587, true
	default:
		// 对于未知域名，使用配置中的默认值
		e.logger.Warn("未识别的邮箱域名，使用默认SMTP配置",
			zap.String("domain", domain),
			zap.String("default_smtp", e.config.GetSMTPHost()))
		return e.config.GetSMTPHost(), e.config.GetSMTPPort(), e.config.GetUseTLS()
	}
}

// validateEmailAddress 验证邮箱地址格式
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
