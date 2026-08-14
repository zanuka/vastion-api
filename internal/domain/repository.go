package domain

import "context"

type SiteRepository interface {
	Insert(ctx context.Context, site *Site) error
	GetByID(ctx context.Context, id string) (*Site, error)
	GetByCode(ctx context.Context, code string) (*Site, error)
	List(ctx context.Context) ([]Site, error)
}

type SensorRepository interface {
	Insert(ctx context.Context, sensor *Sensor) error
	GetByID(ctx context.Context, id string) (*Sensor, error)
	ListBySiteID(ctx context.Context, siteID string) ([]Sensor, error)
}

type DetectionRepository interface {
	Insert(ctx context.Context, detection *Detection) error
	GetByID(ctx context.Context, id string) (*Detection, error)
	List(ctx context.Context, filter DetectionListFilter) ([]Detection, error)
}

type AcknowledgementRepository interface {
	Insert(ctx context.Context, ack *Acknowledgement) error
	GetByID(ctx context.Context, id string) (*Acknowledgement, error)
	ListByDetectionID(ctx context.Context, detectionID string) ([]Acknowledgement, error)
}
