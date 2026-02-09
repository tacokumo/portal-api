package v1alpha1

import (
	"context"
	"fmt"
	"net/http"

	"github.com/tacokumo/portal-api/pkg/apis/v1alpha1/api"
	"github.com/tacokumo/portal-api/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Handler struct {
	*HealthCheckService
	*ApplicationService
}

func NewHandler(
	cfg *config.Config,
	client client.Client) *Handler {
	return &Handler{
		HealthCheckService: &HealthCheckService{},
		ApplicationService: &ApplicationService{config: cfg, client: client},
	}
}

var _ api.Handler = &Handler{}

type ErrorWithCode struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

var _ error = &ErrorWithCode{}

func (e *ErrorWithCode) Error() string {
	return fmt.Sprintf("E%d: %s", e.Code, e.Message)
}

func (h *Handler) NewError(ctx context.Context, err error) *api.ErrorStatusCode {
	if ewc, ok := err.(*ErrorWithCode); ok {
		return &api.ErrorStatusCode{
			StatusCode: ewc.Code,
			Response: api.Error{
				Code:    int32(ewc.Code),
				Message: ewc.Message,
			},
		}
	}

	return &api.ErrorStatusCode{
		StatusCode: http.StatusInternalServerError,
		Response: api.Error{
			Code:    ErrorCodeUnknown,
			Message: err.Error(),
		},
	}
}
