package service

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/zanuka/baluardo-api/internal/domain"
)

func encodeCursor(detectedAt time.Time, id string) string {
	raw := strconv.FormatInt(detectedAt.UTC().UnixMilli(), 10) + "|" + id
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func decodeCursor(raw string) (domain.KeysetCursor, error) {
	b, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return domain.KeysetCursor{}, fmt.Errorf("%w: invalid cursor", domain.ErrValidation)
	}
	parts := strings.SplitN(string(b), "|", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return domain.KeysetCursor{}, fmt.Errorf("%w: invalid cursor", domain.ErrValidation)
	}
	ms, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return domain.KeysetCursor{}, fmt.Errorf("%w: invalid cursor", domain.ErrValidation)
	}
	return domain.KeysetCursor{
		DetectedAt: time.UnixMilli(ms).UTC(),
		ID:         parts[1],
	}, nil
}
