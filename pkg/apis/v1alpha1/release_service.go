package v1alpha1

import (
	"context"
	"fmt"
	"net/http"

	"github.com/tacokumo/portal-api/pkg/apis/v1alpha1/api"
	"github.com/tacokumo/portal-api/pkg/config"
	tacokumov1alpha1 "github.com/tacokumo/portal-controller-kubernetes/api/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ReleaseService struct {
	config *config.Config
	client client.Client
}

func (s *ReleaseService) GetApplicationReleases(
	ctx context.Context,
	params api.GetApplicationReleasesParams,
) (api.GetApplicationReleasesRes, error) {
	if s.client == nil {
		return nil, &ErrorWithCode{
			Code:    http.StatusServiceUnavailable,
			Message: "kubernetes cluster is not available",
		}
	}

	if err := ValidateApplicationName(params.Name); err != nil {
		return nil, err
	}

	// Application CRDの存在確認
	key := types.NamespacedName{
		Namespace: s.config.PortalName,
		Name:      params.Name,
	}
	app := tacokumov1alpha1.Application{}
	if err := s.client.Get(ctx, key, &app); err != nil {
		return nil, err
	}

	releases := make([]api.Release, 0, len(app.Status.Releases))
	for _, ref := range app.Status.Releases {
		ns := ref.Namespace
		if ns == "" {
			ns = s.config.PortalName
		}
		releaseKey := types.NamespacedName{
			Namespace: ns,
			Name:      ref.Name,
		}
		rel := tacokumov1alpha1.Release{}
		if err := s.client.Get(ctx, releaseKey, &rel); err != nil {
			// 削除済みReleaseはスキップ
			continue
		}

		apiRelease := api.Release{
			Name:            rel.Name,
			State:           rel.Status.State,
			RepositoryURL:   rel.Spec.Repo.URL,
			AppconfigPath:   rel.Spec.AppConfigPath,
			AppconfigBranch: rel.Spec.AppConfigBranch,
			CreatedAt:       rel.CreationTimestamp.Time,
		}
		if rel.Spec.Commit != nil {
			apiRelease.Commit = api.NewOptString(*rel.Spec.Commit)
		}
		releases = append(releases, apiRelease)
	}

	result := api.GetApplicationReleasesOKApplicationJSON(releases)
	return &result, nil
}

func (s *ReleaseService) RollbackApplication(
	ctx context.Context,
	req *api.RollbackRequest,
	params api.RollbackApplicationParams,
) (api.RollbackApplicationRes, error) {
	if s.client == nil {
		return nil, &ErrorWithCode{
			Code:    http.StatusServiceUnavailable,
			Message: "kubernetes cluster is not available",
		}
	}

	if err := ValidateApplicationName(params.Name); err != nil {
		return nil, err
	}
	if err := ValidateRollbackRequest(req); err != nil {
		return nil, err
	}

	// Application CRD取得
	key := types.NamespacedName{
		Namespace: s.config.PortalName,
		Name:      params.Name,
	}
	app := tacokumov1alpha1.Application{}
	if err := s.client.Get(ctx, key, &app); err != nil {
		return nil, err
	}

	// 対象ReleaseがApplicationに紐づいているか確認
	found := false
	for _, ref := range app.Status.Releases {
		if ref.Name == req.ReleaseName {
			found = true
			break
		}
	}
	if !found {
		return nil, &ErrorWithCode{
			Code:    http.StatusNotFound,
			Message: fmt.Sprintf("release %q not found in application %q", req.ReleaseName, params.Name),
		}
	}

	// Release CRD取得
	releaseKey := types.NamespacedName{
		Namespace: s.config.PortalName,
		Name:      req.ReleaseName,
	}
	rel := tacokumov1alpha1.Release{}
	if err := s.client.Get(ctx, releaseKey, &rel); err != nil {
		return nil, err
	}

	// ReleaseのSpecをApplicationのReleaseTemplateにコピー（EnvSecretNameは保持）
	app.Spec.ReleaseTemplate.Repo = rel.Spec.Repo
	app.Spec.ReleaseTemplate.AppConfigPath = rel.Spec.AppConfigPath
	app.Spec.ReleaseTemplate.AppConfigBranch = rel.Spec.AppConfigBranch
	app.Spec.ReleaseTemplate.Commit = rel.Spec.Commit

	if err := s.client.Update(ctx, &app); err != nil {
		return nil, err
	}

	return &api.Application{
		Name:            app.Name,
		AppconfigPath:   app.Spec.ReleaseTemplate.AppConfigPath,
		RepositoryURL:   app.Spec.ReleaseTemplate.Repo.URL,
		AppconfigBranch: app.Spec.ReleaseTemplate.AppConfigBranch,
	}, nil
}
