package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zanuka/baluardo-api/internal/domain"
	"github.com/zanuka/baluardo-api/internal/fakes"
)

func TestAckIdempotentAndConflict(t *testing.T) {
	now := time.Date(2026, 8, 13, 18, 0, 0, 0, time.UTC)
	open := domain.Detection{
		ID:         "open-1",
		SiteID:     "site-1",
		SensorID:   "sen-1",
		Status:     domain.DetectionStatusOpen,
		Severity:   domain.DetectionSeverityHigh,
		Confidence: 0.8,
		Summary:    "open",
		DetectedAt: now,
	}
	rejected := domain.Detection{
		ID:         "rej-1",
		SiteID:     "site-1",
		SensorID:   "sen-1",
		Status:     domain.DetectionStatusRejected,
		Severity:   domain.DetectionSeverityLow,
		Confidence: 0.2,
		Summary:    "rejected",
		DetectedAt: now.Add(-time.Hour),
	}

	tests := []struct {
		name       string
		id         string
		wantStatus domain.DetectionStatus
		wantErr    error
	}{
		{name: "ack open", id: "open-1", wantStatus: domain.DetectionStatusAcked},
		{name: "ack rejected", id: "rej-1", wantErr: domain.ErrConflict},
		{name: "missing", id: "nope", wantErr: domain.ErrNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewDetections(fakes.NewDetectionRepo(open, rejected), fakes.NewAckRepo())
			got, err := svc.Ack(context.Background(), tt.id, "ada")
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ack: %v", err)
			}
			if got.Status != tt.wantStatus {
				t.Fatalf("status = %s, want %s", got.Status, tt.wantStatus)
			}
			again, err := svc.Ack(context.Background(), tt.id, "ada")
			if err != nil {
				t.Fatalf("second ack: %v", err)
			}
			if again.Status != tt.wantStatus {
				t.Fatalf("second status = %s, want %s", again.Status, tt.wantStatus)
			}
			if again.ID != got.ID || again.LastUpdated.UnixMilli() != got.LastUpdated.UnixMilli() {
				t.Fatalf("idempotent body mismatch: first=%+v second=%+v", got, again)
			}
		})
	}
}

func TestReject(t *testing.T) {
	now := time.Date(2026, 8, 13, 18, 0, 0, 0, time.UTC)
	open := domain.Detection{
		ID:         "open-1",
		SiteID:     "site-1",
		SensorID:   "sen-1",
		Status:     domain.DetectionStatusOpen,
		Severity:   domain.DetectionSeverityMedium,
		Confidence: 0.5,
		Summary:    "open",
		DetectedAt: now,
	}
	acked := domain.Detection{
		ID:         "ack-1",
		SiteID:     "site-1",
		SensorID:   "sen-1",
		Status:     domain.DetectionStatusAcked,
		Severity:   domain.DetectionSeverityLow,
		Confidence: 0.4,
		Summary:    "acked",
		DetectedAt: now.Add(-time.Minute),
	}

	t.Run("reason required", func(t *testing.T) {
		svc := NewDetections(fakes.NewDetectionRepo(open), fakes.NewAckRepo())
		_, err := svc.Reject(context.Background(), "open-1", "  ", "ada")
		if !errors.Is(err, domain.ErrValidation) {
			t.Fatalf("err = %v, want validation", err)
		}
	})

	t.Run("reject open then idempotent", func(t *testing.T) {
		svc := NewDetections(fakes.NewDetectionRepo(open), fakes.NewAckRepo())
		got, err := svc.Reject(context.Background(), "open-1", "weather", "ada")
		if err != nil {
			t.Fatal(err)
		}
		if got.Status != domain.DetectionStatusRejected {
			t.Fatalf("status = %s", got.Status)
		}
		again, err := svc.Reject(context.Background(), "open-1", "weather", "ada")
		if err != nil {
			t.Fatal(err)
		}
		if again.Status != domain.DetectionStatusRejected {
			t.Fatalf("second status = %s", again.Status)
		}
	})

	t.Run("reject acked", func(t *testing.T) {
		svc := NewDetections(fakes.NewDetectionRepo(acked), fakes.NewAckRepo())
		_, err := svc.Reject(context.Background(), "ack-1", "nope", "ada")
		if !errors.Is(err, domain.ErrConflict) {
			t.Fatalf("err = %v, want conflict", err)
		}
	})
}

func TestListKeyset(t *testing.T) {
	now := time.Date(2026, 8, 13, 18, 0, 0, 0, time.UTC)
	items := []domain.Detection{
		{ID: "c", SiteID: "s1", SensorID: "n", Status: domain.DetectionStatusOpen, Severity: domain.DetectionSeverityHigh, DetectedAt: now},
		{ID: "b", SiteID: "s1", SensorID: "n", Status: domain.DetectionStatusOpen, Severity: domain.DetectionSeverityLow, DetectedAt: now.Add(-time.Minute)},
		{ID: "a", SiteID: "s2", SensorID: "n", Status: domain.DetectionStatusAcked, Severity: domain.DetectionSeverityHigh, DetectedAt: now.Add(-2 * time.Minute)},
	}
	svc := NewDetections(fakes.NewDetectionRepo(items...), fakes.NewAckRepo())

	page, err := svc.List(context.Background(), domain.DetectionListFilter{Limit: 2}, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 2 {
		t.Fatalf("len = %d, want 2", len(page.Items))
	}
	if page.Items[0].ID != "c" || page.Items[1].ID != "b" {
		t.Fatalf("order = %s,%s", page.Items[0].ID, page.Items[1].ID)
	}
	if page.NextCursor == "" {
		t.Fatal("expected next cursor")
	}

	page2, err := svc.List(context.Background(), domain.DetectionListFilter{Limit: 2}, page.NextCursor)
	if err != nil {
		t.Fatal(err)
	}
	if len(page2.Items) != 1 || page2.Items[0].ID != "a" {
		t.Fatalf("page2 = %+v", page2.Items)
	}
	if page2.NextCursor != "" {
		t.Fatalf("unexpected next cursor %q", page2.NextCursor)
	}

	filtered, err := svc.List(context.Background(), domain.DetectionListFilter{SiteID: "s1", Status: domain.DetectionStatusOpen}, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered.Items) != 2 {
		t.Fatalf("filtered len = %d, want 2", len(filtered.Items))
	}
}

func TestDecodeCursor(t *testing.T) {
	_, err := decodeCursor("not-a-cursor")
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("err = %v, want validation", err)
	}
	raw := encodeCursor(time.UnixMilli(1_700_000_000_000).UTC(), "abc")
	got, err := decodeCursor(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "abc" || got.DetectedAt.UnixMilli() != 1_700_000_000_000 {
		t.Fatalf("got %+v", got)
	}
}
