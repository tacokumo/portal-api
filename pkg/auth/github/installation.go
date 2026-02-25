package github

import (
	"context"
	"fmt"
	"strings"

	"github.com/tacokumo/portal-api/pkg/auth"
)

// InstallationAuthenticator Installation Token認証処理
type InstallationAuthenticator struct {
	client       AuthProvider
	organization string
}

// NewInstallationAuthenticator Installation Token認証処理を作成
func NewInstallationAuthenticator(
	client AuthProvider,
	organization string,
) *InstallationAuthenticator {
	return &InstallationAuthenticator{
		client:       client,
		organization: organization,
	}
}

// Authenticate Installation Token認証を実行
func (i *InstallationAuthenticator) Authenticate(ctx context.Context, token string) (*auth.User, error) {
	// GitHub Installation Token prefix確認
	if !strings.HasPrefix(token, "ghs_") {
		return nil, fmt.Errorf("invalid GitHub Installation Token format")
	}

	// Installation情報取得
	installationInfo, err := i.client.ValidateInstallationToken(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("installation token validation failed: %w", err)
	}

	// Organization確認
	if installationInfo.Account != i.organization {
		return nil, fmt.Errorf("installation token is not for organization %s", i.organization)
	}

	// GitHub Actions用の特別なユーザー情報を作成
	user := &auth.User{
		ID:    fmt.Sprintf("installation:%d", installationInfo.ID),
		Login: fmt.Sprintf("github-actions[bot]"),
		Name:  "GitHub Actions",
		Email: "",
		Teams: []string{fmt.Sprintf("%s/github-actions", i.organization)}, // 特別なTeam
	}

	return user, nil
}