package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zanuka/vastion-api/internal/router"
	"github.com/zanuka/vastion-api/internal/service"
)

type stubPinger struct {
	err error
}

func (s stubPinger) Ping(context.Context) error {
	return s.err
}

func TestHealth(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		pingErr    error
		wantStatus int
		wantOK     bool
	}{
		{
			name:       "healthz ok",
			path:       "/healthz",
			pingErr:    nil,
			wantStatus: http.StatusOK,
			wantOK:     true,
		},
		{
			name:       "healthz ignores mongo",
			path:       "/healthz",
			pingErr:    errors.New("down"),
			wantStatus: http.StatusOK,
			wantOK:     true,
		},
		{
			name:       "readyz ok",
			path:       "/readyz",
			pingErr:    nil,
			wantStatus: http.StatusOK,
			wantOK:     true,
		},
		{
			name:       "readyz unavailable",
			path:       "/readyz",
			pingErr:    errors.New("down"),
			wantStatus: http.StatusServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := router.New(router.Deps{
				Health:      service.NewHealth(stubPinger{err: tt.pingErr}),
				CORSOrigins: []string{"http://localhost:5173"},
				Logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
			})

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d body=%s", rec.Code, tt.wantStatus, rec.Body.String())
			}

			if !tt.wantOK {
				return
			}

			var body struct {
				Status string `json:"status"`
			}
			if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if body.Status != "ok" {
				t.Fatalf("status field = %q, want ok", body.Status)
			}
		})
	}
}

func TestOpenAPI(t *testing.T) {
	h := router.New(router.Deps{
		Health:      service.NewHealth(stubPinger{}),
		CORSOrigins: []string{"http://localhost:5173"},
		Logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
	})

	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
}
