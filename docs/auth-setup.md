# Portal API 認証システムセットアップ手順

このドキュメントでは、Portal APIの認証・認可システムのセットアップ手順を説明します。

## 概要

Portal APIは以下の3つの認証方式をサポートします：

1. **GitHub OAuth** - フロントエンドアプリケーション向け
2. **GitHub Personal Access Token (PAT)** - CLI・APIクライアント向け
3. **GitHub Installation Access Token** - GitHub Actions向け

## 前提条件

- Go 1.25.6以降
- Valkey (Redis互換) サーバー
- GitHub Organization (`tacokumo`) への管理者権限
- HTTPS対応のドメイン（本番環境）

## 1. RSAキーペア生成

JWTの署名に使用するRSAキーペアを生成します：

```bash
# 秘密鍵生成 (2048bit)
openssl genrsa -out jwt-private-key.pem 2048

# 公開鍵生成
openssl rsa -in jwt-private-key.pem -pubout -out jwt-public-key.pem

# 権限設定
chmod 600 jwt-private-key.pem
chmod 644 jwt-public-key.pem

# Kubernetes Secretとして作成（Kubernetes環境の場合）
kubectl create secret generic portal-jwt-keys \
  --from-file=private-key=jwt-private-key.pem \
  --from-file=public-key=jwt-public-key.pem
```

## 2. GitHub OAuth Appの作成

1. GitHub Organization設定にアクセス
2. "Developer settings" > "OAuth Apps" > "New OAuth App"
3. 以下の情報を入力：
   - **Application name**: TACOKUMO Portal
   - **Homepage URL**: https://portal.tacokumo.com
   - **Authorization callback URL**: https://portal.tacokumo.com/auth/github/callback
4. Client IDとClient Secretを記録

## 3. GitHub App の作成（GitHub Actions用）

1. GitHub Organization設定 > "Developer settings" > "GitHub Apps" > "New GitHub App"
2. 以下の情報を入力：
   - **GitHub App name**: tacokumo-portal-api
   - **Homepage URL**: https://portal.tacokumo.com
   - **Webhook URL**: https://portal.tacokumo.com/webhooks/github（無効化可能）
   - **Permissions**:
     - Repository permissions:
       - Contents: Read
       - Metadata: Read
     - Organization permissions:
       - Members: Read
       - Team membership: Read
3. Private keyをダウンロード
4. App IDを記録
5. Organization にインストール

## 4. GitHub Team 設定

以下のTeamを作成し、適切なメンバーを追加：

```yaml
teams:
  - name: developers           # 標準開発者権限
    permissions:
      - applications:read
      - applications:write
      - secrets:read
      - metrics:read

  - name: senior-developers    # 上級開発者権限
    permissions:
      - applications:read
      - applications:write
      - secrets:read
      - secrets:write
      - metrics:read

  - name: platform-team        # プラットフォームチーム
    permissions:
      - applications:read
      - applications:write
      - secrets:read
      - secrets:write
      - system:admin
      - audit:read
      - metrics:read

  - name: admins               # 管理者（全権限）
    permissions: all

  - name: github-actions       # 自動的に作成（GitHub Actions用）
    permissions:
      - applications:read
      - applications:write
      - secrets:read
```

## 5. Valkey (Redis) セットアップ

### ローカル開発環境

```bash
# Dockerを使用
docker run -d \
  --name portal-valkey \
  -p 6379:6379 \
  valkey/valkey:latest
```

### Kubernetes環境

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: portal-valkey
spec:
  replicas: 1
  selector:
    matchLabels:
      app: portal-valkey
  template:
    metadata:
      labels:
        app: portal-valkey
    spec:
      containers:
      - name: valkey
        image: valkey/valkey:latest
        ports:
        - containerPort: 6379
        env:
        - name: VALKEY_PASSWORD
          valueFrom:
            secretKeyRef:
              name: portal-valkey-secret
              key: password
---
apiVersion: v1
kind: Service
metadata:
  name: portal-valkey
spec:
  selector:
    app: portal-valkey
  ports:
  - port: 6379
    targetPort: 6379
```

## 6. 環境変数設定

`config/environment-variables.env.example` を参考に環境変数を設定：

```bash
# 開発環境の場合
cp config/environment-variables.env.example .env
# .env ファイルを編集して実際の値を設定

# 本番環境の場合（Kubernetes）
kubectl create secret generic portal-api-config \
  --from-literal=GITHUB_CLIENT_ID='your-client-id' \
  --from-literal=GITHUB_CLIENT_SECRET='your-client-secret' \
  --from-literal=GITHUB_APP_ID='your-app-id' \
  --from-literal=VALKEY_PASSWORD='your-valkey-password'
```

## 7. アプリケーション起動

```bash
# 開発環境
go run cmd/main.go

# 本番環境
go build -o portal-api cmd/main.go
./portal-api
```

## 8. 動作確認

### 1. ヘルスチェック
```bash
curl https://portal.tacokumo.com/health/liveness
```

### 2. OAuth認証フロー
1. ブラウザで `https://portal.tacokumo.com/auth/github/login` にアクセス
2. GitHub認証を完了
3. JWT Cookieが設定されることを確認

### 3. API アクセス（PAT認証）
```bash
# GitHub PATを使用
curl -H "Authorization: Bearer ghp_your-personal-access-token" \
  https://portal.tacokumo.com/v1alpha1/applications
```

### 4. GitHub Actions認証
```yaml
# .github/workflows/example.yml
name: Test Portal API
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
    - name: Call Portal API
      run: |
        curl -H "Authorization: Bearer ${{ secrets.GITHUB_TOKEN }}" \
          https://portal.tacokumo.com/v1alpha1/applications
```

## トラブルシューティング

### JWT関連エラー

- RSAキーペアのパスと権限を確認
- JWT_PRIVATE_KEY_PATH, JWT_PUBLIC_KEY_PATH環境変数を確認
- 秘密鍵がPKCS#1またはPKCS#8形式であることを確認

### GitHub認証エラー

- GitHub OAuth AppのCallback URLを確認
- Client ID/Secretが正しく設定されていることを確認
- Organizationのメンバー権限を確認

### Valkey接続エラー

- Valkeyサーバーが起動していることを確認
- VALKEY_ADDRESS環境変数のホスト名とポートを確認
- ネットワーク接続とファイアウォール設定を確認

### 権限エラー

- ユーザーが適切なTeamに所属していることを確認
- Team名が `organization/team` 形式であることを確認
- RBACロール定義が正しく設定されていることを確認

## セキュリティ考慮事項

1. **HTTPS必須**: 本番環境では必ずHTTPSを使用
2. **環境変数管理**: 機密情報は環境変数で管理し、リポジトリにコミットしない
3. **キー管理**: RSA秘密鍵は適切な権限設定で保護
4. **セッション管理**: Valkeyへのアクセスも認証付きで設定
5. **監査ログ**: 本番環境では構造化ログ出力を有効化
6. **レート制限**: 適切なレート制限設定を本番環境で有効化

## ログとメトリクス

認証システムは以下の情報をログ出力します：

- 認証成功/失敗イベント
- アクセス許可/拒否イベント
- セッション作成/削除イベント
- API呼び出し監査ログ

本番環境では構造化ログ（JSON形式）の使用を推奨します。