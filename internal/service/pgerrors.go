package service

import (
	"errors"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

type pgErrorClassification int

const (
	nonRetriable pgErrorClassification = iota
	retriable
)

type postgresErrorClassifier struct{}

func (c postgresErrorClassifier) classify(err error) pgErrorClassification {
	if err == nil {
		return nonRetriable
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return classifyPgError(pgErr)
	}

	return nonRetriable
}

func classifyPgError(pgErr *pgconn.PgError) pgErrorClassification {
	switch pgErr.Code {
	case pgerrcode.ConnectionException,
		pgerrcode.ConnectionDoesNotExist,
		pgerrcode.ConnectionFailure,
		pgerrcode.TransactionRollback,
		pgerrcode.SerializationFailure,
		pgerrcode.DeadlockDetected,
		pgerrcode.CannotConnectNow:
		return retriable
	}

	return nonRetriable
}
