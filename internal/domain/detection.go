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

type Detection struct {
	ID         string
	SiteID     string
	SensorID   string
	Status     DetectionStatus
	Severity   DetectionSeverity
	Summary    string
	DetectedAt time.Time
}

type DetectionListFilter struct {
	SiteID   string
	SensorID string
	Status   DetectionStatus
	Severity DetectionSeverity
}
