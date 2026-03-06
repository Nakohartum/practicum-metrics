package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	models "github.com/Nakohartum/practicum-metrics/internal/model"
	"github.com/jackc/pgx/v5"
)

var (
	errNoConnectionToClose = errors.New("no connection to close")
)

type PgDatabaseAdapter struct {
	connectionString string
	db               *pgx.Conn
	mu               sync.Mutex
}

func NewPgDatabaseAdapter(connectionString string) *PgDatabaseAdapter {
	return &PgDatabaseAdapter{
		connectionString: connectionString,
	}
}

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
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	files := make([]string, 0)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".up.sql") {
			files = append(files, filepath.Join(dir, name))
		}
	}
	sort.Strings(files)

	for _, f := range files {
		sqlBytes, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", f, err)
		}
		if _, err := dbAdapter.db.Exec(ctx, string(sqlBytes)); err != nil {
			return fmt.Errorf("apply migration %s: %w", f, err)
		}
	}
	return nil
}


func (dbAdapter *PgDatabaseAdapter) Close(ctx context.Context) error{
	dbAdapter.mu.Lock()
	defer dbAdapter.mu.Unlock()

	if dbAdapter.db == nil {
		return errNoConnectionToClose
	}
	err := dbAdapter.db.Close(ctx)
	return err
}

func (dbAdapter *PgDatabaseAdapter) CheckConnection(ctx context.Context) error{
	dbAdapter.mu.Lock()
	defer dbAdapter.mu.Unlock()

	return dbAdapter.db.Ping(ctx)
}

func (dbAdapter *PgDatabaseAdapter) SetData(model models.Metrics) error {
	dbAdapter.mu.Lock()
	defer dbAdapter.mu.Unlock()

	var count int64
	row := dbAdapter.db.QueryRow(context.Background(), "SELECT COUNT(*) FROM metric where id = $1 and metric_type = $2", model.ID, model.MType)
	err := row.Scan(&count)
	if err != nil {
		return err
	}
	if count == 0 {
		_, err = dbAdapter.db.Exec(context.Background(), "INSERT INTO metric(id, metric_type, delta, value) VALUES($1, $2, $3, $4)", model.ID, model.MType, model.Delta, model.Value)
	} else {
		switch model.MType {
		case models.Counter:
			_, err = dbAdapter.db.Exec(
				context.Background(),
				"UPDATE metric SET delta = COALESCE(delta, 0) + $1 WHERE id = $2 AND metric_type = $3",
				model.Delta, model.ID, model.MType,
			)
		case models.Gauge:
			_, err = dbAdapter.db.Exec(
				context.Background(),
				"UPDATE metric SET value = $1 WHERE id = $2 AND metric_type = $3",
				model.Value, model.ID, model.MType,
			)
		default:
			return errors.New("no such metric type")
		}
	}
	return err
}

func (dbAdapter *PgDatabaseAdapter) GetAll() []models.Metrics {
	dbAdapter.mu.Lock()
	defer dbAdapter.mu.Unlock()

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
	dbAdapter.mu.Lock()
	defer dbAdapter.mu.Unlock()

	var res models.Metrics

	row := dbAdapter.db.QueryRow(context.Background(), "select * from metric where id = $1 and metric_type = $2", metricKey, metricType)
	
	err := row.Scan(&res.ID, &res.MType, &res.Delta, &res.Value)

	return res, err
} 

func (dbAdapter *PgDatabaseAdapter) SetMultipleDataViaTransaction(ctx context.Context, metrics []models.Metrics) error {
	dbAdapter.mu.Lock()
	defer dbAdapter.mu.Unlock()

	transaction, err := dbAdapter.db.BeginTx(ctx, pgx.TxOptions{})
	
	if err != nil {
		return err
	}
	defer transaction.Rollback(ctx)
	query, err := transaction.Prepare(ctx, "add query", "insert into metric(id, metric_type, delta, value) values($1, $2, $3, $4)")
	if err != nil {
		return err
	}
	for _, v := range metrics {
		_, err = transaction.Exec(ctx, query.SQL, v.ID, v.MType, v.Delta, v.Value)
		if err != nil {
			return err
		}
	}
	return nil
}
