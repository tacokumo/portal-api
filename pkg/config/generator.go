package config

import (
	"os"
	"reflect"

	"github.com/cockroachdb/errors"
	"gopkg.in/yaml.v3"
)

func GenerateDefaultConfig(filename string) error {
	cfg := &Config{}

	// デフォルト値を適用
	if err := applyDefaults(cfg); err != nil {
		return errors.Wrap(err, "failed to apply defaults")
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return errors.Wrap(err, "failed to marshal config")
	}

	// コメント付きYAMLヘッダーを追加
	header := `# Portal API Configuration
# 機密情報は環境変数で設定してください:
#
# 必須環境変数（認証機能使用時）:
# - GITHUB_CLIENT_ID
# - GITHUB_CLIENT_SECRET
# - JWT_PRIVATE_KEY_PATH
# - JWT_PUBLIC_KEY_PATH
#
# オプション環境変数:
# - GITHUB_APP_ID
# - GITHUB_APP_PRIVATE_KEY_PATH
# - VALKEY_PASSWORD
#
`

	content := header + string(data)

	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		return errors.Wrap(err, "failed to write config file")
	}

	return nil
}

func (c *Config) Display(maskSecrets bool) (string, error) {
	displayCfg := *c

	if maskSecrets {
		maskSecretsRecursive(reflect.ValueOf(&displayCfg).Elem())
	}

	data, err := yaml.Marshal(displayCfg)
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal config for display")
	}

	return string(data), nil
}

func maskSecretsRecursive(v reflect.Value) {
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		if field.Kind() == reflect.Struct {
			maskSecretsRecursive(field)
			continue
		}

		// yaml:"-" タグがある場合は機密情報とみなしてマスク
		yamlTag := fieldType.Tag.Get("yaml")
		if yamlTag == "-" && field.Kind() == reflect.String {
			original := field.String()
			if original != "" {
				field.SetString(maskString(original))
			}
		}
	}
}

func maskString(s string) string {
	if s == "" {
		return ""
	}

	// UTF-8ルーンとして処理
	runes := []rune(s)
	runeLen := len(runes)

	if runeLen <= 4 {
		return "****"
	}

	// 最初の2ルーンと最後の2ルーンを保持
	prefix := string(runes[:2])
	suffix := string(runes[runeLen-2:])

	return prefix + "****" + suffix
}
