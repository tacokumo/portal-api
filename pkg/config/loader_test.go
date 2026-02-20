package config

import (
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {

	tests := []struct {
		name    string
		setup   func(t *testing.T)
		wantErr bool
	}{
		{
			name: "デフォルト値のみで正常に読み込める",
			setup: func(t *testing.T) {
				// 環境変数をクリア
				clearAllEnvVars(t)
			},
			wantErr: false,
		},
		{
			name: "環境変数による上書きが正しく動作する",
			setup: func(t *testing.T) {
				clearAllEnvVars(t)
				t.Setenv("PORTAL_NAME", "Custom Portal")
				t.Setenv("SERVER_PORT", "9090")
				t.Setenv("LOG_LEVEL", "debug")
			},
			wantErr: false,
		},
		{
			name: "YAMLファイルが存在しない場合でも正常に動作する",
			setup: func(t *testing.T) {
				clearAllEnvVars(t)
				// configファイルを明示的に存在しないパスに設定
				os.Args = []string{"test", "-config", "/nonexistent/config.yaml"}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup(t)
			}

			cfg, err := LoadWithConfigPath("")
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, cfg)

			// 基本的なデフォルト値が設定されていることを確認
			assert.NotEmpty(t, cfg.PortalName)
			assert.Greater(t, cfg.Server.Port, 0)
			assert.NotEmpty(t, cfg.Server.LogLevel)
		})
	}
}

func TestLoad_環境変数による上書き(t *testing.T) {

	tests := []struct {
		name     string
		envVars  map[string]string
		validate func(t *testing.T, cfg *Config)
	}{
		{
			name: "string型の環境変数処理",
			envVars: map[string]string{
				"PORTAL_NAME": "環境変数Portal",
				"LOG_LEVEL":   "warn",
			},
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "環境変数Portal", cfg.PortalName)
				assert.Equal(t, "warn", cfg.Server.LogLevel)
			},
		},
		{
			name: "int型の環境変数処理",
			envVars: map[string]string{
				"SERVER_PORT": "3000",
				"VALKEY_DB":   "5",
			},
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, 3000, cfg.Server.Port)
				assert.Equal(t, 5, cfg.Auth.Valkey.DB)
			},
		},
		{
			name: "time.Duration型の環境変数処理",
			envVars: map[string]string{
				"JWT_ACCESS_TOKEN_DURATION":  "2h30m",
				"JWT_REFRESH_TOKEN_DURATION": "48h",
			},
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, 2*time.Hour+30*time.Minute, cfg.Auth.JWT.AccessTokenDuration)
				assert.Equal(t, 48*time.Hour, cfg.Auth.JWT.RefreshTokenDuration)
			},
		},
		{
			name: "スライス型の環境変数処理",
			envVars: map[string]string{
				"CORS_ALLOWED_ORIGINS": "http://localhost:3000,https://example.com,https://test.domain",
			},
			validate: func(t *testing.T, cfg *Config) {
				expected := []string{"http://localhost:3000", "https://example.com", "https://test.domain"}
				assert.Equal(t, expected, cfg.Security.CORS.AllowedOrigins)
			},
		},
		{
			name: "空文字列のスライス処理",
			envVars: map[string]string{
				"CORS_ALLOWED_ORIGINS": "",
			},
			validate: func(t *testing.T, cfg *Config) {
				assert.Empty(t, cfg.Security.CORS.AllowedOrigins)
			},
		},
		{
			name: "スペースを含むスライス処理",
			envVars: map[string]string{
				"CORS_ALLOWED_ORIGINS": " http://localhost:3000 , https://example.com , https://test.domain ",
			},
			validate: func(t *testing.T, cfg *Config) {
				expected := []string{"http://localhost:3000", "https://example.com", "https://test.domain"}
				assert.Equal(t, expected, cfg.Security.CORS.AllowedOrigins)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 環境変数をクリア
			clearAllEnvVars(t)

			// テスト用環境変数設定
			for key, value := range tt.envVars {
				t.Setenv(key, value)
			}

			cfg, err := LoadWithConfigPath("")
			assert.NoError(t, err)
			require.NotNil(t, cfg)

			tt.validate(t, cfg)
		})
	}
}

func TestApplyDefaults(t *testing.T) {
	t.Parallel()

	t.Run("デフォルト値が正しく適用される", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{}
		err := applyDefaults(cfg)

		assert.NoError(t, err)
		assert.Equal(t, "TACOKUMO Portal", cfg.PortalName)
		assert.Equal(t, 8080, cfg.Server.Port)
		assert.Equal(t, "info", cfg.Server.LogLevel)
		assert.Equal(t, "localhost:6379", cfg.Auth.Valkey.Address)
		assert.Equal(t, 0, cfg.Auth.Valkey.DB)
		assert.Equal(t, time.Hour, cfg.Auth.JWT.AccessTokenDuration)
		assert.Equal(t, 8*time.Hour, cfg.Auth.JWT.RefreshTokenDuration)
	})

	t.Run("既存値がある場合はデフォルト値で上書きしない", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{
			PortalName: "既存のポータル名",
			Server: ServerConfig{
				Port:     9000,
				LogLevel: "debug",
			},
		}

		err := applyDefaults(cfg)

		assert.NoError(t, err)
		// 既存値は保持される
		assert.Equal(t, "既存のポータル名", cfg.PortalName)
		assert.Equal(t, 9000, cfg.Server.Port)
		assert.Equal(t, "debug", cfg.Server.LogLevel)
		// ゼロ値のフィールドにはデフォルト値が設定される
		assert.Equal(t, "localhost:6379", cfg.Auth.Valkey.Address)
	})
}

func TestApplyDefaultsRecursive(t *testing.T) {
	t.Parallel()

	t.Run("ネストした構造体のデフォルト値が正しく適用される", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{}
		v := reflect.ValueOf(cfg).Elem()

		err := applyDefaultsRecursive(v)

		assert.NoError(t, err)
		assert.Equal(t, "TACOKUMO Portal", cfg.PortalName)
		assert.Equal(t, 8080, cfg.Server.Port)
		assert.Equal(t, "info", cfg.Server.LogLevel)
		assert.Equal(t, "localhost:6379", cfg.Auth.Valkey.Address)
		assert.Equal(t, time.Hour, cfg.Auth.JWT.AccessTokenDuration)
	})
}

func TestSetFieldValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		fieldType reflect.Type
		value     string
		expected  interface{}
		wantErr   bool
	}{
		{
			name:      "string型の設定",
			fieldType: reflect.TypeOf(""),
			value:     "test string",
			expected:  "test string",
			wantErr:   false,
		},
		{
			name:      "int型の設定",
			fieldType: reflect.TypeOf(int(0)),
			value:     "42",
			expected:  int64(42),
			wantErr:   false,
		},
		{
			name:      "int64型の設定",
			fieldType: reflect.TypeOf(int64(0)),
			value:     "999",
			expected:  int64(999),
			wantErr:   false,
		},
		{
			name:      "bool型の設定（true）",
			fieldType: reflect.TypeOf(false),
			value:     "true",
			expected:  true,
			wantErr:   false,
		},
		{
			name:      "bool型の設定（false）",
			fieldType: reflect.TypeOf(false),
			value:     "false",
			expected:  false,
			wantErr:   false,
		},
		{
			name:      "time.Duration型の設定",
			fieldType: reflect.TypeOf(time.Duration(0)),
			value:     "5m30s",
			expected:  int64(5*time.Minute + 30*time.Second),
			wantErr:   false,
		},
		{
			name:      "string slice型の設定",
			fieldType: reflect.TypeOf([]string{}),
			value:     "a,b,c",
			expected:  []string{"a", "b", "c"},
			wantErr:   false,
		},
		{
			name:      "不正なint値",
			fieldType: reflect.TypeOf(int(0)),
			value:     "invalid",
			wantErr:   true,
		},
		{
			name:      "不正なbool値",
			fieldType: reflect.TypeOf(false),
			value:     "invalid",
			wantErr:   true,
		},
		{
			name:      "不正なtime.Duration値",
			fieldType: reflect.TypeOf(time.Duration(0)),
			value:     "invalid duration",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			field := reflect.New(tt.fieldType).Elem()
			err := setFieldValue(field, tt.value)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)

			switch tt.fieldType.Kind() {
			case reflect.String:
				assert.Equal(t, tt.expected, field.String())
			case reflect.Int, reflect.Int64:
				assert.Equal(t, tt.expected, field.Int())
			case reflect.Bool:
				assert.Equal(t, tt.expected, field.Bool())
			case reflect.Slice:
				if tt.fieldType.Elem().Kind() == reflect.String {
					slice := field.Interface().([]string)
					assert.Equal(t, tt.expected, slice)
				}
			}
		})
	}
}

func TestApplyEnvironmentVariables(t *testing.T) {
	t.Run("envタグがある環境変数が正しく適用される", func(t *testing.T) {

		clearAllEnvVars(t)
		t.Setenv("PORTAL_NAME", "環境変数からのポータル名")
		t.Setenv("SERVER_PORT", "9999")
		t.Setenv("GITHUB_CLIENT_ID", "env-client-id")

		cfg := newTestConfig()
		err := applyEnvironmentVariables(cfg)

		assert.NoError(t, err)
		assert.Equal(t, "環境変数からのポータル名", cfg.PortalName)
		assert.Equal(t, 9999, cfg.Server.Port)
		assert.Equal(t, "env-client-id", cfg.Auth.GitHub.OAuth.ClientID)
	})

	t.Run("環境変数が未設定の場合は既存値が保持される", func(t *testing.T) {

		clearAllEnvVars(t)

		cfg := newTestConfig()
		originalPortalName := cfg.PortalName
		originalPort := cfg.Server.Port

		err := applyEnvironmentVariables(cfg)

		assert.NoError(t, err)
		assert.Equal(t, originalPortalName, cfg.PortalName)
		assert.Equal(t, originalPort, cfg.Server.Port)
	})
}

func TestApplyEnvironmentVariablesRecursive(t *testing.T) {
	t.Run("ネストした構造体の環境変数が正しく適用される", func(t *testing.T) {
		clearAllEnvVars(t)
		t.Setenv("GITHUB_CLIENT_ID", "nested-client-id")
		t.Setenv("JWT_ACCESS_TOKEN_DURATION", "3h")
		t.Setenv("VALKEY_ADDRESS", "redis.example.com:6380")

		cfg := &Config{}
		v := reflect.ValueOf(cfg).Elem()

		err := applyEnvironmentVariablesRecursive(v)

		assert.NoError(t, err)
		assert.Equal(t, "nested-client-id", cfg.Auth.GitHub.OAuth.ClientID)
		assert.Equal(t, 3*time.Hour, cfg.Auth.JWT.AccessTokenDuration)
		assert.Equal(t, "redis.example.com:6380", cfg.Auth.Valkey.Address)
	})
}

func TestLoadYAMLConfig(t *testing.T) {
	t.Parallel()

	t.Run("YAMLファイルが存在しない場合はエラーにならない", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{}
		err := loadYAMLConfig(cfg, "")

		assert.NoError(t, err)
	})

	t.Run("有効なYAMLファイルが正しく読み込まれる", func(t *testing.T) {
		t.Parallel()

		yamlContent := `
portal_name: "YAML Portal"
server:
  port: 8888
  log_level: "error"
auth:
  jwt:
    access_token_duration: "2h"
security:
  cors:
    allowed_origins:
      - "https://yaml.example.com"
`
		configFile := createTempConfigFile(t, yamlContent)

		cfg := &Config{}
		err := loadYAMLConfig(cfg, configFile)

		assert.NoError(t, err)
		assert.Equal(t, "YAML Portal", cfg.PortalName)
		assert.Equal(t, 8888, cfg.Server.Port)
		assert.Equal(t, "error", cfg.Server.LogLevel)
		assert.Equal(t, 2*time.Hour, cfg.Auth.JWT.AccessTokenDuration)
		assert.Contains(t, cfg.Security.CORS.AllowedOrigins, "https://yaml.example.com")
	})

	t.Run("不正なYAMLファイルはエラーになる", func(t *testing.T) {
		t.Parallel()

		invalidYaml := `
portal_name: "Invalid YAML
server:
  port: not_a_number
`
		configFile := createTempConfigFile(t, invalidYaml)

		cfg := &Config{}
		err := loadYAMLConfig(cfg, configFile)

		assert.Error(t, err)
	})

	t.Run("存在しない明示的なファイルパスはエラーになる", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{}
		err := loadYAMLConfig(cfg, "/nonexistent/path/config.yaml")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read config file")
	})
}

// ヘルパー関数

// createTempConfigFile は一時的な設定ファイルを作成する
func createTempConfigFile(t *testing.T, content string) string {
	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = os.Remove(tmpfile.Name())
	})

	_, err = tmpfile.WriteString(content)
	require.NoError(t, err)
	require.NoError(t, tmpfile.Close())

	return tmpfile.Name()
}

// clearAllEnvVars は設定に関連する環境変数をクリアする
func clearAllEnvVars(t *testing.T) {
	envVars := []string{
		"PORTAL_NAME",
		"SERVER_PORT",
		"LOG_LEVEL",
		"GITHUB_CLIENT_ID",
		"GITHUB_CLIENT_SECRET",
		"GITHUB_OAUTH_REDIRECT_URL",
		"GITHUB_APP_ID",
		"GITHUB_APP_PRIVATE_KEY_PATH",
		"JWT_PRIVATE_KEY_PATH",
		"JWT_PUBLIC_KEY_PATH",
		"JWT_ACCESS_TOKEN_DURATION",
		"JWT_REFRESH_TOKEN_DURATION",
		"VALKEY_ADDRESS",
		"VALKEY_PASSWORD",
		"VALKEY_DB",
		"CORS_ALLOWED_ORIGINS",
	}

	for _, env := range envVars {
		t.Setenv(env, "")
	}
}

// エラーハンドリングテスト

func TestLoad_エラーハンドリング(t *testing.T) {
	t.Run("不正な環境変数でのエラー", func(t *testing.T) {

		clearAllEnvVars(t)
		t.Setenv("SERVER_PORT", "invalid_port")

		_, err := LoadWithConfigPath("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to apply environment variables")
	})

	t.Run("不正なDuration値でのエラー", func(t *testing.T) {

		clearAllEnvVars(t)
		t.Setenv("JWT_ACCESS_TOKEN_DURATION", "invalid_duration")

		_, err := LoadWithConfigPath("")
		assert.Error(t, err)
	})
}

// Unicode文字列テスト
func TestSetFieldValue_Unicode処理(t *testing.T) {
	t.Parallel()

	t.Run("Unicode文字列が正しく処理される", func(t *testing.T) {
		t.Parallel()

		field := reflect.New(reflect.TypeOf("")).Elem()
		unicodeValue := "こんにちは世界🌍"

		err := setFieldValue(field, unicodeValue)

		assert.NoError(t, err)
		assert.Equal(t, unicodeValue, field.String())
	})

	t.Run("Unicode文字を含むスライスが正しく処理される", func(t *testing.T) {
		t.Parallel()

		field := reflect.New(reflect.TypeOf([]string{})).Elem()
		unicodeValue := "日本語,English,中文,العربية"

		err := setFieldValue(field, unicodeValue)

		assert.NoError(t, err)
		slice := field.Interface().([]string)
		expected := []string{"日本語", "English", "中文", "العربية"}
		assert.Equal(t, expected, slice)
	})
}

// 境界値テスト
func TestSetFieldValue_境界値テスト(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		fieldType reflect.Type
		value     string
		wantErr   bool
	}{
		{
			name:      "空文字列",
			fieldType: reflect.TypeOf(""),
			value:     "",
			wantErr:   false,
		},
		{
			name:      "最大int64値",
			fieldType: reflect.TypeOf(int64(0)),
			value:     "9223372036854775807",
			wantErr:   false,
		},
		{
			name:      "最小int64値",
			fieldType: reflect.TypeOf(int64(0)),
			value:     "-9223372036854775808",
			wantErr:   false,
		},
		{
			name:      "int64オーバーフロー",
			fieldType: reflect.TypeOf(int64(0)),
			value:     "9223372036854775808",
			wantErr:   true,
		},
		{
			name:      "0秒のDuration",
			fieldType: reflect.TypeOf(time.Duration(0)),
			value:     "0s",
			wantErr:   false,
		},
		{
			name:      "非常に長いDuration",
			fieldType: reflect.TypeOf(time.Duration(0)),
			value:     "8760h", // 1年
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			field := reflect.New(tt.fieldType).Elem()
			err := setFieldValue(field, tt.value)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
