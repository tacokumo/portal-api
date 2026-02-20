package config

import (
	"github.com/cockroachdb/errors"
	"os"
)

func (c *Config) Validate() error {
	// 段階的検証：認証機能が有効な場合のみ厳密検証
	if c.isAuthEnabled() {
		return c.validateAuth()
	}

	// 基本検証（既存の動作を維持）
	return c.validateBasic()
}

func (c *Config) validateBasic() error {
	if c.PortalName == "" {
		return errors.New("PORTAL_NAME is required")
	}

	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return errors.New("server port must be between 1 and 65535")
	}

	return nil
}

func (c *Config) validateAuth() error {
	// 基本検証も実行
	if err := c.validateBasic(); err != nil {
		return err
	}

	// 必須項目の検証
	if c.Auth.GitHub.OAuth.ClientID == "" {
		return errors.New("GITHUB_CLIENT_ID is required for authentication")
	}
	if c.Auth.GitHub.OAuth.ClientSecret == "" {
		return errors.New("GITHUB_CLIENT_SECRET is required for authentication")
	}
	if c.Auth.JWT.PrivateKeyPath == "" {
		return errors.New("JWT_PRIVATE_KEY_PATH is required for authentication")
	}
	if c.Auth.JWT.PublicKeyPath == "" {
		return errors.New("JWT_PUBLIC_KEY_PATH is required for authentication")
	}

	// ファイル存在確認
	if _, err := os.Stat(c.Auth.JWT.PrivateKeyPath); os.IsNotExist(err) {
		return errors.Errorf("JWT private key file not found: %s", c.Auth.JWT.PrivateKeyPath)
	}
	if _, err := os.Stat(c.Auth.JWT.PublicKeyPath); os.IsNotExist(err) {
		return errors.Errorf("JWT public key file not found: %s", c.Auth.JWT.PublicKeyPath)
	}

	return nil
}

func (c *Config) isAuthEnabled() bool {
	// 認証関連の環境変数が設定されている場合は認証有効とみなす
	return c.Auth.GitHub.OAuth.ClientID != "" ||
		c.Auth.JWT.PrivateKeyPath != ""
}
