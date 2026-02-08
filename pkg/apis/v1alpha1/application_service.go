package v1alpha1

import (
	"context"

	"github.com/tacokumo/portal-api/pkg/apis/v1alpha1/api"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ApplicationService struct {
	client client.Client
}

func (s *ApplicationService) GetApplication(
	ctx context.Context,
	params api.GetApplicationParams) (*api.Application, error) {
	return &api.Application{}, nil
}

func (s *ApplicationService) GetApplications(ctx context.Context) ([]api.Application, error) {
	return []api.Application{}, nil
}
