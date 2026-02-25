package rbac

// Permission 権限を表現する型
type Permission string

// システム権限定義
const (
	// Application管理権限
	PermissionApplicationsRead  Permission = "applications:read"
	PermissionApplicationsWrite Permission = "applications:write"

	// Secret管理権限
	PermissionSecretsRead  Permission = "secrets:read"
	PermissionSecretsWrite Permission = "secrets:write"

	// システム管理権限
	PermissionSystemAdmin Permission = "system:admin"

	// 監査ログ権限
	PermissionAuditRead Permission = "audit:read"

	// メトリクス権限
	PermissionMetricsRead Permission = "metrics:read"
)

// AllPermissions 全権限のリスト
var AllPermissions = []Permission{
	PermissionApplicationsRead,
	PermissionApplicationsWrite,
	PermissionSecretsRead,
	PermissionSecretsWrite,
	PermissionSystemAdmin,
	PermissionAuditRead,
	PermissionMetricsRead,
}

// IsValidPermission 権限が有効かどうかを確認
func IsValidPermission(permission Permission) bool {
	for _, p := range AllPermissions {
		if p == permission {
			return true
		}
	}
	return false
}

// PermissionGroup 関連する権限をグループ化
type PermissionGroup struct {
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Permissions []Permission `json:"permissions"`
}

// PredefinedPermissionGroups 事前定義された権限グループ
var PredefinedPermissionGroups = map[string]PermissionGroup{
	"applications": {
		Name:        "Applications Management",
		Description: "Application resources read and write access",
		Permissions: []Permission{
			PermissionApplicationsRead,
			PermissionApplicationsWrite,
		},
	},
	"secrets": {
		Name:        "Secrets Management",
		Description: "Secret resources read and write access",
		Permissions: []Permission{
			PermissionSecretsRead,
			PermissionSecretsWrite,
		},
	},
	"readonly": {
		Name:        "Read-only Access",
		Description: "Read-only access to all resources",
		Permissions: []Permission{
			PermissionApplicationsRead,
			PermissionSecretsRead,
			PermissionAuditRead,
			PermissionMetricsRead,
		},
	},
	"admin": {
		Name:        "System Administrator",
		Description: "Full administrative access",
		Permissions: AllPermissions,
	},
}