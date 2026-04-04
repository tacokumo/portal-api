package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/tacokumo/portal-api/pkg/config"
)

// RateLimitMiddleware provides distributed rate limiting using a RateLimitStore backend.
type RateLimitMiddleware struct {
	store  RateLimitStore
	config config.RateLimitConfig
	logger *slog.Logger
}

// NewRateLimitMiddleware creates a rate limit middleware backed by the given store.
func NewRateLimitMiddleware(store RateLimitStore, cfg config.RateLimitConfig) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		store:  store,
		config: cfg,
		logger: slog.Default(),
	}
}

// IPRateLimit returns middleware that enforces per-IP rate limiting.
func (m *RateLimitMiddleware) IPRateLimit() echo.MiddlewareFunc {
	period := time.Duration(m.config.PeriodSeconds) * time.Second

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			clientIP := c.RealIP()
			key := "ip:" + clientIP

			result, err := m.store.Allow(
				c.Request().Context(),
				key,
				m.config.IPRate,
				m.config.IPBurst,
				period,
			)
			if err != nil {
				m.logger.Warn("rate limit check failed", "error", err, "ip", clientIP)
				if m.config.FailOpen {
					return next(c)
				}
				return echo.NewHTTPError(http.StatusServiceUnavailable, "rate limit service unavailable")
			}

			setRateLimitHeaders(c, result)

			if !result.Allowed {
				return echo.NewHTTPError(http.StatusTooManyRequests,
					fmt.Sprintf("Rate limit exceeded for IP: %s", clientIP))
			}

			return next(c)
		}
	}
}

// UserRateLimit returns middleware that enforces per-user rate limiting.
// Unauthenticated requests are passed through (IP-level limiting still applies).
func (m *RateLimitMiddleware) UserRateLimit() echo.MiddlewareFunc {
	period := time.Duration(m.config.PeriodSeconds) * time.Second

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			authCtx, err := GetAuthContext(c)
			if err != nil {
				// Not authenticated — skip user rate limit
				return next(c)
			}

			key := "user:" + authCtx.User.Login

			result, err := m.store.Allow(
				c.Request().Context(),
				key,
				m.config.UserRate,
				m.config.UserBurst,
				period,
			)
			if err != nil {
				m.logger.Warn("rate limit check failed", "error", err, "user", authCtx.User.Login)
				if m.config.FailOpen {
					return next(c)
				}
				return echo.NewHTTPError(http.StatusServiceUnavailable, "rate limit service unavailable")
			}

			setRateLimitHeaders(c, result)

			if !result.Allowed {
				return echo.NewHTTPError(http.StatusTooManyRequests,
					fmt.Sprintf("Rate limit exceeded for user: %s", authCtx.User.Login))
			}

			return next(c)
		}
	}
}

// setRateLimitHeaders writes standard rate limit headers to the response.
func setRateLimitHeaders(c *echo.Context, result *RateLimitResult) {
	h := c.Response().Header()
	h.Set("X-RateLimit-Limit", strconv.Itoa(result.Limit))
	h.Set("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
	h.Set("X-RateLimit-Reset", strconv.FormatInt(result.ResetAt.Unix(), 10))

	if !result.Allowed {
		retryAfterSec := int64(result.RetryAfter.Seconds())
		if retryAfterSec < 1 {
			retryAfterSec = 1
		}
		h.Set("Retry-After", strconv.FormatInt(retryAfterSec, 10))
	}
}
