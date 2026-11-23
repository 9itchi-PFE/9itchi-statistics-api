package mocks

import (
	"context"

	"9itchi-statistics-api/models"
	"9itchi-statistics-api/models/api"
)

type StatisticsService struct {
	RecordEventFunc   func(context.Context, api.EventRequest) (*api.EventResponse, *models.ApiError)
	ConsumeEventFunc  func(context.Context, string, map[string]any) error
	OverviewFunc      func(context.Context, api.DateRangeQuery) (*api.DashboardResponse, *models.ApiError)
	RegistrationsFunc func(context.Context, api.DateRangeQuery) (*api.DashboardResponse, *models.ApiError)
	EmailsFunc        func(context.Context, api.DateRangeQuery) (*api.DashboardResponse, *models.ApiError)
	SubscriptionsFunc func(context.Context, api.DateRangeQuery) (*api.DashboardResponse, *models.ApiError)
	RevenueFunc       func(context.Context, api.DateRangeQuery) (*api.DashboardResponse, *models.ApiError)
}

func (m *StatisticsService) RecordEvent(ctx context.Context, req api.EventRequest) (*api.EventResponse, *models.ApiError) {
	return m.RecordEventFunc(ctx, req)
}

func (m *StatisticsService) ConsumeEvent(ctx context.Context, event string, payload map[string]any) error {
	return m.ConsumeEventFunc(ctx, event, payload)
}

func (m *StatisticsService) Overview(ctx context.Context, q api.DateRangeQuery) (*api.DashboardResponse, *models.ApiError) {
	return m.OverviewFunc(ctx, q)
}

func (m *StatisticsService) Registrations(ctx context.Context, q api.DateRangeQuery) (*api.DashboardResponse, *models.ApiError) {
	return m.RegistrationsFunc(ctx, q)
}

func (m *StatisticsService) Emails(ctx context.Context, q api.DateRangeQuery) (*api.DashboardResponse, *models.ApiError) {
	return m.EmailsFunc(ctx, q)
}

func (m *StatisticsService) Subscriptions(ctx context.Context, q api.DateRangeQuery) (*api.DashboardResponse, *models.ApiError) {
	return m.SubscriptionsFunc(ctx, q)
}

func (m *StatisticsService) Revenue(ctx context.Context, q api.DateRangeQuery) (*api.DashboardResponse, *models.ApiError) {
	return m.RevenueFunc(ctx, q)
}
