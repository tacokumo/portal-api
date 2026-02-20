# ADR004 認証・認可

## Status

Proposed

## Context

Portal APIでは、GitHubベースの認証・認可システムが必要です。
TACOKUMOプラットフォームにアクセスするユーザーは、GitHubアカウントを持つ開発者であり、
GitHub Organizationへの所属やTeamでの役割に基づいてアクセス制御を行う必要があります。

### 要件

以下のような要件があります：

1. **多様なクライアント対応**
   - フロントエンドアプリケーション（Next.jsなど）
   - CLI
   - GitHub Actions（CI/CD パイプライン）
   - Terraform Provider などのAPIクライアントを内包するツール

2. **GitHub 組織ベースのアクセス制御**
   - GitHub Organization への所属確認（public/private メンバーシップ両方）
   - GitHub Team ベースの役割管理
   - リポジトリレベルでのアクセス制御

3. **セキュリティ要件**
   - セッション管理によるセキュアなアクセス制御
   - トークンの適切なライフサイクル管理
   - CSRF、XSS等の一般的な攻撃への対策

4. **運用要件**
   - 監査ログの記録
   - セッションの強制無効化
   - 障害時の適切なフォールバック

## Decision Drivers

- **開発者体験の向上**: 複数の認証方式をシームレスにサポート
- **セキュリティの確保**: 業界標準のセキュリティプラクティスを採用
- **運用性の担保**: 監査機能とトラブルシューティング機能を提供
- **拡張性の確保**: 将来的な権限モデルの拡張に対応
- **GitHub エコシステムとの統合**: GitHub の既存機能を最大限活用

## Alternatives Considered

### 1. 認証プロバイダーの選択

| オプション | メリット | デメリット | 評価 |
|------------|----------|------------|------|
| GitHub OAuth のみ | GitHub エコシステムとの完全統合 | CLI/Actions での利用が複雑 | ❌ |
| Auth0/Okta 等の外部IdP | 豊富な機能、企業向け機能 | GitHub との連携が複雑、追加コスト | ❌ |
| **GitHub ベースのハイブリッド方式** | 各クライアントタイプに最適化、統一された権限管理 | 実装の複雑性 | ✅ 採用 |

### 2. セッション管理方式

| オプション | メリット | デメリット | 評価 |
|------------|----------|------------|------|
| ステートレス JWT のみ | スケーラビリティ | セッション無効化が困難 | ❌ |
| サーバーサイドセッション | 完全な制御 | スケーラビリティの課題 | ❌ |
| **JWT + Redis ハイブリッド** | スケーラビリティと制御のバランス | 適度な複雑性 | ✅ 採用 |

### 3. 権限管理方式

| オプション | メリット | デメリット | 評価 |
|------------|----------|------------|------|
| 単純な Admin/User | 実装が簡単 | 柔軟性に欠ける | ❌ |
| **GitHub Team ベース RBAC** | GitHub の権限構造と整合、柔軟性 | GitHub Team 管理への依存 | ✅ 採用 |
| カスタム RBAC | 完全な制御 | 実装・運用コストが高い | ❌ |

## Decision Outcome

認証・認可システムとして、GitHubをIdentity Providerとして利用し、
JWTとValkeyを組み合わせたセッション管理を採用します。

### 認証方式

以下の3つの認証方式をサポートします：

#### 1. GitHub OAuth - フロントエンドアプリケーション向け

```
Client App -> Portal API -> GitHub OAuth -> Portal API -> Client App
     |              |                              |
     |              |                              |
   Session      JWT Token                    Refresh Token
  Creation      Generation                    (stored in Valkey)
```

**フロー:**
1. フロントエンドが認証要求を送信
2. Portal API が GitHub OAuth認証URLを生成（state パラメータ付き）
3. ユーザーが GitHub で認証
4. GitHub が認証コード付きでコールバック
5. Portal API が認証コードをアクセストークンに交換
6. GitHub API でユーザー情報・Organization・Team情報を取得
7. JWT（アクセストークン）とリフレッシュトークンを生成
8. リフレッシュトークンを Valkey に保存
9. JWT をクライアントに返却

**必要な GitHub 権限:**
- `read:user` - ユーザー情報取得
- `read:org` - Organization メンバーシップ確認
- `read:team` - Team メンバーシップ確認

**エラーハンドリング:**
- `invalid_request`: OAuth パラメータエラー → 400 Bad Request
- `access_denied`: ユーザーがアクセス拒否 → 401 Unauthorized
- `state_mismatch`: CSRF 攻撃の可能性 → 400 Bad Request
- `github_api_error`: GitHub API エラー → 502 Bad Gateway
- `organization_not_member`: Organization 非メンバー → 403 Forbidden

#### 2. GitHub Personal Access Token (PAT) - CLI/APIクライアント向け

```
CLI Tool -> Portal API -> GitHub API (User/Org/Team info)
    |            |               |
    |            |               |
  PAT Token  Validation     Team Role
                            Mapping
```

**フロー:**
1. CLI/APIクライアント ユーザーが PAT を Portal API に送信
2. Portal API が PAT を使用して GitHub API にアクセス
3. ユーザー情報、Organization、Team 情報を取得
4. 権限を検証し、JWT を生成
5. JWT をレスポンスとして返却

**必要な PAT スコープ:**
- `read:user`
- `read:org`
- `read:team`

**エラーハンドリング:**
- `invalid_token`: 不正な PAT → 401 Unauthorized
- `insufficient_scope`: 必要なスコープが不足 → 403 Forbidden
- `rate_limit_exceeded`: GitHub API レート制限 → 429 Too Many Requests
- `token_expired`: PAT の有効期限切れ → 401 Unauthorized

#### 3. GitHub Installation Access Token - GitHub Actions向け

```
GitHub Actions -> Portal API -> GitHub API (Installation info)
       |             |                |
       |             |                |
  Installation   Validation      Repository
     Token                         Access
```

**フロー:**
1. GitHub Actions が Installation Access Token を送信
2. Portal API が token を検証し Installation 情報を取得
3. Installation が管理するリポジトリ一覧を取得
4. リポジトリベースの権限を設定し JWT を生成
5. JWT をレスポンスとして返却

**必要な GitHub App 権限:**
- `metadata: read` - Installation 情報取得
- `members: read` - Organization メンバー確認（オプション）

**エラーハンドリング:**
- `invalid_installation_token`: 不正な Installation Token → 401 Unauthorized
- `installation_suspended`: Installation が停止状態 → 403 Forbidden
- `insufficient_permissions`: 必要な権限が不足 → 403 Forbidden

### 認可設計

#### GitHub Team ベースの RBAC

**固定ロールマッピング:**
- `viewer`: 読み取り専用アクセス
- `writer`: 読み書きアクセス

**Team マッピング戦略:**
```yaml
# Organization レベル設定例
org_settings:
  tacokumo:
    default_role: viewer
    team_mappings:
      - team: "admin"
        role: writer
      - team: "maintainers"
        role: writer
      - team: "developers"
        role: viewer
```

**権限キャッシュ戦略:**
- **キャッシュ期間**: 15分（バランス重視）
- **キャッシュキー**: `user:{user_id}:permissions`
- **キャッシュ内容**: ユーザーのロール、所属 Organization、Team 情報
- **無効化タイミング**:
  - JWT 有効期限切れ時
  - ログアウト時
  - 手動でのセッション無効化時

**Organization 構造変更への対応:**
- Team メンバーシップの変更は次回認証時に反映
- 緊急時のアクセス無効化機能を提供
- 定期的な Team 情報同期ジョブ（1日1回）

**将来の拡張指針:**
- カスタムロールの追加対応
- リポジトリレベル権限の細分化
- 一時的な権限昇格機能

### セッション管理

#### JWT 設計

**JWT ペイロード構造:**
```json
{
  "sub": "github_user_id",
  "name": "username",
  "roles": ["viewer", "writer"],
  "orgs": ["tacokumo"],
  "teams": ["developers", "maintainers"],
  "auth_method": "oauth|pat|installation",
  "iat": 1640995200,
  "exp": 1640998800,
  "jti": "session_id"
}
```

**トークンライフサイクル:**
- **アクセストークン（JWT）**: 1時間
- **リフレッシュトークン**: 8時間（OAuth のみ）
- **署名アルゴリズム**: RS256
- **キーローテーション**: 月次（自動化）

#### Valkey セッション管理

**セッション情報構造:**
```json
{
  "user_id": "github_user_id",
  "session_id": "unique_session_id",
  "refresh_token": "encrypted_refresh_token",
  "auth_method": "oauth|pat|installation",
  "created_at": "2024-01-01T00:00:00Z",
  "last_accessed": "2024-01-01T01:00:00Z",
  "user_agent": "Mozilla/5.0...",
  "ip_address": "192.168.1.1"
}
```

**キー設計:**
- セッション: `session:{session_id}`
- ユーザーセッション一覧: `user:{user_id}:sessions`
- リフレッシュトークン: `refresh:{refresh_token_hash}`

**セッション更新フロー:**
```
Client -> API: Expired JWT + Refresh Token
API -> Valkey: Validate Refresh Token
API -> GitHub: Refresh User Info (if needed)
API -> Client: New JWT + New Refresh Token
API -> Valkey: Update Session Info
```

**ログアウト・無効化:**
- **通常ログアウト**: 該当セッションを Valkey から削除
- **全セッション無効化**: ユーザーの全セッションを削除
- **強制無効化**: 管理者による特定ユーザーのセッション削除

**Valkey障害時の対応:**
- **Read 失敗**: JWT のみで認証継続（制限時間内）
- **Write 失敗**: 新規ログインを一時停止、既存セッション継続
- **完全障害**: 緊急時用のメンテナンスモードに移行

### 技術スタック

#### 依存関係

**追加必要な依存関係:**
```go
// go.mod に追加
require (
    github.com/golang-jwt/jwt/v5 v5.3.1        // JWT生成・検証
    github.com/redis/go-redis/v9 v9.18.0       // Valkeyクライアント
    // golang.org/x/oauth2 v0.35.0 は既存
)
```

#### 環境変数・設定

**必要な環境変数:**
```bash
# GitHub OAuth
GITHUB_CLIENT_ID=your_client_id
GITHUB_CLIENT_SECRET=your_client_secret
GITHUB_OAUTH_REDIRECT_URL=http://localhost:8080/auth/callback

# GitHub App
GITHUB_APP_ID=123456
GITHUB_APP_PRIVATE_KEY_PATH=/path/to/private-key.pem

# JWT
JWT_PRIVATE_KEY_PATH=/path/to/jwt-private-key.pem
JWT_PUBLIC_KEY_PATH=/path/to/jwt-public-key.pem
JWT_ACCESS_TOKEN_DURATION=1h
JWT_REFRESH_TOKEN_DURATION=8h

# Valkey
VALKEY_URL=redis://localhost:6379
VALKEY_DB=0
VALKEY_PASSWORD=your_password

# Organization 設定
GITHUB_ORGANIZATION=tacokumo
DEFAULT_ROLE=viewer
```

### セキュリティ考慮事項

#### 認証セキュリティ

**JWT セキュリティ:**
- **最小限ペイロード**: ユーザーID、ロール、有効期限のみ
- **署名検証**: RS256 による署名必須検証
- **有効期限厳守**: 期限切れトークンの即座拒否
- **JTI による重複防止**: トークン固有IDでリプレイ攻撃防止

**OAuth セキュリティ:**
- **State パラメータ**: CSRF攻撃防止（128bit ランダム）
- **PKCE**: コード横取り攻撃防止（推奨）
- **Secure Cookie**: セッション情報の安全な保存
- **短期認証コード**: 5分以内の認証コード利用

**Token 保管:**
- **暗号化保存**: Valkey 内のセンシティブデータは暗号化
- **メモリ内処理**: 平文トークンの永続化回避
- **安全な削除**: メモリからの確実なデータ削除

#### インフラセキュリティ

**Transport Security:**
- **HTTPS必須**: 全通信のTLS 1.3以上での暗号化
- **HSTS**: HTTP Strict Transport Security の有効化
- **Certificate Pinning**: 可能な環境では証明書ピニング

**Secrets管理:**
- **GitHub App秘密鍵**: Kubernetes Secret または外部 Vault
- **JWT署名鍵**: 定期ローテーション対応
- **OAuth Client Secret**: 環境変数での安全な管理
- **Valkey認証**: 強固なパスワードとTLS接続

**ネットワークセキュリティ:**
- **CORS設定**: 必要最小限のオリジンのみ許可
- **Rate Limiting**: 認証エンドポイントへの制限
- **IP Whitelist**: 管理機能のIP制限（必要に応じて）

#### 運用セキュリティ

**監査ログ:**
- **認証イベント**: 成功・失敗の全ログ記録
- **権限変更**: ロール昇格・降格のログ
- **セッション操作**: 作成・無効化・期限切れ
- **異常パターン**: 短期間の大量失敗等

**Rate Limiting & Abuse対策:**
- **認証試行制限**: IP単位 10回/分
- **Token生成制限**: ユーザー単位 5回/分
- **GitHub API制限**: レート制限遵守とバックオフ
- **Brute Force対策**: 一定失敗後のアカウントロック

**インシデント対応:**
- **セキュリティ侵害時**: 全セッション強制無効化機能
- **Token漏洩時**: 該当トークンの即座無効化
- **GitHub障害時**: 認証サービス縮退モード
- **監査証跡**: セキュリティ調査のための詳細ログ

### 実装指針

#### API エンドポイント設計

```yaml
# OpenAPI セキュリティ定義追加
components:
  securitySchemes:
    BearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT
    PersonalAccessToken:
      type: http
      scheme: bearer
      description: GitHub Personal Access Token
    InstallationToken:
      type: http
      scheme: bearer
      description: GitHub Installation Access Token

security:
  - BearerAuth: []
  - PersonalAccessToken: []
  - InstallationToken: []
```

**認証関連エンドポイント:**
- `GET /auth/github/login` - OAuth認証開始
- `GET /auth/github/callback` - OAuth コールバック
- `POST /auth/token/refresh` - JWT リフレッシュ
- `POST /auth/logout` - ログアウト
- `POST /auth/verify` - トークン検証

#### 設定ファイル構造

```go
// pkg/config/auth.go
type AuthConfig struct {
    GitHub GitHubConfig `yaml:"github"`
    JWT    JWTConfig    `yaml:"jwt"`
    Valkey ValkeyConfig `yaml:"valkey"`
    Security SecurityConfig `yaml:"security"`
}

type GitHubConfig struct {
    ClientID     string `yaml:"client_id"`
    ClientSecret string `yaml:"client_secret"`
    RedirectURL  string `yaml:"redirect_url"`
    AppID        int64  `yaml:"app_id"`
    PrivateKey   string `yaml:"private_key_path"`
    Organization string `yaml:"organization"`
}
```

#### テスト戦略

**単体テスト:**
- JWT トークン生成・検証ロジック
- GitHub API レスポンスのモック化
- 各認証方式の正常・異常ケース

**統合テスト:**
- 認証フロー全体のエンドツーエンドテスト
- Valkey との連携テスト
- GitHub API モックサーバーとの連携

**セキュリティテスト:**
- 不正トークンでのアクセス試行
- CSRF 攻撃シミュレーション
- Rate Limiting の動作確認

#### デプロイメント考慮事項

**ゼロダウンタイム対応:**
- JWT 署名鍵の段階的ローテーション
- Valkey のクラスター構成でのフェイルオーバー
- ヘルスチェックエンドポイントでの認証サービス状態確認

**スケーリング:**
- ステートレス設計による水平スケーリング対応
- Valkey のマスター・レプリカ構成
- GitHub API レート制限を考慮した分散処理

**モニタリング:**
- 認証成功・失敗率の監視
- JWT 有効期限切れ頻度の追跡
- Valkey 接続プール使用状況
- GitHub API レート制限使用状況

## Consequences

### Positive

- **開発者体験**: 各クライアントタイプに最適化された認証フロー
- **セキュリティ**: 業界標準プラクティスの採用
- **運用性**: 詳細な監査ログと管理機能
- **拡張性**: GitHub エコシステムの機能を活用した柔軟な権限管理
- **パフォーマンス**: JWT とキャッシュによる高速認証

### Negative

- **実装複雑性**: 3つの認証方式への対応
- **外部依存**: GitHub サービスの可用性に依存
- **運用負荷**: トークンローテーション等の定期メンテナンス
- **デバッグ困難**: 分散セッション管理によるトラブルシューティング
- **学習コスト**: チーム向けの認証・認可概念の教育

### Risks & Mitigations

| リスク | 発生確率 | 影響度 | 軽減策 |
|--------|----------|--------|--------|
| GitHub API 障害 | 中 | 高 | キャッシュ延長、縮退モード |
| JWT署名鍵漏洩 | 低 | 高 | 定期ローテーション、検知機能 |
| Valkey障害 | 中 | 中 | レプリカ構成、フォールバック |
| Rate制限到達 | 中 | 中 | バックオフ、複数IP分散 |
| CSRF攻撃 | 低 | 中 | State検証、PKCE |

### Monitoring & Alerting

**重要メトリクス:**
- 認証成功率: >95%
- JWT有効期限切れ率: <10%
- GitHub API エラー率: <5%
- 認証レスポンス時間: <500ms
- Valkey接続プール使用率: <80%

**アラート条件:**
- 認証成功率が90%を下回る
- 連続5分間でGitHub APIエラー率が20%を上回る
- 1分間に同一IPから50回以上の認証失敗
- JWT署名検証失敗が1時間で100件を超える

## Implementation Roadmap

### Phase 1: 基盤実装 (Week 1-2)
- [ ] JWT生成・検証ライブラリの統合
- [ ] Valkey接続とセッション管理の実装
- [ ] 基本的な設定管理の実装
- [ ] ヘルスチェックとメトリクス収集

### Phase 2: GitHub OAuth実装 (Week 3-4)
- [ ] GitHub OAuth フローの実装
- [ ] State パラメータによるCSRF対策
- [ ] ユーザー情報・権限情報の取得
- [ ] JWT トークンの生成・レスポンス

### Phase 3: PAT・Installation Token (Week 5-6)
- [ ] GitHub PAT 検証機能
- [ ] GitHub Installation Token 検証機能
- [ ] 各認証方式の統一インターフェース化
- [ ] エラーハンドリングの標準化

### Phase 4: 認可・セキュリティ (Week 7-8)
- [ ] Team ベース権限マッピング
- [ ] Rate Limiting 機能
- [ ] 監査ログ機能
- [ ] セッション管理画面（管理者向け）

### Phase 5: テスト・運用準備 (Week 9-10)
- [ ] 統合テストの実装
- [ ] セキュリティテストの実施
- [ ] 運用ドキュメントの作成
- [ ] モニタリング・アラートの設定

### Phase 6: 本格運用開始 (Week 11-12)
- [ ] OpenAPI スキーマへのセキュリティ定義追加
- [ ] 既存 API エンドポイントへの認証機能適用
- [ ] 段階的ロールアウトとユーザーフィードバック収集
- [ ] パフォーマンスチューニング