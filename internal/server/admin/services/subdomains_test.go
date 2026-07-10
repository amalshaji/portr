package services

import (
	"errors"
	"testing"
)

func TestIsConstraintErrorMatchesOnlyRequestedConstraint(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		constraint string
		want       bool
	}{
		{
			name:       "postgres reservation constraint",
			err:        errors.New(`duplicate key value violates unique constraint "idx_subdomain_reservation_name_unique"`),
			constraint: reservationSubdomainIndex,
			want:       true,
		},
		{
			name:       "sqlite connection constraint",
			err:        errors.New(`UNIQUE constraint failed: index 'idx_connection_active_subdomain_unique'`),
			constraint: activeConnectionSubdomainIndex,
			want:       true,
		},
		{
			name:       "unrelated connection primary key",
			err:        errors.New(`duplicate key value violates unique constraint "connection_pkey"`),
			constraint: activeConnectionSubdomainIndex,
			want:       false,
		},
		{
			name:       "unrelated sqlite unique constraint",
			err:        errors.New("UNIQUE constraint failed: connection.id"),
			constraint: activeConnectionSubdomainIndex,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isConstraintError(tt.err, tt.constraint); got != tt.want {
				t.Fatalf("isConstraintError() = %t, want %t", got, tt.want)
			}
		})
	}
}
