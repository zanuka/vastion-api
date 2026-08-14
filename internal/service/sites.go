package service

import (
	"context"

	"github.com/zanuka/vastion-api/internal/domain"
)

type Sites struct {
	sites domain.SiteRepository
}

func NewSites(sites domain.SiteRepository) *Sites {
	return &Sites{sites: sites}
}

func (s *Sites) List(ctx context.Context) ([]domain.Site, error) {
	return s.sites.List(ctx)
}
