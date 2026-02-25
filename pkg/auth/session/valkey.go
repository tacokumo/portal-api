package session

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/valkey-io/valkey-go"
	"github.com/tacokumo/portal-api/pkg/auth"
	"github.com/tacokumo/portal-api/pkg/config"
)

// ValkeyManager Valkeyベースのセッション管理
type ValkeyManager struct {
	client valkey.Client
	ctx    context.Context
}

// NewValkeyManager 新しいValkeyセッションマネージャーを作成
func NewValkeyManager(cfg config.ValkeyConfig) (*ValkeyManager, error) {
	client, err := valkey.NewClient(valkey.ClientOption{
		InitAddress: []string{cfg.Address},
		Password:    cfg.Password,
		SelectDB:    cfg.DB,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Valkey client: %w", err)
	}

	// 接続テスト
	ctx := context.Background()
	if err := client.Do(ctx, client.B().Ping().Build()).Error(); err != nil {
		return nil, fmt.Errorf("failed to connect to Valkey: %w", err)
	}

	return &ValkeyManager{
		client: client,
		ctx:    ctx,
	}, nil
}

// CreateSession セッション作成
func (v *ValkeyManager) CreateSession(sessionID string, user auth.User, clientIP, userAgent string) error {
	session := Session{
		UserID:     user.ID,
		Login:      user.Login,
		Teams:      user.Teams,
		CreatedAt:  time.Now(),
		LastAccess: time.Now(),
		ClientIP:   clientIP,
		UserAgent:  userAgent,
	}

	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	key := KeyPrefixSession + sessionID
	// 8時間のTTL設定
	cmd := v.client.B().Setex().Key(key).Seconds(int64(8 * time.Hour / time.Second)).Value(string(data)).Build()

	if err := v.client.Do(v.ctx, cmd).Error(); err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	return nil
}

// GetSession セッション取得
func (v *ValkeyManager) GetSession(sessionID string) (*Session, error) {
	key := KeyPrefixSession + sessionID
	cmd := v.client.B().Get().Key(key).Build()

	result := v.client.Do(v.ctx, cmd)
	if err := result.Error(); err != nil {
		if valkey.IsValkeyNil(err) {
			return nil, fmt.Errorf("session not found")
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	data, err := result.ToString()
	if err != nil {
		return nil, fmt.Errorf("failed to convert session data: %w", err)
	}

	var session Session
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return &session, nil
}

// UpdateLastAccess 最終アクセス時間更新
func (v *ValkeyManager) UpdateLastAccess(sessionID string) error {
	session, err := v.GetSession(sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session for update: %w", err)
	}

	session.LastAccess = time.Now()

	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal updated session: %w", err)
	}

	key := KeyPrefixSession + sessionID
	// 既存のTTLを維持
	cmd := v.client.B().Set().Key(key).Value(string(data)).Keepttl().Build()

	if err := v.client.Do(v.ctx, cmd).Error(); err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	return nil
}

// DeleteSession セッション削除
func (v *ValkeyManager) DeleteSession(sessionID string) error {
	key := KeyPrefixSession + sessionID
	cmd := v.client.B().Del().Key(key).Build()

	if err := v.client.Do(v.ctx, cmd).Error(); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	return nil
}

// CacheUserTeams ユーザーのTeam情報をキャッシュ
func (v *ValkeyManager) CacheUserTeams(userID string, teams []string, duration time.Duration) error {
	data, err := json.Marshal(teams)
	if err != nil {
		return fmt.Errorf("failed to marshal teams: %w", err)
	}

	key := KeyPrefixUserTeams + userID
	cmd := v.client.B().Setex().Key(key).Seconds(int64(duration / time.Second)).Value(string(data)).Build()

	if err := v.client.Do(v.ctx, cmd).Error(); err != nil {
		return fmt.Errorf("failed to cache user teams: %w", err)
	}

	return nil
}

// GetCachedUserTeams キャッシュされたTeam情報取得
func (v *ValkeyManager) GetCachedUserTeams(userID string) ([]string, error) {
	key := KeyPrefixUserTeams + userID
	cmd := v.client.B().Get().Key(key).Build()

	result := v.client.Do(v.ctx, cmd)
	if err := result.Error(); err != nil {
		if valkey.IsValkeyNil(err) {
			return nil, fmt.Errorf("teams cache not found")
		}
		return nil, fmt.Errorf("failed to get cached teams: %w", err)
	}

	data, err := result.ToString()
	if err != nil {
		return nil, fmt.Errorf("failed to convert teams data: %w", err)
	}

	var teams []string
	if err := json.Unmarshal([]byte(data), &teams); err != nil {
		return nil, fmt.Errorf("failed to unmarshal teams: %w", err)
	}

	return teams, nil
}

// Close 接続クローズ
func (v *ValkeyManager) Close() error {
	v.client.Close()
	return nil
}