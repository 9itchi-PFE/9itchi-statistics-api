package balance

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func serve(t *testing.T, status int, body string) (*httptest.Server, *string) {
	t.Helper()
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("X-API-KEY")
		require.Empty(t, r.URL.RawQuery, "the key never goes in the URL")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv, &got
}

func TestHunterReadsCreditsPool(t *testing.T) {
	srv, key := serve(t, http.StatusOK, `{"data":{"reset_date":"2026-12-05","requests":{"credits":{"used":10,"available":50,"remaining":40},"searches":{"used":1,"available":25,"remaining":24}}}}`)
	HunterURL = srv.URL
	b, err := Hunter(context.Background(), srv.Client(), "k1")
	require.NoError(t, err)
	require.Equal(t, &Balance{Left: 40, Limit: 50, ResetsOn: "2026-12-05"}, b)
	require.Equal(t, "k1", *key)
}

func TestHunterFallsBackToSearches(t *testing.T) {
	srv, _ := serve(t, http.StatusOK, `{"data":{"reset_date":"","requests":{"searches":{"used":1,"available":25,"remaining":24}}}}`)
	HunterURL = srv.URL
	b, err := Hunter(context.Background(), srv.Client(), "k")
	require.NoError(t, err)
	require.Equal(t, &Balance{Left: 24, Limit: 25}, b)
}

func TestHunterError(t *testing.T) {
	srv, _ := serve(t, http.StatusUnauthorized, `{"errors":[]}`)
	HunterURL = srv.URL
	_, err := Hunter(context.Background(), srv.Client(), "k")
	require.ErrorContains(t, err, "status 401")
}

func TestApolloReadsLeadCredits(t *testing.T) {
	srv, key := serve(t, http.StatusOK, `{"credit_usage_stats":{"lead_credit":{"limit":100,"consumed":30,"left_over":70},"export_credit":{"limit":5,"consumed":0,"left_over":5}},"current_credit_cycle":{"start_date":"2026-11-01T00:00:00Z","end_date":"2026-12-01T00:00:00Z"}}`)
	ApolloURL = srv.URL
	b, err := Apollo(context.Background(), srv.Client(), "a1")
	require.NoError(t, err)
	require.Equal(t, &Balance{Left: 70, Limit: 100, ResetsOn: "2026-12-01"}, b)
	require.Equal(t, "a1", *key)
}

func TestApolloWithoutLeadCredits(t *testing.T) {
	srv, _ := serve(t, http.StatusOK, `{"credit_usage_stats":{}}`)
	ApolloURL = srv.URL
	_, err := Apollo(context.Background(), srv.Client(), "a")
	require.ErrorContains(t, err, "lead_credit")
}
