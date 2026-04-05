package v1alpha1

import (
	"context"
	"fmt"

	"github.com/samber/lo"
	"github.com/tacokumo/portal-api/pkg/apis/v1alpha1/api"
	"github.com/tacokumo/portal-api/pkg/config"
	tacokumov1alpha1 "github.com/tacokumo/portal-controller-kubernetes/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ApplicationService struct {
	config *config.Config
	client client.Client
}

func (s *ApplicationService) GetApplication(
	ctx context.Context,
	params api.GetApplicationParams,
) (*api.Application, error) {
	key := types.NamespacedName{
		Namespace: s.config.PortalName,
		Name:      params.Name,
	}
	app := tacokumov1alpha1.Application{}
	if err := s.client.Get(ctx, key, &app); err != nil {
		return nil, err
	}

	return &api.Application{
		Name:            app.Name,
		AppconfigPath:   app.Spec.ReleaseTemplate.AppConfigPath,
		RepositoryURL:   app.Spec.ReleaseTemplate.Repo.URL,
		AppconfigBranch: app.Spec.ReleaseTemplate.AppConfigBranch,
	}, nil
}

func (s *ApplicationService) GetApplications(ctx context.Context) ([]api.Application, error) {
	appList := tacokumov1alpha1.ApplicationList{}
	if err := s.client.List(ctx, &appList); err != nil {
		return nil, err
	}

	apps := lo.Map(appList.Items, func(item tacokumov1alpha1.Application, _ int) api.Application {
		return api.Application{
			Name:            item.Name,
			AppconfigPath:   item.Spec.ReleaseTemplate.AppConfigPath,
			RepositoryURL:   item.Spec.ReleaseTemplate.Repo.URL,
			AppconfigBranch: item.Spec.ReleaseTemplate.AppConfigBranch,
		}
	})
	return apps, nil
}

func (s *ApplicationService) CreateApplication(
	ctx context.Context,
	req *api.CreateApplicationRequest,
) (*api.Application, error) {
	if err := ValidateCreateApplicationRequest(req); err != nil {
		return nil, err
	}

	app := tacokumov1alpha1.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: s.config.PortalName,
		},
		Spec: tacokumov1alpha1.ApplicationSpec{
			ReleaseTemplate: tacokumov1alpha1.ReleaseSpec{
				Repo: tacokumov1alpha1.RepositoryRef{
					URL: req.RepositoryURL,
				},
				AppConfigPath:   req.AppconfigPath,
				AppConfigBranch: req.AppconfigBranch,
			},
		},
	}

	if err := s.client.Create(ctx, &app); err != nil {
		return nil, err
	}
	return &api.Application{
		Name:            app.Name,
		AppconfigPath:   app.Spec.ReleaseTemplate.AppConfigPath,
		RepositoryURL:   app.Spec.ReleaseTemplate.Repo.URL,
		AppconfigBranch: app.Spec.ReleaseTemplate.AppConfigBranch,
	}, nil
}

func (s *ApplicationService) DeleteApplication(
	ctx context.Context,
	params api.DeleteApplicationParams,
) (api.DeleteApplicationRes, error) {
	// 1. Application取得（存在確認）
	appKey := types.NamespacedName{
		Namespace: s.config.PortalName,
		Name:      params.Name,
	}
	app := tacokumov1alpha1.Application{}
	if err := s.client.Get(ctx, appKey, &app); err != nil {
		return nil, err // NotFound時は404へ変換される
	}

	// 2. 関連Secret削除（存在する場合）
	secretKey := types.NamespacedName{
		Namespace: s.config.PortalName,
		Name:      fmt.Sprintf("%s-secret", params.Name),
	}
	secret := corev1.Secret{}
	if err := s.client.Get(ctx, secretKey, &secret); err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, err
		}
		// Secretが存在しない場合はスキップ
	} else {
		if err := s.client.Delete(ctx, &secret); err != nil {
			return nil, err
		}
	}

	// 3. Application削除
	if err := s.client.Delete(ctx, &app); err != nil {
		return nil, err
	}

	return &api.DeleteApplicationNoContent{}, nil
}
