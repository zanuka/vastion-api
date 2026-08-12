package domain

import "context"

type SiteRepository interface {
	GetByID(ctx context.Context, id string) (*Site, error)
}

type SensorRepository interface {
	GetByID(ctx context.Context, id string) (*Sensor, error)
}

type DetectionRepository interface {
	GetByID(ctx context.Context, id string) (*Detection, error)
}

type AcknowledgementRepository interface {
	GetByID(ctx context.Context, id string) (*Acknowledgement, error)
}
