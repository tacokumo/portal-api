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
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestApplicationSecretService_CreateApplicationSecret(t *testing.T) {
	tests := []struct {
		name     string
		config   *config.Config
		clientFn func() client.Client
		req      *api.CreateSecretRequest
		params   api.CreateApplicationSecretParams
		isError  bool
	}{
		{
			name: "正常に作成できるケース（Applicationが存在する場合）",
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
			req: &api.CreateSecretRequest{
				Items: []api.SecretItem{
					{Key: "DB_PASSWORD", Value: "secret123"},
					{Key: "API_KEY", Value: "apikey456"},
				},
			},
			params: api.CreateApplicationSecretParams{
				Name: "example-app",
			},
			isError: false,
		},
		{
			name: "Applicationが存在しない場合のエラーケース",
			config: &config.Config{
				PortalName: "portal-namespace",
			},
			clientFn: func() client.Client {
				scheme, err := k8sclient.NewScheme()
				assert.NoError(t, err)
				return fake.NewClientBuilder().WithScheme(scheme).Build()
			},
			req: &api.CreateSecretRequest{
				Items: []api.SecretItem{
					{Key: "DB_PASSWORD", Value: "secret123"},
				},
			},
			params: api.CreateApplicationSecretParams{
				Name: "non-existent-app",
			},
			isError: true,
		},
		{
			name: "既にSecretが存在する場合のエラーケース",
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
				err = c.Create(t.Context(), &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "existing-app-secret",
						Namespace: "portal-namespace",
					},
					StringData: map[string]string{
						"OLD_KEY": "old_value",
					},
				})
				assert.NoError(t, err)
				return c
			},
			req: &api.CreateSecretRequest{
				Items: []api.SecretItem{
					{Key: "DB_PASSWORD", Value: "secret123"},
				},
			},
			params: api.CreateApplicationSecretParams{
				Name: "existing-app",
			},
			isError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service := &ApplicationSecretService{
				config: tt.config,
				client: tt.clientFn(),
			}
			ret, err := service.CreateApplicationSecret(t.Context(), tt.req, tt.params)
			if tt.isError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, ret)
			assert.Len(t, ret.Items, len(tt.req.Items))
			for _, item := range ret.Items {
				assert.Equal(t, "REDACTED", item.Value)
			}
		})
	}
}

func TestApplicationSecretService_GetApplicationSecret(t *testing.T) {
	tests := []struct {
		name     string
		config   *config.Config
		clientFn func() client.Client
		params   api.GetApplicationSecretParams
		isError  bool
	}{
		{
			name: "存在するSecretを取得できるケース",
			config: &config.Config{
				PortalName: "portal-namespace",
			},
			clientFn: func() client.Client {
				scheme, err := k8sclient.NewScheme()
				assert.NoError(t, err)
				c := fake.NewClientBuilder().WithScheme(scheme).Build()
				err = c.Create(t.Context(), &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "example-app-secret",
						Namespace: "portal-namespace",
					},
					Data: map[string][]byte{
						"DB_PASSWORD": []byte("secret123"),
						"API_KEY":     []byte("apikey456"),
					},
				})
				assert.NoError(t, err)
				return c
			},
			params: api.GetApplicationSecretParams{
				Name: "example-app",
			},
			isError: false,
		},
		{
			name: "存在しないSecretを取得しようとした場合のエラーケース",
			config: &config.Config{
				PortalName: "portal-namespace",
			},
			clientFn: func() client.Client {
				scheme, err := k8sclient.NewScheme()
				assert.NoError(t, err)
				return fake.NewClientBuilder().WithScheme(scheme).Build()
			},
			params: api.GetApplicationSecretParams{
				Name: "non-existent-app",
			},
			isError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service := &ApplicationSecretService{
				config: tt.config,
				client: tt.clientFn(),
			}
			ret, err := service.GetApplicationSecret(t.Context(), tt.params)
			if tt.isError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, ret)
			assert.Len(t, ret.Items, 2)
			for _, item := range ret.Items {
				assert.Equal(t, "REDACTED", item.Value)
			}
		})
	}
}

func TestApplicationSecretService_UpdateApplicationSecret(t *testing.T) {
	tests := []struct {
		name     string
		config   *config.Config
		clientFn func() client.Client
		req      *api.CreateSecretRequest
		params   api.UpdateApplicationSecretParams
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
				err = c.Create(t.Context(), &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "example-app-secret",
						Namespace: "portal-namespace",
					},
					Data: map[string][]byte{
						"DB_PASSWORD": []byte("old_secret"),
					},
				})
				assert.NoError(t, err)
				return c
			},
			req: &api.CreateSecretRequest{
				Items: []api.SecretItem{
					{Key: "DB_PASSWORD", Value: "new_secret"},
					{Key: "NEW_KEY", Value: "new_value"},
				},
			},
			params: api.UpdateApplicationSecretParams{
				Name: "example-app",
			},
			isError: false,
		},
		{
			name: "存在しないSecretを更新しようとした場合のエラーケース",
			config: &config.Config{
				PortalName: "portal-namespace",
			},
			clientFn: func() client.Client {
				scheme, err := k8sclient.NewScheme()
				assert.NoError(t, err)
				return fake.NewClientBuilder().WithScheme(scheme).Build()
			},
			req: &api.CreateSecretRequest{
				Items: []api.SecretItem{
					{Key: "DB_PASSWORD", Value: "new_secret"},
				},
			},
			params: api.UpdateApplicationSecretParams{
				Name: "non-existent-app",
			},
			isError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service := &ApplicationSecretService{
				config: tt.config,
				client: tt.clientFn(),
			}
			ret, err := service.UpdateApplicationSecret(t.Context(), tt.req, tt.params)
			if tt.isError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, ret)
			assert.Len(t, ret.Items, len(tt.req.Items))
			for _, item := range ret.Items {
				assert.Equal(t, "REDACTED", item.Value)
			}
		})
	}
}
