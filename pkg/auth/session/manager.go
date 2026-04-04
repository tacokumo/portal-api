package session

import (
	"github.com/tacokumo/portal-api/pkg/config"
	"github.com/valkey-io/valkey-go"
)

// NewManager セッションマネージャーを作成
func NewManager(cfg config.ValkeyConfig) (Manager, error) {
	return NewValkeyManager(cfg)
}

// NewManagerWithClient creates a session manager using an existing Valkey client.
func NewManagerWithClient(client valkey.Client) Manager {
	return NewValkeyManagerWithClient(client)
}