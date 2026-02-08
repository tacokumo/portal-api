package v1alpha1

import (
	"context"

	"github.com/tacokumo/portal-api/pkg/apis/v1alpha1/api"
)

type HealthCheckService struct{}

func (s *HealthCheckService) GetHealthLiveness(ctx context.Context) (*api.HealthCheckStatus, error) {
	return &api.HealthCheckStatus{
		Status: "OK",
	}, nil
}

func (s *HealthCheckService) GetHealthReadiness(ctx context.Context) (*api.HealthCheckStatus, error) {
	return &api.HealthCheckStatus{
		Status: "OK",
	}, nil
}
