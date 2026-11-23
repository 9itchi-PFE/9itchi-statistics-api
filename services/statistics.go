package services

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"time"

	"9itchi-statistics-api/core"
	"9itchi-statistics-api/models"
	"9itchi-statistics-api/models/api"
	"9itchi-statistics-api/models/db"
	"9itchi-statistics-api/models/mapping"
	"9itchi-statistics-api/repositories"
	"9itchi-statistics-api/utils"

	"go.uber.org/zap"
)

const maxRangeDays = 366

const (
	metricUserRegistered      = "user_registered"
	metricEmailSent           = "email_sent"
	metricEmailFailed         = "email_failed"
	metricEmailOpened         = "email_opened"
	metricJobsMatched         = "jobs_matched"
	metricSubscriptionCreated = "subscription_created"
	metricPaymentCompleted    = "payment_completed"
)

type eventRecorder func(context.Context, time.Time) (*db.Metric, error)

// StatisticsService holds all business logic for statistics. Handlers stay thin.
type StatisticsService interface {
	RecordEvent(ctx context.Context, req api.EventRequest) (*api.EventResponse, *models.ApiError)
	ConsumeEvent(ctx context.Context, event string, payload map[string]any) error
	Overview(ctx context.Context, q api.DateRangeQuery) (*api.DashboardResponse, *models.ApiError)
	Registrations(ctx context.Context, q api.DateRangeQuery) (*api.DashboardResponse, *models.ApiError)
	Emails(ctx context.Context, q api.DateRangeQuery) (*api.DashboardResponse, *models.ApiError)
	Subscriptions(ctx context.Context, q api.DateRangeQuery) (*api.DashboardResponse, *models.ApiError)
	Revenue(ctx context.Context, q api.DateRangeQuery) (*api.DashboardResponse, *models.ApiError)
}

type statisticsService struct {
	repo   repositories.StatisticsRepository
	events map[string]eventRecorder
}

func NewStatisticsService(repo repositories.StatisticsRepository) StatisticsService {
	s := &statisticsService{repo: repo}
	s.events = map[string]eventRecorder{
		core.EventUserRegistered:      s.incrementer(metricUserRegistered),
		core.EventEmailSent:           s.incrementer(metricEmailSent),
		core.EventEmailFailed:         s.incrementer(metricEmailFailed),
		core.EventEmailOpened:         s.incrementer(metricEmailOpened),
		core.EventJobsMatched:         s.incrementer(metricJobsMatched),
		core.EventSubscriptionCreated: s.incrementer(metricSubscriptionCreated),
		core.EventPaymentCompleted:    s.incrementer(metricPaymentCompleted),
	}
	return s
}

func (s *statisticsService) incrementer(name string) eventRecorder {
	return func(ctx context.Context, at time.Time) (*db.Metric, error) {
		return s.repo.Increment(ctx, name, at)
	}
}

func (s *statisticsService) RecordEvent(ctx context.Context, req api.EventRequest) (*api.EventResponse, *models.ApiError) {
	dispatch, ok := s.events[req.Type]
	if !ok {
		return nil, badRequest("unsupported event type")
	}
	at := time.Now().UTC()
	if req.TS != nil {
		at = req.TS.UTC()
	}
	metric, err := dispatch(ctx, at)
	if err != nil {
		return nil, internalError("increment metric", err)
	}
	return mapping.MetricToEventResponse(metric), nil
}

func (s *statisticsService) ConsumeEvent(ctx context.Context, event string, payload map[string]any) error {
	req := api.EventRequest{Type: event}
	if raw, exists := payload["ts"]; exists && raw != nil {
		switch value := raw.(type) {
		case string:
			at, err := time.Parse(time.RFC3339, value)
			if err != nil {
				return errors.New("invalid event timestamp")
			}
			req.TS = &at
		case time.Time:
			req.TS = &value
		default:
			return errors.New("invalid event timestamp")
		}
	}
	_, apiErr := s.RecordEvent(ctx, req)
	if apiErr != nil {
		return apiErr.Error
	}
	return nil
}

func (s *statisticsService) Overview(ctx context.Context, q api.DateRangeQuery) (*api.DashboardResponse, *models.ApiError) {
	return s.dashboard(ctx, q, allMetricNames())
}

func (s *statisticsService) Registrations(ctx context.Context, q api.DateRangeQuery) (*api.DashboardResponse, *models.ApiError) {
	return s.dashboard(ctx, q, []string{metricUserRegistered})
}

func (s *statisticsService) Emails(ctx context.Context, q api.DateRangeQuery) (*api.DashboardResponse, *models.ApiError) {
	return s.dashboard(ctx, q, []string{metricEmailFailed, metricEmailOpened, metricEmailSent})
}

func (s *statisticsService) Subscriptions(ctx context.Context, q api.DateRangeQuery) (*api.DashboardResponse, *models.ApiError) {
	return s.dashboard(ctx, q, []string{metricSubscriptionCreated})
}

func (s *statisticsService) Revenue(ctx context.Context, q api.DateRangeQuery) (*api.DashboardResponse, *models.ApiError) {
	return s.dashboard(ctx, q, []string{metricPaymentCompleted})
}

func (s *statisticsService) dashboard(ctx context.Context, q api.DateRangeQuery, names []string) (*api.DashboardResponse, *models.ApiError) {
	from, to, err := parseRange(q)
	if err != nil {
		return nil, badRequest(err.Error())
	}
	rows, err := s.repo.List(ctx, from, to)
	if err != nil {
		return nil, internalError("list metrics", err)
	}
	values := make(map[string]map[string]int64, len(names))
	for _, name := range names {
		values[name] = make(map[string]int64)
	}
	for _, row := range rows {
		if values[row.Name] != nil {
			values[row.Name][row.Day.Format(time.DateOnly)] = row.Value
		}
	}
	result := &api.DashboardResponse{
		From:    from.Format(time.DateOnly),
		To:      to.Format(time.DateOnly),
		Metrics: make(map[string]api.MetricSummary, len(names)),
	}
	for _, name := range names {
		summary := api.MetricSummary{Series: make([]api.MetricPoint, 0, int(to.Sub(from).Hours()/24)+1)}
		for day := from; !day.After(to); day = day.AddDate(0, 0, 1) {
			date := day.Format(time.DateOnly)
			value := values[name][date]
			summary.Total += value
			summary.Series = append(summary.Series, api.MetricPoint{Day: date, Value: value})
		}
		result.Metrics[name] = summary
	}
	return result, nil
}

func parseRange(q api.DateRangeQuery) (time.Time, time.Time, error) {
	today := time.Now().UTC().Truncate(24 * time.Hour)
	from, to := today.AddDate(0, 0, -(maxRangeDaysDefault-1)), today
	var err error
	if q.From != "" {
		from, err = time.Parse(time.DateOnly, q.From)
		if err != nil {
			return time.Time{}, time.Time{}, errors.New("from must be YYYY-MM-DD")
		}
	}
	if q.To != "" {
		to, err = time.Parse(time.DateOnly, q.To)
		if err != nil {
			return time.Time{}, time.Time{}, errors.New("to must be YYYY-MM-DD")
		}
	}
	if q.From == "" && q.To != "" {
		from = to.AddDate(0, 0, -(maxRangeDaysDefault - 1))
	}
	if from.After(to) {
		return time.Time{}, time.Time{}, errors.New("from must be on or before to")
	}
	if to.Sub(from) >= time.Duration(maxRangeDays)*24*time.Hour {
		return time.Time{}, time.Time{}, errors.New("date range must not exceed 366 days")
	}
	return from.UTC(), to.UTC(), nil
}

const maxRangeDaysDefault = 30

func allMetricNames() []string {
	names := []string{
		metricUserRegistered,
		metricEmailSent,
		metricEmailFailed,
		metricEmailOpened,
		metricJobsMatched,
		metricSubscriptionCreated,
		metricPaymentCompleted,
	}
	sort.Strings(names)
	return names
}

func badRequest(message string) *models.ApiError {
	return &models.ApiError{Code: http.StatusBadRequest, Error: errors.New(message)}
}

func internalError(operation string, err error) *models.ApiError {
	utils.Logger.Error(operation+" failed", zap.Error(err))
	return &models.ApiError{Code: http.StatusInternalServerError, Error: errors.New("internal server error")}
}
