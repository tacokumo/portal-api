package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v5"
	"github.com/tacokumo/portal-api/pkg/config"
)

// CORSMiddleware Cross-Origin Resource Sharing保護ミドルウェア
type CORSMiddleware struct {
	allowedOrigins   []string
	allowedMethods   string // カンマ区切りのヘッダー値
	allowedHeaders   string // カンマ区切りのヘッダー値
	allowCredentials bool
	maxAge           string
}

// NewCORSMiddleware CORSミドルウェアを作成
func NewCORSMiddleware(cfg config.CORSConfig) *CORSMiddleware {
	methods := cfg.AllowedMethods
	if len(methods) == 0 {
		methods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	}

	headers := cfg.AllowedHeaders
	if len(headers) == 0 {
		headers = []string{"Authorization", "Content-Type", "X-CSRF-Token", "X-Requested-With"}
	}

	maxAge := cfg.MaxAge
	if maxAge == 0 {
		maxAge = 86400
	}

	return &CORSMiddleware{
		allowedOrigins:   cfg.AllowedOrigins,
		allowedMethods:   strings.Join(methods, ", "),
		allowedHeaders:   strings.Join(headers, ", "),
		allowCredentials: cfg.AllowCredentials,
		maxAge:           strconv.Itoa(maxAge),
	}
}

// Handle CORS保護を提供するミドルウェア
func (m *CORSMiddleware) Handle() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			origin := c.Request().Header.Get("Origin")

			// Originヘッダーなし → CORSリクエストではない
			if origin == "" {
				return next(c)
			}

			// オリジン許可チェック
			if !m.isOriginAllowed(origin) {
				// 不許可: CORSヘッダーを設定せずに通す（ブラウザ側でブロック）
				if c.Request().Method == http.MethodOptions {
					return c.NoContent(http.StatusNoContent)
				}
				return next(c)
			}

			h := c.Response().Header()

			// Access-Control-Allow-Origin設定
			// ワイルドカード"*"とcredentials=trueの組み合わせはW3C仕様で禁止
			// されているため、実際のオリジンを反映する
			if m.allowCredentials {
				h.Set("Access-Control-Allow-Origin", origin)
				h.Set("Vary", "Origin")
			} else if m.hasWildcardOrigin() {
				h.Set("Access-Control-Allow-Origin", "*")
			} else {
				h.Set("Access-Control-Allow-Origin", origin)
				h.Set("Vary", "Origin")
			}

			if m.allowCredentials {
				h.Set("Access-Control-Allow-Credentials", "true")
			}

			// プリフライトリクエスト
			if c.Request().Method == http.MethodOptions {
				h.Set("Access-Control-Allow-Methods", m.allowedMethods)
				h.Set("Access-Control-Allow-Headers", m.allowedHeaders)
				h.Set("Access-Control-Max-Age", m.maxAge)
				return c.NoContent(http.StatusNoContent)
			}

			return next(c)
		}
	}
}

// isOriginAllowed オリジンが許可リストに含まれるかチェック
func (m *CORSMiddleware) isOriginAllowed(origin string) bool {
	if len(m.allowedOrigins) == 0 {
		return false
	}

	for _, pattern := range m.allowedOrigins {
		if pattern == "*" {
			return true
		}
		// 末尾ワイルドカード（例: http://localhost:*）
		if strings.HasSuffix(pattern, "*") {
			prefix := strings.TrimSuffix(pattern, "*")
			if strings.HasPrefix(origin, prefix) {
				return true
			}
			continue
		}
		if pattern == origin {
			return true
		}
	}

	return false
}

// hasWildcardOrigin AllowedOriginsに"*"が含まれるかチェック
func (m *CORSMiddleware) hasWildcardOrigin() bool {
	for _, o := range m.allowedOrigins {
		if o == "*" {
			return true
		}
	}
	return false
}
