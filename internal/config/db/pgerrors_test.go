package config

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
		want PGErrorClassification
	}{
		{name: "nil is non retriable", err: nil, want: NonRetriable},
		{name: "connection failure is retriable", err: &pgconn.PgError{Code: pgerrcode.ConnectionFailure}, want: Retriable},
		{name: "unique violation is non retriable", err: &pgconn.PgError{Code: pgerrcode.UniqueViolation}, want: NonRetriable},
		{name: "generic error is non retriable", err: errors.New("boom"), want: NonRetriable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			classifier := NewPostgresErrorClassifier()
			assert.Equal(t, tt.want, classifier.Classify(tt.err))
		})
	}
}
