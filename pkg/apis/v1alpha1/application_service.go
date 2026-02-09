package v1alpha1

import (
	"context"

	"github.com/samber/lo"
	"github.com/tacokumo/portal-api/pkg/apis/v1alpha1/api"
	"github.com/tacokumo/portal-api/pkg/config"
	tacokumov1alpha1 "github.com/tacokumo/portal-controller-kubernetes/api/v1alpha1"
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
