package domain

import "time"

type DetectionStatus string

const (
	DetectionStatusOpen     DetectionStatus = "open"
	DetectionStatusAcked    DetectionStatus = "acked"
	DetectionStatusRejected DetectionStatus = "rejected"
)

type DetectionSeverity string

const (
	DetectionSeverityCritical DetectionSeverity = "critical"
	DetectionSeverityHigh     DetectionSeverity = "high"
	DetectionSeverityMedium   DetectionSeverity = "medium"
	DetectionSeverityLow      DetectionSeverity = "low"
)

type Provenance struct {
	SensorID     string
	SensorName   string
	Model        string
	ModelVersion string
}

type Detection struct {
	ID          string
	SiteID      string
	SensorID    string
	Status      DetectionStatus
	Severity    DetectionSeverity
	Confidence  float64
	Summary     string
	DetectedAt  time.Time
	LastUpdated time.Time
	Provenance  Provenance
}

type KeysetCursor struct {
	DetectedAt time.Time
	ID         string
}

type DetectionListFilter struct {
	SiteID   string
	SensorID string
	Status   DetectionStatus
	Severity DetectionSeverity
	Limit    int
	Cursor   *KeysetCursor
}
