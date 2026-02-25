# Portal API

TACOKUMO Portal の API サーバーです。GitHub OAuth認証、JWT トークン管理、Kubernetes リソース管理を提供します。

## 特徴

- **認証・認可**: GitHub OAuth + JWT
- **セッション管理**: Valkey (Redis互換)
- **Kubernetes連携**: Kubernetes APIクライアント
- **OpenAPI**: 自動生成されたAPIクライアント/サーバー
- **Docker対応**: マルチステージビルド + Docker Compose

## アーキテクチャ

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Frontend      │    │   Portal API    │    │   Kubernetes    │
│                 │────│                 │────│    Cluster      │
│ (React/Next.js) │    │  (Go + Echo)    │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                              │
                       ┌─────────────────┐
                       │     Valkey      │
                       │ (Session Store) │
                       └─────────────────┘
```

## クイックスタート

### 前提条件

- Docker & Docker Compose
- Make
- GitHub OAuth App の作成

### 1. GitHub OAuth App の設定

1. [GitHub Developer Settings](https://github.com/settings/developers) にアクセス
2. "New OAuth App" をクリック
3. 以下の設定で作成:
   - **Application name**: `TACOKUMO Portal (Local)`
   - **Homepage URL**: `http://localhost:8080`
   - **Authorization callback URL**: `http://localhost:8080/auth/github/callback`
4. Client ID と Client Secret をメモする

### 2. 開発環境のセットアップ

```bash
# リポジトリをクローン
git clone https://github.com/tacokumo/portal-api.git
cd portal-api

# 開発環境をセットアップ
make setup-dev

# .envファイルを編集してGitHub OAuth設定を記入
vim .env
```

`.env`ファイルの例:
```bash
GITHUB_CLIENT_ID=your-client-id-here
GITHUB_CLIENT_SECRET=your-client-secret-here
GITHUB_OAUTH_REDIRECT_URL=http://localhost:8080/auth/github/callback
```

### 3. 開発サーバーの起動

```bash
# 開発環境を起動
make dev-up

# ログを確認
make dev-logs
```

サーバーが起動したら http://localhost:8080 にアクセスしてください。

### 4. 動作確認

```bash
# ヘルスチェック
make health-check

# APIエンドポイント確認
curl http://localhost:8080/healthz
```

## 開発用コマンド

### Docker Compose

```bash
# 開発環境
make dev-up        # 開発環境を起動
make dev-down      # 開発環境を停止
make dev-logs      # ログを表示
make dev-rebuild   # イメージを再ビルドして起動

# 本番用
make docker-up     # 本番環境を起動
make docker-down   # 環境を停止
make docker-logs   # ログを表示
make docker-clean  # 全てのコンテナとボリュームを削除
```

### 開発・ビルド

```bash
make build         # バイナリをビルド
make test          # テストを実行
make lint          # リンターを実行
make format        # コードフォーマット
make generate      # OpenAPI コードを生成
```

### デバッグ・トラブルシューティング

```bash
make health-check  # サービスの健全性確認
make debug-logs    # デバッグログを表示
```

## API仕様

OpenAPI仕様書は `apis/v1alpha1/openapi.yaml` に定義されています。

### 主要エンドポイント

- `GET /healthz` - ヘルスチェック
- `POST /auth/github` - GitHub OAuth認証開始
- `GET /auth/github/callback` - GitHub OAuth コールバック
- `POST /auth/refresh` - JWTトークン更新
- `GET /applications` - アプリケーション一覧
- `POST /applications` - アプリケーション作成

## プロジェクト構造

```
.
├── apis/v1alpha1/           # OpenAPI 仕様
│   └── openapi.yaml
├── cmd/server/              # エントリーポイント
├── config/                  # 設定ファイル
├── docs/                    # ドキュメント
├── internal/                # 内部パッケージ
├── pkg/                     # 公開パッケージ
│   ├── apis/v1alpha1/       # 自動生成APIコード
│   ├── auth/                # 認証・認可
│   ├── config/              # 設定管理
│   ├── k8sclient/          # Kubernetesクライアント
│   ├── platform/           # サーバープラットフォーム
│   └── rbac/               # RBAC
├── scripts/                 # スクリプト
├── docker-compose.yaml      # Docker Compose設定
├── docker-compose.dev.yaml  # 開発用オーバーライド
└── Dockerfile              # Dockerイメージ設定
```

## 設定

### 環境変数

主要な環境変数:

| 変数名 | 説明 | デフォルト |
|--------|------|------------|
| `SERVER_PORT` | サーバーポート | `8080` |
| `LOG_LEVEL` | ログレベル | `info` |
| `GITHUB_CLIENT_ID` | GitHub OAuth Client ID | - |
| `GITHUB_CLIENT_SECRET` | GitHub OAuth Client Secret | - |
| `VALKEY_ADDRESS` | Valkeyサーバーアドレス | `localhost:6379` |
| `JWT_PRIVATE_KEY_PATH` | JWT秘密鍵パス | `/etc/secrets/jwt-private-key.pem` |

詳細は `config/environment-variables.env.example` を参照してください。

### JWT鍵の管理

JWT鍵は自動生成されますが、手動で生成することも可能です:

```bash
# JWT鍵ペアを生成
./scripts/generate-jwt-keys.sh
```

## 認証フロー

1. ユーザーが認証エンドポイントにアクセス
2. GitHub OAuth認証画面にリダイレクト
3. ユーザーが認証を許可
4. GitHub からコールバックでAuthCodeを受信
5. AuthCodeを使ってアクセストークンを取得
6. ユーザー情報を取得してJWTトークンを発行
7. セッション情報をValkeyに保存

## コントリビューション

1. フィーチャーブランチを作成
2. 変更を実装
3. テストを実行: `make test`
4. リンターを実行: `make lint`
5. プルリクエストを作成

## トラブルシューティング

### よくある問題

**Q: GitHub OAuth認証が失敗する**
A: GitHub OAuth Appの設定を確認してください。特にCallback URLが正確であることを確認してください。

**Q: Valkey接続エラーが発生する**
A: Valkeyコンテナが起動していることを確認: `docker ps | grep valkey`

**Q: JWT鍵が見つからないエラー**
A: `make setup-secrets` を実行してJWT鍵を生成してください。

**Q: ポート8080が使用中**
A: 他のプロセスがポートを使用していないか確認: `lsof -i :8080`

詳細なトラブルシューティングは [docs/local-development.md](docs/local-development.md) を参照してください。

## ライセンス

このプロジェクトは MIT ライセンスの下で公開されています。