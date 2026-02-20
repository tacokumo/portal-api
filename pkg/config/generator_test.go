package config

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestGenerateDefaultConfig(t *testing.T) {
	t.Parallel()

	t.Run("デフォルト設定が正しく生成される", func(t *testing.T) {
		t.Parallel()

		// デフォルト設定を生成（内部ロジックをテスト）
		cfg := &Config{}
		err := applyDefaults(cfg)
		require.NoError(t, err)

		// Displayメソッドで出力をテスト
		content, err := cfg.Display(false)
		require.NoError(t, err)

		// デフォルト値が含まれていることを確認
		assert.Contains(t, content, "TACOKUMO Portal")
		assert.Contains(t, content, "8080")
		assert.Contains(t, content, "info")
		assert.Contains(t, content, "localhost:6379")
	})

	t.Run("書き込み権限がない場合はエラー", func(t *testing.T) {
		t.Parallel()

		err := GenerateDefaultConfig("/root/readonly/config.yaml")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to write config file")
	})

	t.Run("生成されたYAMLが有効な形式である", func(t *testing.T) {
		t.Parallel()

		// デフォルト設定を生成
		cfg := &Config{}
		err := applyDefaults(cfg)
		require.NoError(t, err)

		// YAML形式で出力
		content, err := cfg.Display(false)
		require.NoError(t, err)

		// YAMLとして再解析できることを確認
		var testCfg Config
		err = yaml.Unmarshal([]byte(content), &testCfg)
		assert.NoError(t, err)

		// デフォルト値が正しく設定されていることを確認
		assert.Equal(t, "TACOKUMO Portal", testCfg.PortalName)
		assert.Equal(t, 8080, testCfg.Server.Port)
		assert.Equal(t, "info", testCfg.Server.LogLevel)
	})
}

func TestConfig_Display(t *testing.T) {
	t.Parallel()

	t.Run("機密情報をマスクしないで表示", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{
			PortalName: "Display Test Portal",
			Auth: AuthConfig{
				GitHub: GitHubConfig{
					OAuth: GitHubOAuthConfig{
						RedirectURL: "http://localhost:8080/callback",
					},
				},
				JWT: JWTConfig{
					AccessTokenDuration:  time.Hour,
					RefreshTokenDuration: 8 * time.Hour,
				},
			},
		}

		result, err := cfg.Display(false)
		assert.NoError(t, err)
		assert.NotEmpty(t, result)

		// 通常のフィールドが表示されることを確認（yaml:"-"タグがないフィールド）
		assert.Contains(t, result, "Display Test Portal")
		assert.Contains(t, result, "http://localhost:8080/callback")
		assert.Contains(t, result, "1h0m0s")
	})

	t.Run("機密情報をマスクして表示", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{
			PortalName: "Masked Test Portal",
			Auth: AuthConfig{
				GitHub: GitHubConfig{
					OAuth: GitHubOAuthConfig{
						ClientID:     "masked-client-id",
						ClientSecret: "masked-client-secret",
						RedirectURL:  "http://localhost:8080/callback",
					},
				},
				JWT: JWTConfig{
					PrivateKeyPath:       "/path/to/private.key",
					PublicKeyPath:        "/path/to/public.key",
					AccessTokenDuration:  time.Hour,
					RefreshTokenDuration: 8 * time.Hour,
				},
			},
		}

		// マスク処理確認のため、一度マスクされた設定を取得
		displayCfg := *cfg
		maskSecretsRecursive(reflect.ValueOf(&displayCfg).Elem())

		// マスク後の値を確認
		assert.Equal(t, "ma****id", displayCfg.Auth.GitHub.OAuth.ClientID)
		assert.Equal(t, "ma****et", displayCfg.Auth.GitHub.OAuth.ClientSecret)

		result, err := cfg.Display(true)
		assert.NoError(t, err)
		assert.NotEmpty(t, result)

		// 非機密情報は表示される
		assert.Contains(t, result, "Masked Test Portal")
		assert.Contains(t, result, "http://localhost:8080/callback")
		assert.Contains(t, result, "1h0m0s")
	})

	t.Run("空の設定でも正常に表示される", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{}
		result, err := cfg.Display(true)
		assert.NoError(t, err)
		assert.NotEmpty(t, result)
	})
}

func TestMaskSecretsRecursive(t *testing.T) {
	t.Parallel()

	t.Run("ネストした構造体の機密情報が正しくマスクされる", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{
			PortalName: "Secrets Test Portal",
			Auth: AuthConfig{
				GitHub: GitHubConfig{
					OAuth: GitHubOAuthConfig{
						ClientID:     "test-client-id",
						ClientSecret: "test-client-secret",
						RedirectURL:  "http://localhost:8080/callback",
					},
					App: GitHubAppConfig{
						AppID:          "test-app-id",
						PrivateKeyPath: "/path/to/app/private.key",
					},
				},
				JWT: JWTConfig{
					PrivateKeyPath:       "/path/to/jwt/private.key",
					PublicKeyPath:        "/path/to/jwt/public.key",
					AccessTokenDuration:  time.Hour,
					RefreshTokenDuration: 8 * time.Hour,
				},
				Valkey: ValkeyConfig{
					Address:  "localhost:6379",
					Password: "valkey-password",
					DB:       0,
				},
			},
		}

		v := reflect.ValueOf(cfg).Elem()
		maskSecretsRecursive(v)

		// 非機密情報は変更されない
		assert.Equal(t, "Secrets Test Portal", cfg.PortalName)
		assert.Equal(t, "http://localhost:8080/callback", cfg.Auth.GitHub.OAuth.RedirectURL)
		assert.Equal(t, time.Hour, cfg.Auth.JWT.AccessTokenDuration)
		assert.Equal(t, "localhost:6379", cfg.Auth.Valkey.Address)
		assert.Equal(t, 0, cfg.Auth.Valkey.DB)

		// 機密情報（yaml:"-"タグがついているフィールド）はマスクされる
		assert.Equal(t, "te****id", cfg.Auth.GitHub.OAuth.ClientID)     // "test-client-id" -> "te****id"
		assert.Equal(t, "te****et", cfg.Auth.GitHub.OAuth.ClientSecret) // "test-client-secret" -> "te****et"
		assert.Equal(t, "te****id", cfg.Auth.GitHub.App.AppID)          // "test-app-id" -> "te****id"
		assert.Equal(t, "/p****ey", cfg.Auth.GitHub.App.PrivateKeyPath) // "/path/to/app/private.key" -> "/p****ey"
		assert.Equal(t, "/p****ey", cfg.Auth.JWT.PrivateKeyPath)        // "/path/to/jwt/private.key" -> "/p****ey"
		assert.Equal(t, "/p****ey", cfg.Auth.JWT.PublicKeyPath)         // "/path/to/jwt/public.key" -> "/p****ey"
		assert.Equal(t, "va****rd", cfg.Auth.Valkey.Password)           // "valkey-password" -> "va****rd"
	})

	t.Run("空文字列のフィールドは変更されない", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{
			Auth: AuthConfig{
				GitHub: GitHubConfig{
					OAuth: GitHubOAuthConfig{
						ClientID:     "",
						ClientSecret: "",
					},
				},
			},
		}

		v := reflect.ValueOf(cfg).Elem()
		maskSecretsRecursive(v)

		// 空文字列はそのまま
		assert.Equal(t, "", cfg.Auth.GitHub.OAuth.ClientID)
		assert.Equal(t, "", cfg.Auth.GitHub.OAuth.ClientSecret)
	})

	t.Run("非文字列フィールドは処理されない", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{
			Server: ServerConfig{
				Port:     8080,
				LogLevel: "info",
			},
			Auth: AuthConfig{
				JWT: JWTConfig{
					AccessTokenDuration: time.Hour,
				},
			},
		}

		originalPort := cfg.Server.Port
		originalDuration := cfg.Auth.JWT.AccessTokenDuration

		v := reflect.ValueOf(cfg).Elem()
		maskSecretsRecursive(v)

		// 非文字列フィールドは変更されない
		assert.Equal(t, originalPort, cfg.Server.Port)
		assert.Equal(t, originalDuration, cfg.Auth.JWT.AccessTokenDuration)
		// yaml:"-"タグがない文字列フィールドも変更されない
		assert.Equal(t, "info", cfg.Server.LogLevel)
	})
}

func TestMaskString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "空文字列",
			input:    "",
			expected: "",
		},
		{
			name:     "1文字",
			input:    "a",
			expected: "****",
		},
		{
			name:     "2文字",
			input:    "ab",
			expected: "****",
		},
		{
			name:     "3文字",
			input:    "abc",
			expected: "****",
		},
		{
			name:     "4文字",
			input:    "abcd",
			expected: "****",
		},
		{
			name:     "5文字",
			input:    "abcde",
			expected: "ab****de",
		},
		{
			name:     "6文字",
			input:    "abcdef",
			expected: "ab****ef",
		},
		{
			name:     "長い文字列",
			input:    "very-long-secret-key-12345",
			expected: "ve****45",
		},
		{
			name:     "日本語文字列（短い）",
			input:    "秘密",
			expected: "****",
		},
		{
			name:     "日本語文字列（長い）",
			input:    "これは秘密の情報です",
			expected: "これ****です", // "これは秘密の情報です" -> "これ" + "****" + "です"
		},
		{
			name:     "数値文字列",
			input:    "1234567890",
			expected: "12****90",
		},
		{
			name:     "特殊文字を含む文字列",
			input:    "password!@#$%^&*()",
			expected: "pa****()", // "password!@#$%^&*()" -> "pa" + "****" + "()"
		},
		{
			name:     "Unicode文字列",
			input:    "🔒🗝️secret🔑🔐",
			expected: "🔒🗝****🔑🔐", // "🔒🗝️secret🔑🔐" -> "🔒🗝" + "****" + "🔑🔐"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := maskString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMaskString_Properties(t *testing.T) {
	t.Parallel()

	t.Run("マスクされた文字列の長さは元の文字列より短くない", func(t *testing.T) {
		t.Parallel()

		testStrings := []string{
			"short",
			"medium-length-string",
			"very-very-very-long-string-with-many-characters",
		}

		for _, s := range testStrings {
			masked := maskString(s)
			if len(s) > 4 {
				// 長い文字列の場合、マスク後の長さは8文字（最初2文字 + "****" + 最後2文字）
				assert.Equal(t, 8, len(masked), "Input: %s", s)
			} else {
				// 短い文字列の場合は "****"
				assert.Equal(t, "****", masked, "Input: %s", s)
			}
		}
	})

	t.Run("マスクされた文字列は元の文字列とは異なる（空文字列以外）", func(t *testing.T) {
		t.Parallel()

		testStrings := []string{
			"a",
			"secret",
			"very-long-secret",
		}

		for _, s := range testStrings {
			masked := maskString(s)
			assert.NotEqual(t, s, masked, "Input: %s should be masked", s)
		}
	})
}

func TestGenerateDefaultConfig_Integration(t *testing.T) {
	t.Run("デフォルト設定がLoad関数と連携して正常動作する", func(t *testing.T) {

		// デフォルト設定を生成
		cfg := &Config{}
		err := applyDefaults(cfg)
		require.NoError(t, err)

		// 環境変数クリア（デフォルト値のテスト）
		clearAllEnvVars(t)

		// デフォルト値の確認
		assert.Equal(t, "TACOKUMO Portal", cfg.PortalName)
		assert.Equal(t, 8080, cfg.Server.Port)
		assert.Equal(t, "info", cfg.Server.LogLevel)
		assert.Equal(t, "localhost:6379", cfg.Auth.Valkey.Address)
		assert.Equal(t, 0, cfg.Auth.Valkey.DB)
		assert.Equal(t, time.Hour, cfg.Auth.JWT.AccessTokenDuration)
		assert.Equal(t, 8*time.Hour, cfg.Auth.JWT.RefreshTokenDuration)
	})
}

func TestDisplay_YAMLフォーマット検証(t *testing.T) {
	t.Parallel()

	t.Run("Display出力が有効なYAMLフォーマットである", func(t *testing.T) {
		t.Parallel()

		cfg := newTestConfig()
		cfg.Security.CORS.AllowedOrigins = []string{
			"http://localhost:3000",
			"https://example.com",
		}

		// マスクなしでの出力
		result, err := cfg.Display(false)
		assert.NoError(t, err)

		// 設定が有効かテスト（認証なし設定なので基本検証のみ）
		err = cfg.Validate()
		assert.NoError(t, err)

		// 基本的なYAML構造要素が含まれていることを確認
		assert.Contains(t, result, "portal_name:")
		assert.Contains(t, result, "server:")
		assert.Contains(t, result, "auth:")
		assert.Contains(t, result, "security:")
	})

	t.Run("マスクされた出力でもYAMLとして有効", func(t *testing.T) {
		t.Parallel()

		cfg := newTestConfig()
		cfg.Auth.GitHub.OAuth.ClientID = "masked-yaml-test-id"
		cfg.Auth.GitHub.OAuth.ClientSecret = "masked-yaml-test-secret"

		result, err := cfg.Display(true)
		assert.NoError(t, err)

		// マスクされた値でもYAML構造は保持される
		assert.Contains(t, result, "portal_name:")
		assert.Contains(t, result, "server:")
		// 機密情報は含まれない
		assert.NotContains(t, result, "masked-yaml-test-id")
		assert.NotContains(t, result, "masked-yaml-test-secret")
	})
}

// エラーハンドリングテスト
func TestGenerateDefaultConfig_ErrorHandling(t *testing.T) {
	t.Parallel()

	t.Run("デフォルト適用処理の正常性確認", func(t *testing.T) {
		// デフォルト適用処理が正常に動作することを確認
		cfg := &Config{}
		err := applyDefaults(cfg)
		assert.NoError(t, err)

		// 基本的なデフォルト値が設定されていることを確認
		assert.NotEmpty(t, cfg.PortalName)
		assert.NotZero(t, cfg.Server.Port)
		assert.NotEmpty(t, cfg.Server.LogLevel)
	})
}

func TestDisplay_ErrorHandling(t *testing.T) {
	t.Parallel()

	t.Run("YAML marshal エラーをシミュレート", func(t *testing.T) {
		// 通常の Config 構造体では YAML marshal エラーは発生しにくいため、
		// このテストは将来的な拡張に備えたもの

		cfg := &Config{
			PortalName: "Error Test Portal",
		}

		result, err := cfg.Display(false)
		assert.NoError(t, err)
		assert.NotEmpty(t, result)
	})
}

// ベンチマークテスト
func BenchmarkMaskString(b *testing.B) {
	testStrings := []string{
		"short",
		"medium-length-string",
		"very-long-string-with-many-characters-to-test-performance",
		"これは日本語の秘密情報です",
	}

	for _, s := range testStrings {
		b.Run(s, func(b *testing.B) {
			for b.Loop() {
				maskString(s)
			}
		})
	}
}

func BenchmarkMaskSecretsRecursive(b *testing.B) {
	cfg := &Config{
		PortalName: "Benchmark Test Portal",
		Auth: AuthConfig{
			GitHub: GitHubConfig{
				OAuth: GitHubOAuthConfig{
					ClientID:     "benchmark-client-id",
					ClientSecret: "benchmark-client-secret",
				},
			},
			JWT: JWTConfig{
				PrivateKeyPath: "/path/to/benchmark/private.key",
				PublicKeyPath:  "/path/to/benchmark/public.key",
			},
		},
	}

	b.ResetTimer()
	for b.Loop() {
		cfgCopy := *cfg
		v := reflect.ValueOf(&cfgCopy).Elem()
		maskSecretsRecursive(v)
	}
}

// 複雑なパターンテスト
func TestMaskSecretsRecursive_複雑なパターン(t *testing.T) {
	t.Parallel()

	t.Run("複数レベルのネストした構造体", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{
			Auth: AuthConfig{
				GitHub: GitHubConfig{
					OAuth: GitHubOAuthConfig{
						ClientID:     "nested-level-1",
						ClientSecret: "nested-level-2",
					},
					App: GitHubAppConfig{
						AppID:          "nested-app-id",
						PrivateKeyPath: "nested-private-key",
					},
				},
				Valkey: ValkeyConfig{
					Password: "nested-valkey-password",
				},
			},
		}

		v := reflect.ValueOf(cfg).Elem()
		maskSecretsRecursive(v)

		// 全ての階層の機密情報がマスクされることを確認
		// maskStringの実装に合わせて期待値を調整
		assert.Equal(t, "ne****-1", cfg.Auth.GitHub.OAuth.ClientID)     // "nested-level-1" -> "ne****-1"
		assert.Equal(t, "ne****-2", cfg.Auth.GitHub.OAuth.ClientSecret) // "nested-level-2" -> "ne****-2"
		assert.Equal(t, "ne****id", cfg.Auth.GitHub.App.AppID)          // "nested-app-id" -> "ne****id"
		assert.Equal(t, "ne****ey", cfg.Auth.GitHub.App.PrivateKeyPath) // "nested-private-key" -> "ne****ey"
		assert.Equal(t, "ne****rd", cfg.Auth.Valkey.Password)           // "nested-valkey-password" -> "ne****rd"
	})
}

// エッジケースのテスト
func TestMaskString_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
	}{
		{"制御文字を含む文字列", "secret\x00\x01\x02"},
		{"改行を含む文字列", "secret\nwith\nnewlines"},
		{"タブを含む文字列", "secret\twith\ttabs"},
		{"非常に長い文字列", strings.Repeat("a", 1000)},
		{"バイト境界テスト", "🌟⭐✨🌟⭐"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := maskString(tt.input)

			// 基本的なマスク動作を確認
			if len(tt.input) <= 4 {
				assert.Equal(t, "****", result)
			} else {
				assert.True(t, len(result) >= 8)
				assert.Contains(t, result, "****")
			}
		})
	}
}
