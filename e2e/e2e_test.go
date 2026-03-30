package e2e

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tacokumo/portal-api/pkg/auth"
	"github.com/tacokumo/portal-api/pkg/config"
)

func baseURL() string {
	if u := os.Getenv("E2E_BASE_URL"); u != "" {
		return u
	}
	return "http://localhost:8080"
}

func newJWTManager(t *testing.T) *auth.JWTManager {
	t.Helper()

	privateKeyPath := os.Getenv("JWT_PRIVATE_KEY_PATH")
	if privateKeyPath == "" {
		privateKeyPath = "../secrets/jwt-private-key.pem"
	}
	publicKeyPath := os.Getenv("JWT_PUBLIC_KEY_PATH")
	if publicKeyPath == "" {
		publicKeyPath = "../secrets/jwt-public-key.pem"
	}

	cfg := config.JWTConfig{
		PrivateKeyPath:       privateKeyPath,
		PublicKeyPath:        publicKeyPath,
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 8 * time.Hour,
	}

	privateKeyExists, err := fileExists(privateKeyPath)
	require.NoError(t, err)
	publicKeyExists, err := fileExists(publicKeyPath)
	require.NoError(t, err)
	if privateKeyExists && publicKeyExists {
		mgr, err := auth.NewJWTManager(cfg)
		require.NoError(t, err)
		return mgr
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return auth.NewJWTManagerFromKeys(privateKey, &privateKey.PublicKey, cfg)
}

func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func TestE2E_CSRFProtection_CookieAuthWithoutCSRFToken(t *testing.T) {
	jwtMgr := newJWTManager(t)

	user := auth.User{ID: "e2e-user", Login: "e2e-test", Teams: []string{"tacokumo/dev"}}
	tokenPair, err := jwtMgr.GenerateTokenPair(user, "e2e-session-csrf")
	require.NoError(t, err)

	// Cookie認証でCSRFトークンなしのPOSTは403になるべき
	req, err := http.NewRequest("POST", baseURL()+"/auth/logout", nil)
	require.NoError(t, err)
	req.AddCookie(&http.Cookie{Name: "portal_session", Value: tokenPair.AccessToken})

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusForbidden, resp.StatusCode, "Cookie認証でCSRFトークンなしのPOSTは403を返すべき")
}

func TestE2E_CSRFProtection_CookieAuthWithCSRFToken(t *testing.T) {
	jwtMgr := newJWTManager(t)

	user := auth.User{ID: "e2e-user", Login: "e2e-test", Teams: []string{"tacokumo/dev"}}
	tokenPair, err := jwtMgr.GenerateTokenPair(user, "e2e-session-csrf-valid")
	require.NoError(t, err)

	// 1. CSRFトークンを取得
	req, err := http.NewRequest("GET", baseURL()+"/auth/csrf-token", nil)
	require.NoError(t, err)
	req.AddCookie(&http.Cookie{Name: "portal_session", Value: tokenPair.AccessToken})

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "CSRFトークン取得は200を返すべき")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var csrfResp struct {
		CsrfToken string `json:"csrf_token"`
	}
	err = json.Unmarshal(body, &csrfResp)
	require.NoError(t, err)
	assert.NotEmpty(t, csrfResp.CsrfToken, "CSRFトークンが返されるべき")

	// 2. CSRFトークン付きでPOSTリクエスト
	req2, err := http.NewRequest("POST", baseURL()+"/auth/logout", nil)
	require.NoError(t, err)
	req2.AddCookie(&http.Cookie{Name: "portal_session", Value: tokenPair.AccessToken})
	req2.Header.Set("X-CSRF-Token", csrfResp.CsrfToken)

	resp2, err := http.DefaultClient.Do(req2)
	require.NoError(t, err)
	defer resp2.Body.Close()

	// CSRFトークン付きなので403にならないこと
	assert.NotEqual(t, http.StatusForbidden, resp2.StatusCode, "CSRFトークン付きPOSTは403を返さないべき")
}

func TestE2E_CSRFProtection_BearerAuthSkipsCSRF(t *testing.T) {
	jwtMgr := newJWTManager(t)

	user := auth.User{ID: "e2e-user", Login: "e2e-test", Teams: []string{"tacokumo/dev"}}
	tokenPair, err := jwtMgr.GenerateTokenPair(user, "e2e-session-bearer")
	require.NoError(t, err)

	// Bearer認証のPOSTリクエストはCSRFチェックをスキップする
	// logoutエンドポイントを使用（K8sクライアント不要）
	req, err := http.NewRequest("POST", baseURL()+"/auth/logout", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Bearer認証ではCSRF 403にならないこと
	assert.NotEqual(t, http.StatusForbidden, resp.StatusCode, "Bearer認証はCSRFチェックをスキップすべき")
}

func TestE2E_CSRFProtection_InvalidCSRFToken(t *testing.T) {
	jwtMgr := newJWTManager(t)

	user := auth.User{ID: "e2e-user", Login: "e2e-test", Teams: []string{"tacokumo/dev"}}
	tokenPair, err := jwtMgr.GenerateTokenPair(user, "e2e-session-invalid-csrf")
	require.NoError(t, err)

	// 不正なCSRFトークンでPOST → 403
	req, err := http.NewRequest("POST", baseURL()+"/auth/logout", nil)
	require.NoError(t, err)
	req.AddCookie(&http.Cookie{Name: "portal_session", Value: tokenPair.AccessToken})
	req.Header.Set("X-CSRF-Token", "invalid-token-value")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusForbidden, resp.StatusCode, "不正なCSRFトークンは403を返すべき")
}
