package authz

const (
	HeaderRole = "X-Operator-Role"
	HeaderName = "X-Operator-Name"
)

type Role string

const (
	RoleAnalyst    Role = "analyst"
	RoleSupervisor Role = "supervisor"
)
