package service

import (
	"errors"
	"testing"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
)

func TestPostgresErrorClassifierClassify(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want pgErrorClassification
	}{
		{name: "nil is non retriable", err: nil, want: nonRetriable},
		{name: "connection failure is retriable", err: &pgconn.PgError{Code: pgerrcode.ConnectionFailure}, want: retriable},
		{name: "unique violation is non retriable", err: &pgconn.PgError{Code: pgerrcode.UniqueViolation}, want: nonRetriable},
		{name: "generic error is non retriable", err: errors.New("boom"), want: nonRetriable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			classifier := postgresErrorClassifier{}
			assert.Equal(t, tt.want, classifier.classify(tt.err))
		})
	}
}
