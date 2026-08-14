package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/zanuka/vastion-api/internal/config"
	"github.com/zanuka/vastion-api/internal/domain"
	"github.com/zanuka/vastion-api/internal/repository/mongodb"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", "err", err)
		os.Exit(1)
	}

	client, err := mongodb.Connect(cfg.DatabaseURL)
	if err != nil {
		slog.Error("connect", "err", err)
		os.Exit(1)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = client.Disconnect(ctx)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := client.Ping(ctx); err != nil {
		slog.Error("ping", "err", err)
		os.Exit(1)
	}
	if err := client.EnsureIndexes(ctx); err != nil {
		slog.Error("indexes", "err", err)
		os.Exit(1)
	}
	if err := client.ResetCollections(ctx); err != nil {
		slog.Error("reset", "err", err)
		os.Exit(1)
	}
	if err := seed(ctx, client); err != nil {
		slog.Error("seed", "err", err)
		os.Exit(1)
	}
}

func seed(ctx context.Context, client *mongodb.Client) error {
	sites := mongodb.NewSiteRepository(client)
	sensors := mongodb.NewSensorRepository(client)
	detections := mongodb.NewDetectionRepository(client)
	acks := mongodb.NewAcknowledgementRepository(client)

	now := time.Now().UTC()
	ago := func(minutes int) time.Time {
		return now.Add(-time.Duration(minutes) * time.Minute)
	}

	alpha := &domain.Site{Name: "Sector Alpha", Code: "ALPHA", Status: domain.SiteStatusActive}
	beta := &domain.Site{Name: "Outpost Beta", Code: "BETA", Status: domain.SiteStatusActive}
	gamma := &domain.Site{Name: "Relay Gamma", Code: "GAMMA", Status: domain.SiteStatusActive}
	for _, site := range []*domain.Site{alpha, beta, gamma} {
		if err := sites.Insert(ctx, site); err != nil {
			return err
		}
	}

	rf := &domain.Sensor{SiteID: alpha.ID, Name: "RF Array North", Code: "RF-N"}
	aq := &domain.Sensor{SiteID: alpha.ID, Name: "Air Quality Node 3", Code: "AQ-3"}
	thermal := &domain.Sensor{SiteID: beta.ID, Name: "Thermal Cam East", Code: "TH-E"}
	acoustic := &domain.Sensor{SiteID: beta.ID, Name: "Acoustic Pod 7", Code: "AC-7"}
	relay := &domain.Sensor{SiteID: gamma.ID, Name: "Relay Monitor", Code: "NET-1"}
	radar := &domain.Sensor{SiteID: gamma.ID, Name: "Radar Sweep West", Code: "RD-W"}
	for _, sensor := range []*domain.Sensor{rf, aq, thermal, acoustic, relay, radar} {
		if err := sensors.Insert(ctx, sensor); err != nil {
			return err
		}
	}

	openCritical := &domain.Detection{
		SiteID:     alpha.ID,
		SensorID:   rf.ID,
		Status:     domain.DetectionStatusOpen,
		Severity:   domain.DetectionSeverityCritical,
		Summary:    "Anomalous RF burst near perimeter fence",
		DetectedAt: ago(4),
	}
	openHigh := &domain.Detection{
		SiteID:     alpha.ID,
		SensorID:   aq.ID,
		Status:     domain.DetectionStatusOpen,
		Severity:   domain.DetectionSeverityHigh,
		Summary:    "Elevated particulate reading — sector 3",
		DetectedAt: ago(18),
	}
	openMedium := &domain.Detection{
		SiteID:     beta.ID,
		SensorID:   thermal.ID,
		Status:     domain.DetectionStatusOpen,
		Severity:   domain.DetectionSeverityMedium,
		Summary:    "Motion cluster detected in restricted zone",
		DetectedAt: ago(35),
	}
	ackedLow := &domain.Detection{
		SiteID:     beta.ID,
		SensorID:   acoustic.ID,
		Status:     domain.DetectionStatusAcked,
		Severity:   domain.DetectionSeverityLow,
		Summary:    "Routine acoustic signature — logged",
		DetectedAt: ago(90),
	}
	openHighGamma := &domain.Detection{
		SiteID:     gamma.ID,
		SensorID:   relay.ID,
		Status:     domain.DetectionStatusOpen,
		Severity:   domain.DetectionSeverityHigh,
		Summary:    "Link degradation on relay uplink",
		DetectedAt: ago(8),
	}
	rejectedMedium := &domain.Detection{
		SiteID:     gamma.ID,
		SensorID:   radar.ID,
		Status:     domain.DetectionStatusRejected,
		Severity:   domain.DetectionSeverityMedium,
		Summary:    "False positive — weather interference",
		DetectedAt: ago(120),
	}

	queue := []*domain.Detection{openCritical, openHigh, openMedium, ackedLow, openHighGamma, rejectedMedium}
	for _, detection := range queue {
		if err := detections.Insert(ctx, detection); err != nil {
			return err
		}
	}

	if err := acks.Insert(ctx, &domain.Acknowledgement{
		DetectionID: ackedLow.ID,
		Action:      domain.AckActionAck,
		Operator:    "ada",
		FromStatus:  domain.DetectionStatusOpen,
		ToStatus:    domain.DetectionStatusAcked,
		CreatedAt:   ago(45),
	}); err != nil {
		return err
	}
	if err := acks.Insert(ctx, &domain.Acknowledgement{
		DetectionID: rejectedMedium.ID,
		Action:      domain.AckActionReject,
		Reason:      "weather interference on radar sweep",
		Operator:    "ada",
		FromStatus:  domain.DetectionStatusOpen,
		ToStatus:    domain.DetectionStatusRejected,
		CreatedAt:   ago(60),
	}); err != nil {
		return err
	}

	slog.Info("seed complete",
		"sites", 3,
		"sensors", 6,
		"detections", len(queue),
		"acknowledgements", 2,
	)
	return nil
}
