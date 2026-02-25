package github

import "context"

// GitHubUser GitHub APIから取得したユーザー情報
type GitHubUser struct {
	ID    int64  `json:"id"`
	Login string `json:"login"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// GitHubTeam GitHub Teamの情報
type GitHubTeam struct {
	ID   int64  `json:"id"`
	Slug string `json:"slug"`
}

// GitHubOrganization GitHub Organizationの情報
type GitHubOrganization struct {
	ID    int64  `json:"id"`
	Login string `json:"login"`
}

// AuthProvider GitHub認証プロバイダーインターフェース
type AuthProvider interface {
	// AuthorizeURL OAuth認証URLを生成
	AuthorizeURL(state string) string

	// ExchangeCode authorization codeをaccess tokenに交換
	ExchangeCode(ctx context.Context, code string) (string, error)

	// GetUser ユーザー情報を取得
	GetUser(ctx context.Context, token string) (*GitHubUser, error)

	// GetUserTeams ユーザーが所属するTeam情報を取得
	GetUserTeams(ctx context.Context, token string, org string) ([]GitHubTeam, error)

	// ValidatePAT Personal Access Tokenを検証
	ValidatePAT(ctx context.Context, token string) (*GitHubUser, error)

	// ValidateInstallationToken Installation Tokenを検証
	ValidateInstallationToken(ctx context.Context, token string) (*InstallationInfo, error)
}

// InstallationInfo GitHub App Installation情報
type InstallationInfo struct {
	ID           int64  `json:"id"`
	Account      string `json:"account"`
	RepositoryID int64  `json:"repository_id,omitempty"`
}

// OAuthState OAuth state管理
type OAuthState struct {
	State     string `json:"state"`
	Challenge string `json:"challenge"`   // PKCE code_challenge
	Verifier  string `json:"verifier"`    // PKCE code_verifier
	ExpiresAt int64  `json:"expires_at"`
}