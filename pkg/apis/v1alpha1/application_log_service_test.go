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
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestApplicationLogService_GetApplicationLogs(t *testing.T) {
	tests := []struct {
		name     string
		config   *config.Config
		clientFn func() client.Client
		useCS    bool // clientsetを設定するかどうか
		params   api.GetApplicationLogsParams
		isError  bool
		errCode  int
	}{
		{
			name: "K8sクラスタ未接続の場合503エラーとなること",
			config: &config.Config{
				PortalName: "portal-namespace",
			},
			clientFn: func() client.Client {
				return nil
			},
			useCS: false,
			params: api.GetApplicationLogsParams{
				Name: "example-app",
			},
			isError: true,
			errCode: 503,
		},
		{
			name: "clientsetのみnilの場合503エラーとなること",
			config: &config.Config{
				PortalName: "portal-namespace",
			},
			clientFn: func() client.Client {
				scheme, err := k8sclient.NewScheme()
				assert.NoError(t, err)
				return fake.NewClientBuilder().WithScheme(scheme).Build()
			},
			useCS: false,
			params: api.GetApplicationLogsParams{
				Name: "example-app",
			},
			isError: true,
			errCode: 503,
		},
		{
			name: "存在しないApplicationの場合エラーとなること",
			config: &config.Config{
				PortalName: "portal-namespace",
			},
			clientFn: func() client.Client {
				scheme, err := k8sclient.NewScheme()
				assert.NoError(t, err)
				return fake.NewClientBuilder().WithScheme(scheme).Build()
			},
			useCS: true,
			params: api.GetApplicationLogsParams{
				Name: "non-existent-app",
			},
			isError: true,
		},
		{
			name: "不正なアプリケーション名の場合バリデーションエラーとなること",
			config: &config.Config{
				PortalName: "portal-namespace",
			},
			clientFn: func() client.Client {
				scheme, err := k8sclient.NewScheme()
				assert.NoError(t, err)
				return fake.NewClientBuilder().WithScheme(scheme).Build()
			},
			useCS: true,
			params: api.GetApplicationLogsParams{
				Name: "Invalid-Name",
			},
			isError: true,
			errCode: 400,
		},
		{
			name: "tail_linesが範囲外の場合バリデーションエラーとなること",
			config: &config.Config{
				PortalName: "portal-namespace",
			},
			clientFn: func() client.Client {
				scheme, err := k8sclient.NewScheme()
				assert.NoError(t, err)
				return fake.NewClientBuilder().WithScheme(scheme).Build()
			},
			useCS: true,
			params: api.GetApplicationLogsParams{
				Name:      "example-app",
				TailLines: api.NewOptInt32(20000),
			},
			isError: true,
			errCode: 400,
		},
		{
			name: "since_secondsが0以下の場合バリデーションエラーとなること",
			config: &config.Config{
				PortalName: "portal-namespace",
			},
			clientFn: func() client.Client {
				scheme, err := k8sclient.NewScheme()
				assert.NoError(t, err)
				return fake.NewClientBuilder().WithScheme(scheme).Build()
			},
			useCS: true,
			params: api.GetApplicationLogsParams{
				Name:         "example-app",
				SinceSeconds: api.NewOptInt64(0),
			},
			isError: true,
			errCode: 400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service := &ApplicationLogService{
				config: tt.config,
				client: tt.clientFn(),
			}
			if tt.useCS {
				service.clientset = k8sfake.NewSimpleClientset()
			}

			ret, err := service.GetApplicationLogs(t.Context(), tt.params)
			if tt.isError {
				assert.Error(t, err)
				if tt.errCode > 0 {
					ewc, ok := err.(*ErrorWithCode)
					assert.True(t, ok, "ErrorWithCode型であること")
					if ok {
						assert.Equal(t, tt.errCode, ewc.Code)
					}
				}
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, ret)
		})
	}
}

func TestApplicationLogService_GetApplicationLogs_EmptyPods(t *testing.T) {
	scheme, err := k8sclient.NewScheme()
	assert.NoError(t, err)

	c := fake.NewClientBuilder().WithScheme(scheme).Build()

	// Application CRDを作成
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
					URL: "https://github.com/tacokumo/example-app.git",
				},
			},
		},
	})
	assert.NoError(t, err)

	cs := k8sfake.NewSimpleClientset()

	service := &ApplicationLogService{
		config:    &config.Config{PortalName: "portal-namespace"},
		client:    c,
		clientset: cs,
	}

	ret, err := service.GetApplicationLogs(t.Context(), api.GetApplicationLogsParams{
		Name: "example-app",
	})
	assert.NoError(t, err)

	logs, ok := ret.(*api.ApplicationLogs)
	assert.True(t, ok)
	assert.Equal(t, "example-app", logs.ApplicationName)
	assert.Empty(t, logs.PodLogs, "Podが存在しない場合は空のpod_logsを返すこと")
}

func TestApplicationLogService_GetApplicationLogs_WithPods(t *testing.T) {
	scheme, err := k8sclient.NewScheme()
	assert.NoError(t, err)

	c := fake.NewClientBuilder().WithScheme(scheme).Build()

	// Application CRDを作成
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
					URL: "https://github.com/tacokumo/example-app.git",
				},
			},
		},
	})
	assert.NoError(t, err)

	// 対象ラベル付きPodを作成
	err = c.Create(t.Context(), &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-app-pod-1",
			Namespace: "portal-namespace",
			Labels: map[string]string{
				"app.kubernetes.io/name": "example-app",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "app", Image: "example:latest"},
			},
		},
	})
	assert.NoError(t, err)

	// ラベルなしPod（対象外）
	err = c.Create(t.Context(), &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "other-app-pod",
			Namespace: "portal-namespace",
			Labels: map[string]string{
				"app.kubernetes.io/name": "other-app",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "app", Image: "other:latest"},
			},
		},
	})
	assert.NoError(t, err)

	cs := k8sfake.NewSimpleClientset()

	service := &ApplicationLogService{
		config:    &config.Config{PortalName: "portal-namespace"},
		client:    c,
		clientset: cs,
	}

	ret, err := service.GetApplicationLogs(t.Context(), api.GetApplicationLogsParams{
		Name: "example-app",
	})
	assert.NoError(t, err)

	logs, ok := ret.(*api.ApplicationLogs)
	assert.True(t, ok)
	assert.Equal(t, "example-app", logs.ApplicationName)
	// fake clientset はログストリームをサポートしないため、エラーメッセージが返る
	assert.Len(t, logs.PodLogs, 1, "対象Podのログエントリが1つ返ること")
	assert.Equal(t, "example-app-pod-1", logs.PodLogs[0].PodName)
	assert.Equal(t, "app", logs.PodLogs[0].ContainerName)
}

func TestCollectContainerNames(t *testing.T) {
	pod := corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "app"},
				{Name: "sidecar"},
			},
		},
	}

	t.Run("フィルタなしで全コンテナ名を返す", func(t *testing.T) {
		names := collectContainerNames(pod, "")
		assert.Equal(t, []string{"app", "sidecar"}, names)
	})

	t.Run("フィルタ指定で該当コンテナのみ返す", func(t *testing.T) {
		names := collectContainerNames(pod, "sidecar")
		assert.Equal(t, []string{"sidecar"}, names)
	})
}
