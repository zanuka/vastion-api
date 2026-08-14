package authz

import (
	"errors"
	"strings"
)

const (
	HeaderRole = "X-Operator-Role"
	HeaderName = "X-Operator-Name"
)

type Role string

const (
	RoleAnalyst    Role = "analyst"
	RoleSupervisor Role = "supervisor"
)

var (
	ErrUnauthenticated = errors.New("unauthenticated")
	ErrForbidden       = errors.New("forbidden")
)

type Operator struct {
	Role Role
	Name string
}

func ParseOperator(role, name string) (Operator, error) {
	role = strings.ToLower(strings.TrimSpace(role))
	name = strings.TrimSpace(name)
	if role == "" {
		return Operator{}, ErrUnauthenticated
	}
	switch Role(role) {
	case RoleAnalyst, RoleSupervisor:
	default:
		return Operator{}, ErrForbidden
	}
	if name == "" {
		name = "unknown"
	}
	return Operator{Role: Role(role), Name: name}, nil
}
