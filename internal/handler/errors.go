package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/zanuka/vastion-api/internal/authz"
	"github.com/zanuka/vastion-api/internal/domain"
)

func mapError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, authz.ErrUnauthenticated):
		return huma.NewError(http.StatusUnauthorized, "Operator role header is required.")
	case errors.Is(err, authz.ErrForbidden):
		return huma.NewError(http.StatusForbidden, "Forbidden")
	case errors.Is(err, domain.ErrNotFound):
		return huma.NewError(http.StatusNotFound, "Detection not found.")
	case errors.Is(err, domain.ErrInvalidID):
		return huma.NewError(http.StatusBadRequest, "Invalid id.")
	case errors.Is(err, domain.ErrValidation):
		return huma.NewError(http.StatusBadRequest, problemDetail(err, "Invalid request."))
	case errors.Is(err, domain.ErrConflict):
		return huma.NewError(http.StatusConflict, problemDetail(err, "Conflict."))
	default:
		return huma.NewError(http.StatusInternalServerError, "internal error")
	}
}

func problemDetail(err error, fallback string) string {
	msg := err.Error()
	if i := strings.Index(msg, ": "); i >= 0 {
		msg = msg[i+2:]
	} else {
		msg = fallback
	}
	if msg == "" {
		return fallback
	}
	return strings.ToUpper(msg[:1]) + msg[1:]
}

func requireOperator(role, name string) (authz.Operator, error) {
	op, err := authz.ParseOperator(role, name)
	if err != nil {
		return authz.Operator{}, mapError(err)
	}
	return op, nil
}
