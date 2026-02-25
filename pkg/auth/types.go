package auth

import (
	"time"
	"github.com/golang-jwt/jwt/v5"
)

// User GitHub認証されたユーザー情報
type User struct {
	ID    string `json:"id"`
	Login string `json:"login"`
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
	Teams []string `json:"teams"` // "org/team" format
}

// Claims JWT Claims構造
type Claims struct {
	jwt.RegisteredClaims
	UserInfo  User   `json:"user_info"`
	SessionID string `json:"session_id"`
}

// TokenPair アクセストークンとリフレッシュトークンのペア
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// AuthMethod 認証方式
type AuthMethod int

const (
	AuthMethodOAuth AuthMethod = iota
	AuthMethodPAT
	AuthMethodInstallation
)

// AuthContext リクエストコンテキストに設定される認証情報
type AuthContext struct {
	User     User       `json:"user"`
	Method   AuthMethod `json:"method"`
	ClientIP string     `json:"client_ip"`
}