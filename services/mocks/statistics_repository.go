package mocks

import (
	"context"
	"time"

	"9itchi-statistics-api/models/db"
)

type StatisticsRepository struct {
	IncrementFunc func(context.Context, string, time.Time) (*db.Metric, error)
	ListFunc      func(context.Context, time.Time, time.Time) ([]db.Metric, error)
}

func (m *StatisticsRepository) Increment(ctx context.Context, name string, day time.Time) (*db.Metric, error) {
	return m.IncrementFunc(ctx, name, day)
}

func (m *StatisticsRepository) List(ctx context.Context, from, to time.Time) ([]db.Metric, error) {
	return m.ListFunc(ctx, from, to)
}
