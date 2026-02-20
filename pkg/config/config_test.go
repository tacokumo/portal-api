package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConfig_構造体定義の基本テスト(t *testing.T) {
	t.Parallel()

	t.Run("デフォルトのConfig構造体が正しく初期化される", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{}
		assert.Empty(t, cfg.PortalName)
		assert.Equal(t, 0, cfg.Server.Port)
		assert.Empty(t, cfg.Server.LogLevel)
		assert.Empty(t, cfg.Auth.GitHub.OAuth.ClientID)
		assert.Equal(t, time.Duration(0), cfg.Auth.JWT.AccessTokenDuration)
	})

	t.Run("Config構造体のフィールドが設定可能", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{
			PortalName: "Test Portal",
			Server: ServerConfig{
				Port:     9000,
				LogLevel: "debug",
			},
			Auth: AuthConfig{
				GitHub: GitHubConfig{
					OAuth: GitHubOAuthConfig{
						ClientID:     "test-client-id",
						ClientSecret: "test-secret",
						RedirectURL:  "http://localhost:8080/callback",
					},
				},
				JWT: JWTConfig{
					AccessTokenDuration:  2 * time.Hour,
					RefreshTokenDuration: 24 * time.Hour,
				},
			},
			Security: SecurityConfig{
				CORS: CORSConfig{
					AllowedOrigins: []string{"http://localhost:3000", "https://example.com"},
				},
			},
		}

		assert.Equal(t, "Test Portal", cfg.PortalName)
		assert.Equal(t, 9000, cfg.Server.Port)
		assert.Equal(t, "debug", cfg.Server.LogLevel)
		assert.Equal(t, "test-client-id", cfg.Auth.GitHub.OAuth.ClientID)
		assert.Equal(t, 2*time.Hour, cfg.Auth.JWT.AccessTokenDuration)
		assert.Len(t, cfg.Security.CORS.AllowedOrigins, 2)
	})

	t.Run("ネストした構造体のフィールドアクセス", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{}

		// ネストした構造体の各レベルでの設定が可能
		cfg.Auth.GitHub.OAuth.ClientID = "nested-client-id"
		cfg.Auth.JWT.PrivateKeyPath = "/path/to/private.key"
		cfg.Auth.Valkey.Address = "redis:6379"
		cfg.Security.CORS.AllowedOrigins = []string{"https://trusted.domain"}

		assert.Equal(t, "nested-client-id", cfg.Auth.GitHub.OAuth.ClientID)
		assert.Equal(t, "/path/to/private.key", cfg.Auth.JWT.PrivateKeyPath)
		assert.Equal(t, "redis:6379", cfg.Auth.Valkey.Address)
		assert.Contains(t, cfg.Security.CORS.AllowedOrigins, "https://trusted.domain")
	})
}

// 設定生成ヘルパー関数
func newTestConfig() *Config {
	return &Config{
		PortalName: "Test Portal",
		Server: ServerConfig{
			Port:     8080,
			LogLevel: "info",
		},
		Auth: AuthConfig{
			GitHub: GitHubConfig{
				OAuth: GitHubOAuthConfig{
					ClientID:     "",
					ClientSecret: "",
					RedirectURL:  "http://localhost:8080/callback",
				},
			},
			JWT: JWTConfig{
				AccessTokenDuration:  time.Hour,
				RefreshTokenDuration: 8 * time.Hour,
			},
			Valkey: ValkeyConfig{
				Address: "localhost:6379",
				DB:      0,
			},
		},
		Security: SecurityConfig{
			CORS: CORSConfig{
				AllowedOrigins: []string{"http://localhost:3000"},
			},
		},
	}
}

// newAuthEnabledConfig は認証が有効な設定を生成する（他のテストファイルで使用）
func newAuthEnabledConfig() *Config {
	cfg := newTestConfig()
	cfg.Auth.GitHub.OAuth.ClientID = "test-client-id"
	cfg.Auth.GitHub.OAuth.ClientSecret = "test-client-secret"
	cfg.Auth.JWT.PrivateKeyPath = "/path/to/private.key"
	cfg.Auth.JWT.PublicKeyPath = "/path/to/public.key"
	return cfg
}
