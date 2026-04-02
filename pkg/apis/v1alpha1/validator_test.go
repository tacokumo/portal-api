package v1alpha1

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tacokumo/portal-api/pkg/apis/v1alpha1/api"
)

func TestValidateApplicationName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		isError bool
	}{
		{"有効な名前", "my-app", false},
		{"1文字の名前", "a", false},
		{"数字を含む名前", "app-123", false},
		{"最大長63文字", strings.Repeat("a", 63), false},
		{"空文字列", "", true},
		{"大文字を含む", "My-App", true},
		{"ハイフンで始まる", "-app", true},
		{"ハイフンで終わる", "app-", true},
		{"アンダースコアを含む", "app_name", true},
		{"64文字（超過）", strings.Repeat("a", 64), true},
		{"ドットを含む", "app.name", true},
		{"スラッシュを含む", "app/name", true},
		{"スペースを含む", "app name", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateApplicationName(tt.input)
			if tt.isError {
				assert.Error(t, err)
				ewc, ok := err.(*ErrorWithCode)
				assert.True(t, ok)
				assert.Equal(t, http.StatusBadRequest, ewc.Code)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateRepositoryURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		isError bool
	}{
		{"有効なHTTPS URL", "https://github.com/org/repo.git", false},
		{"ポート付きHTTPS", "https://github.com:443/org/repo.git", false},
		{"空文字列", "", true},
		{"HTTPスキーム", "http://github.com/org/repo.git", true},
		{"SSHスキーム", "ssh://git@github.com/org/repo.git", true},
		{"スキームなし", "github.com/org/repo.git", true},
		{"localhost", "https://localhost/repo", true},
		{"サブドメインlocalhost", "https://sub.localhost/repo", true},
		{"ループバックIPv4", "https://127.0.0.1/repo", true},
		{"プライベートIP 10.x", "https://10.0.0.1/repo", true},
		{"プライベートIP 172.16.x", "https://172.16.0.1/repo", true},
		{"プライベートIP 192.168.x", "https://192.168.1.1/repo", true},
		{"リンクローカルIP", "https://169.254.1.1/repo", true},
		{"IPv6ループバック", "https://[::1]/repo", true},
		{"ゼロアドレス", "https://0.0.0.1/repo", true},
		{"fileスキーム", "file:///etc/passwd", true},
		{"ホスト名なし", "https:///path", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateRepositoryURL(tt.input)
			if tt.isError {
				assert.Error(t, err)
				ewc, ok := err.(*ErrorWithCode)
				assert.True(t, ok)
				assert.Equal(t, http.StatusBadRequest, ewc.Code)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateAppconfigPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		isError bool
	}{
		{"有効なパス", "apps/my-app", false},
		{"単一ディレクトリ", "config", false},
		{"空文字列", "", true},
		{"スペースのみ", "   ", true},
		{"パストラバーサル", "apps/../etc/passwd", true},
		{"先頭のパストラバーサル", "../secret", true},
		{"連続ドット", "apps/..hidden", true},
		{"513文字", strings.Repeat("a", 513), true},
		{"ヌルバイト含む", "apps/\x00test", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateAppconfigPath(tt.input)
			if tt.isError {
				assert.Error(t, err)
				ewc, ok := err.(*ErrorWithCode)
				assert.True(t, ok)
				assert.Equal(t, http.StatusBadRequest, ewc.Code)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateAppconfigBranch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		isError bool
	}{
		{"main", "main", false},
		{"feature/branch", "feature/my-branch", false},
		{"v1.0.0", "v1.0.0", false},
		{"空文字列", "", true},
		{"連続ドット", "branch..name", true},
		{"チルダ", "branch~1", true},
		{"キャレット", "branch^2", true},
		{"コロン", "ref:name", true},
		{"スペース", "branch name", true},
		{"バックスラッシュ", "branch\\name", true},
		{"ドットで開始", ".branch", true},
		{"スラッシュで開始", "/branch", true},
		{"ドットで終了", "branch.", true},
		{"スラッシュで終了", "branch/", true},
		{".lockで終了", "branch.lock", true},
		{"アスタリスク", "branch*", true},
		{"疑問符", "branch?", true},
		{"ブラケット", "branch[0]", true},
		{"@{パターン", "branch@{0}", true},
		{"制御文字", "branch\x01name", true},
		{"254文字", strings.Repeat("a", 254), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateAppconfigBranch(tt.input)
			if tt.isError {
				assert.Error(t, err)
				ewc, ok := err.(*ErrorWithCode)
				assert.True(t, ok)
				assert.Equal(t, http.StatusBadRequest, ewc.Code)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateSecretKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		isError bool
	}{
		{"大文字英字キー", "DB_PASSWORD", false},
		{"ハイフン含む", "api-key", false},
		{"ドット含む", "config.json", false},
		{"空文字列", "", true},
		{"スペース含む", "key with spaces", true},
		{"スラッシュ含む", "key/slash", true},
		{"254文字", strings.Repeat("a", 254), true},
		{"253文字", strings.Repeat("a", 253), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateSecretKey(tt.input)
			if tt.isError {
				assert.Error(t, err)
				ewc, ok := err.(*ErrorWithCode)
				assert.True(t, ok)
				assert.Equal(t, http.StatusBadRequest, ewc.Code)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateSecretValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		isError bool
	}{
		{"通常の値", "some-value", false},
		{"空文字列", "", true},
		{"1MB超過", strings.Repeat("a", 1048577), true},
		{"ちょうど1MB", strings.Repeat("a", 1048576), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateSecretValue(tt.input)
			if tt.isError {
				assert.Error(t, err)
				ewc, ok := err.(*ErrorWithCode)
				assert.True(t, ok)
				assert.Equal(t, http.StatusBadRequest, ewc.Code)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateCreateApplicationRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		req     *api.CreateApplicationRequest
		isError bool
	}{
		{
			name: "全て有効",
			req: &api.CreateApplicationRequest{
				Name:            "my-app",
				RepositoryURL:   "https://github.com/org/repo.git",
				AppconfigPath:   "apps/my-app",
				AppconfigBranch: "main",
			},
			isError: false,
		},
		{
			name: "名前が無効",
			req: &api.CreateApplicationRequest{
				Name:            "INVALID",
				RepositoryURL:   "https://github.com/org/repo.git",
				AppconfigPath:   "apps/test",
				AppconfigBranch: "main",
			},
			isError: true,
		},
		{
			name: "URLが無効",
			req: &api.CreateApplicationRequest{
				Name:            "my-app",
				RepositoryURL:   "http://github.com/org/repo.git",
				AppconfigPath:   "apps/test",
				AppconfigBranch: "main",
			},
			isError: true,
		},
		{
			name: "パスが無効",
			req: &api.CreateApplicationRequest{
				Name:            "my-app",
				RepositoryURL:   "https://github.com/org/repo.git",
				AppconfigPath:   "../etc/passwd",
				AppconfigBranch: "main",
			},
			isError: true,
		},
		{
			name: "ブランチが無効",
			req: &api.CreateApplicationRequest{
				Name:            "my-app",
				RepositoryURL:   "https://github.com/org/repo.git",
				AppconfigPath:   "apps/test",
				AppconfigBranch: "branch..name",
			},
			isError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateCreateApplicationRequest(tt.req)
			if tt.isError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateCreateSecretRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		req     *api.CreateSecretRequest
		isError bool
	}{
		{
			name: "有効なリクエスト",
			req: &api.CreateSecretRequest{
				Items: []api.SecretItem{
					{Key: "DB_PASSWORD", Value: "secret123"},
				},
			},
			isError: false,
		},
		{
			name: "空のItems",
			req: &api.CreateSecretRequest{
				Items: []api.SecretItem{},
			},
			isError: true,
		},
		{
			name: "無効なキー",
			req: &api.CreateSecretRequest{
				Items: []api.SecretItem{
					{Key: "invalid/key", Value: "value"},
				},
			},
			isError: true,
		},
		{
			name: "空のキー",
			req: &api.CreateSecretRequest{
				Items: []api.SecretItem{
					{Key: "", Value: "value"},
				},
			},
			isError: true,
		},
		{
			name: "空の値",
			req: &api.CreateSecretRequest{
				Items: []api.SecretItem{
					{Key: "VALID_KEY", Value: ""},
				},
			},
			isError: true,
		},
		{
			name: "複数アイテムで2番目が無効",
			req: &api.CreateSecretRequest{
				Items: []api.SecretItem{
					{Key: "VALID_KEY", Value: "value"},
					{Key: "invalid/key", Value: "value"},
				},
			},
			isError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateCreateSecretRequest(tt.req)
			if tt.isError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
