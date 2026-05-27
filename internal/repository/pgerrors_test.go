package repository

import (
	"errors"
	"testing"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
)

func TestIsRetriablePostgresError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil is non retriable", err: nil, want: false},
		{name: "connection failure is retriable", err: &pgconn.PgError{Code: pgerrcode.ConnectionFailure}, want: true},
		{name: "unique violation is non retriable", err: &pgconn.PgError{Code: pgerrcode.UniqueViolation}, want: false},
		{name: "generic error is non retriable", err: errors.New("boom"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsRetriablePostgresError(tt.err))
		})
	}
}
