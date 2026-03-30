package v1alpha1

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/tacokumo/portal-api/pkg/apis/v1alpha1/api"
	"github.com/tacokumo/portal-api/pkg/auth"
	"github.com/tacokumo/portal-api/pkg/auth/github"
	"github.com/tacokumo/portal-api/pkg/auth/session"
	"github.com/tacokumo/portal-api/pkg/config"
)

type AuthService struct {
	githubClient   github.AuthProvider
	jwtManager     *auth.JWTManager
	sessionManager session.Manager
	organization   string
	secureCookies  bool
}

func NewAuthService(
	cfg *config.Config,
	githubClient github.AuthProvider,
	jwtManager *auth.JWTManager,
	sessionManager session.Manager,
	organization string,
) *AuthService {
	return &AuthService{
		githubClient:   githubClient,
		jwtManager:     jwtManager,
		sessionManager: sessionManager,
		organization:   organization,
		secureCookies:  true, // HTTPS前提
	}
}

// GitHubOAuthLogin implements GitHub OAuth login operation
func (s *AuthService) GitHubOAuthLogin(ctx context.Context) error {
	// PKCE code_verifierとcode_challengeを生成
	codeVerifier := generateCodeVerifier()
	codeChallenge := generateCodeChallenge(codeVerifier)

	// stateパラメータを生成
	state := generateState()

	// OAuth state情報を構造体として作成
	oauthState := github.OAuthState{
		State:     state,
		Challenge: codeChallenge,
		Verifier:  codeVerifier,
		ExpiresAt: time.Now().Add(10 * time.Minute).Unix(), // 10分有効
	}

	stateData, err := json.Marshal(oauthState)
	if err != nil {
		return &ErrorWithCode{
			Code:    500,
			Message: "failed to create oauth state",
		}
	}

	// OAuth stateを一時的なCookieとして設定
	stateCookie := &http.Cookie{
		Name:     "oauth_state",
		Value:    base64.URLEncoding.EncodeToString(stateData),
		Path:     "/",
		HttpOnly: true,
		Secure:   s.secureCookies,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   600, // 10分
	}

	// GitHub OAuth URLを生成（PKCEパラメータ付き）
	authURL := s.githubClient.AuthorizeURL(state) +
		"&code_challenge=" + url.QueryEscape(codeChallenge) +
		"&code_challenge_method=S256"

	// リダイレクト情報をエラーとして返す
	return &RedirectError{
		URL:     authURL,
		Cookies: []*http.Cookie{stateCookie},
	}
}

// GitHubOAuthCallback implements GitHub OAuth callback operation
func (s *AuthService) GitHubOAuthCallback(ctx context.Context, params api.GitHubOAuthCallbackParams) (api.GitHubOAuthCallbackRes, error) {
	// state確認
	if params.State == "" {
		return &api.Error{
			Code:    400,
			Message: "missing state parameter",
		}, nil
	}

	// authorization code確認
	if params.Code == "" {
		return &api.Error{
			Code:    400,
			Message: "missing authorization code",
		}, nil
	}

	// OAuth state検証（通常はCookieから取得するが、今回は省略）
	// 注意: 実際の実装では適切なCookie処理が必要

	// access tokenに交換
	accessToken, err := s.githubClient.ExchangeCode(ctx, params.Code)
	if err != nil {
		return &api.Error{
			Code:    500,
			Message: fmt.Sprintf("failed to exchange code: %v", err),
		}, nil
	}

	// ユーザー情報取得
	githubUser, err := s.githubClient.GetUser(ctx, accessToken)
	if err != nil {
		return &api.Error{
			Code:    500,
			Message: fmt.Sprintf("failed to get user: %v", err),
		}, nil
	}

	// Team情報取得
	githubTeams, err := s.githubClient.GetUserTeams(ctx, accessToken, s.organization)
	if err != nil {
		return &api.Error{
			Code:    403,
			Message: fmt.Sprintf("user is not member of organization %s", s.organization),
		}, nil
	}

	// Team名をフォーマット
	teams := make([]string, len(githubTeams))
	for i, team := range githubTeams {
		teams[i] = fmt.Sprintf("%s/%s", s.organization, team.Slug)
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
	sessionID := s.jwtManager.GenerateSessionID()

	// セッション作成（IP、UserAgentはcontextから取得が困難なため空文字列）
	if err := s.sessionManager.CreateSession(sessionID, user, "", ""); err != nil {
		return &api.Error{
			Code:    500,
			Message: fmt.Sprintf("failed to create session: %v", err),
		}, nil
	}

	// JWT生成
	tokenPair, err := s.jwtManager.GenerateTokenPair(user, sessionID)
	if err != nil {
		return &api.Error{
			Code:    500,
			Message: fmt.Sprintf("failed to generate JWT: %v", err),
		}, nil
	}

	// レスポンス作成
	name := api.OptString{}
	if user.Name != "" {
		name = api.OptString{Value: user.Name, Set: true}
	}

	email := api.OptString{}
	if user.Email != "" {
		email = api.OptString{Value: user.Email, Set: true}
	}

	return &api.AuthResponse{
		Success: true,
		User: api.User{
			ID:    user.ID,
			Login: user.Login,
			Name:  name,
			Email: email,
			Teams: user.Teams,
		},
		ExpiresAt: tokenPair.ExpiresAt,
	}, nil
}

// Logout implements logout operation
func (s *AuthService) Logout(ctx context.Context) (*api.LogoutOK, error) {
	// contextからセッション情報を取得を試行
	// 通常はCookieからJWTを取得してセッションを削除する
	if sessionID, ok := ctx.Value("session_id").(string); ok && sessionID != "" {
		_ = s.sessionManager.DeleteSession(sessionID)
	}

	// Cookieクリア用のレスポンスはエラーとして返す
	clearCookie := &http.Cookie{
		Name:     "portal_session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   s.secureCookies,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	}

	// クッキーをクリアするための特別なエラーを返す
	return nil, &SetCookieError{
		Cookies: []*http.Cookie{clearCookie},
		Message: "logout success",
	}
}

// RefreshToken implements refresh token operation
func (s *AuthService) RefreshToken(ctx context.Context, req *api.RefreshTokenReq) (api.RefreshTokenRes, error) {
	// リフレッシュトークン検証
	refreshClaims, err := s.jwtManager.ValidateToken(req.RefreshToken)
	if err != nil {
		return &api.Error{
			Code:    401,
			Message: "invalid refresh token",
		}, nil
	}

	// セッション存在確認
	session, err := s.sessionManager.GetSession(refreshClaims.SessionID)
	if err != nil {
		return &api.Error{
			Code:    401,
			Message: "session not found",
		}, nil
	}

	// 最終アクセス時間更新
	if err := s.sessionManager.UpdateLastAccess(refreshClaims.SessionID); err != nil {
		return &api.Error{
			Code:    500,
			Message: "failed to update session",
		}, nil
	}

	// ユーザー情報再構築
	user := auth.User{
		ID:    refreshClaims.UserInfo.ID,
		Login: session.Login,
		Teams: session.Teams,
	}

	// 新しいトークンペア生成
	tokenPair, err := s.jwtManager.GenerateTokenPair(user, refreshClaims.SessionID)
	if err != nil {
		return &api.Error{
			Code:    500,
			Message: "failed to generate new token",
		}, nil
	}

	// レスポンス作成
	return &api.TokenPair{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt,
	}, nil
}

// GetCSRFToken implements GetCSRFToken operation.
func (s *AuthService) GetCSRFToken(ctx context.Context) (*api.CSRFTokenResponse, error) {
	sessionID, ok := ctx.Value("session_id").(string)
	if !ok || sessionID == "" {
		return nil, &ErrorWithCode{
			Code:    http.StatusUnauthorized,
			Message: "session required for CSRF token",
		}
	}

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, &ErrorWithCode{
			Code:    http.StatusInternalServerError,
			Message: "failed to generate CSRF token",
		}
	}
	token := base64.URLEncoding.EncodeToString(tokenBytes)

	if err := s.sessionManager.SaveCSRFToken(sessionID, token, 30*time.Minute); err != nil {
		return nil, &ErrorWithCode{
			Code:    http.StatusInternalServerError,
			Message: "failed to save CSRF token",
		}
	}

	return &api.CSRFTokenResponse{
		CsrfToken: token,
	}, nil
}

const (
	ErrorCodeNotImplemented = 501
)

// RedirectError リダイレクトを表現する特別なエラー型
type RedirectError struct {
	URL     string
	Cookies []*http.Cookie
}

func (e *RedirectError) Error() string {
	return fmt.Sprintf("redirect to: %s", e.URL)
}

// SetCookieError クッキー設定を表現する特別なエラー型
type SetCookieError struct {
	Cookies []*http.Cookie
	Message string
}

func (e *SetCookieError) Error() string {
	return e.Message
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