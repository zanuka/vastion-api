package service

import (
	"context"
	"time"
)

type Pinger interface {
	Ping(ctx context.Context) error
}

type Health struct {
	pinger Pinger
}

func NewHealth(pinger Pinger) *Health {
	return &Health{pinger: pinger}
}

func (h *Health) Ready(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return h.pinger.Ping(ctx)
}
