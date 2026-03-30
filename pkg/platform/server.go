package platform

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/tacokumo/portal-api/pkg/apis/v1alpha1"
	"github.com/tacokumo/portal-api/pkg/apis/v1alpha1/api"
	"github.com/tacokumo/portal-api/pkg/auth"
	"github.com/tacokumo/portal-api/pkg/auth/github"
	"github.com/tacokumo/portal-api/pkg/auth/middleware"
	"github.com/tacokumo/portal-api/pkg/auth/session"
	"github.com/tacokumo/portal-api/pkg/config"
	"github.com/tacokumo/portal-api/pkg/k8sclient"
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

	// 新しい設定システムを使用（フォールバックあり）
	cfg := config.LoadFromEnv()
	if err := cfg.Validate(); err != nil {
		s.logger.ErrorContext(ctx, "invalid configuration", "error", err)
		return err
	}

	// 認証システムセットアップ
	jwtManager, err := auth.NewJWTManager(cfg.Auth.JWT)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create JWT manager", "error", err)
		return err
	}

	sessionManager, err := session.NewManager(cfg.Auth.Valkey)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create session manager", "error", err)
		return err
	}
	defer sessionManager.Close()

	// 認証・認可ミドルウェア
	authMw, err := middleware.NewAuthMiddleware(*cfg, jwtManager, sessionManager)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create auth middleware", "error", err)
		return err
	}

	// 追加ミドルウェア
	csrfMw := middleware.NewCSRFMiddleware(sessionManager, jwtManager, true) // HTTPS前提
	rateLimitMw := middleware.NewRateLimitMiddleware(sessionManager)
	auditMw := middleware.NewAuditMiddleware()
	rbacMw := middleware.NewRBACMiddleware(true) // strictMode有効

	// GitHubクライアント作成
	organization := "tacokumo" // TODO: 設定から取得
	githubClient := github.NewGitHubClient(cfg.Auth.GitHub, organization)

	// OAuth認証ハンドラー（将来使用予定）
	_ = github.NewOAuthHandler(
		cfg.Auth.GitHub,
		jwtManager,
		sessionManager,
		organization,
		true, // secureCookies
	)

	// グローバルミドルウェア適用（段階的統合）
	// レート制限ミドルウェア（全体に適用）
	e.Use(rateLimitMw.IPRateLimit())

	// 監査ログミドルウェア（全体に適用）
	e.Use(auditMw.Log())

	// CSRF保護ミドルウェア
	e.Use(csrfMw.Protect())

	// 認証ミドルウェアは特定のルートに適用（現在はAPIサーバー内で処理）
	_ = authMw
	_ = rbacMw

	s.logger.InfoContext(ctx, "middleware enabled",
		"auth_middleware", "configured",
		"rate_limit", "enabled",
		"audit_log", "enabled",
		"csrf", "enabled",
		"rbac", "configured")

	// 認証エンドポイント設定（認証不要）
	// 現在は既存APIサーバーにすべて委任し、段階的に移行
	s.logger.InfoContext(ctx, "authentication system initialized", "jwt_enabled", true, "session_backend", "valkey")

	sc := echo.StartConfig{
		Address:         fmt.Sprintf(":%d", cfg.Server.Port),
		GracefulTimeout: 5 * time.Second,
	}

	// 開発環境ではKubernetesクライアントを無効化
	var k8sClient client.Client
	restConfig, err := rest.InClusterConfig()
	if err != nil {
		s.logger.InfoContext(ctx, "not running in kubernetes cluster, using mock client", "error", err)
		// 開発環境用のモッククライアント（現在はnil）
		k8sClient = nil
	} else {
		scheme, err := k8sclient.NewScheme()
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to create scheme", "error", err)
			return err
		}
		k8sClient, err = client.New(restConfig, client.Options{
			Scheme: scheme,
		})
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to create k8s client", "error", err)
			return err
		}
	}
	handler := v1alpha1.NewHandler(cfg, k8sClient, jwtManager, sessionManager, githubClient, organization)
	securityHandler := v1alpha1.NewSecurityHandler(cfg, jwtManager, githubClient, sessionManager, organization)
	apiServer, err := api.NewServer(handler, securityHandler)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create API server", "error", err)
		return err
	}

	// 現在は既存のAPIサーバーに全て委任
	// 認証・認可は将来的にミドルウェアとして段階的に統合予定
	e.Any("*", echo.WrapHandler(apiServer))

	s.logger.InfoContext(ctx, "server configured",
		"auth_enabled", true,
		"middleware_ready", true,
		"oauth_handler_ready", true)
	if err := sc.Start(ctx, e); err != nil {
		s.logger.ErrorContext(ctx, "failed to start server", "error", err)
		return err
	}
	return nil
}
