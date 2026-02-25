package session

import (
	"time"
	"github.com/tacokumo/portal-api/pkg/auth"
)

// Session セッションデータ構造
type Session struct {
	UserID     string    `json:"user_id"`
	Login      string    `json:"login"`
	Teams      []string  `json:"teams"`
	CreatedAt  time.Time `json:"created_at"`
	LastAccess time.Time `json:"last_access"`
	ClientIP   string    `json:"client_ip"`
	UserAgent  string    `json:"user_agent"`
}

// Manager セッション管理インターフェース
type Manager interface {
	// CreateSession セッション作成
	CreateSession(sessionID string, user auth.User, clientIP, userAgent string) error

	// GetSession セッション取得
	GetSession(sessionID string) (*Session, error)

	// UpdateLastAccess 最終アクセス時間更新
	UpdateLastAccess(sessionID string) error

	// DeleteSession セッション削除
	DeleteSession(sessionID string) error

	// CacheUserTeams ユーザーのTeam情報をキャッシュ
	CacheUserTeams(userID string, teams []string, duration time.Duration) error

	// GetCachedUserTeams キャッシュされたTeam情報取得
	GetCachedUserTeams(userID string) ([]string, error)

	// Close 接続クローズ
	Close() error
}

// KeyNamespace Valkeyキー名前空間
const (
	KeyPrefixSession   = "portal:session:"
	KeyPrefixUserTeams = "portal:user_teams:"
	KeyPrefixRateLimit = "portal:rate_limit:"
	KeyPrefixCSRF      = "portal:csrf:"
)