#!/bin/bash

# JWT用RSAキーペア生成スクリプト
# Portal API認証システム用

set -e

# 出力ディレクトリ
OUTPUT_DIR="${1:-./secrets}"
KEY_SIZE="${2:-2048}"

echo "🔐 JWT用RSAキーペア生成開始..."
echo "出力ディレクトリ: ${OUTPUT_DIR}"
echo "キーサイズ: ${KEY_SIZE} bit"

# 出力ディレクトリ作成
mkdir -p "${OUTPUT_DIR}"

# 秘密鍵生成
echo "秘密鍵を生成中..."
openssl genrsa -out "${OUTPUT_DIR}/jwt-private-key.pem" "${KEY_SIZE}"

# 公開鍵生成
echo "公開鍵を生成中..."
openssl rsa -in "${OUTPUT_DIR}/jwt-private-key.pem" -pubout -out "${OUTPUT_DIR}/jwt-public-key.pem"

# 権限設定
chmod 600 "${OUTPUT_DIR}/jwt-private-key.pem"
chmod 644 "${OUTPUT_DIR}/jwt-public-key.pem"

echo "✅ RSAキーペア生成完了"
echo ""
echo "生成されたファイル:"
echo "  🔑 秘密鍵: ${OUTPUT_DIR}/jwt-private-key.pem (600)"
echo "  🔓 公開鍵: ${OUTPUT_DIR}/jwt-public-key.pem (644)"
echo ""

# 鍵の情報を表示
echo "🔍 キー情報:"
openssl rsa -in "${OUTPUT_DIR}/jwt-private-key.pem" -text -noout | head -n 3

echo ""
echo "📝 次のステップ:"
echo "1. 環境変数を設定:"
echo "   export JWT_PRIVATE_KEY_PATH=\"$(realpath ${OUTPUT_DIR}/jwt-private-key.pem)\""
echo "   export JWT_PUBLIC_KEY_PATH=\"$(realpath ${OUTPUT_DIR}/jwt-public-key.pem)\""
echo ""
echo "2. Kubernetesの場合:"
echo "   kubectl create secret generic portal-jwt-keys \\"
echo "     --from-file=private-key=${OUTPUT_DIR}/jwt-private-key.pem \\"
echo "     --from-file=public-key=${OUTPUT_DIR}/jwt-public-key.pem"
echo ""
echo "⚠️  注意: 秘密鍵は安全に管理し、リポジトリにコミットしないでください"