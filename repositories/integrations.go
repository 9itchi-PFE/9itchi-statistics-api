package repositories

import (
	"context"
	"time"

	"9itchi-statistics-api/models/db"

	"github.com/jackc/pgx/v5"
)

// IntegrationRepository is the only layer that touches PostgreSQL for provider settings and usage.
type IntegrationRepository interface {
	ListSettings(ctx context.Context) ([]db.IntegrationSetting, error)
	// GetSetting returns ErrNotFound when the provider was never configured.
	GetSetting(ctx context.Context, provider string) (*db.IntegrationSetting, error)
	SaveSetting(ctx context.Context, s db.IntegrationSetting) error
	AddUsage(ctx context.Context, provider string, month time.Time, amount int64) error
	ListUsage(ctx context.Context, month time.Time) ([]db.IntegrationUsage, error)
}

type integrationRepository struct {
	store *Store
}

func NewIntegrationRepository(store *Store) IntegrationRepository {
	return &integrationRepository{store: store}
}

func (r *integrationRepository) ListSettings(ctx context.Context) ([]db.IntegrationSetting, error) {
	ctx, cancel := QueryTimeout(ctx)
	defer cancel()
	rows, err := r.store.Conn(ctx).Query(ctx, `
		SELECT provider, quota, token_enc, token_hint, updated_at FROM integration_settings ORDER BY provider`)
	if err != nil {
		return nil, MapError(err)
	}
	settings, err := pgx.CollectRows(rows, pgx.RowToStructByName[db.IntegrationSetting])
	return settings, MapError(err)
}

func (r *integrationRepository) GetSetting(ctx context.Context, provider string) (*db.IntegrationSetting, error) {
	ctx, cancel := QueryTimeout(ctx)
	defer cancel()
	rows, err := r.store.Conn(ctx).Query(ctx, `
		SELECT provider, quota, token_enc, token_hint, updated_at FROM integration_settings WHERE provider = $1`, provider)
	if err != nil {
		return nil, MapError(err)
	}
	setting, err := pgx.CollectOneRow(rows, pgx.RowToAddrOfStructByName[db.IntegrationSetting])
	return setting, MapError(err)
}

func (r *integrationRepository) SaveSetting(ctx context.Context, s db.IntegrationSetting) error {
	ctx, cancel := QueryTimeout(ctx)
	defer cancel()
	_, err := r.store.Conn(ctx).Exec(ctx, `
		INSERT INTO integration_settings (provider, quota, token_enc, token_hint, updated_at)
		VALUES ($1, $2, $3, $4, now())
		ON CONFLICT (provider) DO UPDATE
		SET quota = EXCLUDED.quota, token_enc = EXCLUDED.token_enc, token_hint = EXCLUDED.token_hint, updated_at = now()`,
		s.Provider, s.Quota, s.TokenEnc, s.TokenHint)
	return MapError(err)
}

func (r *integrationRepository) AddUsage(ctx context.Context, provider string, month time.Time, amount int64) error {
	ctx, cancel := QueryTimeout(ctx)
	defer cancel()
	_, err := r.store.Conn(ctx).Exec(ctx, `
		INSERT INTO integration_usage (provider, month, used)
		VALUES ($1, $2::date, $3)
		ON CONFLICT (provider, month) DO UPDATE SET used = integration_usage.used + EXCLUDED.used`,
		provider, month.UTC().Format(time.DateOnly), amount)
	return MapError(err)
}

func (r *integrationRepository) ListUsage(ctx context.Context, month time.Time) ([]db.IntegrationUsage, error) {
	ctx, cancel := QueryTimeout(ctx)
	defer cancel()
	rows, err := r.store.Conn(ctx).Query(ctx, `
		SELECT provider, month, used FROM integration_usage WHERE month = $1::date`, month.UTC().Format(time.DateOnly))
	if err != nil {
		return nil, MapError(err)
	}
	usage, err := pgx.CollectRows(rows, pgx.RowToStructByName[db.IntegrationUsage])
	return usage, MapError(err)
}
