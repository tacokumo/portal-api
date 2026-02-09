package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tacokumo/portal-api/pkg/apis/v1alpha1/api"
	"github.com/tacokumo/portal-api/pkg/config"
	"github.com/tacokumo/portal-api/pkg/k8sclient"
	tacokumov1alpha1 "github.com/tacokumo/portal-controller-kubernetes/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
