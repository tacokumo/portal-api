package auth

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/tacokumo/portal-api/pkg/config"
)

// JWTManager JWT生成・検証を管理
type JWTManager struct {
	privateKey               *rsa.PrivateKey
	publicKey                *rsa.PublicKey
	accessTokenDuration      time.Duration
	refreshTokenDuration     time.Duration
	issuer                   string
}

// NewJWTManager 新しいJWTマネージャーを作成
func NewJWTManager(cfg config.JWTConfig) (*JWTManager, error) {
	privateKey, err := loadRSAPrivateKey(cfg.PrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load private key: %w", err)
	}

	publicKey, err := loadRSAPublicKey(cfg.PublicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load public key: %w", err)
	}

	return &JWTManager{
		privateKey:               privateKey,
		publicKey:                publicKey,
		accessTokenDuration:      cfg.AccessTokenDuration,
		refreshTokenDuration:     cfg.RefreshTokenDuration,
		issuer:                   "portal-api",
	}, nil
}

// GenerateTokenPair アクセストークンとリフレッシュトークンを生成
func (j *JWTManager) GenerateTokenPair(user User, sessionID string) (*TokenPair, error) {
	now := time.Now()
	accessExpiry := now.Add(j.accessTokenDuration)
	refreshExpiry := now.Add(j.refreshTokenDuration)

	// アクセストークン生成
	accessClaims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    j.issuer,
			Subject:   fmt.Sprintf("github:%s", user.ID),
			ExpiresAt: jwt.NewNumericDate(accessExpiry),
			IssuedAt:  jwt.NewNumericDate(now),
		},
		UserInfo:  user,
		SessionID: sessionID,
	}

	accessToken, err := j.generateToken(accessClaims)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// リフレッシュトークン生成（ユーザー情報を含めない）
	refreshClaims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    j.issuer,
			Subject:   fmt.Sprintf("github:%s", user.ID),
			ExpiresAt: jwt.NewNumericDate(refreshExpiry),
			IssuedAt:  jwt.NewNumericDate(now),
		},
		SessionID: sessionID,
		UserInfo:  User{ID: user.ID}, // IDのみ
	}

	refreshToken, err := j.generateToken(refreshClaims)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    accessExpiry,
	}, nil
}

// ValidateToken トークンを検証してClaimsを返す
func (j *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return j.publicKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	// 有効期限チェック
	if claims.ExpiresAt != nil && time.Now().After(claims.ExpiresAt.Time) {
		return nil, fmt.Errorf("token expired")
	}

	return claims, nil
}

// GenerateSessionID セッションIDを生成
func (j *JWTManager) GenerateSessionID() string {
	return uuid.New().String()
}

// generateToken 内部用トークン生成
func (j *JWTManager) generateToken(claims Claims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(j.privateKey)
}

// loadRSAPrivateKey RSA秘密鍵をファイルから読み込み
func loadRSAPrivateKey(path string) (*rsa.PrivateKey, error) {
	keyData, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key file: %w", err)
	}

	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, fmt.Errorf("failed to parse PEM block")
	}

	if block.Type != "RSA PRIVATE KEY" && block.Type != "PRIVATE KEY" {
		return nil, fmt.Errorf("unsupported key type: %s", block.Type)
	}

	var privateKey *rsa.PrivateKey
	if block.Type == "RSA PRIVATE KEY" {
		privateKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	} else {
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse PKCS8 private key: %w", err)
		}
		var ok bool
		privateKey, ok = key.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("key is not RSA private key")
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return privateKey, nil
}

// loadRSAPublicKey RSA公開鍵をファイルから読み込み
func loadRSAPublicKey(path string) (*rsa.PublicKey, error) {
	keyData, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key file: %w", err)
	}

	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, fmt.Errorf("failed to parse PEM block")
	}

	if block.Type != "RSA PUBLIC KEY" && block.Type != "PUBLIC KEY" {
		return nil, fmt.Errorf("unsupported key type: %s", block.Type)
	}

	var publicKey *rsa.PublicKey
	if block.Type == "RSA PUBLIC KEY" {
		publicKey, err = x509.ParsePKCS1PublicKey(block.Bytes)
	} else {
		key, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse PKIX public key: %w", err)
		}
		var ok bool
		publicKey, ok = key.(*rsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("key is not RSA public key")
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	return publicKey, nil
}