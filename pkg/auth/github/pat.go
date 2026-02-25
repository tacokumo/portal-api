package github

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/tacokumo/portal-api/pkg/auth"
	"github.com/tacokumo/portal-api/pkg/auth/session"
)

// PATAuthenticator Personal Access Token認証処理
type PATAuthenticator struct {
	client         AuthProvider
	sessionManager session.Manager
	organization   string
}

// NewPATAuthenticator PAT認証処理を作成
func NewPATAuthenticator(
	client AuthProvider,
	sessionManager session.Manager,
	organization string,
) *PATAuthenticator {
	return &PATAuthenticator{
		client:         client,
		sessionManager: sessionManager,
		organization:   organization,
	}
}

// Authenticate PAT認証を実行
func (p *PATAuthenticator) Authenticate(ctx context.Context, token string) (*auth.User, error) {
	// GitHub PAT prefix確認
	if !strings.HasPrefix(token, "ghp_") && !strings.HasPrefix(token, "github_pat_") {
		return nil, fmt.Errorf("invalid GitHub PAT format")
	}

	// キャッシュから確認
	userID, err := p.extractUserIDFromToken(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to get user ID: %w", err)
	}

	cachedTeams, err := p.sessionManager.GetCachedUserTeams(userID)
	if err == nil {
		// キャッシュヒット
		githubUser, err := p.client.ValidatePAT(ctx, token)
		if err != nil {
			return nil, fmt.Errorf("PAT validation failed: %w", err)
		}

		return &auth.User{
			ID:    fmt.Sprintf("%d", githubUser.ID),
			Login: githubUser.Login,
			Name:  githubUser.Name,
			Email: githubUser.Email,
			Teams: cachedTeams,
		}, nil
	}

	// キャッシュミス: GitHub APIから取得
	githubUser, err := p.client.ValidatePAT(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("PAT validation failed: %w", err)
	}

	// Team情報取得
	githubTeams, err := p.client.GetUserTeams(ctx, token, p.organization)
	if err != nil {
		return nil, fmt.Errorf("failed to get user teams: %w", err)
	}

	if len(githubTeams) == 0 {
		return nil, fmt.Errorf("user is not member of organization %s", p.organization)
	}

	// Team名をフォーマット
	teams := make([]string, len(githubTeams))
	for i, team := range githubTeams {
		teams[i] = fmt.Sprintf("%s/%s", p.organization, team.Slug)
	}

	// Team情報をキャッシュ（5分間）
	if err := p.sessionManager.CacheUserTeams(
		fmt.Sprintf("%d", githubUser.ID),
		teams,
		5*time.Minute,
	); err != nil {
		// キャッシュエラーはログに記録するが処理は継続
		fmt.Printf("failed to cache user teams: %v\n", err)
	}

	return &auth.User{
		ID:    fmt.Sprintf("%d", githubUser.ID),
		Login: githubUser.Login,
		Name:  githubUser.Name,
		Email: githubUser.Email,
		Teams: teams,
	}, nil
}

// extractUserIDFromToken トークンからユーザーIDを抽出
func (p *PATAuthenticator) extractUserIDFromToken(ctx context.Context, token string) (string, error) {
	githubUser, err := p.client.ValidatePAT(ctx, token)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d", githubUser.ID), nil
}