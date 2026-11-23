package services

import (
	"context"

	"9itchi-statistics-api/models"
	"9itchi-statistics-api/repositories"
)

// StatisticsService holds all business logic for statistics. Handlers stay thin.
type StatisticsService interface {
	// GET /admin/statistics/overview: headline numbers
	Overview(ctx context.Context, userID string) (any, *models.ApiError)
	// GET /admin/statistics/registrations: new users over time
	Registrations(ctx context.Context, userID string) (any, *models.ApiError)
	// GET /admin/statistics/emails: sent vs opened
	Emails(ctx context.Context, userID string) (any, *models.ApiError)
	// GET /admin/statistics/subscriptions: active subscriptions by plan
	Subscriptions(ctx context.Context, userID string) (any, *models.ApiError)
	// GET /admin/statistics/revenue: revenue over time
	Revenue(ctx context.Context, userID string) (any, *models.ApiError)
}

type statisticsService struct {
	repo repositories.StatisticsRepository
}

func NewStatisticsService(repo repositories.StatisticsRepository) StatisticsService {
	return &statisticsService{repo: repo}
}

func (s *statisticsService) Overview(_ context.Context, _ string) (any, *models.ApiError) {
	return nil, models.NotImplemented()
}

func (s *statisticsService) Registrations(_ context.Context, _ string) (any, *models.ApiError) {
	return nil, models.NotImplemented()
}

func (s *statisticsService) Emails(_ context.Context, _ string) (any, *models.ApiError) {
	return nil, models.NotImplemented()
}

func (s *statisticsService) Subscriptions(_ context.Context, _ string) (any, *models.ApiError) {
	return nil, models.NotImplemented()
}

func (s *statisticsService) Revenue(_ context.Context, _ string) (any, *models.ApiError) {
	return nil, models.NotImplemented()
}
