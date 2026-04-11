package e2e

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tacokumo/portal-api/pkg/auth"
	"github.com/tacokumo/portal-api/pkg/config"
)

func baseURL(t *testing.T) string {
	t.Helper()

	if u := os.Getenv("E2E_BASE_URL"); u != "" {
		return u
	}

	t.Skip("E2E_BASE_URL is not set")
	return ""
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

// --- Application バリデーションテスト ---

func TestE2E_ApplicationValidation_EmptyName(t *testing.T) {
	jwtMgr := newJWTManager(t)
	user := auth.User{ID: "e2e-user", Login: "e2e-test", Teams: []string{"tacokumo/dev"}}
	tokenPair, err := jwtMgr.GenerateTokenPair(user, "e2e-session-validation")
	require.NoError(t, err)

	body := `{"name":"","repository_url":"https://github.com/tacokumo/test.git","appconfig_path":"apps/test","appconfig_branch":"main"}`
	req, err := http.NewRequest("POST", baseURL(t)+"/v1alpha1/applications", strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "空の名前は400を返すべき")
}

func TestE2E_ApplicationValidation_UppercaseName(t *testing.T) {
	jwtMgr := newJWTManager(t)
	user := auth.User{ID: "e2e-user", Login: "e2e-test", Teams: []string{"tacokumo/dev"}}
	tokenPair, err := jwtMgr.GenerateTokenPair(user, "e2e-session-validation-2")
	require.NoError(t, err)

	body := `{"name":"Invalid-App","repository_url":"https://github.com/tacokumo/test.git","appconfig_path":"apps/test","appconfig_branch":"main"}`
	req, err := http.NewRequest("POST", baseURL(t)+"/v1alpha1/applications", strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "大文字を含む名前は400を返すべき")
}

func TestE2E_ApplicationValidation_HttpURL(t *testing.T) {
	jwtMgr := newJWTManager(t)
	user := auth.User{ID: "e2e-user", Login: "e2e-test", Teams: []string{"tacokumo/dev"}}
	tokenPair, err := jwtMgr.GenerateTokenPair(user, "e2e-session-validation-3")
	require.NoError(t, err)

	body := `{"name":"valid-app","repository_url":"http://github.com/tacokumo/test.git","appconfig_path":"apps/test","appconfig_branch":"main"}`
	req, err := http.NewRequest("POST", baseURL(t)+"/v1alpha1/applications", strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "httpスキームは400を返すべき")
}

func TestE2E_ApplicationValidation_PrivateIPURL(t *testing.T) {
	jwtMgr := newJWTManager(t)
	user := auth.User{ID: "e2e-user", Login: "e2e-test", Teams: []string{"tacokumo/dev"}}
	tokenPair, err := jwtMgr.GenerateTokenPair(user, "e2e-session-validation-4")
	require.NoError(t, err)

	body := `{"name":"valid-app","repository_url":"https://127.0.0.1/repo.git","appconfig_path":"apps/test","appconfig_branch":"main"}`
	req, err := http.NewRequest("POST", baseURL(t)+"/v1alpha1/applications", strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "プライベートIPは400を返すべき")
}

func TestE2E_ApplicationValidation_PathTraversal(t *testing.T) {
	jwtMgr := newJWTManager(t)
	user := auth.User{ID: "e2e-user", Login: "e2e-test", Teams: []string{"tacokumo/dev"}}
	tokenPair, err := jwtMgr.GenerateTokenPair(user, "e2e-session-validation-5")
	require.NoError(t, err)

	body := `{"name":"valid-app","repository_url":"https://github.com/tacokumo/test.git","appconfig_path":"../etc/passwd","appconfig_branch":"main"}`
	req, err := http.NewRequest("POST", baseURL(t)+"/v1alpha1/applications", strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "パストラバーサルは400を返すべき")
}

func TestE2E_ApplicationValidation_InvalidBranch(t *testing.T) {
	jwtMgr := newJWTManager(t)
	user := auth.User{ID: "e2e-user", Login: "e2e-test", Teams: []string{"tacokumo/dev"}}
	tokenPair, err := jwtMgr.GenerateTokenPair(user, "e2e-session-validation-6")
	require.NoError(t, err)

	body := `{"name":"valid-app","repository_url":"https://github.com/tacokumo/test.git","appconfig_path":"apps/test","appconfig_branch":"branch..name"}`
	req, err := http.NewRequest("POST", baseURL(t)+"/v1alpha1/applications", strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "不正なブランチ名は400を返すべき")
}

// --- Secret バリデーションテスト ---

func TestE2E_SecretValidation_EmptyItems(t *testing.T) {
	jwtMgr := newJWTManager(t)
	user := auth.User{ID: "e2e-user", Login: "e2e-test", Teams: []string{"tacokumo/dev"}}
	tokenPair, err := jwtMgr.GenerateTokenPair(user, "e2e-session-secret-1")
	require.NoError(t, err)

	body := `{"items":[]}`
	req, err := http.NewRequest("POST", baseURL(t)+"/v1alpha1/applications/test-app/secret", strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "空のItemsは400を返すべき")
}

func TestE2E_SecretValidation_InvalidKey(t *testing.T) {
	jwtMgr := newJWTManager(t)
	user := auth.User{ID: "e2e-user", Login: "e2e-test", Teams: []string{"tacokumo/dev"}}
	tokenPair, err := jwtMgr.GenerateTokenPair(user, "e2e-session-secret-2")
	require.NoError(t, err)

	body := `{"items":[{"key":"invalid/key","value":"some-value"}]}`
	req, err := http.NewRequest("POST", baseURL(t)+"/v1alpha1/applications/test-app/secret", strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "不正なキーは400を返すべき")
}

func TestE2E_SecretValidation_EmptyValue(t *testing.T) {
	jwtMgr := newJWTManager(t)
	user := auth.User{ID: "e2e-user", Login: "e2e-test", Teams: []string{"tacokumo/dev"}}
	tokenPair, err := jwtMgr.GenerateTokenPair(user, "e2e-session-secret-3")
	require.NoError(t, err)

	body := `{"items":[{"key":"VALID_KEY","value":""}]}`
	req, err := http.NewRequest("POST", baseURL(t)+"/v1alpha1/applications/test-app/secret", strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "空の値は400を返すべき")
}

func TestE2E_SecretValidation_UpdateInvalidKey(t *testing.T) {
	jwtMgr := newJWTManager(t)
	user := auth.User{ID: "e2e-user", Login: "e2e-test", Teams: []string{"tacokumo/dev"}}
	tokenPair, err := jwtMgr.GenerateTokenPair(user, "e2e-session-secret-4")
	require.NoError(t, err)

	body := `{"items":[{"key":"key with spaces","value":"some-value"}]}`
	req, err := http.NewRequest("PUT", baseURL(t)+"/v1alpha1/applications/test-app/secret", strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "更新時の不正なキーは400を返すべき")
}

// --- CSRF テスト ---

func TestE2E_CSRFProtection_CookieAuthWithoutCSRFToken(t *testing.T) {
	jwtMgr := newJWTManager(t)

	user := auth.User{ID: "e2e-user", Login: "e2e-test", Teams: []string{"tacokumo/dev"}}
	tokenPair, err := jwtMgr.GenerateTokenPair(user, "e2e-session-csrf")
	require.NoError(t, err)

	// Cookie認証でCSRFトークンなしのPOSTは403になるべき
	req, err := http.NewRequest("POST", baseURL(t)+"/auth/logout", nil)
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
	req, err := http.NewRequest("GET", baseURL(t)+"/auth/csrf-token", nil)
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
	req2, err := http.NewRequest("POST", baseURL(t)+"/auth/logout", nil)
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
	req, err := http.NewRequest("POST", baseURL(t)+"/auth/logout", nil)
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
	req, err := http.NewRequest("POST", baseURL(t)+"/auth/logout", nil)
	require.NoError(t, err)
	req.AddCookie(&http.Cookie{Name: "portal_session", Value: tokenPair.AccessToken})
	req.Header.Set("X-CSRF-Token", "invalid-token-value")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusForbidden, resp.StatusCode, "不正なCSRFトークンは403を返すべき")
}

// --- CORS テスト ---

func TestE2E_CORS_PreflightReturns204(t *testing.T) {
	req, err := http.NewRequest("OPTIONS", baseURL(t)+"/health/liveness", nil)
	require.NoError(t, err)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNoContent, resp.StatusCode, "プリフライトリクエストは204を返すべき")
	assert.Equal(t, "http://localhost:3000", resp.Header.Get("Access-Control-Allow-Origin"))
	assert.NotEmpty(t, resp.Header.Get("Access-Control-Allow-Methods"), "Allow-Methodsヘッダーが存在するべき")
	assert.NotEmpty(t, resp.Header.Get("Access-Control-Allow-Headers"), "Allow-Headersヘッダーが存在するべき")
	assert.NotEmpty(t, resp.Header.Get("Access-Control-Max-Age"), "Max-Ageヘッダーが存在するべき")
}

func TestE2E_CORS_AllowedOriginSetsHeader(t *testing.T) {
	req, err := http.NewRequest("GET", baseURL(t)+"/health/liveness", nil)
	require.NoError(t, err)
	req.Header.Set("Origin", "http://localhost:3000")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "http://localhost:3000", resp.Header.Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", resp.Header.Get("Access-Control-Allow-Credentials"))
}

func TestE2E_CORS_DisallowedOriginNoHeader(t *testing.T) {
	req, err := http.NewRequest("GET", baseURL(t)+"/health/liveness", nil)
	require.NoError(t, err)
	req.Header.Set("Origin", "http://evil.example.com")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Empty(t, resp.Header.Get("Access-Control-Allow-Origin"), "不許可オリジンにはCORSヘッダーが設定されないべき")
}

func TestE2E_CORS_WildcardPatternMatch(t *testing.T) {
	// docker-compose.yaml: CORS_ALLOWED_ORIGINS には http://localhost:* が含まれる
	req, err := http.NewRequest("GET", baseURL(t)+"/health/liveness", nil)
	require.NoError(t, err)
	req.Header.Set("Origin", "http://localhost:9999")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "http://localhost:9999", resp.Header.Get("Access-Control-Allow-Origin"))
}

func TestE2E_CORS_NoOriginNoHeaders(t *testing.T) {
	req, err := http.NewRequest("GET", baseURL(t)+"/health/liveness", nil)
	require.NoError(t, err)
	// Originヘッダーなし

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Empty(t, resp.Header.Get("Access-Control-Allow-Origin"), "Originなしの場合CORSヘッダーは不要")
}

// --- Application更新 テスト ---

func TestE2E_ApplicationUpdate_Unauthorized(t *testing.T) {
	req, err := http.NewRequest("PUT", baseURL(t)+"/v1alpha1/applications/test-app", strings.NewReader(`{"repository_url":"https://github.com/tacokumo/test.git","appconfig_path":"apps/test","appconfig_branch":"main"}`))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer invalid-token")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.NotEqual(t, http.StatusOK, resp.StatusCode, "不正なトークンでは200を返さないべき")
}

func TestE2E_ApplicationUpdate_ValidationHttpURL(t *testing.T) {
	jwtMgr := newJWTManager(t)
	user := auth.User{ID: "e2e-user", Login: "e2e-test", Teams: []string{"tacokumo/dev"}}
	tokenPair, err := jwtMgr.GenerateTokenPair(user, "e2e-session-update-1")
	require.NoError(t, err)

	body := `{"repository_url":"http://github.com/tacokumo/test.git","appconfig_path":"apps/test","appconfig_branch":"main"}`
	req, err := http.NewRequest("PUT", baseURL(t)+"/v1alpha1/applications/test-app", strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "httpスキームは400を返すべき")
}

func TestE2E_ApplicationUpdate_ValidationPathTraversal(t *testing.T) {
	jwtMgr := newJWTManager(t)
	user := auth.User{ID: "e2e-user", Login: "e2e-test", Teams: []string{"tacokumo/dev"}}
	tokenPair, err := jwtMgr.GenerateTokenPair(user, "e2e-session-update-2")
	require.NoError(t, err)

	body := `{"repository_url":"https://github.com/tacokumo/test.git","appconfig_path":"../etc/passwd","appconfig_branch":"main"}`
	req, err := http.NewRequest("PUT", baseURL(t)+"/v1alpha1/applications/test-app", strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "パストラバーサルは400を返すべき")
}

func TestE2E_ApplicationUpdate_ValidationInvalidBranch(t *testing.T) {
	jwtMgr := newJWTManager(t)
	user := auth.User{ID: "e2e-user", Login: "e2e-test", Teams: []string{"tacokumo/dev"}}
	tokenPair, err := jwtMgr.GenerateTokenPair(user, "e2e-session-update-3")
	require.NoError(t, err)

	body := `{"repository_url":"https://github.com/tacokumo/test.git","appconfig_path":"apps/test","appconfig_branch":"branch..name"}`
	req, err := http.NewRequest("PUT", baseURL(t)+"/v1alpha1/applications/test-app", strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "不正なブランチ名は400を返すべき")
}

// --- Rate Limit テスト ---
// レート制限テストは RATE_LIMIT_ENABLED=true の環境でのみ実行する。
// レート制限が有効だと後続テストが429で妨害されるため、オプトイン方式で分離する。

func requireRateLimitEnabled(t *testing.T) {
	t.Helper()
	if os.Getenv("RATE_LIMIT_ENABLED") != "true" {
		t.Skip("RATE_LIMIT_ENABLED is not true, skipping rate limit test")
	}
}

func TestE2E_RateLimit_HeadersPresent(t *testing.T) {
	requireRateLimitEnabled(t)

	req, err := http.NewRequest("GET", baseURL(t)+"/health/liveness", nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NotEmpty(t, resp.Header.Get("X-RateLimit-Limit"), "X-RateLimit-Limitヘッダーが存在するべき")
	assert.NotEmpty(t, resp.Header.Get("X-RateLimit-Remaining"), "X-RateLimit-Remainingヘッダーが存在するべき")
	assert.NotEmpty(t, resp.Header.Get("X-RateLimit-Reset"), "X-RateLimit-Resetヘッダーが存在するべき")
}

func TestE2E_RateLimit_ZZ_ExceedsLimit(t *testing.T) {
	requireRateLimitEnabled(t)

	// レート制限を超えるまでリクエストを送信
	var got429 bool
	for i := 0; i < 500; i++ {
		req, err := http.NewRequest("GET", baseURL(t)+"/health/liveness", nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		resp.Body.Close()

		if resp.StatusCode == http.StatusTooManyRequests {
			got429 = true
			assert.NotEmpty(t, resp.Header.Get("Retry-After"), "429レスポンスにRetry-Afterヘッダーが存在するべき")
			break
		}
	}
	assert.True(t, got429, "十分なリクエストで429 Too Many Requestsが返るべき")
}

// --- Application削除 テスト ---

func TestE2E_ApplicationDelete_Unauthorized(t *testing.T) {
	// 不正なトークンでのApplication削除は認証エラー
	req, err := http.NewRequest("DELETE", baseURL(t)+"/v1alpha1/applications/test-app", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer invalid-token")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// 不正なトークンなので200/204以外を返すべき
	assert.NotEqual(t, http.StatusOK, resp.StatusCode, "不正なトークンでは200を返さないべき")
	assert.NotEqual(t, http.StatusNoContent, resp.StatusCode, "不正なトークンでは204を返さないべき")
}

// --- Secret削除 テスト ---

func TestE2E_SecretDelete_Unauthorized(t *testing.T) {
	// 不正なトークンでのSecret削除は認証エラー
	req, err := http.NewRequest("DELETE", baseURL(t)+"/v1alpha1/applications/test-app/secret", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer invalid-token")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// 不正なトークンなので200/204以外を返すべき
	assert.NotEqual(t, http.StatusOK, resp.StatusCode, "不正なトークンでは200を返さないべき")
	assert.NotEqual(t, http.StatusNoContent, resp.StatusCode, "不正なトークンでは204を返さないべき")
}

// --- Application Logs テスト ---

// --- Application Releases テスト ---

func TestE2E_ApplicationReleases_Unauthorized(t *testing.T) {
	// 不正なトークンでのリリース一覧取得は認証エラー
	req, err := http.NewRequest("GET", baseURL(t)+"/v1alpha1/applications/example-app/releases", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer invalid-token")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.NotEqual(t, http.StatusOK, resp.StatusCode, "不正なトークンでは200を返さないべき")
}

func TestE2E_ApplicationReleases_ServiceUnavailable(t *testing.T) {
	// K8s未接続の開発環境ではリリース一覧取得が503を返す
	jwtMgr := newJWTManager(t)
	user := auth.User{ID: "e2e-user", Login: "e2e-test", Teams: []string{"tacokumo/dev"}}
	tokenPair, err := jwtMgr.GenerateTokenPair(user, "e2e-session-releases-1")
	require.NoError(t, err)

	req, err := http.NewRequest("GET", baseURL(t)+"/v1alpha1/applications/example-app/releases", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode, "K8s未接続時は503を返すべき")
}

// --- Application Rollback テスト ---

func TestE2E_ApplicationRollback_Unauthorized(t *testing.T) {
	// 不正なトークンでのロールバックは認証エラー
	body := `{"release_name":"some-release"}`
	req, err := http.NewRequest("POST", baseURL(t)+"/v1alpha1/applications/example-app/rollback", strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer invalid-token")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.NotEqual(t, http.StatusOK, resp.StatusCode, "不正なトークンでは200を返さないべき")
}

func TestE2E_ApplicationRollback_ServiceUnavailable(t *testing.T) {
	// K8s未接続の開発環境ではロールバックが503を返す
	jwtMgr := newJWTManager(t)
	user := auth.User{ID: "e2e-user", Login: "e2e-test", Teams: []string{"tacokumo/dev"}}
	tokenPair, err := jwtMgr.GenerateTokenPair(user, "e2e-session-rollback-1")
	require.NoError(t, err)

	body := `{"release_name":"some-release"}`
	req, err := http.NewRequest("POST", baseURL(t)+"/v1alpha1/applications/example-app/rollback", strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode, "K8s未接続時は503を返すべき")
}

func TestE2E_ApplicationRollback_ValidationInvalidReleaseName(t *testing.T) {
	// 不正なrelease_nameでバリデーションエラー
	jwtMgr := newJWTManager(t)
	user := auth.User{ID: "e2e-user", Login: "e2e-test", Teams: []string{"tacokumo/dev"}}
	tokenPair, err := jwtMgr.GenerateTokenPair(user, "e2e-session-rollback-2")
	require.NoError(t, err)

	body := `{"release_name":"Invalid-Release-Name"}`
	req, err := http.NewRequest("POST", baseURL(t)+"/v1alpha1/applications/example-app/rollback", strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// K8s未接続環境では503が先に返る可能性がある
	assert.Contains(t, []int{http.StatusBadRequest, http.StatusServiceUnavailable}, resp.StatusCode,
		"不正なrelease_nameでは400または503を返すべき")
}

// --- Application Logs テスト ---

func TestE2E_ApplicationLogs_Unauthorized(t *testing.T) {
	// 不正なトークンでのログ取得は認証エラー
	req, err := http.NewRequest("GET", baseURL(t)+"/v1alpha1/applications/example-app/logs", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer invalid-token")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.NotEqual(t, http.StatusOK, resp.StatusCode, "不正なトークンでは200を返さないべき")
}

func TestE2E_ApplicationLogs_ServiceUnavailable(t *testing.T) {
	// K8s未接続の開発環境ではログ取得が503を返す
	jwtMgr := newJWTManager(t)
	user := auth.User{ID: "e2e-user", Login: "e2e-test", Teams: []string{"tacokumo/dev"}}
	tokenPair, err := jwtMgr.GenerateTokenPair(user, "e2e-session-logs-1")
	require.NoError(t, err)

	req, err := http.NewRequest("GET", baseURL(t)+"/v1alpha1/applications/example-app/logs", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode, "K8s未接続時は503を返すべき")
}

func TestE2E_ApplicationLogs_ValidationInvalidTailLines(t *testing.T) {
	// 不正なtail_linesパラメータでバリデーションエラー
	jwtMgr := newJWTManager(t)
	user := auth.User{ID: "e2e-user", Login: "e2e-test", Teams: []string{"tacokumo/dev"}}
	tokenPair, err := jwtMgr.GenerateTokenPair(user, "e2e-session-logs-2")
	require.NoError(t, err)

	// K8s未接続の場合は503が先に返るため、バリデーションテストはK8s接続時のみ意味がある
	// ここでは503が返ることを確認（バリデーションよりK8s接続チェックが先に実行される）
	req, err := http.NewRequest("GET", baseURL(t)+"/v1alpha1/applications/example-app/logs?tail_lines=99999", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// 開発環境ではK8s未接続で503、K8s接続環境では400
	assert.Contains(t, []int{http.StatusBadRequest, http.StatusServiceUnavailable}, resp.StatusCode,
		"不正なtail_linesでは400または503を返すべき")
}
