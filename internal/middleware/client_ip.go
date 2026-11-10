package middleware

import (
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	"github.com/gin-gonic/gin"
)

// UseTrustedRealIP 只在直连对端属于可信代理时采用 X-Real-IP。
// 不读取 X-Forwarded-For，避免调用方伪造来源。proxies 为空时使用回环和私有网段。
func UseTrustedRealIP(engine *gin.Engine, proxies []string) error {
	if engine == nil {
		return fmt.Errorf("http 引擎不能为空")
	}
	if len(proxies) == 0 {
		proxies = config.DefaultTrustedProxies()
	}
	engine.RemoteIPHeaders = []string{"X-Real-IP"}
	if err := engine.SetTrustedProxies(proxies); err != nil {
		return fmt.Errorf("设置可信代理失败: %w", err)
	}
	return nil
}
