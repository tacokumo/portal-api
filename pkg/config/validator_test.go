package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  *Config
		setup   func(t *testing.T) func()
		wantErr bool
	}{
		{
			name:    "認証が無効な場合は基本検証のみ実行される",
			config:  newTestConfig(),
			wantErr: false,
		},
		{
			name: "認証が有効で全ての必須フィールドが設定されている場合は成功",
			config: func() *Config {
				cfg := newAuthEnabledConfig()
				return cfg
			}(),
			setup: func(t *testing.T) func() {
				// 必要なキーファイルを作成（実際の設定は後で行う）
				return func() {
					// テスト終了時のクリーンアップは createTempKeyFile で設定済み
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var cleanup func()
			if tt.setup != nil {
				cleanup = tt.setup(t)
				defer cleanup()
			}

			// 認証有効な設定の場合、実際のファイルパスを設定
			if tt.config.Auth.GitHub.OAuth.ClientID != "" {
				privateKeyFile := createTempKeyFile(t, "dummy private key")
				publicKeyFile := createTempKeyFile(t, "dummy public key")
				tt.config.Auth.JWT.PrivateKeyPath = privateKeyFile
				tt.config.Auth.JWT.PublicKeyPath = publicKeyFile
			}

			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfig_validateBasic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "全ての基本項目が有効な場合は成功",
			config:  newTestConfig(),
			wantErr: false,
		},
		{
			name: "PortalNameが空の場合はエラー",
			config: func() *Config {
				cfg := newTestConfig()
				cfg.PortalName = ""
				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "Portが範囲外（0）の場合はエラー",
			config: func() *Config {
				cfg := newTestConfig()
				cfg.Server.Port = 0
				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "Portが範囲外（65536）の場合はエラー",
			config: func() *Config {
				cfg := newTestConfig()
				cfg.Server.Port = 65536
				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "Port境界値（1）の場合は成功",
			config: func() *Config {
				cfg := newTestConfig()
				cfg.Server.Port = 1
				return cfg
			}(),
			wantErr: false,
		},
		{
			name: "Port境界値（65535）の場合は成功",
			config: func() *Config {
				cfg := newTestConfig()
				cfg.Server.Port = 65535
				return cfg
			}(),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.config.validateBasic()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfig_validateAuth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  func(t *testing.T) *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "全ての認証項目が有効な場合は成功",
			config: func(t *testing.T) *Config {
				cfg := newAuthEnabledConfig()
				cfg.Auth.JWT.PrivateKeyPath = createTempKeyFile(t, "dummy private key")
				cfg.Auth.JWT.PublicKeyPath = createTempKeyFile(t, "dummy public key")
				return cfg
			},
			wantErr: false,
		},
		{
			name: "GITHUB_CLIENT_IDが空の場合はエラー",
			config: func(t *testing.T) *Config {
				cfg := newAuthEnabledConfig()
				cfg.Auth.GitHub.OAuth.ClientID = ""
				return cfg
			},
			wantErr: true,
			errMsg:  "GITHUB_CLIENT_ID is required",
		},
		{
			name: "GITHUB_CLIENT_SECRETが空の場合はエラー",
			config: func(t *testing.T) *Config {
				cfg := newAuthEnabledConfig()
				cfg.Auth.GitHub.OAuth.ClientSecret = ""
				return cfg
			},
			wantErr: true,
			errMsg:  "GITHUB_CLIENT_SECRET is required",
		},
		{
			name: "JWT_PRIVATE_KEY_PATHが空の場合はエラー",
			config: func(t *testing.T) *Config {
				cfg := newAuthEnabledConfig()
				cfg.Auth.JWT.PrivateKeyPath = ""
				return cfg
			},
			wantErr: true,
			errMsg:  "JWT_PRIVATE_KEY_PATH is required",
		},
		{
			name: "JWT_PUBLIC_KEY_PATHが空の場合はエラー",
			config: func(t *testing.T) *Config {
				cfg := newAuthEnabledConfig()
				cfg.Auth.JWT.PublicKeyPath = ""
				return cfg
			},
			wantErr: true,
			errMsg:  "JWT_PUBLIC_KEY_PATH is required",
		},
		{
			name: "JWT秘密鍵ファイルが存在しない場合はエラー",
			config: func(t *testing.T) *Config {
				cfg := newAuthEnabledConfig()
				cfg.Auth.JWT.PrivateKeyPath = "/nonexistent/private.key"
				cfg.Auth.JWT.PublicKeyPath = createTempKeyFile(t, "dummy public key")
				return cfg
			},
			wantErr: true,
			errMsg:  "JWT private key file not found",
		},
		{
			name: "JWT公開鍵ファイルが存在しない場合はエラー",
			config: func(t *testing.T) *Config {
				cfg := newAuthEnabledConfig()
				cfg.Auth.JWT.PrivateKeyPath = createTempKeyFile(t, "dummy private key")
				cfg.Auth.JWT.PublicKeyPath = "/nonexistent/public.key"
				return cfg
			},
			wantErr: true,
			errMsg:  "JWT public key file not found",
		},
		{
			name: "基本検証項目でエラーがある場合（PortalName空）",
			config: func(t *testing.T) *Config {
				cfg := newAuthEnabledConfig()
				cfg.PortalName = ""
				return cfg
			},
			wantErr: true,
			errMsg:  "PORTAL_NAME is required",
		},
		{
			name: "基本検証項目でエラーがある場合（Port範囲外）",
			config: func(t *testing.T) *Config {
				cfg := newAuthEnabledConfig()
				cfg.Server.Port = 0
				return cfg
			},
			wantErr: true,
			errMsg:  "server port must be between",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := tt.config(t)
			err := cfg.validateAuth()

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfig_isAuthEnabled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		config   *Config
		expected bool
	}{
		{
			name:     "認証設定が全て空の場合はfalse",
			config:   newTestConfig(),
			expected: false,
		},
		{
			name: "GITHUB_CLIENT_IDのみ設定されている場合はtrue",
			config: func() *Config {
				cfg := newTestConfig()
				cfg.Auth.GitHub.OAuth.ClientID = "test-client-id"
				return cfg
			}(),
			expected: true,
		},
		{
			name: "JWT_PRIVATE_KEY_PATHのみ設定されている場合はtrue",
			config: func() *Config {
				cfg := newTestConfig()
				cfg.Auth.JWT.PrivateKeyPath = "/path/to/private.key"
				return cfg
			}(),
			expected: true,
		},
		{
			name: "両方設定されている場合はtrue",
			config: func() *Config {
				cfg := newTestConfig()
				cfg.Auth.GitHub.OAuth.ClientID = "test-client-id"
				cfg.Auth.JWT.PrivateKeyPath = "/path/to/private.key"
				return cfg
			}(),
			expected: true,
		},
		{
			name: "他の認証フィールドが設定されていても対象フィールドが空ならfalse",
			config: func() *Config {
				cfg := newTestConfig()
				cfg.Auth.GitHub.OAuth.ClientSecret = "secret"
				cfg.Auth.JWT.PublicKeyPath = "/path/to/public.key"
				return cfg
			}(),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.config.isAuthEnabled()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConfig_Validate_統合テスト(t *testing.T) {
	t.Run("Load関数から呼び出される検証が正しく動作する", func(t *testing.T) {

		// 基本的な設定のみの環境変数
		clearAllEnvVars(t)
		t.Setenv("PORTAL_NAME", "Integration Test Portal")
		t.Setenv("SERVER_PORT", "8090")

		cfg, err := LoadWithConfigPath("")
		assert.NoError(t, err)
		assert.NotNil(t, cfg)
		assert.Equal(t, "Integration Test Portal", cfg.PortalName)
		assert.Equal(t, 8090, cfg.Server.Port)
	})

	t.Run("認証有効な環境でのLoad関数検証", func(t *testing.T) {

		// 認証必須の環境変数設定
		clearAllEnvVars(t)
		privateKeyFile := createTempKeyFile(t, "dummy private key")
		publicKeyFile := createTempKeyFile(t, "dummy public key")

		t.Setenv("PORTAL_NAME", "Auth Enabled Portal")
		t.Setenv("GITHUB_CLIENT_ID", "integration-client-id")
		t.Setenv("GITHUB_CLIENT_SECRET", "integration-client-secret")
		t.Setenv("JWT_PRIVATE_KEY_PATH", privateKeyFile)
		t.Setenv("JWT_PUBLIC_KEY_PATH", publicKeyFile)

		cfg, err := LoadWithConfigPath("")
		assert.NoError(t, err)
		assert.NotNil(t, cfg)
		assert.True(t, cfg.isAuthEnabled())
	})

	t.Run("認証設定不完全な環境でのLoad関数エラー", func(t *testing.T) {

		// 不完全な認証設定
		clearAllEnvVars(t)
		t.Setenv("GITHUB_CLIENT_ID", "incomplete-client-id")
		// GITHUB_CLIENT_SECRET が設定されていない

		_, err := LoadWithConfigPath("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "config validation failed")
	})
}

func TestConfig_Validate_ファイル存在確認詳細(t *testing.T) {
	t.Parallel()

	t.Run("ファイル権限エラーのテスト", func(t *testing.T) {
		t.Parallel()

		cfg := newAuthEnabledConfig()

		// 存在しないディレクトリのファイルを指定
		cfg.Auth.JWT.PrivateKeyPath = "/root/inaccessible/private.key"
		cfg.Auth.JWT.PublicKeyPath = createTempKeyFile(t, "dummy public key")

		err := cfg.validateAuth()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "JWT private key file not found")
	})

	t.Run("空のファイルパスの処理", func(t *testing.T) {
		t.Parallel()

		cfg := newTestConfig()
		cfg.Auth.GitHub.OAuth.ClientID = "test-client" // 認証を有効にする
		cfg.Auth.GitHub.OAuth.ClientSecret = "test-secret"
		cfg.Auth.JWT.PrivateKeyPath = "" // 空パス
		cfg.Auth.JWT.PublicKeyPath = ""  // 空パス

		err := cfg.validateAuth()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "JWT_PRIVATE_KEY_PATH is required")
	})
}

// 複雑な設定パターンテスト
func TestConfig_Validate_複雑な設定パターン(t *testing.T) {
	t.Parallel()

	t.Run("部分的な認証設定での判定", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name        string
			setupConfig func() *Config
			expectAuth  bool
		}{
			{
				name: "GitHub OAuth のみ設定",
				setupConfig: func() *Config {
					cfg := newTestConfig()
					cfg.Auth.GitHub.OAuth.ClientID = "github-only"
					return cfg
				},
				expectAuth: true,
			},
			{
				name: "JWT のみ設定",
				setupConfig: func() *Config {
					cfg := newTestConfig()
					cfg.Auth.JWT.PrivateKeyPath = "/path/to/jwt"
					return cfg
				},
				expectAuth: true,
			},
			{
				name: "どちらも未設定",
				setupConfig: func() *Config {
					return newTestConfig()
				},
				expectAuth: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				cfg := tt.setupConfig()
				result := cfg.isAuthEnabled()
				assert.Equal(t, tt.expectAuth, result)
			})
		}
	})
}

// ヘルパー関数

// createTempKeyFile は一時的なキーファイルを作成する
func createTempKeyFile(t *testing.T, content string) string {
	tmpfile, err := os.CreateTemp("", "key-*.pem")
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = os.Remove(tmpfile.Name())
	})

	_, err = tmpfile.WriteString(content)
	require.NoError(t, err)
	require.NoError(t, tmpfile.Close())

	return tmpfile.Name()
}

// エラーメッセージの詳細テスト
func TestConfig_Validate_エラーメッセージ詳細(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		config    func(t *testing.T) *Config
		errSubstr string
	}{
		{
			name: "PortalNameエラーメッセージ",
			config: func(t *testing.T) *Config {
				cfg := newTestConfig()
				cfg.PortalName = ""
				return cfg
			},
			errSubstr: "PORTAL_NAME is required",
		},
		{
			name: "Portエラーメッセージ（下限）",
			config: func(t *testing.T) *Config {
				cfg := newTestConfig()
				cfg.Server.Port = -1
				return cfg
			},
			errSubstr: "server port must be between 1 and 65535",
		},
		{
			name: "Portエラーメッセージ（上限）",
			config: func(t *testing.T) *Config {
				cfg := newTestConfig()
				cfg.Server.Port = 70000
				return cfg
			},
			errSubstr: "server port must be between 1 and 65535",
		},
		{
			name: "認証ClientIDエラーメッセージ",
			config: func(t *testing.T) *Config {
				cfg := newTestConfig()
				cfg.Auth.JWT.PrivateKeyPath = "/trigger/auth/validation" // 認証を有効にする
				cfg.Auth.GitHub.OAuth.ClientID = ""
				return cfg
			},
			errSubstr: "GITHUB_CLIENT_ID is required for authentication",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := tt.config(t)
			err := cfg.Validate()
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.errSubstr)
		})
	}
}
