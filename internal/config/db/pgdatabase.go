package config

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

var (
	noConnectionToCloseError = errors.New("No connection to close")
)

type PgDatabaseAdapter struct {
	connectionString string
	db               *pgx.Conn
}

func NewPgDatabaseAdapter(connectionString string) *PgDatabaseAdapter {
	return &PgDatabaseAdapter{
		connectionString: connectionString,
	}
}

func (dbAdapter *PgDatabaseAdapter) Open(ctx context.Context) error{
	connection, err := pgx.Connect(ctx, dbAdapter.connectionString)

	if err != nil {
		return err
	}
	dbAdapter.db = connection
	return nil
}

func (dbAdapter *PgDatabaseAdapter) Close(ctx context.Context) error{
	if dbAdapter.db == nil {
		return noConnectionToCloseError
	}
	err := dbAdapter.db.Close(ctx)
	return err
}

func (dbAdapter *PgDatabaseAdapter) CheckConnection(ctx context.Context) error{
	return dbAdapter.db.Ping(ctx)
}