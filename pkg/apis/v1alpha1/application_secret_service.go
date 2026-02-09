package v1alpha1

import (
	"context"
	"fmt"

	"github.com/samber/lo"
	"github.com/tacokumo/portal-api/pkg/apis/v1alpha1/api"
	"github.com/tacokumo/portal-api/pkg/config"
	tacokumov1alpha1 "github.com/tacokumo/portal-controller-kubernetes/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ApplicationSecretService struct {
	config *config.Config
	client client.Client
}

func NewApplicationSecretService(
	cfg *config.Config,
	client client.Client,
) *ApplicationSecretService {
	return &ApplicationSecretService{
		config: cfg,
		client: client,
	}
}

func (s *ApplicationSecretService) CreateApplicationSecret(ctx context.Context, req *api.CreateSecretRequest, params api.CreateApplicationSecretParams) (*api.Secret, error) {
	// TODO: Secretを暗号化/復号する鍵を生成して保存する

	app := tacokumov1alpha1.Application{}
	err := s.client.Get(ctx, client.ObjectKey{
		Namespace: s.config.PortalName,
		Name:      params.Name,
	}, &app)
	if err != nil {
		return nil, err
	}

	secretData := lo.Reduce(req.Items, func(acc map[string]string, item api.SecretItem, _ int) map[string]string {
		acc[item.Key] = item.Value
		return acc
	}, map[string]string{})
	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: s.config.PortalName,
			Name:      fmt.Sprintf("%s-secret", params.Name),
		},
		StringData: secretData,
	}

	if err := s.client.Create(ctx, &secret); err != nil {
		return nil, err
	}

	app.Spec.ReleaseTemplate.EnvSecretName = &secret.Name
	if err := s.client.Update(ctx, &app); err != nil {
		return nil, err
	}
	return &api.Secret{
		Items: lo.Map(req.Items, func(item api.SecretItem, _ int) api.SecretItem {
			return api.SecretItem{
				Key:   item.Key,
				Value: "REDACTED",
			}
		}),
	}, nil
}

func (s *ApplicationSecretService) GetApplicationSecret(ctx context.Context, params api.GetApplicationSecretParams) (*api.Secret, error) {
	key := types.NamespacedName{
		Namespace: s.config.PortalName,
		Name:      fmt.Sprintf("%s-secret", params.Name),
	}
	secret := corev1.Secret{}
	if err := s.client.Get(ctx, key, &secret); err != nil {
		return nil, err
	}

	return &api.Secret{
		Items: lo.MapToSlice(secret.Data, func(key string, value []byte) api.SecretItem {
			return api.SecretItem{
				Key:   key,
				Value: "REDACTED",
			}
		}),
	}, nil
}

func (s *ApplicationSecretService) UpdateApplicationSecret(ctx context.Context, req *api.CreateSecretRequest, params api.UpdateApplicationSecretParams) (*api.Secret, error) {
	key := types.NamespacedName{
		Namespace: s.config.PortalName,
		Name:      fmt.Sprintf("%s-secret", params.Name),
	}
	secret := corev1.Secret{}
	if err := s.client.Get(ctx, key, &secret); err != nil {
		return nil, err
	}

	if secret.StringData == nil {
		secret.StringData = make(map[string]string)
	}
	for _, item := range req.Items {
		secret.StringData[item.Key] = item.Value
	}

	if err := s.client.Update(ctx, &secret); err != nil {
		return nil, err
	}

	return &api.Secret{
		Items: lo.Map(req.Items, func(item api.SecretItem, _ int) api.SecretItem {
			return api.SecretItem{
				Key:   item.Key,
				Value: "REDACTED",
			}
		}),
	}, nil
}
