package github

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/go-github/v66/github"
	"golang.org/x/oauth2"
	github_oauth "golang.org/x/oauth2/github"
	"github.com/tacokumo/portal-api/pkg/config"
)

// GitHubClient GitHub APIクライアント
type GitHubClient struct {
	oauthConfig *oauth2.Config
	httpClient  *http.Client
	org         string

	// PAT検証キャッシュ
	patCache   map[string]*cachedPATInfo
	cacheMutex sync.RWMutex
}

// cachedPATInfo キャッシュされたPAT情報
type cachedPATInfo struct {
	User      *GitHubUser
	Teams     []GitHubTeam
	ExpiresAt time.Time
}

// NewGitHubClient 新しいGitHubクライアントを作成
func NewGitHubClient(cfg config.GitHubConfig, org string) *GitHubClient {
	oauthConfig := &oauth2.Config{
		ClientID:     cfg.OAuth.ClientID,
		ClientSecret: cfg.OAuth.ClientSecret,
		RedirectURL:  cfg.OAuth.RedirectURL,
		Scopes: []string{
			"read:user",
			"user:email",
			"read:org",
			"read:team",
		},
		Endpoint: github_oauth.Endpoint,
	}

	return &GitHubClient{
		oauthConfig: oauthConfig,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		org:         org,
		patCache:    make(map[string]*cachedPATInfo),
	}
}

// AuthorizeURL OAuth認証URLを生成
func (g *GitHubClient) AuthorizeURL(state string) string {
	return g.oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

// ExchangeCode authorization codeをaccess tokenに交換
func (g *GitHubClient) ExchangeCode(ctx context.Context, code string) (string, error) {
	token, err := g.oauthConfig.Exchange(ctx, code)
	if err != nil {
		return "", fmt.Errorf("failed to exchange code: %w", err)
	}

	return token.AccessToken, nil
}

// GetUser ユーザー情報を取得
func (g *GitHubClient) GetUser(ctx context.Context, token string) (*GitHubUser, error) {
	client := g.createAuthenticatedClient(token)

	user, _, err := client.Users.Get(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &GitHubUser{
		ID:    user.GetID(),
		Login: user.GetLogin(),
		Name:  user.GetName(),
		Email: user.GetEmail(),
	}, nil
}

// GetUserTeams ユーザーが所属するTeam情報を取得
func (g *GitHubClient) GetUserTeams(ctx context.Context, token string, org string) ([]GitHubTeam, error) {
	client := g.createAuthenticatedClient(token)

	// ユーザーがorganizationのメンバーかチェック
	membership, _, err := client.Organizations.GetOrgMembership(ctx, "", org)
	if err != nil {
		return nil, fmt.Errorf("user is not a member of organization %s: %w", org, err)
	}

	if membership.GetState() != "active" {
		return nil, fmt.Errorf("user membership is not active in organization %s", org)
	}

	// ユーザーのTeam情報を取得
	teams, _, err := client.Teams.ListUserTeams(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get user teams: %w", err)
	}

	var orgTeams []GitHubTeam
	for _, team := range teams {
		if team.Organization.GetLogin() == org {
			orgTeams = append(orgTeams, GitHubTeam{
				ID:   team.GetID(),
				Slug: team.GetSlug(),
			})
		}
	}

	return orgTeams, nil
}

// ValidatePAT Personal Access Tokenを検証
func (g *GitHubClient) ValidatePAT(ctx context.Context, token string) (*GitHubUser, error) {
	// キャッシュから確認
	g.cacheMutex.RLock()
	cached, exists := g.patCache[token]
	g.cacheMutex.RUnlock()

	if exists && time.Now().Before(cached.ExpiresAt) {
		return cached.User, nil
	}

	// GitHub APIで検証
	client := g.createAuthenticatedClient(token)

	// スコープ確認
	if err := g.validateTokenScopes(ctx, client); err != nil {
		return nil, fmt.Errorf("invalid token scopes: %w", err)
	}

	// ユーザー情報取得
	user, err := g.GetUser(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to get user with PAT: %w", err)
	}

	// Team情報取得
	teams, err := g.GetUserTeams(ctx, token, g.org)
	if err != nil {
		return nil, fmt.Errorf("failed to get teams with PAT: %w", err)
	}

	// キャッシュに保存（5分間）
	g.cacheMutex.Lock()
	g.patCache[token] = &cachedPATInfo{
		User:      user,
		Teams:     teams,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	g.cacheMutex.Unlock()

	return user, nil
}

// ValidateInstallationToken Installation Tokenを検証
func (g *GitHubClient) ValidateInstallationToken(ctx context.Context, token string) (*InstallationInfo, error) {
	client := g.createAuthenticatedClient(token)

	// Installation情報を取得してトークンの有効性確認
	installation, _, err := client.Apps.GetInstallation(ctx, 0) // 0はcurrent installation
	if err != nil {
		return nil, fmt.Errorf("failed to get installation info: %w", err)
	}

	return &InstallationInfo{
		ID:      installation.GetID(),
		Account: installation.Account.GetLogin(),
	}, nil
}

// createAuthenticatedClient 認証済みGitHubクライアントを作成
func (g *GitHubClient) createAuthenticatedClient(token string) *github.Client {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(context.Background(), ts)
	return github.NewClient(tc)
}

// validateTokenScopes トークンが必要なスコープを持っているか確認
func (g *GitHubClient) validateTokenScopes(ctx context.Context, client *github.Client) error {
	// ユーザー情報取得でread:userスコープ確認
	_, _, err := client.Users.Get(ctx, "")
	if err != nil {
		return fmt.Errorf("read:user scope required: %w", err)
	}

	// Organization情報取得でread:orgスコープ確認
	_, _, err = client.Organizations.Get(ctx, g.org)
	if err != nil {
		return fmt.Errorf("read:org scope required: %w", err)
	}

	// Team情報取得でread:teamスコープ確認
	_, _, err = client.Teams.ListUserTeams(ctx, nil)
	if err != nil {
		return fmt.Errorf("read:team scope required: %w", err)
	}

	return nil
}

// GetCachedUserTeams キャッシュされたTeam情報を取得
func (g *GitHubClient) GetCachedUserTeams(token string) ([]GitHubTeam, error) {
	g.cacheMutex.RLock()
	defer g.cacheMutex.RUnlock()

	cached, exists := g.patCache[token]
	if !exists || time.Now().After(cached.ExpiresAt) {
		return nil, fmt.Errorf("no cached teams found")
	}

	return cached.Teams, nil
}