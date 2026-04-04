package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tacokumo/portal-api/pkg/auth"
	"github.com/tacokumo/portal-api/pkg/config"
)

// mockRateLimitStore is a test double for RateLimitStore.
type mockRateLimitStore struct {
	AllowFn func(ctx context.Context, key string, rate int, burst int, period time.Duration) (*RateLimitResult, error)
}

func (m *mockRateLimitStore) Allow(ctx context.Context, key string, rate int, burst int, period time.Duration) (*RateLimitResult, error) {
	if m.AllowFn != nil {
		return m.AllowFn(ctx, key, rate, burst, period)
	}
	return &RateLimitResult{
		Allowed:   true,
		Limit:     burst,
		Remaining: burst - 1,
		ResetAt:   time.Now().Add(time.Minute),
	}, nil
}

func defaultRateLimitConfig() config.RateLimitConfig {
	return config.RateLimitConfig{
		IPRate:        60,
		IPBurst:       20,
		UserRate:      300,
		UserBurst:     60,
		PeriodSeconds: 60,
		FailOpen:      true,
	}
}

func setupRateLimitTest(t *testing.T, method, path string) (*echo.Context, *httptest.ResponseRecorder) {
	t.Helper()
	e := echo.New()
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("X-Real-Ip", "1.2.3.4")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	return c, rec
}

func setAuthContext(c *echo.Context, user auth.User) {
	authCtx := &auth.AuthContext{
		User:   user,
		Method: auth.AuthMethodPAT,
	}
	ctx := context.WithValue(c.Request().Context(), AuthContextKey, authCtx)
	c.SetRequest(c.Request().WithContext(ctx))
}

// --- IP Rate Limit Tests ---

func TestIPRateLimit_Allowed(t *testing.T) {
	t.Parallel()

	store := &mockRateLimitStore{
		AllowFn: func(_ context.Context, key string, _ int, burst int, _ time.Duration) (*RateLimitResult, error) {
			assert.Equal(t, "ip:1.2.3.4", key)
			return &RateLimitResult{
				Allowed:   true,
				Limit:     burst,
				Remaining: burst - 1,
				ResetAt:   time.Now().Add(time.Minute),
			}, nil
		},
	}

	cfg := defaultRateLimitConfig()
	mw := NewRateLimitMiddleware(store, cfg)
	c, rec := setupRateLimitTest(t, http.MethodGet, "/health")

	handler := mw.IPRateLimit()(func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.NotEmpty(t, rec.Header().Get("X-RateLimit-Limit"))
	assert.NotEmpty(t, rec.Header().Get("X-RateLimit-Remaining"))
	assert.NotEmpty(t, rec.Header().Get("X-RateLimit-Reset"))
	assert.Empty(t, rec.Header().Get("Retry-After"))
}

func TestIPRateLimit_Blocked(t *testing.T) {
	t.Parallel()

	store := &mockRateLimitStore{
		AllowFn: func(_ context.Context, _ string, _ int, burst int, _ time.Duration) (*RateLimitResult, error) {
			return &RateLimitResult{
				Allowed:    false,
				Limit:      burst,
				Remaining:  0,
				RetryAfter: 5 * time.Second,
				ResetAt:    time.Now().Add(5 * time.Second),
			}, nil
		},
	}

	cfg := defaultRateLimitConfig()
	mw := NewRateLimitMiddleware(store, cfg)
	c, rec := setupRateLimitTest(t, http.MethodGet, "/health")

	handler := mw.IPRateLimit()(func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	require.Error(t, err)

	var httpErr *echo.HTTPError
	require.ErrorAs(t, err, &httpErr)
	assert.Equal(t, http.StatusTooManyRequests, httpErr.Code)
	assert.NotEmpty(t, rec.Header().Get("Retry-After"))
}

func TestIPRateLimit_ValkeyError_FailOpen(t *testing.T) {
	t.Parallel()

	store := &mockRateLimitStore{
		AllowFn: func(_ context.Context, _ string, _ int, _ int, _ time.Duration) (*RateLimitResult, error) {
			return nil, errors.New("valkey connection refused")
		},
	}

	cfg := defaultRateLimitConfig()
	cfg.FailOpen = true
	mw := NewRateLimitMiddleware(store, cfg)
	c, rec := setupRateLimitTest(t, http.MethodGet, "/health")

	handler := mw.IPRateLimit()(func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestIPRateLimit_ValkeyError_FailClosed(t *testing.T) {
	t.Parallel()

	store := &mockRateLimitStore{
		AllowFn: func(_ context.Context, _ string, _ int, _ int, _ time.Duration) (*RateLimitResult, error) {
			return nil, errors.New("valkey connection refused")
		},
	}

	cfg := defaultRateLimitConfig()
	cfg.FailOpen = false
	mw := NewRateLimitMiddleware(store, cfg)
	c, _ := setupRateLimitTest(t, http.MethodGet, "/health")

	handler := mw.IPRateLimit()(func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	require.Error(t, err)

	var httpErr *echo.HTTPError
	require.ErrorAs(t, err, &httpErr)
	assert.Equal(t, http.StatusServiceUnavailable, httpErr.Code)
}

func TestIPRateLimit_KeyContainsIP(t *testing.T) {
	t.Parallel()

	var capturedKey string
	store := &mockRateLimitStore{
		AllowFn: func(_ context.Context, key string, _ int, burst int, _ time.Duration) (*RateLimitResult, error) {
			capturedKey = key
			return &RateLimitResult{Allowed: true, Limit: burst, Remaining: burst - 1, ResetAt: time.Now().Add(time.Minute)}, nil
		},
	}

	cfg := defaultRateLimitConfig()
	mw := NewRateLimitMiddleware(store, cfg)
	c, _ := setupRateLimitTest(t, http.MethodGet, "/health")

	handler := mw.IPRateLimit()(func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	_ = handler(c)
	assert.Equal(t, "ip:1.2.3.4", capturedKey)
}

func TestIPRateLimit_HeaderValues(t *testing.T) {
	t.Parallel()

	resetAt := time.Now().Add(30 * time.Second)
	store := &mockRateLimitStore{
		AllowFn: func(_ context.Context, _ string, _ int, _ int, _ time.Duration) (*RateLimitResult, error) {
			return &RateLimitResult{
				Allowed:   true,
				Limit:     20,
				Remaining: 15,
				ResetAt:   resetAt,
			}, nil
		},
	}

	cfg := defaultRateLimitConfig()
	mw := NewRateLimitMiddleware(store, cfg)
	c, rec := setupRateLimitTest(t, http.MethodGet, "/health")

	handler := mw.IPRateLimit()(func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	_ = handler(c)
	assert.Equal(t, "20", rec.Header().Get("X-RateLimit-Limit"))
	assert.Equal(t, "15", rec.Header().Get("X-RateLimit-Remaining"))
	assert.Equal(t, strconv.FormatInt(resetAt.Unix(), 10), rec.Header().Get("X-RateLimit-Reset"))
}

// --- User Rate Limit Tests ---

func TestUserRateLimit_Unauthenticated_Skips(t *testing.T) {
	t.Parallel()

	store := &mockRateLimitStore{
		AllowFn: func(_ context.Context, _ string, _ int, _ int, _ time.Duration) (*RateLimitResult, error) {
			t.Fatal("Allow should not be called for unauthenticated requests")
			return nil, nil
		},
	}

	cfg := defaultRateLimitConfig()
	mw := NewRateLimitMiddleware(store, cfg)
	c, rec := setupRateLimitTest(t, http.MethodGet, "/health")

	handler := mw.UserRateLimit()(func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestUserRateLimit_Blocked(t *testing.T) {
	t.Parallel()

	store := &mockRateLimitStore{
		AllowFn: func(_ context.Context, key string, _ int, burst int, _ time.Duration) (*RateLimitResult, error) {
			assert.Equal(t, "user:testuser", key)
			return &RateLimitResult{
				Allowed:    false,
				Limit:      burst,
				Remaining:  0,
				RetryAfter: 3 * time.Second,
				ResetAt:    time.Now().Add(3 * time.Second),
			}, nil
		},
	}

	cfg := defaultRateLimitConfig()
	mw := NewRateLimitMiddleware(store, cfg)
	c, _ := setupRateLimitTest(t, http.MethodGet, "/api/test")

	setAuthContext(c, auth.User{ID: "user-1", Login: "testuser", Teams: []string{"dev"}})

	handler := mw.UserRateLimit()(func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	require.Error(t, err)

	var httpErr *echo.HTTPError
	require.ErrorAs(t, err, &httpErr)
	assert.Equal(t, http.StatusTooManyRequests, httpErr.Code)
}

func TestUserRateLimit_KeyContainsLogin(t *testing.T) {
	t.Parallel()

	var capturedKey string
	store := &mockRateLimitStore{
		AllowFn: func(_ context.Context, key string, _ int, burst int, _ time.Duration) (*RateLimitResult, error) {
			capturedKey = key
			return &RateLimitResult{Allowed: true, Limit: burst, Remaining: burst - 1, ResetAt: time.Now().Add(time.Minute)}, nil
		},
	}

	cfg := defaultRateLimitConfig()
	mw := NewRateLimitMiddleware(store, cfg)
	c, _ := setupRateLimitTest(t, http.MethodGet, "/api/test")

	setAuthContext(c, auth.User{ID: "user-1", Login: "myuser", Teams: []string{"dev"}})

	handler := mw.UserRateLimit()(func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	_ = handler(c)
	assert.Equal(t, "user:myuser", capturedKey)
}
