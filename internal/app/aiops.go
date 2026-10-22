package app

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
)

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
			r.Out.Header.Set("X-API-Key", a.cfg.AIOps.APIKey)
		},
		Transport:     transport,
		FlushInterval: -1,
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, _ error) {
			http.Error(w, `{"code":502,"message":"AI service unavailable"}`, http.StatusBadGateway)
		},
	}
	a.server.Any("/api/v1/*path", gin.WrapH(proxy))
	return nil
}
