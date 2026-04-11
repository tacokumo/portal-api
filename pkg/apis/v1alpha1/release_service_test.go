package v1alpha1

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tacokumo/portal-api/pkg/apis/v1alpha1/api"
	"github.com/tacokumo/portal-api/pkg/config"
	"github.com/tacokumo/portal-api/pkg/k8sclient"
	tacokumov1alpha1 "github.com/tacokumo/portal-controller-kubernetes/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const testNamespace = "portal-namespace"

func TestReleaseService_GetApplicationReleases(t *testing.T) {
	t.Run("K8sクラスタ未接続の場合503エラーとなること", func(t *testing.T) {
		t.Parallel()
		service := &ReleaseService{
			config: &config.Config{PortalName: testNamespace},
			client: nil,
		}
		_, err := service.GetApplicationReleases(t.Context(), api.GetApplicationReleasesParams{Name: "example-app"})
		assert.Error(t, err)
		ewc, ok := err.(*ErrorWithCode)
		assert.True(t, ok)
		assert.Equal(t, 503, ewc.Code)
	})

	t.Run("不正なアプリケーション名の場合400エラーとなること", func(t *testing.T) {
		t.Parallel()
		scheme, err := k8sclient.NewScheme()
		require.NoError(t, err)
		service := &ReleaseService{
			config: &config.Config{PortalName: testNamespace},
			client: fake.NewClientBuilder().WithScheme(scheme).Build(),
		}
		_, err = service.GetApplicationReleases(t.Context(), api.GetApplicationReleasesParams{Name: "Invalid-Name"})
		assert.Error(t, err)
		ewc, ok := err.(*ErrorWithCode)
		assert.True(t, ok)
		assert.Equal(t, 400, ewc.Code)
	})

	t.Run("存在しないApplicationの場合エラーとなること", func(t *testing.T) {
		t.Parallel()
		scheme, err := k8sclient.NewScheme()
		require.NoError(t, err)
		service := &ReleaseService{
			config: &config.Config{PortalName: testNamespace},
			client: fake.NewClientBuilder().WithScheme(scheme).Build(),
		}
		_, err = service.GetApplicationReleases(t.Context(), api.GetApplicationReleasesParams{Name: "non-existent-app"})
		assert.Error(t, err)
	})

	t.Run("Releaseが0件の場合空配列を返すこと", func(t *testing.T) {
		t.Parallel()
		scheme, err := k8sclient.NewScheme()
		require.NoError(t, err)

		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&tacokumov1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "example-app",
					Namespace: testNamespace,
				},
				Spec: tacokumov1alpha1.ApplicationSpec{
					ReleaseTemplate: tacokumov1alpha1.ReleaseSpec{
						Repo:            tacokumov1alpha1.RepositoryRef{URL: "https://github.com/tacokumo/example.git"},
						AppConfigPath:   "apps/example",
						AppConfigBranch: "main",
					},
				},
			},
		).Build()

		service := &ReleaseService{
			config: &config.Config{PortalName: testNamespace},
			client: c,
		}

		ret, err := service.GetApplicationReleases(t.Context(), api.GetApplicationReleasesParams{Name: "example-app"})
		require.NoError(t, err)
		releases, ok := ret.(*api.GetApplicationReleasesOKApplicationJSON)
		require.True(t, ok)
		assert.Empty(t, *releases)
	})

	t.Run("複数Releaseが存在する場合すべて返却すること", func(t *testing.T) {
		t.Parallel()
		scheme, err := k8sclient.NewScheme()
		require.NoError(t, err)

		commit1 := "abc123"
		commit2 := "def456"
		now := metav1.Now()

		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&tacokumov1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "example-app",
					Namespace: testNamespace,
				},
				Spec: tacokumov1alpha1.ApplicationSpec{
					ReleaseTemplate: tacokumov1alpha1.ReleaseSpec{
						Repo:            tacokumov1alpha1.RepositoryRef{URL: "https://github.com/tacokumo/example.git"},
						AppConfigPath:   "apps/example",
						AppConfigBranch: "main",
					},
				},
				Status: tacokumov1alpha1.ApplicationStatus{
					Releases: []corev1.ObjectReference{
						{Name: "example-app-release-1", Namespace: testNamespace},
						{Name: "example-app-release-2", Namespace: testNamespace},
					},
				},
			},
			&tacokumov1alpha1.Release{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "example-app-release-1",
					Namespace:         testNamespace,
					CreationTimestamp: now,
				},
				Spec: tacokumov1alpha1.ReleaseSpec{
					Repo:            tacokumov1alpha1.RepositoryRef{URL: "https://github.com/tacokumo/example.git"},
					AppConfigPath:   "apps/example",
					AppConfigBranch: "main",
					Commit:          &commit1,
				},
				Status: tacokumov1alpha1.ReleaseStatus{
					State: tacokumov1alpha1.ReleaseStateDeployed,
				},
			},
			&tacokumov1alpha1.Release{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "example-app-release-2",
					Namespace:         testNamespace,
					CreationTimestamp: now,
				},
				Spec: tacokumov1alpha1.ReleaseSpec{
					Repo:            tacokumov1alpha1.RepositoryRef{URL: "https://github.com/tacokumo/example.git"},
					AppConfigPath:   "apps/example",
					AppConfigBranch: "develop",
					Commit:          &commit2,
				},
				Status: tacokumov1alpha1.ReleaseStatus{
					State: tacokumov1alpha1.ReleaseStateDeploying,
				},
			},
		).WithStatusSubresource(&tacokumov1alpha1.Application{}, &tacokumov1alpha1.Release{}).Build()

		service := &ReleaseService{
			config: &config.Config{PortalName: testNamespace},
			client: c,
		}

		ret, err := service.GetApplicationReleases(t.Context(), api.GetApplicationReleasesParams{Name: "example-app"})
		require.NoError(t, err)
		releases, ok := ret.(*api.GetApplicationReleasesOKApplicationJSON)
		require.True(t, ok)
		assert.Len(t, *releases, 2)

		r1 := (*releases)[0]
		assert.Equal(t, "example-app-release-1", r1.Name)
		assert.Equal(t, "abc123", r1.Commit.Value)
		assert.Equal(t, "https://github.com/tacokumo/example.git", r1.RepositoryURL)
		assert.Equal(t, "apps/example", r1.AppconfigPath)
		assert.Equal(t, "main", r1.AppconfigBranch)

		r2 := (*releases)[1]
		assert.Equal(t, "example-app-release-2", r2.Name)
		assert.Equal(t, "def456", r2.Commit.Value)
		assert.Equal(t, "develop", r2.AppconfigBranch)
	})

	t.Run("削除済みReleaseの参照はスキップすること", func(t *testing.T) {
		t.Parallel()
		scheme, err := k8sclient.NewScheme()
		require.NoError(t, err)

		commit := "abc123"

		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&tacokumov1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "example-app",
					Namespace: testNamespace,
				},
				Spec: tacokumov1alpha1.ApplicationSpec{
					ReleaseTemplate: tacokumov1alpha1.ReleaseSpec{
						Repo:            tacokumov1alpha1.RepositoryRef{URL: "https://github.com/tacokumo/example.git"},
						AppConfigPath:   "apps/example",
						AppConfigBranch: "main",
					},
				},
				Status: tacokumov1alpha1.ApplicationStatus{
					Releases: []corev1.ObjectReference{
						{Name: "existing-release", Namespace: testNamespace},
						{Name: "deleted-release", Namespace: testNamespace},
					},
				},
			},
			&tacokumov1alpha1.Release{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "existing-release",
					Namespace: testNamespace,
				},
				Spec: tacokumov1alpha1.ReleaseSpec{
					Repo:            tacokumov1alpha1.RepositoryRef{URL: "https://github.com/tacokumo/example.git"},
					AppConfigPath:   "apps/example",
					AppConfigBranch: "main",
					Commit:          &commit,
				},
				Status: tacokumov1alpha1.ReleaseStatus{
					State: tacokumov1alpha1.ReleaseStateDeployed,
				},
			},
		).WithStatusSubresource(&tacokumov1alpha1.Application{}, &tacokumov1alpha1.Release{}).Build()

		service := &ReleaseService{
			config: &config.Config{PortalName: testNamespace},
			client: c,
		}

		ret, err := service.GetApplicationReleases(t.Context(), api.GetApplicationReleasesParams{Name: "example-app"})
		require.NoError(t, err)
		releases, ok := ret.(*api.GetApplicationReleasesOKApplicationJSON)
		require.True(t, ok)
		assert.Len(t, *releases, 1, "削除済みReleaseはスキップされ1件のみ返ること")
		assert.Equal(t, "existing-release", (*releases)[0].Name)
	})

	t.Run("Commitがnilの場合OptStringが未設定であること", func(t *testing.T) {
		t.Parallel()
		scheme, err := k8sclient.NewScheme()
		require.NoError(t, err)

		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&tacokumov1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "example-app",
					Namespace: testNamespace,
				},
				Spec: tacokumov1alpha1.ApplicationSpec{
					ReleaseTemplate: tacokumov1alpha1.ReleaseSpec{
						Repo:            tacokumov1alpha1.RepositoryRef{URL: "https://github.com/tacokumo/example.git"},
						AppConfigPath:   "apps/example",
						AppConfigBranch: "main",
					},
				},
				Status: tacokumov1alpha1.ApplicationStatus{
					Releases: []corev1.ObjectReference{
						{Name: "no-commit-release", Namespace: testNamespace},
					},
				},
			},
			&tacokumov1alpha1.Release{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "no-commit-release",
					Namespace: testNamespace,
				},
				Spec: tacokumov1alpha1.ReleaseSpec{
					Repo:            tacokumov1alpha1.RepositoryRef{URL: "https://github.com/tacokumo/example.git"},
					AppConfigPath:   "apps/example",
					AppConfigBranch: "main",
					Commit:          nil,
				},
				Status: tacokumov1alpha1.ReleaseStatus{
					State: tacokumov1alpha1.ReleaseStateDeployed,
				},
			},
		).WithStatusSubresource(&tacokumov1alpha1.Application{}, &tacokumov1alpha1.Release{}).Build()

		service := &ReleaseService{
			config: &config.Config{PortalName: testNamespace},
			client: c,
		}

		ret, err := service.GetApplicationReleases(t.Context(), api.GetApplicationReleasesParams{Name: "example-app"})
		require.NoError(t, err)
		releases, ok := ret.(*api.GetApplicationReleasesOKApplicationJSON)
		require.True(t, ok)
		assert.Len(t, *releases, 1)
		assert.False(t, (*releases)[0].Commit.Set, "CommitがnilのときOptStringはSetがfalseであること")
	})
}

func TestReleaseService_RollbackApplication(t *testing.T) {
	t.Run("K8sクラスタ未接続の場合503エラーとなること", func(t *testing.T) {
		t.Parallel()
		service := &ReleaseService{
			config: &config.Config{PortalName: testNamespace},
			client: nil,
		}
		_, err := service.RollbackApplication(t.Context(),
			&api.RollbackRequest{ReleaseName: "some-release"},
			api.RollbackApplicationParams{Name: "example-app"},
		)
		assert.Error(t, err)
		ewc, ok := err.(*ErrorWithCode)
		assert.True(t, ok)
		assert.Equal(t, 503, ewc.Code)
	})

	t.Run("不正なアプリケーション名の場合400エラーとなること", func(t *testing.T) {
		t.Parallel()
		scheme, err := k8sclient.NewScheme()
		require.NoError(t, err)
		service := &ReleaseService{
			config: &config.Config{PortalName: testNamespace},
			client: fake.NewClientBuilder().WithScheme(scheme).Build(),
		}
		_, err = service.RollbackApplication(t.Context(),
			&api.RollbackRequest{ReleaseName: "some-release"},
			api.RollbackApplicationParams{Name: "Invalid-Name"},
		)
		assert.Error(t, err)
		ewc, ok := err.(*ErrorWithCode)
		assert.True(t, ok)
		assert.Equal(t, 400, ewc.Code)
	})

	t.Run("不正なリリース名の場合400エラーとなること", func(t *testing.T) {
		t.Parallel()
		scheme, err := k8sclient.NewScheme()
		require.NoError(t, err)
		service := &ReleaseService{
			config: &config.Config{PortalName: testNamespace},
			client: fake.NewClientBuilder().WithScheme(scheme).Build(),
		}
		_, err = service.RollbackApplication(t.Context(),
			&api.RollbackRequest{ReleaseName: "Invalid-Release"},
			api.RollbackApplicationParams{Name: "example-app"},
		)
		assert.Error(t, err)
		ewc, ok := err.(*ErrorWithCode)
		assert.True(t, ok)
		assert.Equal(t, 400, ewc.Code)
	})

	t.Run("存在しないApplicationの場合エラーとなること", func(t *testing.T) {
		t.Parallel()
		scheme, err := k8sclient.NewScheme()
		require.NoError(t, err)
		service := &ReleaseService{
			config: &config.Config{PortalName: testNamespace},
			client: fake.NewClientBuilder().WithScheme(scheme).Build(),
		}
		_, err = service.RollbackApplication(t.Context(),
			&api.RollbackRequest{ReleaseName: "some-release"},
			api.RollbackApplicationParams{Name: "non-existent-app"},
		)
		assert.Error(t, err)
	})

	t.Run("Applicationに紐づかないReleaseの場合404エラーとなること", func(t *testing.T) {
		t.Parallel()
		scheme, err := k8sclient.NewScheme()
		require.NoError(t, err)

		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&tacokumov1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "example-app",
					Namespace: testNamespace,
				},
				Spec: tacokumov1alpha1.ApplicationSpec{
					ReleaseTemplate: tacokumov1alpha1.ReleaseSpec{
						Repo:            tacokumov1alpha1.RepositoryRef{URL: "https://github.com/tacokumo/example.git"},
						AppConfigPath:   "apps/example",
						AppConfigBranch: "main",
					},
				},
				Status: tacokumov1alpha1.ApplicationStatus{
					Releases: []corev1.ObjectReference{
						{Name: "release-1", Namespace: testNamespace},
					},
				},
			},
		).WithStatusSubresource(&tacokumov1alpha1.Application{}).Build()

		service := &ReleaseService{
			config: &config.Config{PortalName: testNamespace},
			client: c,
		}

		_, err = service.RollbackApplication(t.Context(),
			&api.RollbackRequest{ReleaseName: "non-existent-release"},
			api.RollbackApplicationParams{Name: "example-app"},
		)
		assert.Error(t, err)
		ewc, ok := err.(*ErrorWithCode)
		assert.True(t, ok)
		assert.Equal(t, 404, ewc.Code)
	})

	t.Run("正常にロールバックできること", func(t *testing.T) {
		t.Parallel()
		scheme, err := k8sclient.NewScheme()
		require.NoError(t, err)

		oldCommit := "old-commit-hash"
		envSecret := "example-app-secret"

		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&tacokumov1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "example-app",
					Namespace: testNamespace,
				},
				Spec: tacokumov1alpha1.ApplicationSpec{
					ReleaseTemplate: tacokumov1alpha1.ReleaseSpec{
						Repo:            tacokumov1alpha1.RepositoryRef{URL: "https://github.com/tacokumo/example.git"},
						AppConfigPath:   "apps/example",
						AppConfigBranch: "main",
						EnvSecretName:   &envSecret,
					},
				},
				Status: tacokumov1alpha1.ApplicationStatus{
					Releases: []corev1.ObjectReference{
						{Name: "old-release", Namespace: testNamespace},
					},
				},
			},
			&tacokumov1alpha1.Release{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "old-release",
					Namespace:         testNamespace,
					CreationTimestamp:  metav1.NewTime(time.Now().Add(-1 * time.Hour)),
				},
				Spec: tacokumov1alpha1.ReleaseSpec{
					Repo:            tacokumov1alpha1.RepositoryRef{URL: "https://github.com/tacokumo/example.git"},
					AppConfigPath:   "apps/example-v1",
					AppConfigBranch: "release/v1",
					Commit:          &oldCommit,
				},
				Status: tacokumov1alpha1.ReleaseStatus{
					State: tacokumov1alpha1.ReleaseStateDeployed,
				},
			},
		).WithStatusSubresource(&tacokumov1alpha1.Application{}, &tacokumov1alpha1.Release{}).Build()

		service := &ReleaseService{
			config: &config.Config{PortalName: testNamespace},
			client: c,
		}

		ret, err := service.RollbackApplication(t.Context(),
			&api.RollbackRequest{ReleaseName: "old-release"},
			api.RollbackApplicationParams{Name: "example-app"},
		)
		require.NoError(t, err)

		app, ok := ret.(*api.Application)
		require.True(t, ok)
		assert.Equal(t, "example-app", app.Name)
		assert.Equal(t, "apps/example-v1", app.AppconfigPath)
		assert.Equal(t, "release/v1", app.AppconfigBranch)
		assert.Equal(t, "https://github.com/tacokumo/example.git", app.RepositoryURL)
	})

	t.Run("ロールバック後にEnvSecretNameが保持されること", func(t *testing.T) {
		t.Parallel()
		scheme, err := k8sclient.NewScheme()
		require.NoError(t, err)

		oldCommit := "old-commit"
		envSecret := "my-env-secret"

		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&tacokumov1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "example-app",
					Namespace: testNamespace,
				},
				Spec: tacokumov1alpha1.ApplicationSpec{
					ReleaseTemplate: tacokumov1alpha1.ReleaseSpec{
						Repo:            tacokumov1alpha1.RepositoryRef{URL: "https://github.com/tacokumo/example.git"},
						AppConfigPath:   "apps/example",
						AppConfigBranch: "main",
						EnvSecretName:   &envSecret,
					},
				},
				Status: tacokumov1alpha1.ApplicationStatus{
					Releases: []corev1.ObjectReference{
						{Name: "target-release", Namespace: testNamespace},
					},
				},
			},
			&tacokumov1alpha1.Release{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "target-release",
					Namespace: testNamespace,
				},
				Spec: tacokumov1alpha1.ReleaseSpec{
					Repo:            tacokumov1alpha1.RepositoryRef{URL: "https://github.com/tacokumo/example.git"},
					AppConfigPath:   "apps/old",
					AppConfigBranch: "old-branch",
					Commit:          &oldCommit,
					EnvSecretName:   nil, // Releaseには設定なし
				},
				Status: tacokumov1alpha1.ReleaseStatus{
					State: tacokumov1alpha1.ReleaseStateDeployed,
				},
			},
		).WithStatusSubresource(&tacokumov1alpha1.Application{}, &tacokumov1alpha1.Release{}).Build()

		service := &ReleaseService{
			config: &config.Config{PortalName: testNamespace},
			client: c,
		}

		_, err = service.RollbackApplication(t.Context(),
			&api.RollbackRequest{ReleaseName: "target-release"},
			api.RollbackApplicationParams{Name: "example-app"},
		)
		require.NoError(t, err)

		// 更新後のApplicationを再取得してEnvSecretNameが保持されていることを確認
		var updatedApp tacokumov1alpha1.Application
		err = c.Get(t.Context(), client.ObjectKeyFromObject(&tacokumov1alpha1.Application{
			ObjectMeta: metav1.ObjectMeta{Name: "example-app", Namespace: testNamespace},
		}), &updatedApp)
		require.NoError(t, err)
		require.NotNil(t, updatedApp.Spec.ReleaseTemplate.EnvSecretName)
		assert.Equal(t, "my-env-secret", *updatedApp.Spec.ReleaseTemplate.EnvSecretName)
		assert.Equal(t, "apps/old", updatedApp.Spec.ReleaseTemplate.AppConfigPath)
		assert.Equal(t, "old-branch", updatedApp.Spec.ReleaseTemplate.AppConfigBranch)
		require.NotNil(t, updatedApp.Spec.ReleaseTemplate.Commit)
		assert.Equal(t, "old-commit", *updatedApp.Spec.ReleaseTemplate.Commit)
	})
}
