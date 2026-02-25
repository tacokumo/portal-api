package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/tacokumo/portal-api/pkg/auth/session"
	"golang.org/x/time/rate"
)

// RateLimitMiddleware レート制限ミドルウェア
type RateLimitMiddleware struct {
	sessionManager session.Manager
	ipLimiter      *rate.Limiter  // IP単位のグローバル制限
	userLimiter    *rate.Limiter  // ユーザー単位の制限
}

// NewRateLimitMiddleware レート制限ミドルウェアを作成
func NewRateLimitMiddleware(sessionManager session.Manager) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		sessionManager: sessionManager,
		// IP単位: 1秒間に10リクエスト、バースト20
		ipLimiter: rate.NewLimiter(rate.Every(100*time.Millisecond), 20),
		// ユーザー単位: 1秒間に50リクエスト、バースト100
		userLimiter: rate.NewLimiter(rate.Every(20*time.Millisecond), 100),
	}
}

// IPRateLimit IP単位のレート制限
func (m *RateLimitMiddleware) IPRateLimit() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			clientIP := c.RealIP()

			// IPごとのレート制限チェック（簡易版）
			if !m.ipLimiter.Allow() {
				return echo.NewHTTPError(http.StatusTooManyRequests,
					fmt.Sprintf("Rate limit exceeded for IP: %s", clientIP))
			}

			return next(c)
		}
	}
}

// UserRateLimit ユーザー単位のレート制限
func (m *RateLimitMiddleware) UserRateLimit() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			// 認証コンテキストから情報取得
			authCtx, err := GetAuthContext(c)
			if err != nil {
				// 認証されていない場合は IP制限のみ
				return next(c)
			}

			// ユーザーごとのレート制限チェック（簡易版）
			if !m.userLimiter.Allow() {
				return echo.NewHTTPError(http.StatusTooManyRequests,
					fmt.Sprintf("Rate limit exceeded for user: %s", authCtx.User.Login))
			}

			return next(c)
		}
	}
}