package services

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"9itchi-statistics-api/models"
	"9itchi-statistics-api/models/api"
	"9itchi-statistics-api/models/db"
	"9itchi-statistics-api/pkg/balance"
	"9itchi-statistics-api/repositories"
	"9itchi-statistics-api/utils"

	"go.uber.org/zap"
)

const (
	KindAI     = "ai"
	KindPlugin = "plugin"

	maxTokenLength = 512
)

// Integration describes a paid provider the platform consumes: the catalog is the only place to register a new one.
type Integration struct {
	Provider string
	Label    string
	Kind     string
	Unit     string
}

// Integrations is the catalog of providers whose consumption is tracked, in display order.
var Integrations = []Integration{
	{Provider: "gemini", Label: "Gemini", Kind: KindAI, Unit: "tokens"},
	{Provider: "openai", Label: "OpenAI", Kind: KindAI, Unit: "tokens"},
	{Provider: "llamaindex", Label: "LlamaIndex Extract", Kind: KindPlugin, Unit: "documents"},
	{Provider: "hunter", Label: "Hunter.io", Kind: KindPlugin, Unit: "credits"},
	{Provider: "apollo", Label: "Apollo", Kind: KindPlugin, Unit: "credits"},
	{Provider: "quickenrich", Label: "QuickEnrich", Kind: KindPlugin, Unit: "credits"},
	// Job boards read by job-api: one unit per API call (what each provider bills or rate-limits).
	{Provider: "adzuna", Label: "Adzuna", Kind: KindPlugin, Unit: "calls"},
	{Provider: "jsearch", Label: "JSearch (RapidAPI)", Kind: KindPlugin, Unit: "calls"},
	{Provider: "theirstack", Label: "TheirStack", Kind: KindPlugin, Unit: "calls"},
	{Provider: "careerjet", Label: "Careerjet", Kind: KindPlugin, Unit: "calls"},
}

// IntegrationService tracks the monthly consumption of AI providers and plugins against their quota, and holds
// the API tokens an admin can replace without a deploy.
type IntegrationService interface {
	List(ctx context.Context) (*api.IntegrationsResponse, *models.ApiError)
	Update(ctx context.Context, provider string, req api.UpdateIntegrationRequest) (*api.IntegrationResponse, *models.ApiError)
	RecordUsage(ctx context.Context, provider string, amount int64) *models.ApiError
	Token(ctx context.Context, provider string) (*api.TokenResponse, *models.ApiError)
}

const (
	balanceTTL     = 5 * time.Minute
	balanceTimeout = 5 * time.Second
)

type integrationService struct {
	repo   repositories.IntegrationRepository
	config utils.AppConfig
	now    func() time.Time
	// fetchers read a provider's live balance (balance.Fetchers); client is their HTTP client.
	fetchers map[string]balance.Fetcher
	client   *http.Client

	mu       sync.Mutex
	balances map[string]cachedBalance
}

// cachedBalance keeps a provider answer (or its failure, as nil) for balanceTTL, so the page does not ask the
// provider on every load.
type cachedBalance struct {
	value *api.Balance
	at    time.Time
}

func NewIntegrationService(repo repositories.IntegrationRepository, config utils.AppConfig) IntegrationService {
	return &integrationService{
		repo: repo, config: config, now: time.Now,
		fetchers: balance.Fetchers, client: &http.Client{Timeout: balanceTimeout},
		balances: map[string]cachedBalance{},
	}
}

func (s *integrationService) month() time.Time {
	n := s.now().UTC()
	return time.Date(n.Year(), n.Month(), 1, 0, 0, 0, 0, time.UTC)
}

func findIntegration(provider string) (Integration, bool) {
	for _, i := range Integrations {
		if i.Provider == provider {
			return i, true
		}
	}
	return Integration{}, false
}

func (s *integrationService) List(ctx context.Context) (*api.IntegrationsResponse, *models.ApiError) {
	month := s.month()
	settings, err := s.repo.ListSettings(ctx)
	if err != nil {
		return nil, internalError("list integration settings", err)
	}
	usage, err := s.repo.ListUsage(ctx, month)
	if err != nil {
		return nil, internalError("list integration usage", err)
	}
	settingOf := make(map[string]db.IntegrationSetting, len(settings))
	for _, st := range settings {
		settingOf[st.Provider] = st
	}
	usedOf := make(map[string]int64, len(usage))
	for _, u := range usage {
		usedOf[u.Provider] = u.Used
	}
	out := &api.IntegrationsResponse{Month: month.Format("2006-01"), Integrations: make([]api.IntegrationResponse, 0, len(Integrations))}
	for _, i := range Integrations {
		out.Integrations = append(out.Integrations, toIntegrationResponse(i, settingOf[i.Provider], usedOf[i.Provider]))
	}
	var wg sync.WaitGroup
	for n := range out.Integrations {
		r := &out.Integrations[n]
		if _, ok := s.fetchers[r.Provider]; !ok {
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.Balance = s.balance(ctx, r.Provider, settingOf[r.Provider])
		}()
	}
	wg.Wait()
	return out, nil
}

// balance is the provider's live balance, read with the admin token when one is set, else the env key; nil when
// there is no key or the provider fails (logged with the provider name and the error only).
func (s *integrationService) balance(ctx context.Context, provider string, st db.IntegrationSetting) *api.Balance {
	s.mu.Lock()
	c, ok := s.balances[provider]
	s.mu.Unlock()
	if ok && s.now().Sub(c.at) < balanceTTL {
		return c.value
	}
	key := s.config.ProviderKeys[provider]
	if st.TokenEnc != "" {
		token, err := utils.Open(s.config, st.TokenEnc)
		if err != nil {
			utils.Logger.Error("open integration token failed", zap.String("provider", provider), zap.Error(err))
			return nil
		}
		key = token
	}
	if key == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, balanceTimeout)
	defer cancel()
	var value *api.Balance
	b, err := s.fetchers[provider](ctx, s.client, key)
	if err != nil {
		utils.Logger.Warn("integration balance failed", zap.String("provider", provider), zap.Error(err))
	} else {
		value = &api.Balance{Left: b.Left, Limit: b.Limit, ResetsOn: b.ResetsOn}
	}
	s.mu.Lock()
	s.balances[provider] = cachedBalance{value: value, at: s.now()}
	s.mu.Unlock()
	return value
}

func toIntegrationResponse(i Integration, st db.IntegrationSetting, used int64) api.IntegrationResponse {
	r := api.IntegrationResponse{
		Provider: i.Provider, Label: i.Label, Kind: i.Kind, Unit: i.Unit,
		Quota: st.Quota, Used: used, HasToken: st.TokenEnc != "", TokenHint: st.TokenHint,
	}
	if st.Quota > 0 {
		r.Percent = int(used * 100 / st.Quota)
		r.Remaining = max(st.Quota-used, 0)
	}
	return r
}

func (s *integrationService) Update(ctx context.Context, provider string, req api.UpdateIntegrationRequest) (*api.IntegrationResponse, *models.ApiError) {
	integration, ok := findIntegration(provider)
	if !ok {
		return nil, &models.ApiError{Code: http.StatusNotFound, Error: errors.New("unknown integration")}
	}
	if req.Quota == nil && req.Token == nil {
		return nil, badRequest("nothing to update")
	}
	if req.Quota != nil && *req.Quota < 0 {
		return nil, badRequest("quota must be 0 or more")
	}
	current, err := s.repo.GetSetting(ctx, provider)
	if err != nil && !errors.Is(err, repositories.ErrNotFound) {
		return nil, internalError("get integration setting", err)
	}
	setting := db.IntegrationSetting{Provider: provider}
	if current != nil {
		setting = *current
	}
	if req.Quota != nil {
		setting.Quota = *req.Quota
	}
	if req.Token != nil {
		token := strings.TrimSpace(*req.Token)
		if len(token) > maxTokenLength || strings.ContainsAny(token, " \t\r\n") {
			return nil, badRequest("token must be a single value of at most 512 characters")
		}
		setting.TokenEnc, setting.TokenHint = "", ""
		if token != "" {
			sealed, err := utils.Seal(s.config, token)
			if err != nil {
				return nil, internalError("seal integration token", err)
			}
			setting.TokenEnc, setting.TokenHint = sealed, tokenHint(token)
		}
	}
	if err := s.repo.SaveSetting(ctx, setting); err != nil {
		return nil, internalError("save integration setting", err)
	}
	if req.Token != nil {
		s.mu.Lock()
		delete(s.balances, provider)
		s.mu.Unlock()
	}
	usage, err := s.repo.ListUsage(ctx, s.month())
	if err != nil {
		return nil, internalError("list integration usage", err)
	}
	var used int64
	for _, u := range usage {
		if u.Provider == provider {
			used = u.Used
		}
	}
	r := toIntegrationResponse(integration, setting, used)
	return &r, nil
}

// tokenHint shows the last 4 characters of a long token: enough to tell two tokens apart, useless to an attacker.
func tokenHint(token string) string {
	if len(token) < 12 {
		return "••••"
	}
	return "••••" + token[len(token)-4:]
}

func (s *integrationService) RecordUsage(ctx context.Context, provider string, amount int64) *models.ApiError {
	if _, ok := findIntegration(provider); !ok {
		return &models.ApiError{Code: http.StatusNotFound, Error: errors.New("unknown integration")}
	}
	if amount <= 0 {
		return badRequest("amount must be above 0")
	}
	if err := s.repo.AddUsage(ctx, provider, s.month(), amount); err != nil {
		return internalError("add integration usage", err)
	}
	return nil
}

func (s *integrationService) Token(ctx context.Context, provider string) (*api.TokenResponse, *models.ApiError) {
	if _, ok := findIntegration(provider); !ok {
		return nil, &models.ApiError{Code: http.StatusNotFound, Error: errors.New("unknown integration")}
	}
	setting, err := s.repo.GetSetting(ctx, provider)
	if errors.Is(err, repositories.ErrNotFound) || (err == nil && setting.TokenEnc == "") {
		return nil, &models.ApiError{Code: http.StatusNotFound, Error: errors.New("no token set")}
	}
	if err != nil {
		return nil, internalError("get integration setting", err)
	}
	token, err := utils.Open(s.config, setting.TokenEnc)
	if err != nil {
		return nil, internalError("open integration token", err)
	}
	return &api.TokenResponse{Token: token}, nil
}
