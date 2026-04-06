package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tacokumo/portal-api/pkg/apis/v1alpha1/api"
	"github.com/tacokumo/portal-api/pkg/config"
	"github.com/tacokumo/portal-api/pkg/k8sclient"
	tacokumov1alpha1 "github.com/tacokumo/portal-controller-kubernetes/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestApplicationService_GetApplication(t *testing.T) {
	tests := []struct {
		name     string
		config   *config.Config
		clientFn func() client.Client
		params   api.GetApplicationParams
		isError  bool
	}{
		{
			name: "存在するApplicationを取得できること",
			config: &config.Config{
				PortalName: "portal-namespace",
			},
			params: api.GetApplicationParams{
				Name: "example-app",
			},
			clientFn: func() client.Client {
				scheme, err := k8sclient.NewScheme()

				assert.NoError(t, err)
				c := fake.NewClientBuilder().WithScheme(scheme).Build()
				err = c.Create(t.Context(), &tacokumov1alpha1.Application{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "example-app",
						Namespace: "portal-namespace",
					},
					Spec: tacokumov1alpha1.ApplicationSpec{
						ReleaseTemplate: tacokumov1alpha1.ReleaseSpec{
							AppConfigPath:   "apps/example-app",
							AppConfigBranch: "main",
							Repo: tacokumov1alpha1.RepositoryRef{
								URL: "https://github.com/tacokumo/tacokumo-bot.git",
							},
						},
					},
				})
				assert.NoError(t, err)
				return c
			},
		},
		{
			name: "存在しないApplicationを取得しようとした場合、エラーとなること",
			config: &config.Config{
				PortalName: "portal-namespace",
			},
			params: api.GetApplicationParams{
				Name: "non-existent-app",
			},
			clientFn: func() client.Client {
				scheme, err := k8sclient.NewScheme()
				assert.NoError(t, err)
				return fake.NewClientBuilder().WithScheme(scheme).Build()
			},
			isError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service := &ApplicationService{
				config: tt.config,
				client: tt.clientFn(),
			}
			ret, err := service.GetApplication(t.Context(), tt.params)
			if tt.isError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, ret)
		})
	}
}

func TestApplicationService_GetApplications(t *testing.T) {
	tests := []struct {
		name     string
		config   *config.Config
		clientFn func() client.Client
		expected int
	}{
		{
			name: "空の一覧を取得するケース",
			config: &config.Config{
				PortalName: "portal-namespace",
			},
			clientFn: func() client.Client {
				scheme, err := k8sclient.NewScheme()
				assert.NoError(t, err)
				return fake.NewClientBuilder().WithScheme(scheme).Build()
			},
			expected: 0,
		},
		{
			name: "複数のApplicationが存在するケース",
			config: &config.Config{
				PortalName: "portal-namespace",
			},
			clientFn: func() client.Client {
				scheme, err := k8sclient.NewScheme()
				assert.NoError(t, err)
				c := fake.NewClientBuilder().WithScheme(scheme).Build()

				apps := []tacokumov1alpha1.Application{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "app-1",
							Namespace: "portal-namespace",
						},
						Spec: tacokumov1alpha1.ApplicationSpec{
							ReleaseTemplate: tacokumov1alpha1.ReleaseSpec{
								AppConfigPath:   "apps/app-1",
								AppConfigBranch: "main",
								Repo: tacokumov1alpha1.RepositoryRef{
									URL: "https://github.com/tacokumo/app-1.git",
								},
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "app-2",
							Namespace: "portal-namespace",
						},
						Spec: tacokumov1alpha1.ApplicationSpec{
							ReleaseTemplate: tacokumov1alpha1.ReleaseSpec{
								AppConfigPath:   "apps/app-2",
								AppConfigBranch: "develop",
								Repo: tacokumov1alpha1.RepositoryRef{
									URL: "https://github.com/tacokumo/app-2.git",
								},
							},
						},
					},
				}
				for _, app := range apps {
					err := c.Create(t.Context(), &app)
					assert.NoError(t, err)
				}
				return c
			},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service := &ApplicationService{
				config: tt.config,
				client: tt.clientFn(),
			}
			ret, err := service.GetApplications(t.Context())
			assert.NoError(t, err)
			assert.Len(t, ret, tt.expected)
		})
	}
}

func TestApplicationService_CreateApplication(t *testing.T) {
	tests := []struct {
		name     string
		config   *config.Config
		clientFn func() client.Client
		req      *api.CreateApplicationRequest
		isError  bool
	}{
		{
			name: "正常に作成できるケース",
			config: &config.Config{
				PortalName: "portal-namespace",
			},
			clientFn: func() client.Client {
				scheme, err := k8sclient.NewScheme()
				assert.NoError(t, err)
				return fake.NewClientBuilder().WithScheme(scheme).Build()
			},
			req: &api.CreateApplicationRequest{
				Name:            "new-app",
				AppconfigPath:   "apps/new-app",
				RepositoryURL:   "https://github.com/tacokumo/new-app.git",
				AppconfigBranch: "main",
			},
			isError: false,
		},
		{
			name: "名前が空の場合バリデーションエラーとなること",
			config: &config.Config{
				PortalName: "portal-namespace",
			},
			clientFn: func() client.Client {
				scheme, err := k8sclient.NewScheme()
				assert.NoError(t, err)
				return fake.NewClientBuilder().WithScheme(scheme).Build()
			},
			req: &api.CreateApplicationRequest{
				Name:            "",
				AppconfigPath:   "apps/test",
				RepositoryURL:   "https://github.com/tacokumo/test.git",
				AppconfigBranch: "main",
			},
			isError: true,
		},
		{
			name: "リポジトリURLがhttpの場合バリデーションエラーとなること",
			config: &config.Config{
				PortalName: "portal-namespace",
			},
			clientFn: func() client.Client {
				scheme, err := k8sclient.NewScheme()
				assert.NoError(t, err)
				return fake.NewClientBuilder().WithScheme(scheme).Build()
			},
			req: &api.CreateApplicationRequest{
				Name:            "valid-app",
				AppconfigPath:   "apps/test",
				RepositoryURL:   "http://github.com/tacokumo/test.git",
				AppconfigBranch: "main",
			},
			isError: true,
		},
		{
			name: "リポジトリURLがプライベートIPの場合バリデーションエラーとなること",
			config: &config.Config{
				PortalName: "portal-namespace",
			},
			clientFn: func() client.Client {
				scheme, err := k8sclient.NewScheme()
				assert.NoError(t, err)
				return fake.NewClientBuilder().WithScheme(scheme).Build()
			},
			req: &api.CreateApplicationRequest{
				Name:            "valid-app",
				AppconfigPath:   "apps/test",
				RepositoryURL:   "https://127.0.0.1/repo",
				AppconfigBranch: "main",
			},
			isError: true,
		},
		{
			name: "appconfigPathにパストラバーサルが含まれる場合バリデーションエラーとなること",
			config: &config.Config{
				PortalName: "portal-namespace",
			},
			clientFn: func() client.Client {
				scheme, err := k8sclient.NewScheme()
				assert.NoError(t, err)
				return fake.NewClientBuilder().WithScheme(scheme).Build()
			},
			req: &api.CreateApplicationRequest{
				Name:            "valid-app",
				AppconfigPath:   "../etc/passwd",
				RepositoryURL:   "https://github.com/tacokumo/test.git",
				AppconfigBranch: "main",
			},
			isError: true,
		},
		{
			name: "既に同名のApplicationが存在する場合のエラーケース",
			config: &config.Config{
				PortalName: "portal-namespace",
			},
			clientFn: func() client.Client {
				scheme, err := k8sclient.NewScheme()
				assert.NoError(t, err)
				c := fake.NewClientBuilder().WithScheme(scheme).Build()
				err = c.Create(t.Context(), &tacokumov1alpha1.Application{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "existing-app",
						Namespace: "portal-namespace",
					},
					Spec: tacokumov1alpha1.ApplicationSpec{
						ReleaseTemplate: tacokumov1alpha1.ReleaseSpec{
							AppConfigPath:   "apps/existing-app",
							AppConfigBranch: "main",
							Repo: tacokumov1alpha1.RepositoryRef{
								URL: "https://github.com/tacokumo/existing-app.git",
							},
						},
					},
				})
				assert.NoError(t, err)
				return c
			},
			req: &api.CreateApplicationRequest{
				Name:            "existing-app",
				AppconfigPath:   "apps/existing-app",
				RepositoryURL:   "https://github.com/tacokumo/existing-app.git",
				AppconfigBranch: "main",
			},
			isError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service := &ApplicationService{
				config: tt.config,
				client: tt.clientFn(),
			}
			ret, err := service.CreateApplication(t.Context(), tt.req)
			if tt.isError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, ret)
			assert.Equal(t, tt.req.Name, ret.Name)
			assert.Equal(t, tt.req.AppconfigPath, ret.AppconfigPath)
			assert.Equal(t, tt.req.RepositoryURL, ret.RepositoryURL)
			assert.Equal(t, tt.req.AppconfigBranch, ret.AppconfigBranch)
		})
	}
}

func TestApplicationService_UpdateApplication(t *testing.T) {
	tests := []struct {
		name     string
		config   *config.Config
		clientFn func() client.Client
		req      *api.UpdateApplicationRequest
		params   api.UpdateApplicationParams
		isError  bool
	}{
		{
			name: "正常に更新できるケース",
			config: &config.Config{
				PortalName: "portal-namespace",
			},
			clientFn: func() client.Client {
				scheme, err := k8sclient.NewScheme()
				assert.NoError(t, err)
				c := fake.NewClientBuilder().WithScheme(scheme).Build()
				err = c.Create(t.Context(), &tacokumov1alpha1.Application{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "example-app",
						Namespace: "portal-namespace",
					},
					Spec: tacokumov1alpha1.ApplicationSpec{
						ReleaseTemplate: tacokumov1alpha1.ReleaseSpec{
							AppConfigPath:   "apps/example-app",
							AppConfigBranch: "main",
							Repo: tacokumov1alpha1.RepositoryRef{
								URL: "https://github.com/tacokumo/tacokumo-bot.git",
							},
						},
					},
				})
				assert.NoError(t, err)
				return c
			},
			req: &api.UpdateApplicationRequest{
				RepositoryURL:   "https://github.com/tacokumo/updated-app.git",
				AppconfigPath:   "apps/updated-path",
				AppconfigBranch: "develop",
			},
			params: api.UpdateApplicationParams{
				Name: "example-app",
			},
			isError: false,
		},
		{
			name: "存在しないApplicationを更新しようとした場合、エラーとなること",
			config: &config.Config{
				PortalName: "portal-namespace",
			},
			clientFn: func() client.Client {
				scheme, err := k8sclient.NewScheme()
				assert.NoError(t, err)
				return fake.NewClientBuilder().WithScheme(scheme).Build()
			},
			req: &api.UpdateApplicationRequest{
				RepositoryURL:   "https://github.com/tacokumo/test.git",
				AppconfigPath:   "apps/test",
				AppconfigBranch: "main",
			},
			params: api.UpdateApplicationParams{
				Name: "non-existent-app",
			},
			isError: true,
		},
		{
			name: "リポジトリURLがhttpの場合バリデーションエラーとなること",
			config: &config.Config{
				PortalName: "portal-namespace",
			},
			clientFn: func() client.Client {
				scheme, err := k8sclient.NewScheme()
				assert.NoError(t, err)
				return fake.NewClientBuilder().WithScheme(scheme).Build()
			},
			req: &api.UpdateApplicationRequest{
				RepositoryURL:   "http://github.com/tacokumo/test.git",
				AppconfigPath:   "apps/test",
				AppconfigBranch: "main",
			},
			params: api.UpdateApplicationParams{
				Name: "example-app",
			},
			isError: true,
		},
		{
			name: "リポジトリURLがプライベートIPの場合バリデーションエラーとなること",
			config: &config.Config{
				PortalName: "portal-namespace",
			},
			clientFn: func() client.Client {
				scheme, err := k8sclient.NewScheme()
				assert.NoError(t, err)
				return fake.NewClientBuilder().WithScheme(scheme).Build()
			},
			req: &api.UpdateApplicationRequest{
				RepositoryURL:   "https://127.0.0.1/repo",
				AppconfigPath:   "apps/test",
				AppconfigBranch: "main",
			},
			params: api.UpdateApplicationParams{
				Name: "example-app",
			},
			isError: true,
		},
		{
			name: "appconfigPathにパストラバーサルが含まれる場合バリデーションエラーとなること",
			config: &config.Config{
				PortalName: "portal-namespace",
			},
			clientFn: func() client.Client {
				scheme, err := k8sclient.NewScheme()
				assert.NoError(t, err)
				return fake.NewClientBuilder().WithScheme(scheme).Build()
			},
			req: &api.UpdateApplicationRequest{
				RepositoryURL:   "https://github.com/tacokumo/test.git",
				AppconfigPath:   "../etc/passwd",
				AppconfigBranch: "main",
			},
			params: api.UpdateApplicationParams{
				Name: "example-app",
			},
			isError: true,
		},
		{
			name: "不正なブランチ名の場合バリデーションエラーとなること",
			config: &config.Config{
				PortalName: "portal-namespace",
			},
			clientFn: func() client.Client {
				scheme, err := k8sclient.NewScheme()
				assert.NoError(t, err)
				return fake.NewClientBuilder().WithScheme(scheme).Build()
			},
			req: &api.UpdateApplicationRequest{
				RepositoryURL:   "https://github.com/tacokumo/test.git",
				AppconfigPath:   "apps/test",
				AppconfigBranch: "branch..name",
			},
			params: api.UpdateApplicationParams{
				Name: "example-app",
			},
			isError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := tt.clientFn()
			service := &ApplicationService{
				config: tt.config,
				client: c,
			}
			ret, err := service.UpdateApplication(t.Context(), tt.req, tt.params)
			if tt.isError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, ret)

			app, ok := ret.(*api.Application)
			assert.True(t, ok)
			assert.Equal(t, tt.params.Name, app.Name)
			assert.Equal(t, tt.req.RepositoryURL, app.RepositoryURL)
			assert.Equal(t, tt.req.AppconfigPath, app.AppconfigPath)
			assert.Equal(t, tt.req.AppconfigBranch, app.AppconfigBranch)

			// k8sリソースが実際に更新されていることを確認
			k8sApp := tacokumov1alpha1.Application{}
			err = c.Get(t.Context(), types.NamespacedName{
				Namespace: tt.config.PortalName,
				Name:      tt.params.Name,
			}, &k8sApp)
			assert.NoError(t, err)
			assert.Equal(t, tt.req.RepositoryURL, k8sApp.Spec.ReleaseTemplate.Repo.URL)
			assert.Equal(t, tt.req.AppconfigPath, k8sApp.Spec.ReleaseTemplate.AppConfigPath)
			assert.Equal(t, tt.req.AppconfigBranch, k8sApp.Spec.ReleaseTemplate.AppConfigBranch)
		})
	}
}

func TestApplicationService_DeleteApplication(t *testing.T) {
	tests := []struct {
		name     string
		config   *config.Config
		clientFn func() client.Client
		params   api.DeleteApplicationParams
		isError  bool
	}{
		{
			name: "正常に削除できるケース（Application+Secret存在）",
			config: &config.Config{
				PortalName: "portal-namespace",
			},
			clientFn: func() client.Client {
				scheme, err := k8sclient.NewScheme()
				assert.NoError(t, err)
				c := fake.NewClientBuilder().WithScheme(scheme).Build()
				err = c.Create(t.Context(), &tacokumov1alpha1.Application{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "example-app",
						Namespace: "portal-namespace",
					},
					Spec: tacokumov1alpha1.ApplicationSpec{
						ReleaseTemplate: tacokumov1alpha1.ReleaseSpec{
							AppConfigPath:   "apps/example-app",
							AppConfigBranch: "main",
							Repo: tacokumov1alpha1.RepositoryRef{
								URL: "https://github.com/tacokumo/tacokumo-bot.git",
							},
						},
					},
				})
				assert.NoError(t, err)
				err = c.Create(t.Context(), &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "example-app-secret",
						Namespace: "portal-namespace",
					},
					Data: map[string][]byte{
						"DB_PASSWORD": []byte("secret123"),
					},
				})
				assert.NoError(t, err)
				return c
			},
			params: api.DeleteApplicationParams{
				Name: "example-app",
			},
			isError: false,
		},
		{
			name: "正常に削除できるケース（Secretなし）",
			config: &config.Config{
				PortalName: "portal-namespace",
			},
			clientFn: func() client.Client {
				scheme, err := k8sclient.NewScheme()
				assert.NoError(t, err)
				c := fake.NewClientBuilder().WithScheme(scheme).Build()
				err = c.Create(t.Context(), &tacokumov1alpha1.Application{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "no-secret-app",
						Namespace: "portal-namespace",
					},
					Spec: tacokumov1alpha1.ApplicationSpec{
						ReleaseTemplate: tacokumov1alpha1.ReleaseSpec{
							AppConfigPath:   "apps/no-secret-app",
							AppConfigBranch: "main",
							Repo: tacokumov1alpha1.RepositoryRef{
								URL: "https://github.com/tacokumo/no-secret-app.git",
							},
						},
					},
				})
				assert.NoError(t, err)
				return c
			},
			params: api.DeleteApplicationParams{
				Name: "no-secret-app",
			},
			isError: false,
		},
		{
			name: "Application不存在でエラー",
			config: &config.Config{
				PortalName: "portal-namespace",
			},
			clientFn: func() client.Client {
				scheme, err := k8sclient.NewScheme()
				assert.NoError(t, err)
				return fake.NewClientBuilder().WithScheme(scheme).Build()
			},
			params: api.DeleteApplicationParams{
				Name: "non-existent-app",
			},
			isError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := tt.clientFn()
			service := &ApplicationService{
				config: tt.config,
				client: c,
			}
			ret, err := service.DeleteApplication(t.Context(), tt.params)
			if tt.isError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, ret)

			// Application が削除されていることを確認
			app := tacokumov1alpha1.Application{}
			err = c.Get(t.Context(), types.NamespacedName{
				Namespace: tt.config.PortalName,
				Name:      tt.params.Name,
			}, &app)
			assert.Error(t, err, "削除後にApplicationが取得できないこと")

			// Secret も削除されていることを確認
			secret := corev1.Secret{}
			err = c.Get(t.Context(), types.NamespacedName{
				Namespace: tt.config.PortalName,
				Name:      tt.params.Name + "-secret",
			}, &secret)
			assert.Error(t, err, "削除後にSecretが取得できないこと")
		})
	}
}
