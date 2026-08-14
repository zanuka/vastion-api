package handler_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zanuka/baluardo-api/internal/domain"
	"github.com/zanuka/baluardo-api/internal/fakes"
	"github.com/zanuka/baluardo-api/internal/router"
	"github.com/zanuka/baluardo-api/internal/service"
)

func restHandler() http.Handler {
	now := time.Date(2026, 8, 13, 18, 0, 0, 0, time.UTC)
	detections := fakes.NewDetectionRepo(
		domain.Detection{
			ID:         "open-1",
			SiteID:     "site-alpha",
			SensorID:   "sen-a1",
			Status:     domain.DetectionStatusOpen,
			Severity:   domain.DetectionSeverityCritical,
			Confidence: 0.94,
			Summary:    "Anomalous RF burst",
			DetectedAt: now,
			Provenance: domain.Provenance{SensorID: "sen-a1", SensorName: "RF Array North"},
		},
		domain.Detection{
			ID:         "rej-1",
			SiteID:     "site-gamma",
			SensorID:   "sen-g2",
			Status:     domain.DetectionStatusRejected,
			Severity:   domain.DetectionSeverityMedium,
			Confidence: 0.41,
			Summary:    "False positive",
			DetectedAt: now.Add(-2 * time.Hour),
			Provenance: domain.Provenance{SensorID: "sen-g2", SensorName: "Radar Sweep West"},
		},
	)
	sites := fakes.NewSiteRepo(
		domain.Site{ID: "site-alpha", Name: "Sector Alpha", Code: "ALPHA"},
		domain.Site{ID: "site-beta", Name: "Outpost Beta", Code: "BETA"},
	)
	return router.New(router.Deps{
		Health:      service.NewHealth(stubPinger{}),
		Detections:  service.NewDetections(detections, fakes.NewAckRepo()),
		Sites:       service.NewSites(sites),
		CORSOrigins: []string{"http://localhost:5173"},
		Logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
}

func doRequest(t *testing.T, h http.Handler, method, path string, body []byte, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var rdr io.Reader
	if body != nil {
		rdr = bytes.NewReader(body)
	}
	req := httptest.NewRequest(method, path, rdr)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func operatorHeaders() map[string]string {
	return map[string]string{
		"X-Operator-Role": "analyst",
		"X-Operator-Name": "ada",
	}
}

func TestRESTVerticalSlice(t *testing.T) {
	h := restHandler()

	tests := []struct {
		name       string
		method     string
		path       string
		body       []byte
		headers    map[string]string
		wantStatus int
	}{
		{
			name:       "list 401 missing role",
			method:     http.MethodGet,
			path:       "/api/v1/detections",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "list 403 bad role",
			method:     http.MethodGet,
			path:       "/api/v1/detections",
			headers:    map[string]string{"X-Operator-Role": "intern"},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "list 200",
			method:     http.MethodGet,
			path:       "/api/v1/detections",
			headers:    operatorHeaders(),
			wantStatus: http.StatusOK,
		},
		{
			name:       "get 404",
			method:     http.MethodGet,
			path:       "/api/v1/detections/missing",
			headers:    operatorHeaders(),
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "get 200",
			method:     http.MethodGet,
			path:       "/api/v1/detections/open-1",
			headers:    operatorHeaders(),
			wantStatus: http.StatusOK,
		},
		{
			name:       "ack 409 rejected",
			method:     http.MethodPost,
			path:       "/api/v1/detections/rej-1/ack",
			headers:    operatorHeaders(),
			wantStatus: http.StatusConflict,
		},
		{
			name:       "ack 200",
			method:     http.MethodPost,
			path:       "/api/v1/detections/open-1/ack",
			headers:    operatorHeaders(),
			wantStatus: http.StatusOK,
		},
		{
			name:       "reject 400 empty reason",
			method:     http.MethodPost,
			path:       "/api/v1/detections/open-1/reject",
			headers:    operatorHeaders(),
			body:       []byte(`{"reason":""}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "sites 200",
			method:     http.MethodGet,
			path:       "/api/v1/sites",
			headers:    operatorHeaders(),
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := doRequest(t, h, tt.method, tt.path, tt.body, tt.headers)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d body=%s", rec.Code, tt.wantStatus, rec.Body.String())
			}
		})
	}
}

func TestAckIdempotentHTTP(t *testing.T) {
	h := restHandler()
	first := doRequest(t, h, http.MethodPost, "/api/v1/detections/open-1/ack", nil, operatorHeaders())
	if first.Code != http.StatusOK {
		t.Fatalf("first ack status = %d body=%s", first.Code, first.Body.String())
	}
	second := doRequest(t, h, http.MethodPost, "/api/v1/detections/open-1/ack", nil, operatorHeaders())
	if second.Code != http.StatusOK {
		t.Fatalf("second ack status = %d body=%s", second.Code, second.Body.String())
	}
	if first.Body.String() != second.Body.String() {
		t.Fatalf("bodies differ:\n%s\n%s", first.Body.String(), second.Body.String())
	}
	var body struct {
		Status     string  `json:"status"`
		Confidence float64 `json:"confidence"`
		Provenance struct {
			SensorID string `json:"sensorId"`
		} `json:"provenance"`
	}
	if err := json.Unmarshal(first.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Status != "acked" {
		t.Fatalf("status = %s", body.Status)
	}
	if body.Provenance.SensorID == "" {
		t.Fatal("expected provenance.sensorId")
	}
}

func TestListContract(t *testing.T) {
	h := restHandler()
	rec := doRequest(t, h, http.MethodGet, "/api/v1/detections", nil, operatorHeaders())
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		Items      []map[string]any `json:"items"`
		NextCursor *string          `json:"nextCursor"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Items) != 2 {
		t.Fatalf("items = %d, want 2", len(body.Items))
	}
	if body.NextCursor != nil {
		t.Fatalf("nextCursor = %v, want null", body.NextCursor)
	}
	item := body.Items[0]
	for _, key := range []string{"id", "siteId", "sensorId", "status", "severity", "confidence", "summary", "detectedAt", "lastUpdated", "provenance"} {
		if _, ok := item[key]; !ok {
			t.Fatalf("missing %s in detection", key)
		}
	}
}

func TestOpenAPIListsRESTOperations(t *testing.T) {
	h := restHandler()
	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var spec struct {
		Paths map[string]map[string]any `json:"paths"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &spec); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"/api/v1/detections":             "get",
		"/api/v1/detections/{id}":        "get",
		"/api/v1/detections/{id}/ack":    "post",
		"/api/v1/detections/{id}/reject": "post",
		"/api/v1/sites":                  "get",
	}
	for path, method := range want {
		ops, ok := spec.Paths[path]
		if !ok {
			t.Fatalf("missing path %s", path)
		}
		if _, ok := ops[method]; !ok {
			t.Fatalf("missing %s %s", method, path)
		}
	}
}
