package mocks

import (
	"context"
	"time"

	"9itchi-statistics-api/models/db"
)

type StatisticsRepository struct {
	IncrementFunc func(context.Context, string, time.Time) (*db.Metric, error)
	AddFunc       func(context.Context, string, time.Time, int64) (*db.Metric, error)
	ListFunc      func(context.Context, time.Time, time.Time) ([]db.Metric, error)
}

func (m *StatisticsRepository) Increment(ctx context.Context, name string, day time.Time) (*db.Metric, error) {
	return m.IncrementFunc(ctx, name, day)
}

func (m *StatisticsRepository) List(ctx context.Context, from, to time.Time) ([]db.Metric, error) {
	return m.ListFunc(ctx, from, to)
}

func (m *StatisticsRepository) Add(ctx context.Context, name string, day time.Time, amount int64) (*db.Metric, error) {
	if m.AddFunc == nil {
		return &db.Metric{Name: name, Day: day, Value: amount}, nil
	}
	return m.AddFunc(ctx, name, day, amount)
}
