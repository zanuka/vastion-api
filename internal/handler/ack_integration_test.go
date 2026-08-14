//go:build integration

package handler_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/zanuka/vastion-api/internal/domain"
	"github.com/zanuka/vastion-api/internal/repository/mongodb"
	"github.com/zanuka/vastion-api/internal/router"
	"github.com/zanuka/vastion-api/internal/service"
)

func TestAckAgainstMongo(t *testing.T) {
	uri := os.Getenv("DATABASE_URL")
	if uri == "" {
		t.Skip("DATABASE_URL not set")
	}

	client, err := mongodb.Connect(uri)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	t.Cleanup(func() {
		cctx, ccancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer ccancel()
		_ = client.Disconnect(cctx)
	})
	if err := client.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	if err := client.EnsureIndexes(ctx); err != nil {
		t.Fatal(err)
	}

	sites := mongodb.NewSiteRepository(client)
	sensors := mongodb.NewSensorRepository(client)
	detections := mongodb.NewDetectionRepository(client)
	acks := mongodb.NewAcknowledgementRepository(client)

	site := &domain.Site{Name: "Integration Site", Code: "INT-P2-" + time.Now().UTC().Format("150405.000"), Status: domain.SiteStatusActive}
	if err := sites.Insert(ctx, site); err != nil {
		t.Fatal(err)
	}
	sensor := &domain.Sensor{SiteID: site.ID, Name: "Integration Sensor", Code: "INT-S"}
	if err := sensors.Insert(ctx, sensor); err != nil {
		t.Fatal(err)
	}
	open := &domain.Detection{
		SiteID:      site.ID,
		SensorID:    sensor.ID,
		Status:      domain.DetectionStatusOpen,
		Severity:    domain.DetectionSeverityHigh,
		Confidence:  0.77,
		Summary:     "integration open",
		DetectedAt:  time.Now().UTC(),
		LastUpdated: time.Now().UTC(),
		Provenance:  domain.Provenance{SensorID: sensor.ID, SensorName: sensor.Name},
	}
	if err := detections.Insert(ctx, open); err != nil {
		t.Fatal(err)
	}
	rejected := &domain.Detection{
		SiteID:      site.ID,
		SensorID:    sensor.ID,
		Status:      domain.DetectionStatusRejected,
		Severity:    domain.DetectionSeverityLow,
		Confidence:  0.2,
		Summary:     "integration rejected",
		DetectedAt:  time.Now().UTC().Add(-time.Minute),
		LastUpdated: time.Now().UTC().Add(-time.Minute),
		Provenance:  domain.Provenance{SensorID: sensor.ID, SensorName: sensor.Name},
	}
	if err := detections.Insert(ctx, rejected); err != nil {
		t.Fatal(err)
	}

	h := router.New(router.Deps{
		Health:      service.NewHealth(client),
		Detections:  service.NewDetections(detections, acks),
		Sites:       service.NewSites(sites),
		CORSOrigins: []string{"http://localhost:5173"},
		Logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
	})

	first := doRequest(t, h, http.MethodPost, "/api/v1/detections/"+open.ID+"/ack", nil, operatorHeaders())
	if first.Code != http.StatusOK {
		t.Fatalf("first ack = %d body=%s", first.Code, first.Body.String())
	}
	second := doRequest(t, h, http.MethodPost, "/api/v1/detections/"+open.ID+"/ack", nil, operatorHeaders())
	if second.Code != http.StatusOK {
		t.Fatalf("second ack = %d body=%s", second.Code, second.Body.String())
	}
	if first.Body.String() != second.Body.String() {
		t.Fatalf("idempotent bodies differ")
	}
	var body struct {
		Status string `json:"status"`
		ID     string `json:"id"`
	}
	if err := json.Unmarshal(first.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Status != "acked" || body.ID != open.ID {
		t.Fatalf("body = %+v", body)
	}

	conflict := doRequest(t, h, http.MethodPost, "/api/v1/detections/"+rejected.ID+"/ack", nil, operatorHeaders())
	if conflict.Code != http.StatusConflict {
		t.Fatalf("ack rejected = %d body=%s", conflict.Code, conflict.Body.String())
	}
}
