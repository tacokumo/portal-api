package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"
	"github.com/tacokumo/portal-api/pkg/auth/session"
)

// CSRFMiddleware CSRF保護ミドルウェア
type CSRFMiddleware struct {
	sessionManager session.Manager
	secureCookies  bool
	skipPaths      map[string]bool
}

// NewCSRFMiddleware CSRF保護ミドルウェアを作成
func NewCSRFMiddleware(sessionManager session.Manager, secureCookies bool) *CSRFMiddleware {
	// CSRF保護をスキップするパス
	skipPaths := map[string]bool{
		"/auth/github/login":    true,
		"/auth/github/callback": true,
		"/health":               true,
		"/metrics":              true,
	}

	return &CSRFMiddleware{
		sessionManager: sessionManager,
		secureCookies:  secureCookies,
		skipPaths:      skipPaths,
	}
}

// Protect CSRF保護を提供するミドルウェア
func (m *CSRFMiddleware) Protect() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			// スキップパスの確認
			if m.skipPaths[c.Request().URL.Path] {
				return next(c)
			}

			// GET、HEAD、OPTIONSリクエストはスキップ
			method := c.Request().Method
			if method == "GET" || method == "HEAD" || method == "OPTIONS" {
				return next(c)
			}

			// API認証（Bearer token）の場合はスキップ
			authHeader := c.Request().Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				return next(c)
			}

			// セッションCookieがない場合はスキップ
			sessionCookie, err := c.Cookie("portal_session")
			if err != nil {
				return next(c)
			}

			// CSRFトークンを取得
			var csrfToken string

			// ヘッダーから取得
			csrfToken = c.Request().Header.Get("X-CSRF-Token")

			// ヘッダーにない場合はフォームデータから取得
			if csrfToken == "" {
				csrfToken = c.FormValue("_csrf_token")
			}

			if csrfToken == "" {
				return echo.NewHTTPError(http.StatusForbidden, "CSRF token required")
			}

			// CSRFトークン検証
			if err := m.validateCSRFToken(sessionCookie.Value, csrfToken); err != nil {
				return echo.NewHTTPError(http.StatusForbidden, "Invalid CSRF token")
			}

			return next(c)
		}
	}
}

// GenerateCSRFToken CSRFトークンを生成してセッションに保存
func (m *CSRFMiddleware) GenerateCSRFToken(sessionID string) (string, error) {
	// ランダムなCSRFトークンを生成
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("failed to generate CSRF token: %w", err)
	}

	token := base64.URLEncoding.EncodeToString(tokenBytes)

	// Valkeyに保存（30分有効）
	// key := session.KeyPrefixCSRF + sessionID
	// TODO: ValkeyClientでCSRFトークン保存実装

	return token, nil
}

// validateCSRFToken CSRFトークンを検証
func (m *CSRFMiddleware) validateCSRFToken(sessionToken string, csrfToken string) error {
	// TODO: セッションIDを取得してValkeyからCSRFトークンを確認
	// 現在は簡易実装として常に検証成功
	if csrfToken == "" {
		return fmt.Errorf("empty CSRF token")
	}

	return nil
}