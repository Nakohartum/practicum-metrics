package repository

import (
	"context"
	"database/sql"
	"sync"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/pressly/goose/v3"

	_ "github.com/jackc/pgx/v5/stdlib"

	models "github.com/Nakohartum/practicum-metrics/internal/model"
)

// PgDatabaseAdapter stores metrics in PostgreSQL using pgx.
type PgDatabaseAdapter struct {
	connectionString string
	db               *pgx.Conn
	mu               sync.Mutex
}

const dbOperationTimeout = 5 * time.Second

var psql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

// NewPgDatabaseAdapter creates a PostgreSQL adapter for the connection string.
func NewPgDatabaseAdapter(connectionString string) *PgDatabaseAdapter {
	return &PgDatabaseAdapter{
		connectionString: connectionString,
	}
}

// Open connects to PostgreSQL and applies migrations.
func (dbAdapter *PgDatabaseAdapter) Open(ctx context.Context) error {
	connection, err := pgx.Connect(ctx, dbAdapter.connectionString)
	if err != nil {
		return err
	}
	dbAdapter.db = connection

	if err := dbAdapter.runMigrations(ctx, "migrations"); err != nil {
		return err
	}
	return nil
}

func (dbAdapter *PgDatabaseAdapter) runMigrations(ctx context.Context, dir string) error {
	db, err := sql.Open("pgx", dbAdapter.connectionString)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}

	return goose.UpContext(ctx, db, dir)
}

// Close closes the PostgreSQL connection.
func (dbAdapter *PgDatabaseAdapter) Close(ctx context.Context) error {
	dbAdapter.mu.Lock()
	defer dbAdapter.mu.Unlock()

	if dbAdapter.db == nil {
		return errNoConnectionToClose
	}
	err := dbAdapter.db.Close(ctx)
	return err
}

// CheckConnection pings the PostgreSQL connection.
func (dbAdapter *PgDatabaseAdapter) CheckConnection(ctx context.Context) error {
	dbAdapter.mu.Lock()
	defer dbAdapter.mu.Unlock()

	return dbAdapter.db.Ping(ctx)
}

// SetData inserts or updates one metric in PostgreSQL.
func (dbAdapter *PgDatabaseAdapter) SetData(model models.Metrics) error {
	dbAdapter.mu.Lock()
	defer dbAdapter.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), dbOperationTimeout)
	defer cancel()

	switch model.MType {
	case models.Counter:
		return dbAdapter.upsertMetric(ctx, model, counterConflictSuffix)
	case models.Gauge:
		return dbAdapter.upsertMetric(ctx, model, gaugeConflictSuffix)
	default:
		return errMetricTypeNotSupported
	}
}

const (
	counterConflictSuffix = `ON CONFLICT (id, metric_type) DO UPDATE
SET delta = COALESCE(metric.delta, 0) + COALESCE(EXCLUDED.delta, 0),
	value = NULL,
	updated_at = now()`
	gaugeConflictSuffix = `ON CONFLICT (id, metric_type) DO UPDATE
SET delta = NULL,
	value = EXCLUDED.value,
	updated_at = now()`
	batchConflictSuffix = `ON CONFLICT (id, metric_type) DO UPDATE
SET delta = CASE
		WHEN EXCLUDED.metric_type = 'counter' THEN COALESCE(metric.delta, 0) + COALESCE(EXCLUDED.delta, 0)
		ELSE NULL
	END,
	value = CASE
		WHEN EXCLUDED.metric_type = 'gauge' THEN EXCLUDED.value
		ELSE NULL
	END,
	updated_at = now()`
)

func (dbAdapter *PgDatabaseAdapter) upsertMetric(ctx context.Context, model models.Metrics, conflictSuffix string) error {
	delta, value := metricValues(model)
	query, args, err := psql.
		Insert("metric").
		Columns("id", "metric_type", "delta", "value", "updated_at").
		Values(model.ID, model.MType, delta, value, sq.Expr("now()")).
		Suffix(conflictSuffix).
		ToSql()
	if err != nil {
		return err
	}

	_, err = dbAdapter.db.Exec(ctx, query, args...)
	return err
}

func metricValues(model models.Metrics) (*int64, *float64) {
	switch model.MType {
	case models.Counter:
		return model.Delta, nil
	case models.Gauge:
		return nil, model.Value
	default:
		return nil, nil
	}
}

// GetAll returns all metrics from PostgreSQL.
func (dbAdapter *PgDatabaseAdapter) GetAll() []models.Metrics {
	dbAdapter.mu.Lock()
	defer dbAdapter.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), dbOperationTimeout)
	defer cancel()

	results := make([]models.Metrics, 0)
	rows, err := dbAdapter.db.Query(ctx, "SELECT id, metric_type, delta, value FROM metric")
	if err != nil {
		return results
	}
	defer rows.Close()

	for rows.Next() {
		var m models.Metrics

		rows.Scan(&m.ID, &m.MType, &m.Delta, &m.Value)
		results = append(results, m)
	}
	return results
}

// GetData returns one metric from PostgreSQL by type and name.
func (dbAdapter *PgDatabaseAdapter) GetData(metricType string, metricKey string) (models.Metrics, error) {
	dbAdapter.mu.Lock()
	defer dbAdapter.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), dbOperationTimeout)
	defer cancel()

	var res models.Metrics

	row := dbAdapter.db.QueryRow(ctx, "SELECT id, metric_type, delta, value FROM metric WHERE id = $1 AND metric_type = $2", metricKey, metricType)

	err := row.Scan(&res.ID, &res.MType, &res.Delta, &res.Value)

	return res, err
}

// SetMultipleDataViaTransaction inserts or updates metrics with one PostgreSQL query.
func (dbAdapter *PgDatabaseAdapter) SetMultipleDataViaTransaction(ctx context.Context, metrics []models.Metrics) error {
	dbAdapter.mu.Lock()
	defer dbAdapter.mu.Unlock()

	if len(metrics) == 0 {
		return nil
	}

	builder := psql.Insert("metric").Columns("id", "metric_type", "delta", "value", "updated_at")
	for _, v := range metrics {
		switch v.MType {
		case models.Counter, models.Gauge:
		default:
			return errMetricTypeNotSupported
		}
		delta, value := metricValues(v)
		builder = builder.Values(v.ID, v.MType, delta, value, sq.Expr("now()"))
	}

	query, args, err := builder.Suffix(batchConflictSuffix).ToSql()
	if err != nil {
		return err
	}

	_, err = dbAdapter.db.Exec(ctx, query, args...)
	return err
}
