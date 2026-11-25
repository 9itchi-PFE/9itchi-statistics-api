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

func TestCompletedPaymentAlsoCountsWhatItBrought(t *testing.T) {
	type added struct {
		name   string
		day    string
		amount int64
	}
	var counted []string
	var sums []added
	repo := &mocks.StatisticsRepository{
		IncrementFunc: func(_ context.Context, name string, day time.Time) (*db.Metric, error) {
			counted = append(counted, name)
			return &db.Metric{Name: name, Day: day, Value: 1}, nil
		},
		AddFunc: func(_ context.Context, name string, day time.Time, amount int64) (*db.Metric, error) {
			sums = append(sums, added{name, day.UTC().Format(time.DateOnly), amount})
			return &db.Metric{Name: name, Day: day, Value: amount}, nil
		},
	}
	svc := NewStatisticsService(repo)
	ctx := context.Background()

	// As it arrives from RabbitMQ: a JSON number, with the day of the event.
	require.NoError(t, svc.ConsumeEvent(ctx, core.EventPaymentCompleted, map[string]any{"amount_millimes": float64(35000), "ts": "2026-11-20T10:00:00Z"}))
	// A payment a code covered entirely brings nothing; another event with an amount is not revenue.
	require.NoError(t, svc.ConsumeEvent(ctx, core.EventPaymentCompleted, map[string]any{"amount_millimes": float64(0)}))
	require.NoError(t, svc.ConsumeEvent(ctx, core.EventPaymentCompleted, map[string]any{}))
	require.NoError(t, svc.ConsumeEvent(ctx, core.EventEmailSent, map[string]any{"amount_millimes": float64(999)}))

	require.Equal(t, []string{"payment_completed", "payment_completed", "payment_completed", "email_sent"}, counted)
	require.Equal(t, []added{{"revenue_millimes", "2026-11-20", 35000}}, sums)

	repo.AddFunc = func(context.Context, string, time.Time, int64) (*db.Metric, error) { return nil, errors.New("db down") }
	require.Error(t, svc.ConsumeEvent(ctx, core.EventPaymentCompleted, map[string]any{"amount_millimes": int64(1000)}), "asked again by the broker")
}

func TestRevenueAndOverviewCarryTheSum(t *testing.T) {
	day := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)
	repo := &mocks.StatisticsRepository{ListFunc: func(context.Context, time.Time, time.Time) ([]db.Metric, error) {
		return []db.Metric{{Name: "revenue_millimes", Day: day, Value: 70000}, {Name: "payment_completed", Day: day, Value: 2}}, nil
	}}
	svc := NewStatisticsService(repo)
	q := api.DateRangeQuery{From: "2026-11-19", To: "2026-11-21"}
	rev, apiErr := svc.Revenue(context.Background(), q)
	require.Nil(t, apiErr)
	require.Equal(t, int64(70000), rev.Metrics["revenue_millimes"].Total)
	require.Equal(t, int64(2), rev.Metrics["payment_completed"].Total)
	all, apiErr := svc.Overview(context.Background(), q)
	require.Nil(t, apiErr)
	require.Equal(t, int64(70000), all.Metrics["revenue_millimes"].Total)
}

// The steps between signing up and subscribing: each « first time » event counts one student for its day, whether
// it is read from the queue or posted, and the dashboard carries the three figures.
func TestFirstTimeEventsCountStudents(t *testing.T) {
	counted := map[string]int{}
	repo := &mocks.StatisticsRepository{
		IncrementFunc: func(_ context.Context, name string, day time.Time) (*db.Metric, error) {
			counted[name]++
			return &db.Metric{Name: name, Day: day, Value: int64(counted[name])}, nil
		},
	}
	svc := NewStatisticsService(repo)
	for event, metric := range map[string]string{
		core.EventCVFirstImported: "first_cv", core.EventPreferencesFirstSaved: "first_preferences", core.EventApplicationFirstSent: "first_application",
	} {
		require.NoError(t, svc.ConsumeEvent(context.Background(), event, map[string]any{"user_id": "u-1"}))
		res, apiErr := svc.RecordEvent(context.Background(), api.EventRequest{Type: event})
		require.Nil(t, apiErr)
		require.Equal(t, metric, res.Name)
		require.Equal(t, 2, counted[metric])
		require.Contains(t, CountedEvents, event, "the queue is bound to it")
		require.Contains(t, allMetricNames(), metric)
	}
	require.Len(t, counted, 3, "nothing else is counted for them")
}
