package session

import (
	"github.com/tacokumo/portal-api/pkg/config"
)

// NewManager セッションマネージャーを作成
func NewManager(cfg config.ValkeyConfig) (Manager, error) {
	return NewValkeyManager(cfg)
}