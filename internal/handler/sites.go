package handler

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/zanuka/baluardo-api/internal/service"
)

type Sites struct {
	svc *service.Sites
}

func NewSites(svc *service.Sites) *Sites {
	return &Sites{svc: svc}
}

type listSitesInput struct {
	Role string `header:"X-Operator-Role"`
	Name string `header:"X-Operator-Name"`
}

type listSitesOutput struct {
	Body struct {
		Items []SiteBody `json:"items"`
	}
}

func (h *Sites) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID:   "list-sites",
		Method:        http.MethodGet,
		Path:          "/api/v1/sites",
		Summary:       "List sites",
		Tags:          []string{"Sites"},
		DefaultStatus: http.StatusOK,
		Errors:        []int{http.StatusUnauthorized, http.StatusForbidden},
	}, h.List)
}

func (h *Sites) List(ctx context.Context, input *listSitesInput) (*listSitesOutput, error) {
	if _, err := requireOperator(input.Role, input.Name); err != nil {
		return nil, err
	}
	sites, err := h.svc.List(ctx)
	if err != nil {
		return nil, mapError(err)
	}
	out := &listSitesOutput{}
	out.Body.Items = make([]SiteBody, 0, len(sites))
	for _, site := range sites {
		out.Body.Items = append(out.Body.Items, siteBody(site))
	}
	return out, nil
}
