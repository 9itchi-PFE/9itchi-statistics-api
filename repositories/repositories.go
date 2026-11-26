package repositories

import (
	"context"

	"9itchi-statistics-api/utils"
)

// Repositories holds every repository plus the shared Store (pool + transactions).
type Repositories struct {
	Store        *Store
	Statistics   StatisticsRepository
	Integrations IntegrationRepository
}

// InitRepositories connects to PostgreSQL, applies migrations and builds the repositories.
func InitRepositories(ctx context.Context, config utils.AppConfig) (*Repositories, error) {
	if err := RunMigrations(ctx, config.DatabaseURL); err != nil {
		return nil, err
	}
	store, err := NewStore(ctx, config.DatabaseURL)
	if err != nil {
		return nil, err
	}
	return &Repositories{
		Store:        store,
		Statistics:   NewStatisticsRepository(store),
		Integrations: NewIntegrationRepository(store),
	}, nil
}

func (r *Repositories) Close() { r.Store.Close() }
