package v1alpha1

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tacokumo/portal-api/pkg/auth/session"
	"github.com/tacokumo/portal-api/pkg/config"
)

func TestAuthService_GetCSRFToken(t *testing.T) {
	tests := []struct {
		name         string
		ctx          context.Context
		sessionMgr   session.Manager
		expectError  bool
		expectedCode int
	}{
		{
			name: "セッションIDがcontextにある場合、CSRFトークンが生成されること",
			ctx:  context.WithValue(context.Background(), "session_id", "test-session-123"),
			sessionMgr: &session.MockManager{
				SaveCSRFTokenFn: func(sessionID string, token string, ttl time.Duration) error {
					assert.Equal(t, "test-session-123", sessionID)
					assert.NotEmpty(t, token)
					assert.Equal(t, 30*time.Minute, ttl)
					return nil
				},
			},
			expectError: false,
		},
		{
			name:         "セッションIDがcontextにない場合、401エラーが返ること",
			ctx:          context.Background(),
			sessionMgr:   &session.MockManager{},
			expectError:  true,
			expectedCode: http.StatusUnauthorized,
		},
		{
			name: "空のセッションIDの場合、401エラーが返ること",
			ctx:  context.WithValue(context.Background(), "session_id", ""),
			sessionMgr: &session.MockManager{},
			expectError:  true,
			expectedCode: http.StatusUnauthorized,
		},
		{
			name: "SaveCSRFTokenが失敗した場合、500エラーが返ること",
			ctx:  context.WithValue(context.Background(), "session_id", "test-session-123"),
			sessionMgr: &session.MockManager{
				SaveCSRFTokenFn: func(sessionID string, token string, ttl time.Duration) error {
					return fmt.Errorf("valkey connection error")
				},
			},
			expectError:  true,
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := NewAuthService(
				&config.Config{},
				nil, // githubClient not needed
				nil, // jwtManager not needed
				tt.sessionMgr,
				"tacokumo",
			)

			result, err := svc.GetCSRFToken(tt.ctx)

			if tt.expectError {
				assert.Error(t, err)
				ewc, ok := err.(*ErrorWithCode)
				assert.True(t, ok)
				assert.Equal(t, tt.expectedCode, ewc.Code)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.NotEmpty(t, result.CsrfToken)
			}
		})
	}
}
