package app

import (
	"context"
	"fmt"
	jwt "github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	ws "github.com/GoSimplicity/AI-CloudOps/pkg/websocket"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type aiopsPrincipalKey struct{}

const aiopsPrincipalHeader = "X-CloudOps-User-ID"

// registerAIOps keeps the service credential behind the existing JWT/RBAC middleware.
func (a *App) registerAIOps() error {
	if a.cfg.AIOps.BaseURL == "" {
		return nil
	}
	target, err := url.Parse(a.cfg.AIOps.BaseURL)
	if err != nil || target.Host == "" {
		return fmt.Errorf("invalid AIOps base URL")
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ResponseHeaderTimeout = 120 * time.Second
	proxy := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(target)
			r.SetXForwarded()
			r.Out.Header.Del("Authorization")
			r.Out.Header.Del("Cookie")
			r.Out.Header.Del(aiopsPrincipalHeader)
			r.Out.Header.Set(aiopsPrincipalHeader, r.In.Context().Value(aiopsPrincipalKey{}).(string))
			protocols := []string{}
			for _, protocol := range strings.Split(r.Out.Header.Get("Sec-WebSocket-Protocol"), ",") {
				protocol = strings.TrimSpace(protocol)
				if protocol != "" && !strings.HasPrefix(protocol, ws.CredentialProtocolPrefix) {
					protocols = append(protocols, protocol)
				}
			}
			r.Out.Header.Del("Sec-WebSocket-Protocol")
			if len(protocols) > 0 {
				r.Out.Header.Set("Sec-WebSocket-Protocol", strings.Join(protocols, ", "))
			}
			if query := r.Out.URL.Query(); query.Has("token") {
				query.Del("token")
				r.Out.URL.RawQuery = query.Encode()
			}
			r.Out.Header.Set("X-API-Key", a.cfg.AIOps.APIKey)
		},
		Transport:     transport,
		FlushInterval: -1,
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, _ error) {
			http.Error(w, `{"code":502,"message":"AI service unavailable"}`, http.StatusBadGateway)
		},
	}
	a.server.Any("/api/v1/*path", func(c *gin.Context) {
		value, exists := c.Get("user")
		claims, valid := value.(jwt.UserClaims)
		if !exists || !valid || claims.Uid <= 0 {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		request := c.Request.WithContext(context.WithValue(c.Request.Context(), aiopsPrincipalKey{}, strconv.Itoa(claims.Uid)))
		proxy.ServeHTTP(c.Writer, request)
	})
	return nil
}
