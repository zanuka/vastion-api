package domain

type DetectionStatus string

const (
	DetectionStatusOpen     DetectionStatus = "open"
	DetectionStatusAcked    DetectionStatus = "acked"
	DetectionStatusRejected DetectionStatus = "rejected"
)

type Detection struct {
	ID       string
	SiteID   string
	SensorID string
	Status   DetectionStatus
}
