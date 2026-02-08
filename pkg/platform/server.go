package platform

import (
	"context"
	"log/slog"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/tacokumo/portal-api/pkg/apis/v1alpha1"
	"github.com/tacokumo/portal-api/pkg/apis/v1alpha1/api"
	tacokumov1alpha1 "github.com/tacokumo/portal-controller-kubernetes/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Server struct {
	logger *slog.Logger
}

func NewServer(logger *slog.Logger) *Server {
	return &Server{
		logger: logger,
	}
}

func (s *Server) Start(ctx context.Context) error {
	e := echo.New()
	sc := echo.StartConfig{
		Address:         ":1323",
		GracefulTimeout: 5 * time.Second,
	}

	restConfig, err := rest.InClusterConfig()
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get in-cluster config", "error", err)
		return err
	}

	scheme, err := newScheme()
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create scheme", "error", err)
		return err
	}
	k8sClient, err := client.New(restConfig, client.Options{
		Scheme: scheme,
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create k8s client", "error", err)
		return err
	}
	apiServer, err := api.NewServer(v1alpha1.NewHandler(k8sClient))
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create API server", "error", err)
		return err
	}

	e.Any("*", echo.WrapHandler(apiServer))
	if err := sc.Start(ctx, e); err != nil {
		s.logger.ErrorContext(ctx, "failed to start server", "error", err)
		return err
	}
	return nil
}

func newScheme() (*runtime.Scheme, error) {
	s := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(s); err != nil {
		return nil, err
	}
	if err := tacokumov1alpha1.AddToScheme(s); err != nil {
		return nil, err
	}
	return s, nil
}
