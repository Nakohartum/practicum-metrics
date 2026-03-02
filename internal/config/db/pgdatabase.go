package config

import (
	"context"
	"errors"

	models "github.com/Nakohartum/practicum-metrics/internal/model"
	"github.com/jackc/pgx/v5"
)

var (
	errNoConnectionToClose = errors.New("no connection to close")
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
		return errNoConnectionToClose
	}
	err := dbAdapter.db.Close(ctx)
	return err
}

func (dbAdapter *PgDatabaseAdapter) CheckConnection(ctx context.Context) error{
	return dbAdapter.db.Ping(ctx)
}

func (dbAdapter *PgDatabaseAdapter) SetData(model models.Metrics) error {
	var count int64
	row := dbAdapter.db.QueryRow(context.Background(), "SELECT COUNT(*) FROM metric where id = $1 and metric_type = $2", model.ID, model.MType)
	err := row.Scan(count)
	if err != nil {
		return err
	}
	if count == 0 {
		_, err = dbAdapter.db.Exec(context.Background(), "INSERT INTO metric(id, metric_type, delta, value) VALUES($1, $2, $3, $4)", model.ID, model.MType, model.Delta, model.Value)
	} else {
		_, err = dbAdapter.db.Exec(context.Background(), "UPDATE metric SET delta = $1, value = $2 where id = $3 and metric_type = $4", model.Delta, model.Value, model.ID, model.MType)
	}
	return err
}

func (dbAdapter *PgDatabaseAdapter) GetAll() []models.Metrics {
	results := make([]models.Metrics, 0)
	rows, err := dbAdapter.db.Query(context.Background(), "select * from metric")
	if err != nil {
		return results
	}

	for rows.Next(){
		var m models.Metrics

		rows.Scan(&m.ID, &m.MType, &m.Delta, &m.Value)
		results = append(results, m)
	}
	return results
}

func (dbAdapter *PgDatabaseAdapter) GetData(metricType string, metricKey string) (models.Metrics, error) {
	var res models.Metrics

	row := dbAdapter.db.QueryRow(context.Background(), "select * from metric where id = $1 and metric_type = $2", metricKey, metricType)
	
	err := row.Scan(&res.ID, &res.MType, &res.Delta, &res.Value)

	return res, err
} 