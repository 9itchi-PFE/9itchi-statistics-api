-- +goose Up
CREATE TABLE integration_settings (
  provider text PRIMARY KEY,
  quota bigint NOT NULL DEFAULT 0 CHECK (quota >= 0),
  token_enc text NOT NULL DEFAULT '',
  token_hint text NOT NULL DEFAULT '',
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE integration_usage (
  provider text NOT NULL,
  month date NOT NULL,
  used bigint NOT NULL DEFAULT 0 CHECK (used >= 0),
  PRIMARY KEY (provider, month)
);

-- +goose Down
DROP TABLE integration_usage;
DROP TABLE integration_settings;
