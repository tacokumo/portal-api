# ADR005 設定パッケージ

## Status

Proposed

## Context

Portal APIの設定管理システムは現在、環境変数のみに依存した単純なアプローチを採用しています。
現在の実装（`pkg/config/config.go`）では、単一の環境変数（`PORTAL_NAME`）のみを管理していますが、
ADR004で定義された認証・認可システムでは15項目以上の環境変数が必要となります。

### 現在の設定システムの課題

**環境変数の爆発的増加:**
```bash
# ADR004で要求される環境変数（抜粋）
GITHUB_CLIENT_ID=your_client_id
GITHUB_CLIENT_SECRET=your_client_secret
GITHUB_OAUTH_REDIRECT_URL=http://localhost:8080/auth/callback
GITHUB_APP_ID=123456
GITHUB_APP_PRIVATE_KEY_PATH=/path/to/private-key.pem
JWT_PRIVATE_KEY_PATH=/path/to/jwt-private-key.pem
JWT_PUBLIC_KEY_PATH=/path/to/jwt-public-key.pem
JWT_ACCESS_TOKEN_DURATION=1h
JWT_REFRESH_TOKEN_DURATION=8h
VALKEY_URL=redis://localhost:6379
VALKEY_DB=0
VALKEY_PASSWORD=your_password
GITHUB_ORGANIZATION=tacokumo
DEFAULT_ROLE=viewer
LOG_LEVEL=info
```

**現在のアプローチの問題点:**
1. **管理の複雑性**: 15項目以上の環境変数を個別に管理する必要
2. **設定ミスのリスク**: タイポや設定漏れによる起動失敗
3. **開発環境の構築困難**: 新しい開発者が環境構築に時間を要する
4. **デフォルト値の不在**: 環境変数が未設定の場合の適切な動作が困難
5. **階層構造の欠如**: 関連する設定をグループ化できない
6. **ドキュメンテーションの困難**: 設定項目の説明や例を提供できない

### 運用環境での課題

**Kubernetes での運用:**
- ConfigMap と Secret の使い分けが複雑
- 環境変数の注入順序や依存関係の管理
- 設定変更時のローリングアップデートの複雑性

**セキュリティ課題:**
- 機密情報（JWT秘密鍵、GitHub Client Secret）と非機密情報の明確な分離が困難
- 環境変数のログ出力時の意図しない漏洩リスク

**保守性の課題:**
- 設定の検証ロジックが分散化
- 設定値の型安全性の不足
- テスト環境での設定管理の複雑性

## Decision Drivers

- **開発者体験の向上**: 設定の理解と管理を簡単にしたい
- **運用効率の改善**: デプロイメントとメンテナンスコストを削減したい
- **セキュリティの強化**: 機密情報と非機密情報の明確な分離
- **可読性の向上**: 設定の構造を直感的に理解できるようにしたい
- **拡張性の確保**: 将来の機能拡張に柔軟に対応したい
- **Kubernetes エコシステムとの親和性**: Cloud Native な運用に適応したい

## Alternatives Considered

### 1. 設定ファイル形式の選択

| オプション | メリット | デメリット | 評価 |
|------------|----------|------------|------|
| JSON | 構造化、広範囲サポート | コメント不可、人間可読性低 | ❌ |
| TOML | 設定ファイルに特化、可読性高 | Go生態系での採用率低 | ❌ |
| **YAML** | Kubernetesとの整合、コメント対応、階層構造 | インデントエラーのリスク | ✅ 採用 |
| 環境変数のみ | シンプル、12-Factor App準拠 | 大量設定の管理困難 | ❌ |

### 2. 環境変数管理方式

| オプション | メリット | デメリット | 評価 |
|------------|----------|------------|------|
| 固定プレフィックス | 統一性、名前空間分離 | 柔軟性不足、既存変数との不整合 | ❌ |
| **構造体タグベース** | 柔軟性、明示性、後方互換性 | わずかな実装複雑性 | ✅ 採用 |
| 設定ファイルのみ | 一元管理、可読性 | 機密情報管理の課題 | ❌ |

### 3. 設定ライブラリの選択

| オプション | メリット | デメリット | 評価 |
|------------|----------|------------|------|
| Viper | 豊富な機能、広く採用 | 重厚、構造体タグの制限 | ❌ |
| **Cleanenv** | YAML + envタグ統合、軽量、バリデーション内蔵 | 比較的新しい | ✅ 採用 |
| 標準ライブラリ | 依存なし | 低機能、実装コスト高 | ❌ |
| Envconfig | 環境変数特化 | YAMLサポートなし | ❌ |

### 4. 機密情報管理方式

| オプション | メリット | デメリット | 評価 |
|------------|----------|------------|------|
| **環境変数のみ（推奨）** | Kubernetes Secret整合、シンプル | 設定ファイルに記録できない | ✅ 採用 |
| 設定ファイル + 暗号化 | 一元管理 | 鍵管理の複雑性 | ❌ |
| 外部Vault | 高セキュリティ | インフラ複雑性増大 | ❌ |

## Decision Outcome

**YAML ベース設定ファイル + 構造体タグによる環境変数オーバーライド方式** を採用します。

### アーキテクチャ概要

```
設定の優先順位:
1. コマンドラインフラグ (--config /path/to/config.yaml)
2. 環境変数 (env:"CUSTOM_VAR_NAME")
3. 設定ファイル (config.yaml)
4. 構造体のデフォルト値
```

### 設定ファイル構造設計

#### config.yaml（サンプル）

```yaml
# Portal API Configuration
# 機密情報は環境変数で上書きしてください

server:
  port: 8080
  portal_name: "TACOKUMO Portal"
  log_level: "info"

auth:
  github:
    oauth:
      # client_id: "設定ファイルには記載しない（環境変数のみ）"
      # client_secret: "設定ファイルには記載しない（環境変数のみ）"
      redirect_url: "http://localhost:8080/auth/callback"
    app:
      # app_id: "設定ファイルには記載しない（環境変数のみ）"
      # private_key_path: "設定ファイルには記載しない（環境変数のみ）"
    organization: "tacokumo"
    default_role: "viewer"

  jwt:
    # private_key_path: "設定ファイルには記載しない（環境変数のみ）"
    # public_key_path: "設定ファイルには記載しない（環境変数のみ）"
    access_token_duration: "1h"
    refresh_token_duration: "8h"

  valkey:
    url: "redis://localhost:6379"
    db: 0
    # password: "設定ファイルには記載しない（環境変数のみ）"

security:
  rate_limit:
    auth_attempts: 10  # per minute per IP
    token_generation: 5  # per minute per user
  csrf:
    state_length: 16
  cors:
    allowed_origins: ["http://localhost:3000"]
```

#### Go構造体設計（構造体タグベース）

```go
// pkg/config/config.go
package config

import (
    "time"
)

type Config struct {
    Server   ServerConfig   `yaml:"server"`
    Auth     AuthConfig     `yaml:"auth"`
    Security SecurityConfig `yaml:"security"`
}

type ServerConfig struct {
    Port       int    `yaml:"port" env:"SERVER_PORT" default:"8080"`
    PortalName string `yaml:"portal_name" env:"PORTAL_NAME" default:"TACOKUMO Portal"`
    LogLevel   string `yaml:"log_level" env:"LOG_LEVEL" default:"info"`
}

type AuthConfig struct {
    GitHub GitHubConfig `yaml:"github"`
    JWT    JWTConfig    `yaml:"jwt"`
    Valkey ValkeyConfig `yaml:"valkey"`
}

type GitHubConfig struct {
    OAuth        GitHubOAuthConfig `yaml:"oauth"`
    App          GitHubAppConfig   `yaml:"app"`
    Organization string            `yaml:"organization" env:"GITHUB_ORGANIZATION" default:"tacokumo"`
    DefaultRole  string            `yaml:"default_role" env:"AUTH_GITHUB_DEFAULT_ROLE" default:"viewer"`
}

type GitHubOAuthConfig struct {
    ClientID     string `yaml:"-" env:"GITHUB_CLIENT_ID"`  // 機密情報：設定ファイル無視
    ClientSecret string `yaml:"-" env:"GITHUB_CLIENT_SECRET"`  // 機密情報：設定ファイル無視
    RedirectURL  string `yaml:"redirect_url" env:"GITHUB_OAUTH_REDIRECT_URL"`
}

type GitHubAppConfig struct {
    AppID          int64  `yaml:"-" env:"GITHUB_APP_ID"`  // 機密情報：設定ファイル無視
    PrivateKeyPath string `yaml:"-" env:"GITHUB_APP_PRIVATE_KEY_PATH"`  // 機密情報：設定ファイル無視
}

type JWTConfig struct {
    PrivateKeyPath       string        `yaml:"-" env:"JWT_PRIVATE_KEY_PATH"`  // 機密情報：設定ファイル無視
    PublicKeyPath        string        `yaml:"-" env:"JWT_PUBLIC_KEY_PATH"`   // 機密情報：設定ファイル無視
    AccessTokenDuration  time.Duration `yaml:"access_token_duration" env:"JWT_ACCESS_TOKEN_DURATION" default:"1h"`
    RefreshTokenDuration time.Duration `yaml:"refresh_token_duration" env:"JWT_REFRESH_TOKEN_DURATION" default:"8h"`
}

type ValkeyConfig struct {
    URL      string `yaml:"url" env:"VALKEY_URL" default:"redis://localhost:6379"`
    DB       int    `yaml:"db" env:"VALKEY_DB" default:"0"`
    Password string `yaml:"-" env:"VALKEY_PASSWORD"`  // 機密情報：設定ファイル無視
}

type SecurityConfig struct {
    RateLimit RateLimitConfig `yaml:"rate_limit"`
    CSRF      CSRFConfig      `yaml:"csrf"`
    CORS      CORSConfig      `yaml:"cors"`
}

type RateLimitConfig struct {
    AuthAttempts     int `yaml:"auth_attempts" env:"RATE_LIMIT_AUTH_ATTEMPTS" default:"10"`
    TokenGeneration  int `yaml:"token_generation" env:"RATE_LIMIT_TOKEN_GENERATION" default:"5"`
}

type CSRFConfig struct {
    StateLength int `yaml:"state_length" env:"CSRF_STATE_LENGTH" default:"16"`
}

type CORSConfig struct {
    AllowedOrigins []string `yaml:"allowed_origins" env:"CORS_ALLOWED_ORIGINS" env-separator:","`
}
```

### 実装アプローチ

#### 設定ローダーの実装

```go
// pkg/config/loader.go
package config

import (
    "flag"
    "os"

    "github.com/ilyakaznacheev/cleanenv"
    "github.com/cockroachdb/errors"
)

// Load 設定を読み込みます
// 優先順位: 1. コマンドラインフラグ 2. 環境変数 3. 設定ファイル 4. デフォルト値
func Load() (*Config, error) {
    var configPath string
    flag.StringVar(&configPath, "config", "", "path to config file")
    flag.Parse()

    cfg := &Config{}

    // 設定ファイルが指定されている場合は読み込み
    if configPath != "" {
        if err := cleanenv.ReadConfig(configPath, cfg); err != nil {
            return nil, errors.Wrap(err, "failed to read config file")
        }
    } else {
        // デフォルト設定ファイルの検索
        defaultPaths := []string{
            "./config.yaml",
            "./config/config.yaml",
            "/etc/portal-api/config.yaml",
        }

        for _, path := range defaultPaths {
            if _, err := os.Stat(path); err == nil {
                if err := cleanenv.ReadConfig(path, cfg); err != nil {
                    return nil, errors.Wrapf(err, "failed to read default config file: %s", path)
                }
                break
            }
        }
    }

    // 環境変数からの読み込み（設定ファイルの値を上書き）
    if err := cleanenv.ReadEnv(cfg); err != nil {
        return nil, errors.Wrap(err, "failed to read environment variables")
    }

    // 設定検証
    if err := cfg.Validate(); err != nil {
        return nil, errors.Wrap(err, "config validation failed")
    }

    return cfg, nil
}

// Validate 設定の検証を行います
func (c *Config) Validate() error {
    // 必須項目の検証
    if c.Auth.GitHub.OAuth.ClientID == "" {
        return errors.New("GITHUB_CLIENT_ID is required")
    }
    if c.Auth.GitHub.OAuth.ClientSecret == "" {
        return errors.New("GITHUB_CLIENT_SECRET is required")
    }
    if c.Auth.JWT.PrivateKeyPath == "" {
        return errors.New("JWT_PRIVATE_KEY_PATH is required")
    }
    if c.Auth.JWT.PublicKeyPath == "" {
        return errors.New("JWT_PUBLIC_KEY_PATH is required")
    }

    // 値の範囲検証
    if c.Server.Port < 1 || c.Server.Port > 65535 {
        return errors.New("server port must be between 1 and 65535")
    }

    if c.Security.RateLimit.AuthAttempts < 1 {
        return errors.New("auth attempts rate limit must be greater than 0")
    }

    return nil
}
```

### 構造体タグベース方式の利点

#### 1. 柔軟な環境変数名マッピング
```go
// 既存の環境変数名を維持可能
PortalName string `yaml:"portal_name" env:"PORTAL_NAME"`

// ADR004の新しい環境変数名も自由に設定
DefaultRole string `yaml:"default_role" env:"AUTH_GITHUB_DEFAULT_ROLE"`
```

#### 2. 機密情報の明確な分離
```go
// 設定ファイルには出力されない機密情報
ClientSecret string `yaml:"-" env:"GITHUB_CLIENT_SECRET"`

// 設定ファイルと環境変数両方で設定可能な非機密情報
RedirectURL string `yaml:"redirect_url" env:"GITHUB_OAUTH_REDIRECT_URL"`
```

#### 3. 後方互換性の保持
```go
// 既存のPORTAL_NAME環境変数はそのまま利用可能
PortalName string `yaml:"portal_name" env:"PORTAL_NAME" default:"TACOKUMO Portal"`
```

### 使用方法の例

#### 開発環境
```bash
# config.yaml でベース設定を定義
# 機密情報のみ環境変数で設定
export GITHUB_CLIENT_ID="your_dev_client_id"
export GITHUB_CLIENT_SECRET="your_dev_client_secret"
export JWT_PRIVATE_KEY_PATH="./keys/jwt-private-key.pem"
export JWT_PUBLIC_KEY_PATH="./keys/jwt-public-key.pem"

./portal-api --config ./config/dev.yaml
```

#### 本番環境（Kubernetes）
```yaml
# ConfigMap で非機密設定
apiVersion: v1
kind: ConfigMap
metadata:
  name: portal-api-config
data:
  config.yaml: |
    server:
      port: 8080
      log_level: "info"
    auth:
      github:
        organization: "tacokumo"
        oauth:
          redirect_url: "https://portal.tacokumo.com/auth/callback"

---
# Secret で機密設定
apiVersion: v1
kind: Secret
metadata:
  name: portal-api-secrets
type: Opaque
stringData:
  GITHUB_CLIENT_ID: "your_prod_client_id"
  GITHUB_CLIENT_SECRET: "your_prod_client_secret"
  JWT_PRIVATE_KEY_PATH: "/keys/jwt-private-key.pem"
  JWT_PUBLIC_KEY_PATH: "/keys/jwt-public-key.pem"
  VALKEY_PASSWORD: "your_prod_password"
```

### 設定生成ツールの提供

#### config init コマンドの実装
```go
// cmd/config.go
func initConfigCommand() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "init",
        Short: "Generate sample configuration file",
        RunE: func(cmd *cobra.Command, args []string) error {
            // サンプル設定ファイルを生成
            cfg := &config.Config{
                Server: config.ServerConfig{
                    Port:       8080,
                    PortalName: "TACOKUMO Portal",
                    LogLevel:   "info",
                },
                // ... デフォルト値で初期化
            }

            return generateConfigFile("config.yaml", cfg)
        },
    }
    return cmd
}
```

使用例:
```bash
./portal-api config init  # config.yaml を生成
./portal-api config validate --config ./config.yaml  # 設定ファイル検証
./portal-api config show --config ./config.yaml  # 現在の設定表示（機密情報マスク）
```

## Consequences

### Positive

#### 開発者体験の大幅改善
- **設定の可視化**: YAML形式により設定構造が一目で理解可能
- **コメント活用**: 各設定項目の説明と例を設定ファイルに記述
- **エラー削減**: タイポや設定ミスの削減（IDEのYAMLサポート活用）
- **新人開発者の導入簡化**: サンプル設定ファイルで即座に開発環境構築

#### 運用効率の向上
- **階層的設定管理**: 関連設定のグループ化により管理が簡単
- **環境差分の明確化**: 開発・ステージング・本番の設定差分が明確
- **デプロイメント簡素化**: ConfigMapとSecretの適切な使い分け
- **設定変更の影響範囲限定**: 構造化により変更箇所の特定が容易

#### セキュリティの強化
- **機密情報の明確分離**: `yaml:"-"`により設定ファイルへの機密情報出力を防止
- **監査性向上**: どの設定が機密でどれが非機密かが構造体で明確
- **設定漏洩リスク低減**: 設定ファイルを安全に共有・バージョン管理可能

#### 拡張性とメンテナンス性
- **構造体タグの柔軟性**: 環境変数名を自由に設定、既存システムとの整合
- **型安全性**: Go の型システムによる設定値の安全性確保
- **バリデーション統合**: 設定読み込み時の自動検証
- **テスタビリティ**: 設定のモックや差し替えが容易

### Negative

#### 実装コストの増加
- **学習曲線**: 開発チーム向けの新しい設定システム教育
- **初期実装工数**: cleanenv ライブラリの統合と既存コードの移行
- **ドキュメンテーション**: 新しい設定システムの運用ドキュメント作成

#### 複雑性の追加
- **YAML構文エラー**: インデントミスによる設定読み込みエラーのリスク
- **設定の分散**: 設定ファイルと環境変数の2箇所での管理
- **デバッグ複雑化**: 設定値がどこから読み込まれたかの追跡が必要

#### 依存関係の追加
- **外部ライブラリ**: cleanenv への依存追加
- **バージョン管理**: ライブラリ更新時の互換性確認が必要

### Risks & Mitigations

| リスク | 発生確率 | 影響度 | 軽減策 |
|--------|----------|--------|--------|
| YAML構文エラー | 中 | 中 | 設定検証コマンド、CI/CDでの自動検証 |
| 機密情報の誤設定 | 低 | 高 | yaml:"-" タグ、設定表示時のマスク |
| 設定の不整合 | 中 | 中 | バリデーション機能、単体テスト |
| 既存環境変数の破損 | 低 | 高 | 後方互換性テスト、段階的移行 |
| ライブラリの脆弱性 | 低 | 中 | 定期的な依存関係スキャン |

### Migration Strategy

#### Phase 1: 新設定システムの並行実装 (Week 1-2)
1. **cleanenvライブラリの統合**
2. **新しい設定構造体の定義**
3. **設定ローダーの実装**
4. **既存PORTAL_NAME環境変数の後方互換性確保**

#### Phase 2: 設定ファイルとツールの準備 (Week 3)
1. **サンプル設定ファイルの作成**
2. **設定検証・生成ツールの実装**
3. **開発環境での動作確認**

#### Phase 3: ADR004環境変数の段階的移行 (Week 4-5)
1. **認証関連設定の新システム対応**
2. **既存環境変数と新システムの並行運用**
3. **統合テストでの動作確認**

#### Phase 4: 運用ドキュメントと最終移行 (Week 6)
1. **運用ドキュメントの作成**
2. **チーム向けトレーニングの実施**
3. **本格運用開始**

## 技術仕様詳細

### 依存関係

```go
// go.mod への追加
require (
    github.com/ilyakaznacheev/cleanenv v1.6.0  // 設定ファイル + 環境変数管理
)
```

### ファイル構成

```
pkg/config/
├── config.go       # 設定構造体定義
├── loader.go       # 設定読み込み機能
├── validator.go    # 設定検証機能
└── generator.go    # 設定ファイル生成機能

config/
├── config.yaml     # デフォルト設定ファイル
├── dev.yaml        # 開発環境用設定
├── staging.yaml    # ステージング環境用設定
└── prod.yaml       # 本番環境用設定（機密情報除く）

cmd/
└── config.go       # 設定関連CLIコマンド
```

### パフォーマンス考慮

#### 設定読み込み最適化
- **一度だけ読み込み**: アプリケーション起動時にのみ設定を読み込み
- **設定キャッシュ**: 読み込んだ設定をメモリ内でキャッシュ
- **遅延バリデーション**: 使用される設定のみを検証

#### メモリ使用量の最適化
- **構造体最適化**: 必要最小限のフィールド定義
- **環境変数の即座解放**: 読み込み後の環境変数アクセス回避

この設計により、Portal APIの設定管理システムは大幅に改善され、ADR004で要求される認証・認可システムの複雑な設定を効率的に管理できるようになります。

## Implementation Roadmap

### Phase 1: 基盤実装 (Week 1-2)
- [ ] cleanenv ライブラリの統合
- [ ] 新しい設定構造体の定義
- [ ] 基本的な設定ローダーの実装
- [ ] 既存 PORTAL_NAME 環境変数の後方互換性テスト

### Phase 2: 設定ファイル対応 (Week 3)
- [ ] サンプルconfig.yamlの作成
- [ ] 設定検証機能の実装
- [ ] 設定生成・表示ツールの実装
- [ ] 開発環境での動作確認

### Phase 3: ADR004対応 (Week 4-5)
- [ ] 認証関連の全環境変数をサポート
- [ ] 機密情報分離機能の実装
- [ ] 統合テストの作成と実行
- [ ] ドキュメンテーションの更新

### Phase 4: 運用準備 (Week 6)
- [ ] Kubernetes ConfigMap/Secret サンプルの作成
- [ ] 運用ドキュメントの完成
- [ ] チーム向けマイグレーションガイドの作成
- [ ] 本格運用開始の準備完了