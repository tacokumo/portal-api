package middleware

import (
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/tacokumo/portal-api/pkg/rbac"
)

// RBACMiddleware ロールベースアクセス制御ミドルウェア
type RBACMiddleware struct {
	evaluator *rbac.Evaluator
}

// NewRBACMiddleware RBACミドルウェアを作成
func NewRBACMiddleware(strictMode bool) *RBACMiddleware {
	return &RBACMiddleware{
		evaluator: rbac.NewEvaluator(strictMode),
	}
}

// RequirePermission 特定の権限を要求するミドルウェア
func (m *RBACMiddleware) RequirePermission(permission rbac.Permission) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			// 認証コンテキスト取得
			authCtx, err := GetAuthContext(c)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "authentication required")
			}

			// 権限チェック
			if err := m.evaluator.RequirePermission(authCtx.User, permission); err != nil {
				return echo.NewHTTPError(http.StatusForbidden, err.Error())
			}

			return next(c)
		}
	}
}

// RequireAllPermissions 複数の権限をすべて要求するミドルウェア
func (m *RBACMiddleware) RequireAllPermissions(permissions []rbac.Permission) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			// 認証コンテキスト取得
			authCtx, err := GetAuthContext(c)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "authentication required")
			}

			// 権限チェック
			if err := m.evaluator.RequireAllPermissions(authCtx.User, permissions); err != nil {
				return echo.NewHTTPError(http.StatusForbidden, err.Error())
			}

			return next(c)
		}
	}
}

// RequireAnyPermission 複数の権限のいずれかを要求するミドルウェア
func (m *RBACMiddleware) RequireAnyPermission(permissions []rbac.Permission) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			// 認証コンテキスト取得
			authCtx, err := GetAuthContext(c)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "authentication required")
			}

			// 権限チェック
			if err := m.evaluator.RequireAnyPermission(authCtx.User, permissions); err != nil {
				return echo.NewHTTPError(http.StatusForbidden, err.Error())
			}

			return next(c)
		}
	}
}

// RequireSystemAdmin システム管理者権限を要求するミドルウェア
func (m *RBACMiddleware) RequireSystemAdmin() echo.MiddlewareFunc {
	return m.RequirePermission(rbac.PermissionSystemAdmin)
}

// RequireApplicationsRead アプリケーション読み取り権限を要求
func (m *RBACMiddleware) RequireApplicationsRead() echo.MiddlewareFunc {
	return m.RequirePermission(rbac.PermissionApplicationsRead)
}

// RequireApplicationsWrite アプリケーション書き込み権限を要求
func (m *RBACMiddleware) RequireApplicationsWrite() echo.MiddlewareFunc {
	return m.RequirePermission(rbac.PermissionApplicationsWrite)
}

// RequireSecretsRead シークレット読み取り権限を要求
func (m *RBACMiddleware) RequireSecretsRead() echo.MiddlewareFunc {
	return m.RequirePermission(rbac.PermissionSecretsRead)
}

// RequireSecretsWrite シークレット書き込み権限を要求
func (m *RBACMiddleware) RequireSecretsWrite() echo.MiddlewareFunc {
	return m.RequirePermission(rbac.PermissionSecretsWrite)
}

// GetUserPermissions ユーザーの権限情報を取得するヘルパー関数
func (m *RBACMiddleware) GetUserPermissions(c *echo.Context) ([]rbac.Permission, error) {
	authCtx, err := GetAuthContext(c)
	if err != nil {
		return nil, err
	}

	return m.evaluator.GetUserPermissions(authCtx.User)
}