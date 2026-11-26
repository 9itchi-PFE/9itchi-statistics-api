package db

import "time"

// IntegrationSetting maps table integration_settings. TokenEnc is the AES-GCM sealed API token.
type IntegrationSetting struct {
	Provider  string    `db:"provider"`
	Quota     int64     `db:"quota"`
	TokenEnc  string    `db:"token_enc"`
	TokenHint string    `db:"token_hint"`
	UpdatedAt time.Time `db:"updated_at"`
}

// IntegrationUsage maps table integration_usage: what a provider consumed in a month (first day).
type IntegrationUsage struct {
	Provider string    `db:"provider"`
	Month    time.Time `db:"month"`
	Used     int64     `db:"used"`
}
