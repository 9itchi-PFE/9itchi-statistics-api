// Package balance reads what is left on a paid provider account (free calls, no credit spent).
package balance

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Balance is the account's credits for the current cycle: Left of Limit, renewed on ResetsOn (YYYY-MM-DD or "").
type Balance struct {
	Left     int64
	Limit    int64
	ResetsOn string
}

// Fetcher reads the balance of one provider with its API key.
type Fetcher func(ctx context.Context, client *http.Client, key string) (*Balance, error)

// Fetchers holds the providers whose balance can be read; register a new provider here.
var Fetchers = map[string]Fetcher{
	"hunter": Hunter,
	"apollo": Apollo,
}

var (
	HunterURL = "https://api.hunter.io/v2/account"
	ApolloURL = "https://api.apollo.io/api/v1/usage_stats/credit_usage_stats"
)

type hunterQuota struct {
	Available float64 `json:"available"`
	Remaining float64 `json:"remaining"`
}

// Hunter reads GET /v2/account. The key goes in the X-API-KEY header (never in the URL, so errors can't leak it).
// Teams with one credits pool have requests.credits; others have separate searches (the ones domain-search uses).
func Hunter(ctx context.Context, client *http.Client, key string) (*Balance, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, HunterURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-KEY", key)
	var body struct {
		Data struct {
			ResetDate string `json:"reset_date"`
			Requests  struct {
				Credits  *hunterQuota `json:"credits"`
				Searches *hunterQuota `json:"searches"`
			} `json:"requests"`
		} `json:"data"`
	}
	if err := call(client, req, &body); err != nil {
		return nil, fmt.Errorf("hunter account: %w", err)
	}
	q := body.Data.Requests.Credits
	if q == nil {
		q = body.Data.Requests.Searches
	}
	if q == nil {
		return nil, errors.New("hunter account: no credits in the answer")
	}
	return &Balance{Left: int64(q.Remaining), Limit: int64(q.Available), ResetsOn: day(body.Data.ResetDate)}, nil
}

// Apollo reads POST /api/v1/usage_stats/credit_usage_stats (needs a master key or that endpoint in the key's scope).
// lead_credit is the email credits pool the people match reveal spends.
func Apollo(ctx context.Context, client *http.Client, key string) (*Balance, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ApolloURL, bytes.NewReader([]byte("{}")))
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", key)
	req.Header.Set("Content-Type", "application/json")
	var body struct {
		Stats map[string]struct {
			Limit    float64 `json:"limit"`
			LeftOver float64 `json:"left_over"`
		} `json:"credit_usage_stats"`
		Cycle struct {
			EndDate string `json:"end_date"`
		} `json:"current_credit_cycle"`
	}
	if err := call(client, req, &body); err != nil {
		return nil, fmt.Errorf("apollo credit usage: %w", err)
	}
	lead, ok := body.Stats["lead_credit"]
	if !ok {
		return nil, errors.New("apollo credit usage: no lead_credit in the answer")
	}
	return &Balance{Left: int64(lead.LeftOver), Limit: int64(lead.Limit), ResetsOn: day(body.Cycle.EndDate)}, nil
}

func call(client *http.Client, req *http.Request, out any) error {
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(out)
}

// day keeps the YYYY-MM-DD part of a date or timestamp, "" when it is not one.
func day(s string) string {
	if len(s) < 10 {
		return ""
	}
	if _, err := time.Parse("2006-01-02", s[:10]); err != nil {
		return ""
	}
	return s[:10]
}
