package rbac

import (
	"fmt"
	"strings"
)

// Role ロールを表現する構造体
type Role struct {
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Permissions []Permission `json:"permissions"`
}

// HasPermission ロールが特定の権限を持っているかチェック
func (r *Role) HasPermission(permission Permission) bool {
	for _, p := range r.Permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// HasAllPermissions ロールが指定されたすべての権限を持っているかチェック
func (r *Role) HasAllPermissions(permissions []Permission) bool {
	for _, permission := range permissions {
		if !r.HasPermission(permission) {
			return false
		}
	}
	return true
}

// HasAnyPermission ロールが指定された権限のいずれかを持っているかチェック
func (r *Role) HasAnyPermission(permissions []Permission) bool {
	for _, permission := range permissions {
		if r.HasPermission(permission) {
			return true
		}
	}
	return false
}

// DefaultTeamRoles GitHub Teamに基づくデフォルトロール定義
var DefaultTeamRoles = map[string]Role{
	"developers": {
		Name:        "Developer",
		Description: "Standard developer access to applications and secrets",
		Permissions: []Permission{
			PermissionApplicationsRead,
			PermissionApplicationsWrite,
			PermissionSecretsRead,
			PermissionMetricsRead,
		},
	},
	"senior-developers": {
		Name:        "Senior Developer",
		Description: "Enhanced developer access including secret management",
		Permissions: []Permission{
			PermissionApplicationsRead,
			PermissionApplicationsWrite,
			PermissionSecretsRead,
			PermissionSecretsWrite,
			PermissionMetricsRead,
		},
	},
	"platform-team": {
		Name:        "Platform Team",
		Description: "Platform team access including system administration",
		Permissions: []Permission{
			PermissionApplicationsRead,
			PermissionApplicationsWrite,
			PermissionSecretsRead,
			PermissionSecretsWrite,
			PermissionSystemAdmin,
			PermissionAuditRead,
			PermissionMetricsRead,
		},
	},
	"admins": {
		Name:        "Administrator",
		Description: "Full administrative access to all resources",
		Permissions: AllPermissions,
	},
	"github-actions": {
		Name:        "GitHub Actions",
		Description: "Automated deployment and CI/CD access",
		Permissions: []Permission{
			PermissionApplicationsRead,
			PermissionApplicationsWrite,
			PermissionSecretsRead,
		},
	},
}

// GetRoleFromTeam GitHub Teamからロールを取得
func GetRoleFromTeam(teamName string) (*Role, error) {
	// "organization/team" 形式から team 部分を抽出
	parts := strings.Split(teamName, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid team name format: %s", teamName)
	}

	team := parts[1]
	role, exists := DefaultTeamRoles[team]
	if !exists {
		return nil, fmt.Errorf("no role defined for team: %s", team)
	}

	// コピーを返して元のロール定義を保護
	roleCopy := Role{
		Name:        role.Name,
		Description: role.Description,
		Permissions: make([]Permission, len(role.Permissions)),
	}
	copy(roleCopy.Permissions, role.Permissions)

	return &roleCopy, nil
}

// CombineRoles 複数のロールの権限を結合
func CombineRoles(roles []Role) Role {
	permissionSet := make(map[Permission]bool)
	var roleNames []string

	// 全ロールの権限を収集
	for _, role := range roles {
		roleNames = append(roleNames, role.Name)
		for _, permission := range role.Permissions {
			permissionSet[permission] = true
		}
	}

	// 重複を除いた権限リストを作成
	var permissions []Permission
	for permission := range permissionSet {
		permissions = append(permissions, permission)
	}

	return Role{
		Name:        strings.Join(roleNames, ", "),
		Description: fmt.Sprintf("Combined role from: %s", strings.Join(roleNames, ", ")),
		Permissions: permissions,
	}
}

// GetRolesFromTeams 複数のTeamからロールを取得して結合
func GetRolesFromTeams(teamNames []string) (*Role, error) {
	if len(teamNames) == 0 {
		return nil, fmt.Errorf("no teams provided")
	}

	var roles []Role
	for _, teamName := range teamNames {
		role, err := GetRoleFromTeam(teamName)
		if err != nil {
			// Teamにロールが定義されていない場合はスキップ
			continue
		}
		roles = append(roles, *role)
	}

	if len(roles) == 0 {
		return nil, fmt.Errorf("no valid roles found for teams: %v", teamNames)
	}

	combinedRole := CombineRoles(roles)
	return &combinedRole, nil
}