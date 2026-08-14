package authz

import "testing"

func TestParseOperator(t *testing.T) {
	tests := []struct {
		role    string
		name    string
		wantErr error
		want    Role
	}{
		{role: "", wantErr: ErrUnauthenticated},
		{role: "intern", wantErr: ErrForbidden},
		{role: "analyst", name: "ada", want: RoleAnalyst},
		{role: "SUPERVISOR", name: "bee", want: RoleSupervisor},
		{role: " analyst ", want: RoleAnalyst},
	}
	for _, tt := range tests {
		got, err := ParseOperator(tt.role, tt.name)
		if tt.wantErr != nil {
			if err != tt.wantErr {
				t.Fatalf("role=%q err=%v want %v", tt.role, err, tt.wantErr)
			}
			continue
		}
		if err != nil {
			t.Fatalf("role=%q err=%v", tt.role, err)
		}
		if got.Role != tt.want {
			t.Fatalf("role=%q got %s want %s", tt.role, got.Role, tt.want)
		}
	}
}
