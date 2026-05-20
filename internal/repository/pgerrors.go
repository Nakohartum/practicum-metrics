package repository

import (
	"errors"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

// IsRetriablePostgresError reports whether a PostgreSQL error can be retried.
func IsRetriablePostgresError(err error) bool {
	if err == nil {
		return false
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}

	switch pgErr.Code {
	case pgerrcode.ConnectionException,
		pgerrcode.ConnectionDoesNotExist,
		pgerrcode.ConnectionFailure,
		pgerrcode.TransactionRollback,
		pgerrcode.SerializationFailure,
		pgerrcode.DeadlockDetected,
		pgerrcode.CannotConnectNow:
		return true
	}

	return false
}

func retryRetriablePostgresError(operation func() error) error {
	err := operation()
	if err == nil {
		return nil
	}
	if !IsRetriablePostgresError(err) {
		return err
	}

	cooldown := 1 * time.Second
	for i := 0; i < 3; i++ {
		time.Sleep(cooldown)
		err = operation()
		if err == nil {
			return nil
		}
		if !IsRetriablePostgresError(err) {
			return err
		}
		cooldown += 2 * time.Second
	}
	return err
}
