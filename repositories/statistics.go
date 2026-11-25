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
	// Add adds an amount (above 0) to a metric of a day: a sum, where Increment counts.
	Add(ctx context.Context, name string, day time.Time, amount int64) (*db.Metric, error)
	List(ctx context.Context, from, to time.Time) ([]db.Metric, error)
	// Claim records that an event was counted: false when its id was already there (a second delivery).
	Claim(ctx context.Context, eventID string) (bool, error)
}

type statisticsRepository struct {
	store *Store
}

func NewStatisticsRepository(store *Store) StatisticsRepository {
	return &statisticsRepository{store: store}
}

func (r *statisticsRepository) Increment(ctx context.Context, name string, day time.Time) (*db.Metric, error) {
	return r.Add(ctx, name, day, 1)
}

func (r *statisticsRepository) Add(ctx context.Context, name string, day time.Time, amount int64) (*db.Metric, error) {
	ctx, cancel := QueryTimeout(ctx)
	defer cancel()
	rows, err := r.store.Conn(ctx).Query(ctx, `
		INSERT INTO metrics (name, day, value)
		VALUES ($1, $2::date, $3)
		ON CONFLICT (name, day) DO UPDATE SET value = metrics.value + EXCLUDED.value
		RETURNING name, day, value`, name, day.UTC().Format(time.DateOnly), amount)
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

func (r *statisticsRepository) Claim(ctx context.Context, eventID string) (bool, error) {
	ctx, cancel := QueryTimeout(ctx)
	defer cancel()
	tag, err := r.store.Conn(ctx).Exec(ctx, `INSERT INTO counted_events (event_id) VALUES ($1) ON CONFLICT DO NOTHING`, eventID)
	if err != nil {
		return false, MapError(err)
	}
	return tag.RowsAffected() == 1, nil
}
