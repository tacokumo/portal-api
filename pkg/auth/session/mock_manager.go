package session

import (
	"time"

	"github.com/tacokumo/portal-api/pkg/auth"
)

// MockManager テスト用のセッションマネージャーモック
type MockManager struct {
	CreateSessionFn     func(sessionID string, user auth.User, clientIP, userAgent string) error
	GetSessionFn        func(sessionID string) (*Session, error)
	UpdateLastAccessFn  func(sessionID string) error
	DeleteSessionFn     func(sessionID string) error
	CacheUserTeamsFn    func(userID string, teams []string, duration time.Duration) error
	GetCachedUserTeamsFn func(userID string) ([]string, error)
	SaveCSRFTokenFn     func(sessionID string, token string, ttl time.Duration) error
	GetCSRFTokenFn      func(sessionID string) (string, error)
	CloseFn             func() error
}

func (m *MockManager) CreateSession(sessionID string, user auth.User, clientIP, userAgent string) error {
	if m.CreateSessionFn != nil {
		return m.CreateSessionFn(sessionID, user, clientIP, userAgent)
	}
	return nil
}

func (m *MockManager) GetSession(sessionID string) (*Session, error) {
	if m.GetSessionFn != nil {
		return m.GetSessionFn(sessionID)
	}
	return nil, nil
}

func (m *MockManager) UpdateLastAccess(sessionID string) error {
	if m.UpdateLastAccessFn != nil {
		return m.UpdateLastAccessFn(sessionID)
	}
	return nil
}

func (m *MockManager) DeleteSession(sessionID string) error {
	if m.DeleteSessionFn != nil {
		return m.DeleteSessionFn(sessionID)
	}
	return nil
}

func (m *MockManager) CacheUserTeams(userID string, teams []string, duration time.Duration) error {
	if m.CacheUserTeamsFn != nil {
		return m.CacheUserTeamsFn(userID, teams, duration)
	}
	return nil
}

func (m *MockManager) GetCachedUserTeams(userID string) ([]string, error) {
	if m.GetCachedUserTeamsFn != nil {
		return m.GetCachedUserTeamsFn(userID)
	}
	return nil, nil
}

func (m *MockManager) SaveCSRFToken(sessionID string, token string, ttl time.Duration) error {
	if m.SaveCSRFTokenFn != nil {
		return m.SaveCSRFTokenFn(sessionID, token, ttl)
	}
	return nil
}

func (m *MockManager) GetCSRFToken(sessionID string) (string, error) {
	if m.GetCSRFTokenFn != nil {
		return m.GetCSRFTokenFn(sessionID)
	}
	return "", nil
}

func (m *MockManager) Close() error {
	if m.CloseFn != nil {
		return m.CloseFn()
	}
	return nil
}
