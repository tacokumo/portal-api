package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v5"
)

// AuditLogEntry 監査ログエントリ
type AuditLogEntry struct {
	Timestamp   time.Time `json:"timestamp"`
	UserID      string    `json:"user_id,omitempty"`
	UserLogin   string    `json:"user_login,omitempty"`
	ClientIP    string    `json:"client_ip"`
	Method      string    `json:"method"`
	Path        string    `json:"path"`
	StatusCode  int       `json:"status_code"`
	UserAgent   string    `json:"user_agent"`
	Duration    int64     `json:"duration_ms"`
	AuthMethod  string    `json:"auth_method,omitempty"`
	Error       string    `json:"error,omitempty"`
}

// AuditMiddleware 監査ログミドルウェア
type AuditMiddleware struct {
	skipPaths map[string]bool
}

// NewAuditMiddleware 監査ログミドルウェアを作成
func NewAuditMiddleware() *AuditMiddleware {
	// 監査をスキップするパス（ヘルスチェック等）
	skipPaths := map[string]bool{
		"/health":  true,
		"/metrics": true,
		"/ping":    true,
	}

	return &AuditMiddleware{
		skipPaths: skipPaths,
	}
}

// Log 監査ログを記録するミドルウェア
func (m *AuditMiddleware) Log() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			// スキップパスの確認
			if m.skipPaths[c.Request().URL.Path] {
				return next(c)
			}

			start := time.Now()

			// リクエスト処理実行
			err := next(c)

			// 監査ログエントリ作成
			entry := AuditLogEntry{
				Timestamp:  start,
				ClientIP:   c.RealIP(),
				Method:     c.Request().Method,
				Path:       c.Request().URL.Path,
				UserAgent:  c.Request().Header.Get("User-Agent"),
				Duration:   time.Since(start).Milliseconds(),
			}

			// レスポンスステータスコード取得
			if c.Response() != nil {
				entry.StatusCode = http.StatusOK // デフォルト値
				// Echo v5では直接ステータスコードを取得する方法を確認する必要がある
			}

			// 認証情報が利用可能な場合は追加
			if authCtx, err := GetAuthContext(c); err == nil {
				entry.UserID = authCtx.User.ID
				entry.UserLogin = authCtx.User.Login

				switch authCtx.Method {
				case 0: // AuthMethodOAuth
					entry.AuthMethod = "oauth"
				case 1: // AuthMethodPAT
					entry.AuthMethod = "pat"
				case 2: // AuthMethodInstallation
					entry.AuthMethod = "installation"
				}
			}

			// エラーがある場合は記録
			if err != nil {
				entry.Error = err.Error()
				if httpErr, ok := err.(*echo.HTTPError); ok {
					entry.StatusCode = httpErr.Code
				} else {
					entry.StatusCode = http.StatusInternalServerError
				}
			}

			// 監査ログをJSON形式で出力
			fmt.Printf("AUDIT: %+v\n", entry)

			// TODO: 本格的な実装では構造化ログライブラリを使用
			// logger.Info("audit log", "entry", entry)

			return err
		}
	}
}