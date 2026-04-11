package v1alpha1

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/tacokumo/portal-api/pkg/apis/v1alpha1/api"
	"github.com/tacokumo/portal-api/pkg/auth"
	"github.com/tacokumo/portal-api/pkg/auth/github"
	"github.com/tacokumo/portal-api/pkg/auth/session"
	"github.com/tacokumo/portal-api/pkg/config"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Handler struct {
	*HealthCheckService
	*ApplicationService
	*ApplicationSecretService
	*ApplicationLogService
	*AuthService
}

func NewHandler(
	cfg *config.Config,
	client client.Client,
	clientset kubernetes.Interface,
	jwtManager *auth.JWTManager,
	sessionManager session.Manager,
	githubClient github.AuthProvider,
	organization string,
) *Handler {
	return &Handler{
		HealthCheckService:       &HealthCheckService{},
		ApplicationService:       &ApplicationService{config: cfg, client: client},
		ApplicationSecretService: NewApplicationSecretService(cfg, client),
		ApplicationLogService:    &ApplicationLogService{config: cfg, client: client, clientset: clientset},
		AuthService: NewAuthService(
			cfg,
			githubClient,
			jwtManager,
			sessionManager,
			organization,
		),
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
	var ewc *ErrorWithCode
	if errors.As(err, &ewc) {
		return &api.ErrorStatusCode{
			StatusCode: ewc.Code,
			Response: api.Error{
				Code:    int32(ewc.Code),
				Message: ewc.Message,
			},
		}
	}

	// Kubernetes NotFoundエラーを404に変換
	if apierrors.IsNotFound(err) {
		return &api.ErrorStatusCode{
			StatusCode: http.StatusNotFound,
			Response: api.Error{
				Code:    ErrorCodeNotFound,
				Message: "resource not found",
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
