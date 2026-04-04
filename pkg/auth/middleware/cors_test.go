package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/tacokumo/portal-api/pkg/config"
)

func setupCORSTest(t *testing.T, method, path string, headers map[string]string) (*echo.Context, *httptest.ResponseRecorder) {
	t.Helper()

	e := echo.New()
	req := httptest.NewRequest(method, path, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	return c, rec
}

func defaultCORSConfig() config.CORSConfig {
	return config.CORSConfig{
		AllowedOrigins:   []string{"http://localhost:3000", "http://localhost:8080"},
		AllowCredentials: true,
		MaxAge:           86400,
	}
}

func TestCORSMiddleware_NoOriginHeader_PassThrough(t *testing.T) {
	t.Parallel()

	mw := NewCORSMiddleware(defaultCORSConfig())
	c, rec := setupCORSTest(t, http.MethodGet, "/health/liveness", nil)

	handler := mw.Handle()(func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORSMiddleware_AllowedOrigin_SetsHeaders(t *testing.T) {
	t.Parallel()

	mw := NewCORSMiddleware(defaultCORSConfig())
	c, rec := setupCORSTest(t, http.MethodGet, "/health/liveness", map[string]string{
		"Origin": "http://localhost:3000",
	})

	handler := mw.Handle()(func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "http://localhost:3000", rec.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", rec.Header().Get("Access-Control-Allow-Credentials"))
	assert.Equal(t, "Origin", rec.Header().Get("Vary"))
}

func TestCORSMiddleware_DisallowedOrigin_NoCORSHeaders(t *testing.T) {
	t.Parallel()

	mw := NewCORSMiddleware(defaultCORSConfig())
	c, rec := setupCORSTest(t, http.MethodGet, "/health/liveness", map[string]string{
		"Origin": "http://evil.example.com",
	})

	handler := mw.Handle()(func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORSMiddleware_Preflight_ReturnsNoContent(t *testing.T) {
	t.Parallel()

	mw := NewCORSMiddleware(defaultCORSConfig())
	c, rec := setupCORSTest(t, http.MethodOptions, "/v1alpha1/applications", map[string]string{
		"Origin":                        "http://localhost:3000",
		"Access-Control-Request-Method": "POST",
	})

	handler := mw.Handle()(func(c *echo.Context) error {
		return c.String(http.StatusOK, "should not reach")
	})

	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, "http://localhost:3000", rec.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, rec.Header().Get("Access-Control-Allow-Methods"), "POST")
	assert.Contains(t, rec.Header().Get("Access-Control-Allow-Headers"), "Authorization")
	assert.Equal(t, "86400", rec.Header().Get("Access-Control-Max-Age"))
	assert.Equal(t, "true", rec.Header().Get("Access-Control-Allow-Credentials"))
}

func TestCORSMiddleware_Preflight_DisallowedOrigin(t *testing.T) {
	t.Parallel()

	mw := NewCORSMiddleware(defaultCORSConfig())
	c, rec := setupCORSTest(t, http.MethodOptions, "/v1alpha1/applications", map[string]string{
		"Origin":                        "http://evil.example.com",
		"Access-Control-Request-Method": "POST",
	})

	handler := mw.Handle()(func(c *echo.Context) error {
		return c.String(http.StatusOK, "should not reach")
	})

	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORSMiddleware_WildcardOrigin_WithCredentials(t *testing.T) {
	t.Parallel()

	cfg := config.CORSConfig{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
		MaxAge:           86400,
	}
	mw := NewCORSMiddleware(cfg)
	c, rec := setupCORSTest(t, http.MethodGet, "/health/liveness", map[string]string{
		"Origin": "http://any-origin.example.com",
	})

	handler := mw.Handle()(func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	assert.NoError(t, err)
	// credentials=trueの場合、"*"ではなく実オリジンを返す（W3C仕様）
	assert.Equal(t, "http://any-origin.example.com", rec.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", rec.Header().Get("Access-Control-Allow-Credentials"))
	assert.Equal(t, "Origin", rec.Header().Get("Vary"))
}

func TestCORSMiddleware_WildcardOrigin_WithoutCredentials(t *testing.T) {
	t.Parallel()

	cfg := config.CORSConfig{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: false,
		MaxAge:           86400,
	}
	mw := NewCORSMiddleware(cfg)
	c, rec := setupCORSTest(t, http.MethodGet, "/health/liveness", map[string]string{
		"Origin": "http://any-origin.example.com",
	})

	handler := mw.Handle()(func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, "*", rec.Header().Get("Access-Control-Allow-Origin"))
	assert.Empty(t, rec.Header().Get("Access-Control-Allow-Credentials"))
}

func TestCORSMiddleware_EmptyAllowedOrigins_DeniesAll(t *testing.T) {
	t.Parallel()

	cfg := config.CORSConfig{
		AllowedOrigins:   []string{},
		AllowCredentials: true,
	}
	mw := NewCORSMiddleware(cfg)
	c, rec := setupCORSTest(t, http.MethodGet, "/health/liveness", map[string]string{
		"Origin": "http://localhost:3000",
	})

	handler := mw.Handle()(func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORSMiddleware_PatternMatch_LocalhostWildcard(t *testing.T) {
	t.Parallel()

	cfg := config.CORSConfig{
		AllowedOrigins:   []string{"http://localhost:*"},
		AllowCredentials: true,
		MaxAge:           86400,
	}
	mw := NewCORSMiddleware(cfg)

	origins := []string{"http://localhost:3000", "http://localhost:8080", "http://localhost:9999"}
	for _, origin := range origins {
		t.Run(origin, func(t *testing.T) {
			t.Parallel()
			c, rec := setupCORSTest(t, http.MethodGet, "/health/liveness", map[string]string{
				"Origin": origin,
			})

			handler := mw.Handle()(func(c *echo.Context) error {
				return c.String(http.StatusOK, "ok")
			})

			err := handler(c)
			assert.NoError(t, err)
			assert.Equal(t, origin, rec.Header().Get("Access-Control-Allow-Origin"))
		})
	}
}

func TestCORSMiddleware_PatternMatch_NoMatch(t *testing.T) {
	t.Parallel()

	cfg := config.CORSConfig{
		AllowedOrigins:   []string{"http://localhost:*"},
		AllowCredentials: true,
	}
	mw := NewCORSMiddleware(cfg)
	c, rec := setupCORSTest(t, http.MethodGet, "/health/liveness", map[string]string{
		"Origin": "http://example.com:3000",
	})

	handler := mw.Handle()(func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	assert.NoError(t, err)
	assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORSMiddleware_DefaultMethods(t *testing.T) {
	t.Parallel()

	cfg := defaultCORSConfig()
	mw := NewCORSMiddleware(cfg)
	c, rec := setupCORSTest(t, http.MethodOptions, "/v1alpha1/applications", map[string]string{
		"Origin":                        "http://localhost:3000",
		"Access-Control-Request-Method": "POST",
	})

	handler := mw.Handle()(func(c *echo.Context) error {
		return c.String(http.StatusOK, "should not reach")
	})

	err := handler(c)
	assert.NoError(t, err)
	methods := rec.Header().Get("Access-Control-Allow-Methods")
	for _, m := range []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"} {
		assert.Contains(t, methods, m)
	}
}

func TestCORSMiddleware_DefaultHeaders(t *testing.T) {
	t.Parallel()

	cfg := defaultCORSConfig()
	mw := NewCORSMiddleware(cfg)
	c, rec := setupCORSTest(t, http.MethodOptions, "/v1alpha1/applications", map[string]string{
		"Origin":                        "http://localhost:3000",
		"Access-Control-Request-Method": "POST",
	})

	handler := mw.Handle()(func(c *echo.Context) error {
		return c.String(http.StatusOK, "should not reach")
	})

	err := handler(c)
	assert.NoError(t, err)
	headers := rec.Header().Get("Access-Control-Allow-Headers")
	for _, h := range []string{"Authorization", "Content-Type", "X-CSRF-Token", "X-Requested-With"} {
		assert.Contains(t, headers, h)
	}
}

func TestCORSMiddleware_CustomMethodsAndHeaders(t *testing.T) {
	t.Parallel()

	cfg := config.CORSConfig{
		AllowedOrigins:   []string{"http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST"},
		AllowedHeaders:   []string{"Authorization", "X-Custom-Header"},
		AllowCredentials: true,
		MaxAge:           3600,
	}
	mw := NewCORSMiddleware(cfg)
	c, rec := setupCORSTest(t, http.MethodOptions, "/v1alpha1/applications", map[string]string{
		"Origin":                        "http://localhost:3000",
		"Access-Control-Request-Method": "POST",
	})

	handler := mw.Handle()(func(c *echo.Context) error {
		return c.String(http.StatusOK, "should not reach")
	})

	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, "GET, POST", rec.Header().Get("Access-Control-Allow-Methods"))
	assert.Equal(t, "Authorization, X-Custom-Header", rec.Header().Get("Access-Control-Allow-Headers"))
	assert.Equal(t, "3600", rec.Header().Get("Access-Control-Max-Age"))
}

func TestCORSMiddleware_MaxAge(t *testing.T) {
	t.Parallel()

	cfg := config.CORSConfig{
		AllowedOrigins:   []string{"http://localhost:3000"},
		AllowCredentials: true,
		MaxAge:           7200,
	}
	mw := NewCORSMiddleware(cfg)
	c, rec := setupCORSTest(t, http.MethodOptions, "/health/liveness", map[string]string{
		"Origin":                        "http://localhost:3000",
		"Access-Control-Request-Method": "GET",
	})

	handler := mw.Handle()(func(c *echo.Context) error {
		return c.String(http.StatusOK, "should not reach")
	})

	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, "7200", rec.Header().Get("Access-Control-Max-Age"))
}
