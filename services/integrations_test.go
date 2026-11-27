package services

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"9itchi-statistics-api/models/api"
	"9itchi-statistics-api/pkg/balance"
	"9itchi-statistics-api/services/mocks"
	"9itchi-statistics-api/utils"

	"github.com/stretchr/testify/require"
)

func newIntegrationTestService(repo *mocks.IntegrationRepository) IntegrationService {
	svc := NewIntegrationService(repo, utils.AppConfig{EncryptionKey: "0123456789abcdef0123456789abcdef"}).(*integrationService)
	svc.now = func() time.Time { return time.Date(2026, 11, 26, 10, 0, 0, 0, time.UTC) }
	return svc
}

func ptr[T any](v T) *T { return &v }

func TestIntegrationsListComputesUsageAgainstQuota(t *testing.T) {
	repo := mocks.NewIntegrationRepository()
	svc := newIntegrationTestService(repo)
	ctx := context.Background()

	_, apiErr := svc.Update(ctx, "gemini", api.UpdateIntegrationRequest{Quota: ptr(int64(1000))})
	require.Nil(t, apiErr)
	require.Nil(t, svc.RecordUsage(ctx, "gemini", 250))
	require.Nil(t, svc.RecordUsage(ctx, "gemini", 100))
	require.Nil(t, svc.RecordUsage(ctx, "llamaindex", 5))

	list, apiErr := svc.List(ctx)
	require.Nil(t, apiErr)
	require.Equal(t, "2026-11", list.Month)
	require.Len(t, list.Integrations, len(Integrations))
	gemini := list.Integrations[0]
	require.Equal(t, int64(350), gemini.Used)
	require.Equal(t, 35, gemini.Percent)
	require.Equal(t, int64(650), gemini.Remaining)
	llama := list.Integrations[2]
	require.Equal(t, int64(5), llama.Used)
	require.Zero(t, llama.Percent, "no quota set")

	// Email-api reports the contact finders in credits.
	require.Nil(t, svc.RecordUsage(ctx, "hunter", 1))
	require.Nil(t, svc.RecordUsage(ctx, "apollo", 2))
	list, apiErr = svc.List(ctx)
	require.Nil(t, apiErr)
	require.Equal(t, "hunter", list.Integrations[3].Provider)
	require.Equal(t, "credits", list.Integrations[3].Unit)
	require.Equal(t, int64(1), list.Integrations[3].Used)
	require.Equal(t, "apollo", list.Integrations[4].Provider)
	require.Equal(t, int64(2), list.Integrations[4].Used)
}

func TestIntegrationTokenIsSealedAndNeverListed(t *testing.T) {
	repo := mocks.NewIntegrationRepository()
	svc := newIntegrationTestService(repo)
	ctx := context.Background()

	_, apiErr := svc.Token(ctx, "openai")
	require.Equal(t, http.StatusNotFound, apiErr.Code)

	r, apiErr := svc.Update(ctx, "openai", api.UpdateIntegrationRequest{Token: ptr("  sk-secret-token-1234 ")})
	require.Nil(t, apiErr)
	require.True(t, r.HasToken)
	require.Equal(t, "••••1234", r.TokenHint)
	require.NotContains(t, repo.Settings["openai"].TokenEnc, "sk-secret")

	got, apiErr := svc.Token(ctx, "openai")
	require.Nil(t, apiErr)
	require.Equal(t, "sk-secret-token-1234", got.Token)

	// A quota update keeps the token; an empty token removes it.
	r, apiErr = svc.Update(ctx, "openai", api.UpdateIntegrationRequest{Quota: ptr(int64(10))})
	require.Nil(t, apiErr)
	require.True(t, r.HasToken)
	r, apiErr = svc.Update(ctx, "openai", api.UpdateIntegrationRequest{Token: ptr("")})
	require.Nil(t, apiErr)
	require.False(t, r.HasToken)
	require.Equal(t, int64(10), r.Quota)
}

func TestIntegrationRejectsBadInput(t *testing.T) {
	svc := newIntegrationTestService(mocks.NewIntegrationRepository())
	ctx := context.Background()

	_, apiErr := svc.Update(ctx, "nope", api.UpdateIntegrationRequest{Quota: ptr(int64(1))})
	require.Equal(t, http.StatusNotFound, apiErr.Code)
	_, apiErr = svc.Update(ctx, "gemini", api.UpdateIntegrationRequest{})
	require.Equal(t, http.StatusBadRequest, apiErr.Code)
	_, apiErr = svc.Update(ctx, "gemini", api.UpdateIntegrationRequest{Quota: ptr(int64(-1))})
	require.Equal(t, http.StatusBadRequest, apiErr.Code)
	_, apiErr = svc.Update(ctx, "gemini", api.UpdateIntegrationRequest{Token: ptr("two words")})
	require.Equal(t, http.StatusBadRequest, apiErr.Code)
	require.Equal(t, http.StatusNotFound, svc.RecordUsage(ctx, "nope", 1).Code)
	require.Equal(t, http.StatusBadRequest, svc.RecordUsage(ctx, "gemini", 0).Code)
}

func TestIntegrationsListAddsLiveBalance(t *testing.T) {
	repo := mocks.NewIntegrationRepository()
	svc := NewIntegrationService(repo, utils.AppConfig{
		EncryptionKey: "0123456789abcdef0123456789abcdef",
		ProviderKeys:  map[string]string{"hunter": "env-hunter-key", "apollo": ""},
	}).(*integrationService)
	clock := time.Date(2026, 11, 26, 10, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return clock }
	var keys []string
	calls := 0
	svc.fetchers = map[string]balance.Fetcher{
		"hunter": func(_ context.Context, _ *http.Client, key string) (*balance.Balance, error) {
			calls++
			keys = append(keys, key)
			return &balance.Balance{Left: 40, Limit: 50, ResetsOn: "2026-12-05"}, nil
		},
		"apollo": func(context.Context, *http.Client, string) (*balance.Balance, error) {
			t.Fatal("apollo has no key: never asked")
			return nil, nil
		},
	}
	ctx := context.Background()

	list, apiErr := svc.List(ctx)
	require.Nil(t, apiErr)
	byProvider := map[string]api.IntegrationResponse{}
	for _, i := range list.Integrations {
		byProvider[i.Provider] = i
	}
	require.Equal(t, &api.Balance{Left: 40, Limit: 50, ResetsOn: "2026-12-05"}, byProvider["hunter"].Balance)
	require.Nil(t, byProvider["apollo"].Balance, "no key")
	require.Nil(t, byProvider["gemini"].Balance, "no balance API")
	require.Equal(t, []string{"env-hunter-key"}, keys)

	// Cached for 5 minutes.
	_, apiErr = svc.List(ctx)
	require.Nil(t, apiErr)
	require.Equal(t, 1, calls)

	// An admin token wins over the env key and drops the cache.
	_, apiErr = svc.Update(ctx, "hunter", api.UpdateIntegrationRequest{Token: ptr("admin-hunter-token")})
	require.Nil(t, apiErr)
	_, apiErr = svc.List(ctx)
	require.Nil(t, apiErr)
	require.Equal(t, []string{"env-hunter-key", "admin-hunter-token"}, keys)

	// A provider failure is no balance, not an error, and is retried after the TTL.
	svc.fetchers["hunter"] = func(context.Context, *http.Client, string) (*balance.Balance, error) {
		calls++
		return nil, errors.New("status 401")
	}
	clock = clock.Add(6 * time.Minute)
	list, apiErr = svc.List(ctx)
	require.Nil(t, apiErr)
	require.Nil(t, list.Integrations[3].Balance)
	require.Equal(t, 3, calls)
}
