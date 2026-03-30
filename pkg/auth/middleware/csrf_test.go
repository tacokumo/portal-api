package middleware

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/tacokumo/portal-api/pkg/auth"
	"github.com/tacokumo/portal-api/pkg/auth/session"
	"github.com/tacokumo/portal-api/pkg/config"
)

func newTestJWTManager(t *testing.T) *auth.JWTManager {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.NoError(t, err)

	cfg := config.JWTConfig{
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 8 * time.Hour,
	}

	return auth.NewJWTManagerFromKeys(privateKey, &privateKey.PublicKey, cfg)
}

func setupCSRFTest(t *testing.T, method, path string, body string, headers map[string]string, cookies []*http.Cookie) (*echo.Context, *httptest.ResponseRecorder) {
	t.Helper()

	e := echo.New()
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
	} else {
		req = httptest.NewRequest(method, path, nil)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}
	for _, c := range cookies {
		req.AddCookie(c)
	}

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	return c, rec
}

func TestCSRFMiddleware_Protect_SkipPaths(t *testing.T) {
	t.Parallel()

	jwtMgr := newTestJWTManager(t)
	mockSM := &session.MockManager{}
	csrfMw := NewCSRFMiddleware(mockSM, jwtMgr, true)

	skipPaths := []string{"/auth/github/login", "/auth/github/callback", "/health", "/metrics"}

	for _, path := range skipPaths {
		t.Run(path, func(t *testing.T) {
			t.Parallel()
			c, _ := setupCSRFTest(t, http.MethodPost, path, "", nil, nil)

			handler := csrfMw.Protect()(func(c *echo.Context) error {
				return c.String(http.StatusOK, "ok")
			})

			err := handler(c)
			assert.NoError(t, err)
		})
	}
}

func TestCSRFMiddleware_Protect_SkipSafeMethods(t *testing.T) {
	t.Parallel()

	jwtMgr := newTestJWTManager(t)
	mockSM := &session.MockManager{}
	csrfMw := NewCSRFMiddleware(mockSM, jwtMgr, true)

	safeMethods := []string{http.MethodGet, http.MethodHead, http.MethodOptions}

	for _, method := range safeMethods {
		t.Run(method, func(t *testing.T) {
			t.Parallel()
			c, _ := setupCSRFTest(t, method, "/v1alpha1/applications", "", nil, nil)

			handler := csrfMw.Protect()(func(c *echo.Context) error {
				return c.String(http.StatusOK, "ok")
			})

			err := handler(c)
			assert.NoError(t, err)
		})
	}
}

func TestCSRFMiddleware_Protect_SkipBearerAuth(t *testing.T) {
	t.Parallel()

	jwtMgr := newTestJWTManager(t)
	mockSM := &session.MockManager{}
	csrfMw := NewCSRFMiddleware(mockSM, jwtMgr, true)

	c, _ := setupCSRFTest(t, http.MethodPost, "/v1alpha1/applications", "", map[string]string{
		"Authorization": "Bearer some-token",
	}, nil)

	handler := csrfMw.Protect()(func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	assert.NoError(t, err)
}

func TestCSRFMiddleware_Protect_SkipNoSessionCookie(t *testing.T) {
	t.Parallel()

	jwtMgr := newTestJWTManager(t)
	mockSM := &session.MockManager{}
	csrfMw := NewCSRFMiddleware(mockSM, jwtMgr, true)

	c, _ := setupCSRFTest(t, http.MethodPost, "/v1alpha1/applications", "", nil, nil)

	handler := csrfMw.Protect()(func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	assert.NoError(t, err)
}

func TestCSRFMiddleware_Protect_MissingCSRFToken(t *testing.T) {
	t.Parallel()

	jwtMgr := newTestJWTManager(t)
	mockSM := &session.MockManager{}
	csrfMw := NewCSRFMiddleware(mockSM, jwtMgr, true)

	user := auth.User{ID: "123", Login: "testuser", Teams: []string{"team1"}}
	tokenPair, err := jwtMgr.GenerateTokenPair(user, "session-123")
	assert.NoError(t, err)

	cookies := []*http.Cookie{
		{Name: "portal_session", Value: tokenPair.AccessToken},
	}
	c, _ := setupCSRFTest(t, http.MethodPost, "/v1alpha1/applications", "", nil, cookies)

	handler := csrfMw.Protect()(func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err = handler(c)
	assert.Error(t, err)
	httpErr, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusForbidden, httpErr.Code)
}

func TestCSRFMiddleware_Protect_InvalidCSRFToken(t *testing.T) {
	t.Parallel()

	jwtMgr := newTestJWTManager(t)
	mockSM := &session.MockManager{
		GetCSRFTokenFn: func(sessionID string) (string, error) {
			return "correct-token", nil
		},
	}
	csrfMw := NewCSRFMiddleware(mockSM, jwtMgr, true)

	user := auth.User{ID: "123", Login: "testuser", Teams: []string{"team1"}}
	tokenPair, err := jwtMgr.GenerateTokenPair(user, "session-123")
	assert.NoError(t, err)

	cookies := []*http.Cookie{
		{Name: "portal_session", Value: tokenPair.AccessToken},
	}
	c, _ := setupCSRFTest(t, http.MethodPost, "/v1alpha1/applications", "", map[string]string{
		"X-CSRF-Token": "wrong-token",
	}, cookies)

	handler := csrfMw.Protect()(func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err = handler(c)
	assert.Error(t, err)
	httpErr, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusForbidden, httpErr.Code)
}

func TestCSRFMiddleware_Protect_ValidCSRFToken_Header(t *testing.T) {
	t.Parallel()

	jwtMgr := newTestJWTManager(t)
	csrfToken := "valid-csrf-token-123"
	mockSM := &session.MockManager{
		GetCSRFTokenFn: func(sessionID string) (string, error) {
			assert.Equal(t, "session-123", sessionID)
			return csrfToken, nil
		},
	}
	csrfMw := NewCSRFMiddleware(mockSM, jwtMgr, true)

	user := auth.User{ID: "123", Login: "testuser", Teams: []string{"team1"}}
	tokenPair, err := jwtMgr.GenerateTokenPair(user, "session-123")
	assert.NoError(t, err)

	cookies := []*http.Cookie{
		{Name: "portal_session", Value: tokenPair.AccessToken},
	}
	c, _ := setupCSRFTest(t, http.MethodPost, "/v1alpha1/applications", "", map[string]string{
		"X-CSRF-Token": csrfToken,
	}, cookies)

	handler := csrfMw.Protect()(func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err = handler(c)
	assert.NoError(t, err)
}

func TestCSRFMiddleware_Protect_ValidCSRFToken_FormField(t *testing.T) {
	t.Parallel()

	jwtMgr := newTestJWTManager(t)
	csrfToken := "valid-csrf-token-456"
	mockSM := &session.MockManager{
		GetCSRFTokenFn: func(sessionID string) (string, error) {
			return csrfToken, nil
		},
	}
	csrfMw := NewCSRFMiddleware(mockSM, jwtMgr, true)

	user := auth.User{ID: "123", Login: "testuser", Teams: []string{"team1"}}
	tokenPair, err := jwtMgr.GenerateTokenPair(user, "session-123")
	assert.NoError(t, err)

	cookies := []*http.Cookie{
		{Name: "portal_session", Value: tokenPair.AccessToken},
	}
	c, _ := setupCSRFTest(t, http.MethodPost, "/v1alpha1/applications", "_csrf_token="+csrfToken, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, cookies)

	handler := csrfMw.Protect()(func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err = handler(c)
	assert.NoError(t, err)
}

func TestCSRFMiddleware_Protect_ExpiredCSRFToken(t *testing.T) {
	t.Parallel()

	jwtMgr := newTestJWTManager(t)
	mockSM := &session.MockManager{
		GetCSRFTokenFn: func(sessionID string) (string, error) {
			return "", fmt.Errorf("csrf token not found")
		},
	}
	csrfMw := NewCSRFMiddleware(mockSM, jwtMgr, true)

	user := auth.User{ID: "123", Login: "testuser", Teams: []string{"team1"}}
	tokenPair, err := jwtMgr.GenerateTokenPair(user, "session-123")
	assert.NoError(t, err)

	cookies := []*http.Cookie{
		{Name: "portal_session", Value: tokenPair.AccessToken},
	}
	c, _ := setupCSRFTest(t, http.MethodPost, "/v1alpha1/applications", "", map[string]string{
		"X-CSRF-Token": "some-token",
	}, cookies)

	handler := csrfMw.Protect()(func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err = handler(c)
	assert.Error(t, err)
	httpErr, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusForbidden, httpErr.Code)
}

func TestCSRFMiddleware_GenerateCSRFToken(t *testing.T) {
	t.Parallel()

	jwtMgr := newTestJWTManager(t)
	var savedToken string
	var savedSessionID string
	mockSM := &session.MockManager{
		SaveCSRFTokenFn: func(sessionID string, token string, ttl time.Duration) error {
			savedSessionID = sessionID
			savedToken = token
			assert.Equal(t, 30*time.Minute, ttl)
			return nil
		},
	}
	csrfMw := NewCSRFMiddleware(mockSM, jwtMgr, true)

	token, err := csrfMw.GenerateCSRFToken("test-session")
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.Equal(t, "test-session", savedSessionID)
	assert.Equal(t, token, savedToken)
}

func TestCSRFMiddleware_GenerateCSRFToken_SaveError(t *testing.T) {
	t.Parallel()

	jwtMgr := newTestJWTManager(t)
	mockSM := &session.MockManager{
		SaveCSRFTokenFn: func(sessionID string, token string, ttl time.Duration) error {
			return fmt.Errorf("valkey connection error")
		},
	}
	csrfMw := NewCSRFMiddleware(mockSM, jwtMgr, true)

	token, err := csrfMw.GenerateCSRFToken("test-session")
	assert.Error(t, err)
	assert.Empty(t, token)
	assert.Contains(t, err.Error(), "failed to save CSRF token")
}
