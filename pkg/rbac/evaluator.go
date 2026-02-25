package rbac

import (
	"fmt"

	"github.com/tacokumo/portal-api/pkg/auth"
)

// Evaluator 権限評価器
type Evaluator struct {
	strictMode bool // 厳密モード: 未定義のTeamに対してエラーを返すか
}

// NewEvaluator 権限評価器を作成
func NewEvaluator(strictMode bool) *Evaluator {
	return &Evaluator{
		strictMode: strictMode,
	}
}

// HasPermission ユーザーが指定された権限を持っているかチェック
func (e *Evaluator) HasPermission(user auth.User, permission Permission) (bool, error) {
	role, err := e.GetUserRole(user)
	if err != nil {
		if e.strictMode {
			return false, fmt.Errorf("failed to get user role: %w", err)
		}
		// 非厳密モードでは権限なしとして処理
		return false, nil
	}

	return role.HasPermission(permission), nil
}

// HasAllPermissions ユーザーが指定されたすべての権限を持っているかチェック
func (e *Evaluator) HasAllPermissions(user auth.User, permissions []Permission) (bool, error) {
	role, err := e.GetUserRole(user)
	if err != nil {
		if e.strictMode {
			return false, fmt.Errorf("failed to get user role: %w", err)
		}
		return false, nil
	}

	return role.HasAllPermissions(permissions), nil
}

// HasAnyPermission ユーザーが指定された権限のいずれかを持っているかチェック
func (e *Evaluator) HasAnyPermission(user auth.User, permissions []Permission) (bool, error) {
	role, err := e.GetUserRole(user)
	if err != nil {
		if e.strictMode {
			return false, fmt.Errorf("failed to get user role: %w", err)
		}
		return false, nil
	}

	return role.HasAnyPermission(permissions), nil
}

// GetUserRole ユーザーのTeam情報からロールを取得
func (e *Evaluator) GetUserRole(user auth.User) (*Role, error) {
	return GetRolesFromTeams(user.Teams)
}

// RequirePermission 権限チェックを行い、権限がない場合はエラーを返す
func (e *Evaluator) RequirePermission(user auth.User, permission Permission) error {
	hasPermission, err := e.HasPermission(user, permission)
	if err != nil {
		return fmt.Errorf("permission check failed: %w", err)
	}

	if !hasPermission {
		return fmt.Errorf("user %s does not have permission %s", user.Login, permission)
	}

	return nil
}

// RequireAllPermissions 全権限チェックを行い、権限がない場合はエラーを返す
func (e *Evaluator) RequireAllPermissions(user auth.User, permissions []Permission) error {
	hasAllPermissions, err := e.HasAllPermissions(user, permissions)
	if err != nil {
		return fmt.Errorf("permissions check failed: %w", err)
	}

	if !hasAllPermissions {
		return fmt.Errorf("user %s does not have all required permissions %v", user.Login, permissions)
	}

	return nil
}

// RequireAnyPermission いずれかの権限チェックを行い、権限がない場合はエラーを返す
func (e *Evaluator) RequireAnyPermission(user auth.User, permissions []Permission) error {
	hasAnyPermission, err := e.HasAnyPermission(user, permissions)
	if err != nil {
		return fmt.Errorf("permissions check failed: %w", err)
	}

	if !hasAnyPermission {
		return fmt.Errorf("user %s does not have any of the required permissions %v", user.Login, permissions)
	}

	return nil
}

// GetUserPermissions ユーザーが持つ全権限のリストを取得
func (e *Evaluator) GetUserPermissions(user auth.User) ([]Permission, error) {
	role, err := e.GetUserRole(user)
	if err != nil {
		if e.strictMode {
			return nil, fmt.Errorf("failed to get user role: %w", err)
		}
		return []Permission{}, nil
	}

	return role.Permissions, nil
}

// IsSystemAdmin ユーザーがシステム管理者権限を持っているかチェック
func (e *Evaluator) IsSystemAdmin(user auth.User) (bool, error) {
	return e.HasPermission(user, PermissionSystemAdmin)
}