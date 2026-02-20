package config

import (
	"flag"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"gopkg.in/yaml.v3"
)

// Load 設定を優先順位に従って読み込み
// 1. コマンドライン > 2. 環境変数 > 3. YAML > 4. デフォルト値
func Load() (*Config, error) {
	return LoadWithConfigPath("")
}

// LoadWithConfigPath 指定されたconfigPathで設定を読み込み
func LoadWithConfigPath(configPath string) (*Config, error) {
	// テスト環境ではflag.Parseを避ける
	if configPath == "" && !flag.Parsed() {
		var path string
		flag.StringVar(&path, "config", "", "path to config file")
		flag.Parse()
		configPath = path
	}

	cfg := &Config{}

	// Step 1: デフォルト値の設定
	if err := applyDefaults(cfg); err != nil {
		return nil, errors.Wrap(err, "failed to apply default values")
	}

	// Step 2: YAML ファイルの読み込み
	if err := loadYAMLConfig(cfg, configPath); err != nil {
		return nil, errors.Wrap(err, "failed to load YAML config")
	}

	// Step 3: 環境変数のオーバーライド
	if err := applyEnvironmentVariables(cfg); err != nil {
		return nil, errors.Wrap(err, "failed to apply environment variables")
	}

	// Step 4: コマンドライン引数のオーバーライド
	// (必要に応じて実装)

	// Step 5: 設定検証
	if err := cfg.Validate(); err != nil {
		return nil, errors.Wrap(err, "config validation failed")
	}

	return cfg, nil
}

// applyDefaults は構造体の default タグからデフォルト値を設定
func applyDefaults(cfg interface{}) error {
	return applyDefaultsRecursive(reflect.ValueOf(cfg).Elem())
}

func applyDefaultsRecursive(v reflect.Value) error {
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		// 構造体フィールドの場合は再帰処理
		if field.Kind() == reflect.Struct {
			if err := applyDefaultsRecursive(field); err != nil {
				return err
			}
			continue
		}

		// default タグからデフォルト値を取得
		defaultValue := fieldType.Tag.Get("default")
		if defaultValue == "" {
			continue
		}

		// フィールドがゼロ値の場合のみデフォルト値を設定
		if field.IsZero() {
			if err := setFieldValue(field, defaultValue); err != nil {
				return errors.Wrapf(err, "failed to set default value for field %s", fieldType.Name)
			}
		}
	}

	return nil
}

// loadYAMLConfig はYAMLファイルから設定を読み込み
func loadYAMLConfig(cfg *Config, configPath string) error {
	if configPath == "" {
		// デフォルト設定ファイルの検索
		defaultPaths := []string{
			"./config.yaml",
			"./config/config.yaml",
			"/etc/portal-api/config.yaml",
		}

		for _, path := range defaultPaths {
			if _, err := os.Stat(path); err == nil {
				configPath = path
				break
			}
		}

		if configPath == "" {
			// 設定ファイルが見つからない場合は続行（環境変数・デフォルト値のみ）
			return nil
		}
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return errors.Wrapf(err, "failed to read config file: %s", configPath)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return errors.Wrapf(err, "failed to unmarshal YAML config: %s", configPath)
	}

	return nil
}

// applyEnvironmentVariables は env タグに基づいて環境変数を適用
func applyEnvironmentVariables(cfg interface{}) error {
	return applyEnvironmentVariablesRecursive(reflect.ValueOf(cfg).Elem())
}

func applyEnvironmentVariablesRecursive(v reflect.Value) error {
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		// 構造体フィールドの場合は再帰処理
		if field.Kind() == reflect.Struct {
			if err := applyEnvironmentVariablesRecursive(field); err != nil {
				return err
			}
			continue
		}

		// env タグから環境変数名を取得
		envName := fieldType.Tag.Get("env")
		if envName == "" {
			continue
		}

		envValue := os.Getenv(envName)
		if envValue == "" {
			continue
		}

		// 環境変数の値をフィールドに設定
		if err := setFieldValue(field, envValue); err != nil {
			return errors.Wrapf(err, "failed to set env value %s for field %s", envName, fieldType.Name)
		}
	}

	return nil
}

// setFieldValue はリフレクションを使って型に応じた値の設定を行う
func setFieldValue(field reflect.Value, value string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if field.Type() == reflect.TypeOf(time.Duration(0)) {
			// time.Duration の特別処理
			duration, err := time.ParseDuration(value)
			if err != nil {
				return errors.Wrapf(err, "invalid duration format: %s", value)
			}
			field.SetInt(int64(duration))
		} else {
			// 通常の整数型
			intVal, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return errors.Wrapf(err, "invalid integer format: %s", value)
			}
			field.SetInt(intVal)
		}

	case reflect.Bool:
		boolVal, err := strconv.ParseBool(value)
		if err != nil {
			return errors.Wrapf(err, "invalid boolean format: %s", value)
		}
		field.SetBool(boolVal)

	case reflect.Slice:
		// スライスの処理（env-separator タグ対応）
		if field.Type().Elem().Kind() == reflect.String {
			separator := ","
			// env-separator タグを取得するために、フィールドタイプが必要
			// ここでは簡単にデフォルトのカンマ区切りを使用

			var values []string
			if strings.TrimSpace(value) != "" {
				values = strings.Split(value, separator)
				for i, v := range values {
					values[i] = strings.TrimSpace(v)
				}
			}

			slice := reflect.MakeSlice(field.Type(), len(values), len(values))
			for i, v := range values {
				slice.Index(i).SetString(v)
			}
			field.Set(slice)
		}

	default:
		return errors.Errorf("unsupported field type: %s", field.Kind())
	}

	return nil
}

// LoadFromEnv は後方互換性のために残している（非推奨）
// 新しいコードでは Load() を使用してください
func LoadFromEnv() *Config {
	cfg, err := LoadWithConfigPath("")
	if err != nil {
		// フォールバック: 既存の動作（最小限の設定）
		portalName := os.Getenv("PORTAL_NAME")
		if portalName == "" {
			portalName = "TACOKUMO Portal"
		}
		return &Config{
			PortalName: portalName,
			Server: ServerConfig{
				Port:     8080,
				LogLevel: "info",
			},
		}
	}
	return cfg
}
