package v1alpha1

import (
	"context"
	"fmt"
	"strings"

	"github.com/tacokumo/portal-api/pkg/apis/v1alpha1/api"
	"github.com/tacokumo/portal-api/pkg/auth"
	"github.com/tacokumo/portal-api/pkg/auth/github"
	"github.com/tacokumo/portal-api/pkg/auth/session"
	"github.com/tacokumo/portal-api/pkg/config"
)

type SecurityHandler struct {
	jwtManager            *auth.JWTManager
	patAuthenticator      *github.PATAuthenticator
	installationAuth      *github.InstallationAuthenticator
	organization          string
}

func NewSecurityHandler(
	cfg *config.Config,
	jwtManager *auth.JWTManager,
	githubClient github.AuthProvider,
	sessionManager session.Manager,
	organization string,
) *SecurityHandler {
	return &SecurityHandler{
		jwtManager:       jwtManager,
		patAuthenticator: github.NewPATAuthenticator(githubClient, sessionManager, organization),
		installationAuth: github.NewInstallationAuthenticator(githubClient, organization),
		organization:     organization,
	}
}

// HandleBearerAuth handles BearerAuth security
func (h *SecurityHandler) HandleBearerAuth(ctx context.Context, operationName api.OperationName, t api.BearerAuth) (context.Context, error) {
	if t.Token == "" {
		return ctx, &ErrorWithCode{
			Code:    401,
			Message: "Bearer token required",
		}
	}

	// 1. JWTトークン検証を試行
	claims, err := h.jwtManager.ValidateToken(t.Token)
	if err == nil {
		// JWT認証成功
		ctx = context.WithValue(ctx, "user", claims.UserInfo)
		ctx = context.WithValue(ctx, "session_id", claims.SessionID)
		return ctx, nil
	}

	// 2. JWT認証失敗時、GitHub認証にフォールバック
	return h.handleGitHubAuth(ctx, t.Token)
}

// handleGitHubAuth GitHub認証（PAT/Installation Token）を処理
func (h *SecurityHandler) handleGitHubAuth(ctx context.Context, token string) (context.Context, error) {
	// GitHub PAT認証を試行
	if h.isGitHubPAT(token) {
		user, err := h.patAuthenticator.Authenticate(ctx, token)
		if err != nil {
			return ctx, &ErrorWithCode{
				Code:    401,
				Message: fmt.Sprintf("GitHub PAT authentication failed: %v", err),
			}
		}

		// ユーザー情報をcontextに追加
		ctx = context.WithValue(ctx, "user", user)
		ctx = context.WithValue(ctx, "auth_type", "github_pat")
		return ctx, nil
	}

	// GitHub Installation Token認証を試行
	if h.isGitHubInstallationToken(token) {
		user, err := h.installationAuth.Authenticate(ctx, token)
		if err != nil {
			return ctx, &ErrorWithCode{
				Code:    401,
				Message: fmt.Sprintf("GitHub Installation Token authentication failed: %v", err),
			}
		}

		// ユーザー情報をcontextに追加
		ctx = context.WithValue(ctx, "user", user)
		ctx = context.WithValue(ctx, "auth_type", "github_installation")
		return ctx, nil
	}

	// いずれの認証にも該当しない場合
	return ctx, &ErrorWithCode{
		Code:    401,
		Message: "Invalid bearer token: must be a valid JWT, GitHub PAT, or GitHub Installation Token",
	}
}

// isGitHubPAT トークンがGitHub PATかどうか判定
func (h *SecurityHandler) isGitHubPAT(token string) bool {
	return strings.HasPrefix(token, "ghp_") || strings.HasPrefix(token, "github_pat_")
}

// isGitHubInstallationToken トークンがGitHub Installation Tokenかどうか判定
func (h *SecurityHandler) isGitHubInstallationToken(token string) bool {
	return strings.HasPrefix(token, "ghs_")
}

// HandleCookieAuth handles CookieAuth security
func (h *SecurityHandler) HandleCookieAuth(ctx context.Context, operationName api.OperationName, t api.CookieAuth) (context.Context, error) {
	// Note: 現在は簡易実装
	if t.APIKey == "" {
		return ctx, &ErrorWithCode{
			Code:    401,
			Message: "Cookie authentication required",
		}
	}

	// JWTトークン検証
	claims, err := h.jwtManager.ValidateToken(t.APIKey)
	if err != nil {
		return ctx, &ErrorWithCode{
			Code:    401,
			Message: "Invalid session cookie",
		}
	}

	// ユーザー情報をcontextに追加
	ctx = context.WithValue(ctx, "user", claims.UserInfo)
	ctx = context.WithValue(ctx, "session_id", claims.SessionID)

	return ctx, nil
}