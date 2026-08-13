package handler

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/zanuka/vastion-api/internal/service"
)

type HealthOutput struct {
	Body struct {
		Status string `json:"status"`
	}
}

type Health struct {
	svc *service.Health
}

func NewHealth(svc *service.Health) *Health {
	return &Health{svc: svc}
}

func (h *Health) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "get-healthz",
		Method:      http.MethodGet,
		Path:        "/healthz",
		Summary:     "Process liveness",
		Tags:        []string{"Health"},
	}, h.Healthz)

	huma.Register(api, huma.Operation{
		OperationID: "get-readyz",
		Method:      http.MethodGet,
		Path:        "/readyz",
		Summary:     "Mongo readiness",
		Tags:        []string{"Health"},
	}, h.Readyz)
}

func (h *Health) Healthz(_ context.Context, _ *struct{}) (*HealthOutput, error) {
	out := &HealthOutput{}
	out.Body.Status = "ok"
	return out, nil
}

func (h *Health) Readyz(ctx context.Context, _ *struct{}) (*HealthOutput, error) {
	if err := h.svc.Ready(ctx); err != nil {
		return nil, huma.NewError(http.StatusServiceUnavailable, "unavailable")
	}
	out := &HealthOutput{}
	out.Body.Status = "ok"
	return out, nil
}
