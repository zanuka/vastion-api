package handler

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/zanuka/baluardo-api/internal/domain"
	"github.com/zanuka/baluardo-api/internal/service"
)

type Detections struct {
	svc *service.Detections
}

func NewDetections(svc *service.Detections) *Detections {
	return &Detections{svc: svc}
}

type listDetectionsInput struct {
	Role     string `header:"X-Operator-Role"`
	Name     string `header:"X-Operator-Name"`
	Site     string `query:"site"`
	Severity string `query:"severity"`
	Status   string `query:"status"`
	Cursor   string `query:"cursor"`
}

type listDetectionsOutput struct {
	Body struct {
		Items      []DetectionBody `json:"items"`
		NextCursor *string         `json:"nextCursor"`
	}
}

type getDetectionInput struct {
	Role string `header:"X-Operator-Role"`
	Name string `header:"X-Operator-Name"`
	ID   string `path:"id"`
}

type detectionOutput struct {
	Body DetectionBody
}

type ackDetectionInput struct {
	Role string `header:"X-Operator-Role"`
	Name string `header:"X-Operator-Name"`
	ID   string `path:"id"`
}

type rejectDetectionInput struct {
	Role string `header:"X-Operator-Role"`
	Name string `header:"X-Operator-Name"`
	ID   string `path:"id"`
	Body struct {
		Reason string `json:"reason"`
	}
}

func (h *Detections) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID:   "list-detections",
		Method:        http.MethodGet,
		Path:          "/api/v1/detections",
		Summary:       "List detections",
		Tags:          []string{"Detections"},
		DefaultStatus: http.StatusOK,
		Errors:        []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusBadRequest},
	}, h.List)

	huma.Register(api, huma.Operation{
		OperationID:   "get-detection",
		Method:        http.MethodGet,
		Path:          "/api/v1/detections/{id}",
		Summary:       "Get detection",
		Tags:          []string{"Detections"},
		DefaultStatus: http.StatusOK,
		Errors:        []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound, http.StatusBadRequest},
	}, h.Get)

	huma.Register(api, huma.Operation{
		OperationID:   "ack-detection",
		Method:        http.MethodPost,
		Path:          "/api/v1/detections/{id}/ack",
		Summary:       "Acknowledge detection",
		Description:   "Idempotent: repeating ack on an already acked detection returns 200 with the same body. Illegal transitions return 409.",
		Tags:          []string{"Detections"},
		DefaultStatus: http.StatusOK,
		Errors:        []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound, http.StatusConflict, http.StatusBadRequest},
	}, h.Ack)

	huma.Register(api, huma.Operation{
		OperationID:   "reject-detection",
		Method:        http.MethodPost,
		Path:          "/api/v1/detections/{id}/reject",
		Summary:       "Reject detection",
		Description:   "Reason is required. Idempotent if already rejected. Illegal transitions return 409.",
		Tags:          []string{"Detections"},
		DefaultStatus: http.StatusOK,
		Errors:        []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound, http.StatusConflict, http.StatusBadRequest},
	}, h.Reject)
}

func (h *Detections) List(ctx context.Context, input *listDetectionsInput) (*listDetectionsOutput, error) {
	if _, err := requireOperator(input.Role, input.Name); err != nil {
		return nil, err
	}
	status, err := parseStatus(input.Status)
	if err != nil {
		return nil, mapError(err)
	}
	severity, err := parseSeverity(input.Severity)
	if err != nil {
		return nil, mapError(err)
	}
	page, err := h.svc.List(ctx, domain.DetectionListFilter{
		SiteID:   input.Site,
		Status:   status,
		Severity: severity,
	}, input.Cursor)
	if err != nil {
		return nil, mapError(err)
	}
	out := &listDetectionsOutput{}
	out.Body.Items = make([]DetectionBody, 0, len(page.Items))
	for _, item := range page.Items {
		out.Body.Items = append(out.Body.Items, detectionBody(item))
	}
	if page.NextCursor != "" {
		out.Body.NextCursor = &page.NextCursor
	}
	return out, nil
}

func (h *Detections) Get(ctx context.Context, input *getDetectionInput) (*detectionOutput, error) {
	if _, err := requireOperator(input.Role, input.Name); err != nil {
		return nil, err
	}
	d, err := h.svc.Get(ctx, input.ID)
	if err != nil {
		return nil, mapError(err)
	}
	return &detectionOutput{Body: detectionBody(*d)}, nil
}

func (h *Detections) Ack(ctx context.Context, input *ackDetectionInput) (*detectionOutput, error) {
	op, err := requireOperator(input.Role, input.Name)
	if err != nil {
		return nil, err
	}
	d, err := h.svc.Ack(ctx, input.ID, op.Name)
	if err != nil {
		return nil, mapError(err)
	}
	return &detectionOutput{Body: detectionBody(*d)}, nil
}

func (h *Detections) Reject(ctx context.Context, input *rejectDetectionInput) (*detectionOutput, error) {
	op, err := requireOperator(input.Role, input.Name)
	if err != nil {
		return nil, err
	}
	d, err := h.svc.Reject(ctx, input.ID, input.Body.Reason, op.Name)
	if err != nil {
		return nil, mapError(err)
	}
	return &detectionOutput{Body: detectionBody(*d)}, nil
}
