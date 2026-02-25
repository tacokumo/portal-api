package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"
	"github.com/tacokumo/portal-api/pkg/auth"
	"github.com/tacokumo/portal-api/pkg/auth/github"
	"github.com/tacokumo/portal-api/pkg/auth/session"
	"github.com/tacokumo/portal-api/pkg/config"
)

// AuthContextKey リクエストコンテキストのキー
const AuthContextKey = "auth_context"

// AuthMiddleware 認証ミドルウェア
type AuthMiddleware struct {
	jwtManager            *auth.JWTManager
	sessionManager        session.Manager
	patAuthenticator      *github.PATAuthenticator
	installAuthenticator  *github.InstallationAuthenticator
	organization          string
}

// NewAuthMiddleware 認証ミドルウェアを作成
func NewAuthMiddleware(
	cfg config.Config,
	jwtManager *auth.JWTManager,
	sessionManager session.Manager,
) (*AuthMiddleware, error) {
	// GitHub clientを作成
	githubClient := github.NewGitHubClient(cfg.Auth.GitHub, "tacokumo") // TODO: 設定から取得

	// PAT認証処理
	patAuth := github.NewPATAuthenticator(githubClient, sessionManager, "tacokumo")

	// Installation Token認証処理
	installAuth := github.NewInstallationAuthenticator(githubClient, "tacokumo")

	return &AuthMiddleware{
		jwtManager:            jwtManager,
		sessionManager:        sessionManager,
		patAuthenticator:      patAuth,
		installAuthenticator:  installAuth,
		organization:          "tacokumo",
	}, nil
}

// RequireAuth 認証を必須とするミドルウェア
func (m *AuthMiddleware) RequireAuth() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			// 認証処理実行
			authCtx, err := m.authenticate(c)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
			}

			// リクエストコンテキストに認証情報設定
			ctx := context.WithValue(c.Request().Context(), AuthContextKey, authCtx)
			c.SetRequest(c.Request().WithContext(ctx))

			return next(c)
		}
	}
}

// OptionalAuth 認証をオプションとするミドルウェア
func (m *AuthMiddleware) OptionalAuth() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			// 認証処理実行（エラーでも継続）
			authCtx, err := m.authenticate(c)
			if err == nil {
				// 認証成功時のみコンテキストに設定
				ctx := context.WithValue(c.Request().Context(), AuthContextKey, authCtx)
				c.SetRequest(c.Request().WithContext(ctx))
			}

			return next(c)
		}
	}
}

// authenticate 認証処理実行
func (m *AuthMiddleware) authenticate(c *echo.Context) (*auth.AuthContext, error) {
	ctx := c.Request().Context()
	clientIP := c.RealIP()

	// 1. Cookie JWT認証（OAuth）を試す
	if sessionCookie, err := c.Cookie("portal_session"); err == nil {
		if authCtx, err := m.authenticateWithJWT(ctx, sessionCookie.Value, clientIP); err == nil {
			return authCtx, nil
		}
	}

	// 2. Authorization headerをチェック
	authHeader := c.Request().Header.Get("Authorization")
	if authHeader == "" {
		return nil, fmt.Errorf("no authentication provided")
	}

	if !strings.HasPrefix(authHeader, "Bearer ") {
		return nil, fmt.Errorf("invalid authorization header format")
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")

	// 3. PAT認証を試す
	if strings.HasPrefix(token, "ghp_") || strings.HasPrefix(token, "github_pat_") {
		return m.authenticateWithPAT(ctx, token, clientIP)
	}

	// 4. Installation Token認証を試す
	if strings.HasPrefix(token, "ghs_") {
		return m.authenticateWithInstallation(ctx, token, clientIP)
	}

	return nil, fmt.Errorf("unsupported token type")
}

// authenticateWithJWT JWT認証処理
func (m *AuthMiddleware) authenticateWithJWT(ctx context.Context, token string, clientIP string) (*auth.AuthContext, error) {
	// JWT検証
	claims, err := m.jwtManager.ValidateToken(token)
	if err != nil {
		return nil, fmt.Errorf("invalid JWT: %w", err)
	}

	// セッション存在確認
	session, err := m.sessionManager.GetSession(claims.SessionID)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	// 最終アクセス時間更新
	if err := m.sessionManager.UpdateLastAccess(claims.SessionID); err != nil {
		// ログに記録するが処理は継続
		fmt.Printf("failed to update session last access: %v\n", err)
	}

	return &auth.AuthContext{
		User: auth.User{
			ID:    session.UserID,
			Login: session.Login,
			Teams: session.Teams,
		},
		Method:   auth.AuthMethodOAuth,
		ClientIP: clientIP,
	}, nil
}

// authenticateWithPAT PAT認証処理
func (m *AuthMiddleware) authenticateWithPAT(ctx context.Context, token string, clientIP string) (*auth.AuthContext, error) {
	user, err := m.patAuthenticator.Authenticate(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("PAT authentication failed: %w", err)
	}

	return &auth.AuthContext{
		User:     *user,
		Method:   auth.AuthMethodPAT,
		ClientIP: clientIP,
	}, nil
}

// authenticateWithInstallation Installation Token認証処理
func (m *AuthMiddleware) authenticateWithInstallation(ctx context.Context, token string, clientIP string) (*auth.AuthContext, error) {
	user, err := m.installAuthenticator.Authenticate(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("installation token authentication failed: %w", err)
	}

	return &auth.AuthContext{
		User:     *user,
		Method:   auth.AuthMethodInstallation,
		ClientIP: clientIP,
	}, nil
}

// GetAuthContext リクエストコンテキストから認証情報を取得
func GetAuthContext(c *echo.Context) (*auth.AuthContext, error) {
	authCtx, ok := c.Request().Context().Value(AuthContextKey).(*auth.AuthContext)
	if !ok {
		return nil, fmt.Errorf("no authentication context found")
	}
	return authCtx, nil
}

// MustGetAuthContext リクエストコンテキストから認証情報を取得（パニックあり）
func MustGetAuthContext(c *echo.Context) *auth.AuthContext {
	authCtx, err := GetAuthContext(c)
	if err != nil {
		panic(err)
	}
	return authCtx
}