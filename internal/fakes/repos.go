package fakes

import (
	"context"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/zanuka/baluardo-api/internal/domain"
)

var (
	_ domain.DetectionRepository       = (*DetectionRepo)(nil)
	_ domain.AcknowledgementRepository = (*AckRepo)(nil)
	_ domain.SiteRepository            = (*SiteRepo)(nil)
)

type DetectionRepo struct {
	mu      sync.Mutex
	items   map[string]domain.Detection
	counter int
}

func NewDetectionRepo(items ...domain.Detection) *DetectionRepo {
	r := &DetectionRepo{items: make(map[string]domain.Detection)}
	for _, item := range items {
		if item.ID == "" {
			r.counter++
			item.ID = fakeID("det", r.counter)
		}
		if item.LastUpdated.IsZero() {
			item.LastUpdated = item.DetectedAt
		}
		if item.Provenance.SensorID == "" {
			item.Provenance.SensorID = item.SensorID
		}
		r.items[item.ID] = item
	}
	return r
}

func (r *DetectionRepo) Insert(_ context.Context, detection *domain.Detection) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if detection.ID == "" {
		r.counter++
		detection.ID = fakeID("det", r.counter)
	}
	if detection.Status == "" {
		detection.Status = domain.DetectionStatusOpen
	}
	if detection.LastUpdated.IsZero() {
		detection.LastUpdated = detection.DetectedAt
	}
	if detection.Provenance.SensorID == "" {
		detection.Provenance.SensorID = detection.SensorID
	}
	cloned := *detection
	r.items[detection.ID] = cloned
	return nil
}

func (r *DetectionRepo) GetByID(_ context.Context, id string) (*domain.Detection, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.items[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cloned := item
	return &cloned, nil
}

func (r *DetectionRepo) List(_ context.Context, filter domain.DetectionListFilter) ([]domain.Detection, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]domain.Detection, 0, len(r.items))
	for _, item := range r.items {
		if filter.SiteID != "" && item.SiteID != filter.SiteID {
			continue
		}
		if filter.SensorID != "" && item.SensorID != filter.SensorID {
			continue
		}
		if filter.Status != "" && item.Status != filter.Status {
			continue
		}
		if filter.Severity != "" && item.Severity != filter.Severity {
			continue
		}
		if filter.Cursor != nil {
			c := filter.Cursor
			if item.DetectedAt.After(c.DetectedAt) {
				continue
			}
			if item.DetectedAt.Equal(c.DetectedAt) && item.ID >= c.ID {
				continue
			}
		}
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].DetectedAt.Equal(out[j].DetectedAt) {
			return out[i].DetectedAt.After(out[j].DetectedAt)
		}
		return out[i].ID > out[j].ID
	})
	if filter.Limit > 0 && len(out) > filter.Limit+1 {
		out = out[:filter.Limit+1]
	}
	return cloneDetections(out), nil
}

func (r *DetectionRepo) UpdateStatus(_ context.Context, id string, from, to domain.DetectionStatus, at time.Time) (*domain.Detection, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.items[id]
	if !ok || item.Status != from {
		return nil, domain.ErrNotFound
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}
	item.Status = to
	item.LastUpdated = at.UTC()
	r.items[id] = item
	cloned := item
	return &cloned, nil
}

type AckRepo struct {
	mu      sync.Mutex
	items   []domain.Acknowledgement
	counter int
}

func NewAckRepo() *AckRepo {
	return &AckRepo{}
}

func (r *AckRepo) Insert(_ context.Context, ack *domain.Acknowledgement) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if ack.ID == "" {
		r.counter++
		ack.ID = fakeID("ack", r.counter)
	}
	if ack.CreatedAt.IsZero() {
		ack.CreatedAt = time.Now().UTC()
	}
	r.items = append(r.items, *ack)
	return nil
}

func (r *AckRepo) GetByID(_ context.Context, id string) (*domain.Acknowledgement, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.items {
		if r.items[i].ID == id {
			cloned := r.items[i]
			return &cloned, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *AckRepo) ListByDetectionID(_ context.Context, detectionID string) ([]domain.Acknowledgement, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]domain.Acknowledgement, 0)
	for _, item := range r.items {
		if item.DetectionID == detectionID {
			out = append(out, item)
		}
	}
	return out, nil
}

type SiteRepo struct {
	mu    sync.Mutex
	items []domain.Site
}

func NewSiteRepo(items ...domain.Site) *SiteRepo {
	cloned := make([]domain.Site, len(items))
	copy(cloned, items)
	return &SiteRepo{items: cloned}
}

func (r *SiteRepo) Insert(_ context.Context, site *domain.Site) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items = append(r.items, *site)
	return nil
}

func (r *SiteRepo) GetByID(_ context.Context, id string) (*domain.Site, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.items {
		if r.items[i].ID == id {
			cloned := r.items[i]
			return &cloned, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *SiteRepo) GetByCode(_ context.Context, code string) (*domain.Site, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.items {
		if r.items[i].Code == code {
			cloned := r.items[i]
			return &cloned, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *SiteRepo) List(context.Context) ([]domain.Site, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]domain.Site, len(r.items))
	copy(out, r.items)
	return out, nil
}

func cloneDetections(in []domain.Detection) []domain.Detection {
	out := make([]domain.Detection, len(in))
	copy(out, in)
	return out
}

func fakeID(prefix string, n int) string {
	return prefix + "-" + strconv.Itoa(n)
}
