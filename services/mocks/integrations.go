package mocks

import (
	"context"
	"time"

	"9itchi-statistics-api/models"
	"9itchi-statistics-api/models/api"
	"9itchi-statistics-api/models/db"
	"9itchi-statistics-api/repositories"
)

type IntegrationRepository struct {
	Settings map[string]db.IntegrationSetting
	Usage    map[string]int64
}

func NewIntegrationRepository() *IntegrationRepository {
	return &IntegrationRepository{Settings: map[string]db.IntegrationSetting{}, Usage: map[string]int64{}}
}

func (m *IntegrationRepository) ListSettings(context.Context) ([]db.IntegrationSetting, error) {
	var out []db.IntegrationSetting
	for _, s := range m.Settings {
		out = append(out, s)
	}
	return out, nil
}

func (m *IntegrationRepository) GetSetting(_ context.Context, provider string) (*db.IntegrationSetting, error) {
	s, ok := m.Settings[provider]
	if !ok {
		return nil, repositories.ErrNotFound
	}
	return &s, nil
}

func (m *IntegrationRepository) SaveSetting(_ context.Context, s db.IntegrationSetting) error {
	m.Settings[s.Provider] = s
	return nil
}

func (m *IntegrationRepository) AddUsage(_ context.Context, provider string, _ time.Time, amount int64) error {
	m.Usage[provider] += amount
	return nil
}

func (m *IntegrationRepository) ListUsage(context.Context, time.Time) ([]db.IntegrationUsage, error) {
	var out []db.IntegrationUsage
	for p, u := range m.Usage {
		out = append(out, db.IntegrationUsage{Provider: p, Used: u})
	}
	return out, nil
}

type IntegrationService struct {
	ListFunc        func(context.Context) (*api.IntegrationsResponse, *models.ApiError)
	UpdateFunc      func(context.Context, string, api.UpdateIntegrationRequest) (*api.IntegrationResponse, *models.ApiError)
	RecordUsageFunc func(context.Context, string, int64) *models.ApiError
	TokenFunc       func(context.Context, string) (*api.TokenResponse, *models.ApiError)
}

func (m *IntegrationService) List(ctx context.Context) (*api.IntegrationsResponse, *models.ApiError) {
	return m.ListFunc(ctx)
}

func (m *IntegrationService) Update(ctx context.Context, provider string, req api.UpdateIntegrationRequest) (*api.IntegrationResponse, *models.ApiError) {
	return m.UpdateFunc(ctx, provider, req)
}

func (m *IntegrationService) RecordUsage(ctx context.Context, provider string, amount int64) *models.ApiError {
	return m.RecordUsageFunc(ctx, provider, amount)
}

func (m *IntegrationService) Token(ctx context.Context, provider string) (*api.TokenResponse, *models.ApiError) {
	return m.TokenFunc(ctx, provider)
}
