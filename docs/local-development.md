# ローカル開発環境セットアップガイド

このドキュメントでは、Portal API のローカル開発環境を詳細にセットアップする方法を説明します。

## 目次

- [前提条件](#前提条件)
- [GitHub OAuth App の設定](#github-oauth-app-の設定)
- [開発環境のセットアップ](#開発環境のセットアップ)
- [開発ワークフロー](#開発ワークフロー)
- [デバッグ方法](#デバッグ方法)
- [トラブルシューティング](#トラブルシューティング)

## 前提条件

### 必要なソフトウェア

- **Docker**: version 20.10 以上
- **Docker Compose**: version 2.0 以上
- **Make**: GNU Make
- **Git**: version 2.30 以上
- **curl** または **wget**: API テスト用

### 確認コマンド

```bash
# バージョン確認
docker --version
docker compose version
make --version
git --version
```

## GitHub OAuth App の設定

### 1. GitHub OAuth App の作成

1. [GitHub Developer Settings](https://github.com/settings/developers) にアクセス
2. "OAuth Apps" タブを選択
3. "New OAuth App" ボタンをクリック
4. 以下の情報を入力:

| フィールド | 値 |
|-----------|---|
| Application name | `TACOKUMO Portal (Local)` |
| Homepage URL | `http://localhost:8080` |
| Application description | `ローカル開発用TACOKUMO Portal` |
| Authorization callback URL | `http://localhost:8080/auth/github/callback` |

5. "Register application" をクリック

### 2. Client ID と Client Secret の取得

作成されたOAuth Appの詳細画面で:

1. **Client ID** をコピーしてメモ
2. **Generate a new client secret** をクリック
3. 表示される **Client Secret** をコピーしてメモ

   ⚠️ **重要**: Client Secret は一度しか表示されません。必ずメモしてください。

### 3. GitHub App の設定（オプション）

より高度な機能を使用する場合、GitHub App も作成できます:

1. [GitHub Developer Settings](https://github.com/settings/developers) の "GitHub Apps" タブ
2. "New GitHub App" をクリック
3. 必要な権限を設定
4. Private Key をダウンロードして `secrets/github-app-private-key.pem` に配置

## 開発環境のセットアップ

### 1. リポジトリのクローン

```bash
git clone https://github.com/tacokumo/portal-api.git
cd portal-api
```

### 2. 自動セットアップ（推奨）

```bash
# 全てを自動でセットアップ
make setup-dev
```

このコマンドは以下を実行します:
- `.env` ファイルの作成
- `secrets/` ディレクトリの作成
- JWT鍵ペアの生成

### 3. 手動セットアップ

#### 3.1 環境変数ファイルの作成

```bash
# .env ファイルを作成
cp .env.development.example .env
```

#### 3.2 .env ファイルの編集

```bash
# エディタで .env を開く
vim .env
# または
nano .env
```

最低限、以下の項目を設定してください:

```bash
# GitHub OAuth設定 (必須)
GITHUB_CLIENT_ID=your-actual-client-id-here
GITHUB_CLIENT_SECRET=your-actual-client-secret-here
GITHUB_OAUTH_REDIRECT_URL=http://localhost:8080/auth/github/callback

# GitHub App設定 (オプション)
GITHUB_APP_ID=your-github-app-id
```

#### 3.3 JWT鍵の生成

```bash
# secrets ディレクトリを作成
mkdir -p secrets

# JWT鍵ペアを生成
./scripts/generate-jwt-keys.sh
```

### 4. 開発環境の起動

```bash
# 開発環境を起動
make dev-up
```

初回起動時は以下が実行されます:
1. Docker イメージのビルド
2. Valkey コンテナの起動
3. Portal API コンテナの起動（開発モード）

### 5. 動作確認

```bash
# ヘルスチェック
make health-check

# または手動確認
curl http://localhost:8080/healthz
```

期待される応答:
```json
{
  "status": "healthy",
  "timestamp": "2026-02-24T10:00:00Z"
}
```

## 開発ワークフロー

### 基本的な開発サイクル

1. **ソースコードの変更**
   - `cmd/`, `internal/`, `pkg/` 配下のファイルを編集
   - 変更後は手動で開発環境を再起動

2. **API テスト**
   ```bash
   # ログを確認
   make dev-logs

   # 特定のエンドポイントをテスト
   curl -X GET http://localhost:8080/applications
   ```

3. **データベースの確認**
   ```bash
   # Valkey に接続
   docker exec -it portal-valkey valkey-cli

   # セッションデータを確認
   127.0.0.1:6379> KEYS *
   127.0.0.1:6379> GET "session:some-session-id"
   ```

### OpenAPI の変更

OpenAPI仕様を変更した場合:

```bash
# API コードを再生成
make generate

# 開発環境を再起動
make dev-rebuild
```

### 依存関係の更新

```bash
# Go modules を更新
go mod tidy

# 開発環境を再ビルド
make dev-rebuild
```

## デバッグ方法

### 1. ログの確認

```bash
# リアルタイムログ
make dev-logs

# 特定のサービスのログ
docker logs portal-api -f
docker logs portal-valkey -f

# 過去のログを確認
make debug-logs
```

### 2. Delve デバッガーの使用

開発用 Docker Compose では Delve デバッガーのポート（2345）を公開しています:

```bash
# デバッガー用のビルド設定
# docker-compose.dev.yaml が自動的に設定済み

# VS Code または GoLand でリモートデバッグを設定
# Connect to: localhost:2345
```

### 3. プロファイリング

```bash
# CPU プロファイル
go tool pprof http://localhost:8080/debug/pprof/profile

# メモリプロファイル
go tool pprof http://localhost:8080/debug/pprof/heap

# goroutine プロファイル
go tool pprof http://localhost:8080/debug/pprof/goroutine
```

### 4. データベースデバッグ

```bash
# Valkey コンソールに接続
docker exec -it portal-valkey valkey-cli

# よく使うコマンド
127.0.0.1:6379> INFO                    # サーバー情報
127.0.0.1:6379> DBSIZE                  # キー数
127.0.0.1:6379> KEYS session:*          # セッションキー一覧
127.0.0.1:6379> FLUSHDB                 # テスト用データクリア
```

## トラブルシューティング

### よくある問題と解決方法

#### 1. GitHub OAuth 認証エラー

**症状**: 認証時に `invalid_client` エラー

**解決方法**:
1. `.env` ファイルの `GITHUB_CLIENT_ID` と `GITHUB_CLIENT_SECRET` を確認
2. GitHub OAuth App の Callback URL が正確か確認
   - 正: `http://localhost:8080/auth/github/callback`
   - 誤: `https://localhost:8080/auth/github/callback` (httpsは不可)

**デバッグコマンド**:
```bash
# 環境変数の確認
docker exec portal-api env | grep GITHUB
```

#### 2. JWT鍵エラー

**症状**: `JWT private key not found` エラー

**解決方法**:
```bash
# JWT鍵が存在するか確認
ls -la secrets/

# 鍵が無い場合は生成
make setup-secrets

# 権限を確認
chmod 600 secrets/*.pem
```

#### 3. Valkey 接続エラー

**症状**: `connection refused` to valkey:6379

**解決方法**:
```bash
# Valkey コンテナの状態確認
docker ps | grep valkey

# ネットワーク接続確認
docker exec portal-api ping valkey

# Valkey ログ確認
docker logs portal-valkey
```

#### 4. ポート競合エラー

**症状**: `port 8080 already in use`

**解決方法**:
```bash
# ポートを使用しているプロセスを確認
lsof -i :8080

# 該当プロセスを停止
kill -9 <PID>

# または別のポートを使用
# docker-compose.yaml の ports を変更
ports:
  - "8081:8080"  # ホスト:8081, コンテナ:8080
```

#### 5. Docker イメージビルドエラー

**症状**: Go modules のダウンロードエラー

**解決方法**:
```bash
# Go module キャッシュをクリア
go clean -modcache

# Docker ビルドキャッシュをクリア
docker system prune -f

# 完全に再ビルド
make docker-clean
make dev-up
```

#### 6. ソースコード変更が反映されない

**症状**: ソースコード変更が反映されない

**解決方法**:
```bash
# 開発環境を再起動
make dev-down
make dev-up
```

### ログレベルの調整

デバッグ時により詳細なログが必要な場合:

```bash
# .env ファイルでログレベルを変更
echo "LOG_LEVEL=debug" >> .env

# 開発環境を再起動
make dev-rebuild
```

### パフォーマンス分析

開発環境でのパフォーマンス問題の分析:

```bash
# コンテナのリソース使用量
docker stats

# Go アプリケーションの詳細分析
docker exec portal-api go tool pprof -http=:8081 /debug/pprof/profile
```

### データベースのリセット

開発中にデータをリセットしたい場合:

```bash
# Valkey データのみリセット
docker exec portal-valkey valkey-cli FLUSHALL

# 完全なリセット（ボリューム含む）
make docker-clean
make dev-up
```

## 高度な設定

### HTTPS対応（オプション）

本番環境に近い環境でテストする場合:

1. SSL証明書を生成:
   ```bash
   # 自己署名証明書を作成
   openssl req -x509 -newkey rsa:4096 -keyout secrets/server-key.pem -out secrets/server-cert.pem -days 365 -nodes
   ```

2. nginx プロキシを追加:
   ```bash
   # docker-compose.https.yaml を作成して nginx サービスを追加
   docker compose -f docker-compose.yaml -f docker-compose.dev.yaml -f docker-compose.https.yaml up -d
   ```

### 外部データベース接続

外部のValkey/Redisサーバーを使用する場合:

```bash
# .env ファイルで外部サーバーを指定
VALKEY_ADDRESS=your-external-redis:6379
VALKEY_PASSWORD=your-password

# ローカルのValkeyサービスを無効化
docker compose up -d portal-api  # valkey サービスは起動しない
```

### プロダクションビルドのテスト

```bash
# プロダクション用イメージでテスト
docker compose -f docker-compose.yaml up -d

# パフォーマンステスト
ab -n 1000 -c 10 http://localhost:8080/healthz
```

## 参考資料

- [Echo Framework ドキュメント](https://echo.labstack.com/)
- [GitHub OAuth Apps ドキュメント](https://docs.github.com/en/developers/apps/building-oauth-apps)
- [Valkey ドキュメント](https://valkey.io/docs/)

## 質問・サポート

問題が解決しない場合:

1. [プロジェクトのIssues](https://github.com/tacokumo/portal-api/issues) を検索
2. 新しい Issue を作成（ログとエラーメッセージを含める）
3. 開発チームに相談