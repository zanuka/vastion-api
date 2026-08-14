package handler

import (
	"fmt"
	"time"

	"github.com/zanuka/baluardo-api/internal/domain"
)

type ProvenanceBody struct {
	SensorID     string `json:"sensorId"`
	SensorName   string `json:"sensorName,omitempty"`
	Model        string `json:"model,omitempty"`
	ModelVersion string `json:"modelVersion,omitempty"`
}

type DetectionBody struct {
	ID          string         `json:"id"`
	SiteID      string         `json:"siteId"`
	SensorID    string         `json:"sensorId"`
	Status      string         `json:"status"`
	Severity    string         `json:"severity"`
	Confidence  float64        `json:"confidence"`
	Summary     string         `json:"summary"`
	DetectedAt  time.Time      `json:"detectedAt"`
	LastUpdated time.Time      `json:"lastUpdated"`
	Provenance  ProvenanceBody `json:"provenance"`
}

type SiteBody struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Code string `json:"code"`
}

func detectionBody(d domain.Detection) DetectionBody {
	last := d.LastUpdated
	if last.IsZero() {
		last = d.DetectedAt
	}
	prov := d.Provenance
	if prov.SensorID == "" {
		prov.SensorID = d.SensorID
	}
	return DetectionBody{
		ID:          d.ID,
		SiteID:      d.SiteID,
		SensorID:    d.SensorID,
		Status:      string(d.Status),
		Severity:    string(d.Severity),
		Confidence:  d.Confidence,
		Summary:     d.Summary,
		DetectedAt:  d.DetectedAt.UTC(),
		LastUpdated: last.UTC(),
		Provenance: ProvenanceBody{
			SensorID:     prov.SensorID,
			SensorName:   prov.SensorName,
			Model:        prov.Model,
			ModelVersion: prov.ModelVersion,
		},
	}
}

func siteBody(s domain.Site) SiteBody {
	return SiteBody{ID: s.ID, Name: s.Name, Code: s.Code}
}

func parseStatus(s string) (domain.DetectionStatus, error) {
	if s == "" {
		return "", nil
	}
	v := domain.DetectionStatus(s)
	switch v {
	case domain.DetectionStatusOpen, domain.DetectionStatusAcked, domain.DetectionStatusRejected:
		return v, nil
	default:
		return "", fmt.Errorf("%w: invalid status", domain.ErrValidation)
	}
}

func parseSeverity(s string) (domain.DetectionSeverity, error) {
	if s == "" {
		return "", nil
	}
	v := domain.DetectionSeverity(s)
	switch v {
	case domain.DetectionSeverityCritical, domain.DetectionSeverityHigh, domain.DetectionSeverityMedium, domain.DetectionSeverityLow:
		return v, nil
	default:
		return "", fmt.Errorf("%w: invalid severity", domain.ErrValidation)
	}
}
