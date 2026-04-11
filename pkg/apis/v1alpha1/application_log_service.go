package v1alpha1

import (
	"context"
	"io"
	"net/http"

	"github.com/tacokumo/portal-api/pkg/apis/v1alpha1/api"
	"github.com/tacokumo/portal-api/pkg/config"
	tacokumov1alpha1 "github.com/tacokumo/portal-controller-kubernetes/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ApplicationLogService struct {
	config    *config.Config
	client    client.Client
	clientset kubernetes.Interface
}

func (s *ApplicationLogService) GetApplicationLogs(
	ctx context.Context,
	params api.GetApplicationLogsParams,
) (api.GetApplicationLogsRes, error) {
	// K8sクラスタ未接続の場合は503
	if s.client == nil || s.clientset == nil {
		return nil, &ErrorWithCode{
			Code:    http.StatusServiceUnavailable,
			Message: "kubernetes cluster is not available",
		}
	}

	if err := ValidateGetApplicationLogsParams(params); err != nil {
		return nil, err
	}

	// Application CRDの存在確認
	key := types.NamespacedName{
		Namespace: s.config.PortalName,
		Name:      params.Name,
	}
	app := tacokumov1alpha1.Application{}
	if err := s.client.Get(ctx, key, &app); err != nil {
		return nil, err // NotFound → 404に変換される
	}

	// アプリケーションに属するPod一覧を取得
	podList := corev1.PodList{}
	labelSelector := labels.SelectorFromSet(labels.Set{
		"app.kubernetes.io/name": params.Name,
	})
	if err := s.client.List(ctx, &podList, &client.ListOptions{
		Namespace:     s.config.PortalName,
		LabelSelector: labelSelector,
	}); err != nil {
		return nil, err
	}

	// 各Podのログを取得
	tailLines := int64(100)
	if v, ok := params.TailLines.Get(); ok {
		tailLines = int64(v)
	}

	logOpts := &corev1.PodLogOptions{
		TailLines: &tailLines,
	}
	if v, ok := params.SinceSeconds.Get(); ok {
		logOpts.SinceSeconds = &v
	}
	if v, ok := params.Container.Get(); ok {
		logOpts.Container = v
	}

	podLogs := make([]api.PodLog, 0, len(podList.Items))
	for _, pod := range podList.Items {
		containers := collectContainerNames(pod, logOpts.Container)
		for _, containerName := range containers {
			opts := logOpts.DeepCopy()
			opts.Container = containerName

			req := s.clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, opts)
			stream, err := req.Stream(ctx)
			if err != nil {
				// 個別Podのログエラーはスキップしてログとしてエラーメッセージを返す
				podLogs = append(podLogs, api.PodLog{
					PodName:       pod.Name,
					ContainerName: containerName,
					Logs:          "error: " + err.Error(),
				})
				continue
			}

			logBytes, err := io.ReadAll(stream)
			stream.Close()
			if err != nil {
				podLogs = append(podLogs, api.PodLog{
					PodName:       pod.Name,
					ContainerName: containerName,
					Logs:          "error: " + err.Error(),
				})
				continue
			}

			podLogs = append(podLogs, api.PodLog{
				PodName:       pod.Name,
				ContainerName: containerName,
				Logs:          string(logBytes),
			})
		}
	}

	return &api.ApplicationLogs{
		ApplicationName: params.Name,
		PodLogs:         podLogs,
	}, nil
}

// collectContainerNames はPodのコンテナ名一覧を返す。
// containerFilter が指定されている場合はそのコンテナのみ返す。
func collectContainerNames(pod corev1.Pod, containerFilter string) []string {
	if containerFilter != "" {
		return []string{containerFilter}
	}

	names := make([]string, 0, len(pod.Spec.Containers))
	for _, c := range pod.Spec.Containers {
		names = append(names, c.Name)
	}
	return names
}
