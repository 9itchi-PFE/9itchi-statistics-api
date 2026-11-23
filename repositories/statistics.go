package repositories

import (
	"context"
	"time"

	"9itchi-statistics-api/models/db"

	"github.com/jackc/pgx/v5"
)

// StatisticsRepository is the only layer that touches PostgreSQL for statistics data.
type StatisticsRepository interface {
	Increment(ctx context.Context, name string, day time.Time) (*db.Metric, error)
	List(ctx context.Context, from, to time.Time) ([]db.Metric, error)
}

type statisticsRepository struct {
	store *Store
}

func NewStatisticsRepository(store *Store) StatisticsRepository {
	return &statisticsRepository{store: store}
}

func (r *statisticsRepository) Increment(ctx context.Context, name string, day time.Time) (*db.Metric, error) {
	ctx, cancel := QueryTimeout(ctx)
	defer cancel()
	rows, err := r.store.Conn(ctx).Query(ctx, `
		INSERT INTO metrics (name, day, value)
		VALUES ($1, $2::date, 1)
		ON CONFLICT (name, day) DO UPDATE SET value = metrics.value + 1
		RETURNING name, day, value`, name, day.UTC().Format(time.DateOnly))
	if err != nil {
		return nil, MapError(err)
	}
	metric, err := pgx.CollectOneRow(rows, pgx.RowToAddrOfStructByName[db.Metric])
	return metric, MapError(err)
}

func (r *statisticsRepository) List(ctx context.Context, from, to time.Time) ([]db.Metric, error) {
	ctx, cancel := QueryTimeout(ctx)
	defer cancel()
	rows, err := r.store.Conn(ctx).Query(ctx, `
		SELECT name, day, value FROM metrics
		WHERE day >= $1::date AND day <= $2::date
		ORDER BY name, day`, from.UTC().Format(time.DateOnly), to.UTC().Format(time.DateOnly))
	if err != nil {
		return nil, MapError(err)
	}
	metrics, err := pgx.CollectRows(rows, pgx.RowToStructByName[db.Metric])
	return metrics, MapError(err)
}
