package github

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/tacokumo/portal-api/pkg/auth"
	"github.com/tacokumo/portal-api/pkg/auth/session"
	"github.com/tacokumo/portal-api/pkg/config"
)

// OAuthHandler OAuth認証ハンドラー
type OAuthHandler struct {
	client         AuthProvider
	jwtManager     *auth.JWTManager
	sessionManager session.Manager
	organization   string
	secureCookies  bool
}

// NewOAuthHandler OAuth認証ハンドラーを作成
func NewOAuthHandler(
	cfg config.GitHubConfig,
	jwtManager *auth.JWTManager,
	sessionManager session.Manager,
	organization string,
	secureCookies bool,
) *OAuthHandler {
	client := NewGitHubClient(cfg, organization)

	return &OAuthHandler{
		client:         client,
		jwtManager:     jwtManager,
		sessionManager: sessionManager,
		organization:   organization,
		secureCookies:  secureCookies,
	}
}

// HandleLogin OAuth認証開始エンドポイント
func (h *OAuthHandler) HandleLogin(c *echo.Context) error {
	// PKCE code_verifierとcode_challengeを生成
	codeVerifier := generateCodeVerifier()
	codeChallenge := generateCodeChallenge(codeVerifier)

	// stateパラメータを生成
	state := generateState()

	// OAuth state情報をセッションに保存
	oauthState := OAuthState{
		State:     state,
		Challenge: codeChallenge,
		Verifier:  codeVerifier,
		ExpiresAt: time.Now().Add(10 * time.Minute).Unix(), // 10分有効
	}

	stateData, err := json.Marshal(oauthState)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create oauth state")
	}

	// OAuth stateを一時的なCookieに保存
	cookie := &http.Cookie{
		Name:     "oauth_state",
		Value:    base64.URLEncoding.EncodeToString(stateData),
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secureCookies,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   600, // 10分
	}
	c.SetCookie(cookie)

	// GitHub OAuth URLを生成（PKCEパラメータ付き）
	authURL := h.client.AuthorizeURL(state) +
		"&code_challenge=" + url.QueryEscape(codeChallenge) +
		"&code_challenge_method=S256"

	// GitHubにリダイレクト
	return c.Redirect(http.StatusFound, authURL)
}

// HandleCallback OAuth認証コールバック処理
func (h *OAuthHandler) HandleCallback(c *echo.Context) error {
	// stateパラメータ確認
	state := c.QueryParam("state")
	if state == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing state parameter")
	}

	// OAuth state情報をCookieから取得
	stateCookie, err := c.Cookie("oauth_state")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "missing oauth state")
	}

	stateData, err := base64.URLEncoding.DecodeString(stateCookie.Value)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid oauth state")
	}

	var oauthState OAuthState
	if err := json.Unmarshal(stateData, &oauthState); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid oauth state format")
	}

	// state検証
	if oauthState.State != state {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid state parameter")
	}

	// 有効期限確認
	if time.Now().Unix() > oauthState.ExpiresAt {
		return echo.NewHTTPError(http.StatusBadRequest, "oauth state expired")
	}

	// authorization codeを取得
	code := c.QueryParam("code")
	if code == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing authorization code")
	}

	// errorパラメータ確認
	if errorParam := c.QueryParam("error"); errorParam != "" {
		errorDesc := c.QueryParam("error_description")
		return echo.NewHTTPError(http.StatusBadRequest,
			fmt.Sprintf("OAuth error: %s (%s)", errorParam, errorDesc))
	}

	// access tokenに交換
	ctx := c.Request().Context()
	accessToken, err := h.client.ExchangeCode(ctx, code)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("failed to exchange code: %v", err))
	}

	// ユーザー情報取得
	githubUser, err := h.client.GetUser(ctx, accessToken)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("failed to get user: %v", err))
	}

	// Team情報取得
	githubTeams, err := h.client.GetUserTeams(ctx, accessToken, h.organization)
	if err != nil {
		return echo.NewHTTPError(http.StatusForbidden,
			fmt.Sprintf("user is not member of organization %s", h.organization))
	}

	// Team名をフォーマット
	teams := make([]string, len(githubTeams))
	for i, team := range githubTeams {
		teams[i] = fmt.Sprintf("%s/%s", h.organization, team.Slug)
	}

	// ユーザー情報構造体作成
	user := auth.User{
		ID:    fmt.Sprintf("%d", githubUser.ID),
		Login: githubUser.Login,
		Name:  githubUser.Name,
		Email: githubUser.Email,
		Teams: teams,
	}

	// セッションID生成
	sessionID := h.jwtManager.GenerateSessionID()

	// セッション作成
	clientIP := c.RealIP()
	userAgent := c.Request().Header.Get("User-Agent")
	if err := h.sessionManager.CreateSession(sessionID, user, clientIP, userAgent); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("failed to create session: %v", err))
	}

	// JWT生成
	tokenPair, err := h.jwtManager.GenerateTokenPair(user, sessionID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("failed to generate JWT: %v", err))
	}

	// JWTをHTTPOnly Cookieとして設定
	accessCookie := &http.Cookie{
		Name:     "portal_session",
		Value:    tokenPair.AccessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secureCookies,
		SameSite: http.SameSiteLaxMode,
		Expires:  tokenPair.ExpiresAt,
	}
	c.SetCookie(accessCookie)

	// OAuth state cookieを削除
	stateCookie.MaxAge = -1
	c.SetCookie(stateCookie)

	// 成功レスポンス
	return c.JSON(http.StatusOK, map[string]interface{}{
		"success":    true,
		"user":       user,
		"expires_at": tokenPair.ExpiresAt,
	})
}

// HandleLogout ログアウト処理
func (h *OAuthHandler) HandleLogout(c *echo.Context) error {
	// Cookieからセッション取得
	sessionCookie, err := c.Cookie("portal_session")
	if err != nil {
		// すでにログアウト済み
		return c.JSON(http.StatusOK, map[string]bool{"success": true})
	}

	// JWTからセッションID取得
	claims, err := h.jwtManager.ValidateToken(sessionCookie.Value)
	if err != nil {
		// 無効なトークンだが、Cookieは削除
	} else {
		// セッション削除
		_ = h.sessionManager.DeleteSession(claims.SessionID)
	}

	// Cookieを削除
	cookie := &http.Cookie{
		Name:     "portal_session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secureCookies,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	}
	c.SetCookie(cookie)

	return c.JSON(http.StatusOK, map[string]bool{"success": true})
}

// HandleRefresh JWT更新処理
func (h *OAuthHandler) HandleRefresh(c *echo.Context) error {
	// リクエストボディからリフレッシュトークン取得
	var request struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := c.Bind(&request); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	// リフレッシュトークン検証
	refreshClaims, err := h.jwtManager.ValidateToken(request.RefreshToken)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid refresh token")
	}

	// セッション存在確認
	session, err := h.sessionManager.GetSession(refreshClaims.SessionID)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "session not found")
	}

	// 最終アクセス時間更新
	if err := h.sessionManager.UpdateLastAccess(refreshClaims.SessionID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update session")
	}

	// 新しいアクセストークン生成
	user := auth.User{
		ID:    refreshClaims.UserInfo.ID,
		Login: session.Login,
		Teams: session.Teams,
	}

	tokenPair, err := h.jwtManager.GenerateTokenPair(user, refreshClaims.SessionID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate new token")
	}

	return c.JSON(http.StatusOK, tokenPair)
}

// generateCodeVerifier PKCE code_verifierを生成
func generateCodeVerifier() string {
	data := make([]byte, 32)
	rand.Read(data)
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(data)
}

// generateCodeChallenge PKCE code_challengeを生成
func generateCodeChallenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(h[:])
}

// generateState OAuth stateパラメータを生成
func generateState() string {
	data := make([]byte, 16)
	rand.Read(data)
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(data)
}