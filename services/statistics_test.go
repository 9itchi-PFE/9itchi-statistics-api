package services

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"9itchi-statistics-api/core"
	"9itchi-statistics-api/models/api"
	"9itchi-statistics-api/models/db"
	"9itchi-statistics-api/services/mocks"

	"github.com/stretchr/testify/require"
)

func TestRecordEventMapsAndIncrements(t *testing.T) {
	at := time.Date(2026, 1, 2, 1, 0, 0, 0, time.FixedZone("offset", 2*60*60))
	repo := &mocks.StatisticsRepository{
		IncrementFunc: func(_ context.Context, name string, day time.Time) (*db.Metric, error) {
			require.Equal(t, "email_sent", name)
			require.Equal(t, "2026-01-01", day.Format(time.DateOnly))
			return &db.Metric{Name: name, Day: day, Value: 7}, nil
		},
	}
	result, apiErr := NewStatisticsService(repo).RecordEvent(context.Background(), api.EventRequest{Type: core.EventEmailSent, TS: &at})
	require.Nil(t, apiErr)
	require.Equal(t, int64(7), result.Value)
	require.Equal(t, "email_sent", result.Name)
}

func TestRecordEventRejectsUnknownTypeAndHidesRepositoryErrors(t *testing.T) {
	svc := NewStatisticsService(&mocks.StatisticsRepository{})
	_, apiErr := svc.RecordEvent(context.Background(), api.EventRequest{Type: "unknown"})
	require.Equal(t, http.StatusBadRequest, apiErr.Code)

	repo := &mocks.StatisticsRepository{
		IncrementFunc: func(context.Context, string, time.Time) (*db.Metric, error) {
			return nil, errors.New("database details")
		},
	}
	_, apiErr = NewStatisticsService(repo).RecordEvent(context.Background(), api.EventRequest{Type: core.EventEmailSent})
	require.Equal(t, http.StatusInternalServerError, apiErr.Code)
	require.Equal(t, "internal server error", apiErr.Error.Error())
}

func TestConsumeEventRejectsMalformedTimestamp(t *testing.T) {
	svc := NewStatisticsService(&mocks.StatisticsRepository{})
	require.Error(t, svc.ConsumeEvent(context.Background(), core.EventEmailSent, map[string]any{"ts": 123}))
	require.Error(t, svc.ConsumeEvent(context.Background(), core.EventEmailSent, map[string]any{"ts": "not-a-time"}))
}

func TestDashboardProducesDenseDailySeriesAndValidatesRange(t *testing.T) {
	repo := &mocks.StatisticsRepository{
		ListFunc: func(_ context.Context, from, to time.Time) ([]db.Metric, error) {
			require.Equal(t, "2026-01-01", from.Format(time.DateOnly))
			require.Equal(t, "2026-01-03", to.Format(time.DateOnly))
			return []db.Metric{{Name: "user_registered", Day: from, Value: 2}}, nil
		},
	}
	svc := NewStatisticsService(repo)
	result, apiErr := svc.Registrations(context.Background(), api.DateRangeQuery{From: "2026-01-01", To: "2026-01-03"})
	require.Nil(t, apiErr)
	summary := result.Metrics["user_registered"]
	require.Equal(t, int64(2), summary.Total)
	require.Len(t, summary.Series, 3)
	require.Equal(t, int64(0), summary.Series[2].Value)

	_, apiErr = svc.Registrations(context.Background(), api.DateRangeQuery{From: "2026-02-02", To: "2026-01-01"})
	require.Equal(t, http.StatusBadRequest, apiErr.Code)
	_, apiErr = svc.Registrations(context.Background(), api.DateRangeQuery{From: "bad"})
	require.Equal(t, http.StatusBadRequest, apiErr.Code)
}
