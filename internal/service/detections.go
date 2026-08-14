package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/zanuka/vastion-api/internal/domain"
)

const defaultListLimit = 50

type DetectionPage struct {
	Items      []domain.Detection
	NextCursor string
}

type Detections struct {
	detections domain.DetectionRepository
	acks       domain.AcknowledgementRepository
}

func NewDetections(detections domain.DetectionRepository, acks domain.AcknowledgementRepository) *Detections {
	return &Detections{detections: detections, acks: acks}
}

func (s *Detections) List(ctx context.Context, filter domain.DetectionListFilter, rawCursor string) (DetectionPage, error) {
	if rawCursor != "" {
		cursor, err := decodeCursor(rawCursor)
		if err != nil {
			return DetectionPage{}, err
		}
		filter.Cursor = &cursor
	}
	if filter.Limit <= 0 {
		filter.Limit = defaultListLimit
	}
	items, err := s.detections.List(ctx, filter)
	if err != nil {
		return DetectionPage{}, err
	}
	next := ""
	if len(items) > filter.Limit {
		items = items[:filter.Limit]
		last := items[len(items)-1]
		next = encodeCursor(last.DetectedAt, last.ID)
	}
	return DetectionPage{Items: items, NextCursor: next}, nil
}

func (s *Detections) Get(ctx context.Context, id string) (*domain.Detection, error) {
	return s.detections.GetByID(ctx, id)
}

func (s *Detections) Ack(ctx context.Context, id, operator string) (*domain.Detection, error) {
	return s.transition(ctx, id, operator, "", domain.AckActionAck, domain.DetectionStatusAcked)
}

func (s *Detections) Reject(ctx context.Context, id, reason, operator string) (*domain.Detection, error) {
	if strings.TrimSpace(reason) == "" {
		return nil, fmt.Errorf("%w: reject reason is required", domain.ErrValidation)
	}
	return s.transition(ctx, id, operator, reason, domain.AckActionReject, domain.DetectionStatusRejected)
}

func (s *Detections) transition(ctx context.Context, id, operator, reason string, action domain.AckAction, to domain.DetectionStatus) (*domain.Detection, error) {
	current, err := s.detections.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if current.Status == to {
		return current, nil
	}
	if current.Status != domain.DetectionStatusOpen {
		return nil, illegalTransition(to)
	}
	now := time.Now().UTC()
	updated, err := s.detections.UpdateStatus(ctx, id, domain.DetectionStatusOpen, to, now)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return s.replayOrConflict(ctx, id, to)
		}
		return nil, err
	}
	_ = s.acks.Insert(ctx, &domain.Acknowledgement{
		DetectionID: id,
		Action:      action,
		Reason:      strings.TrimSpace(reason),
		Operator:    operator,
		FromStatus:  domain.DetectionStatusOpen,
		ToStatus:    to,
		CreatedAt:   now,
	})
	return updated, nil
}

func (s *Detections) replayOrConflict(ctx context.Context, id string, to domain.DetectionStatus) (*domain.Detection, error) {
	current, err := s.detections.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if current.Status == to {
		return current, nil
	}
	return nil, illegalTransition(to)
}

func illegalTransition(to domain.DetectionStatus) error {
	if to == domain.DetectionStatusAcked {
		return fmt.Errorf("%w: cannot acknowledge a rejected detection", domain.ErrConflict)
	}
	return fmt.Errorf("%w: cannot reject an acknowledged detection", domain.ErrConflict)
}
