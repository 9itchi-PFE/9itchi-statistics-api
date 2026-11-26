package api

// IntegrationResponse is one AI provider or plugin with its monthly quota and consumption.
// The token itself is never returned, only whether one is set and its last characters.
type IntegrationResponse struct {
	Provider  string `json:"provider"`
	Label     string `json:"label"`
	Kind      string `json:"kind"`
	Unit      string `json:"unit"`
	Quota     int64  `json:"quota"`
	Used      int64  `json:"used"`
	Remaining int64  `json:"remaining"`
	// Percent is used/quota in percent, 0 without a quota; it can exceed 100.
	Percent   int    `json:"percent"`
	HasToken  bool   `json:"has_token"`
	TokenHint string `json:"token_hint"`
}

type IntegrationsResponse struct {
	Month        string                `json:"month"`
	Integrations []IntegrationResponse `json:"integrations"`
}

// UpdateIntegrationRequest changes what is sent: Quota sets the monthly quota (0 = none), Token replaces the
// API token ("" removes the admin token).
type UpdateIntegrationRequest struct {
	Quota *int64  `json:"quota"`
	Token *string `json:"token"`
}

type UsageRequest struct {
	Amount int64 `json:"amount" binding:"required"`
}

type TokenResponse struct {
	Token string `json:"token"`
}
