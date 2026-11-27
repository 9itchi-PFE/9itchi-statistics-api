# 9itchi Statistics API
Admin dashboard counters: daily rows in Postgres fed by RabbitMQ events. No Redis, no user data. Port 8090, DB `statistics`.

## Conventions (shared by every 9itchi Go service)
- `handlers/` thin (validate, authorize, call service, JSON) -> `services/` all logic, returns `(T, *models.ApiError)` -> `repositories/` only Postgres, plain errors (`ErrNotFound`, `ErrConflict`). Error body `{"error": "..."}`.
- DB calls use `repositories.QueryTimeout`; multi-table writes `Store.WithTx`.
- Schema: add `migrations/NNNN_name.sql` (goose, embedded, runs at startup). Never edit a merged migration.
- Auth: `Authorization: Bearer` AES-GCM-wrapped HS256 JWT (claims `id`, `email`, `role`, `typ`, `exp`; access only). `/admin/*` = `RequireRole("admin")`. `/internal/*` = `x-api-key` (`INTERNAL_API_KEY`), blocked by nginx.
- Events: exchange `9itchi.events` (topic, routing key = event name), names in `core/events.go`. Durable queue per consumer; handler error = one redelivery, then dropped. Empty `RABBITMQ_URL` = no-op consumer.
- Env only, validated in `utils/config.go`. Common: `PORT`, `ENVIRONMENT`, `DATABASE_URL`, `ITCHI_SECRET_KEY`, `ITCHI_ENCRYPTION_KEY`, `INTERNAL_API_KEY`, `INTERNAL_API_KEY_REQUIRED`. Never log tokens, passwords, PII.
- Tests: `{resource}_test.go`, testify, func-field fakes (no testify/mock), success + error paths. Repo tests: `//go:build integration`, embedded Postgres.
- Check: `gofmt -l .; go build ./... && go vet ./... && go test ./...`; `go test -tags integration -count=1 ./repositories/...`; `golangci-lint run`.

## Routes
- Admin: `GET /admin/statistics/overview|registrations|emails|subscriptions|revenue` with `from`, `to` (`YYYY-MM-DD`, default last 30 days, max 366). Credit tracker: `GET /admin/integrations` (month's `used`, `quota`, `percent`, `has_token`, `token_hint` per provider, plus live `balance {left, limit, resets_on}` or null), `PUT /admin/integrations/:provider` (`{quota?, token?}`, `token: ""` removes it).
- Internal: `POST /internal/events` (`{type, ts?}`, 400 on unknown type; no event id, no amount). `POST /internal/integrations/:provider/usage` (`{amount}`, 202) and `GET /internal/integrations/:provider/token` (404 = none set), called by ai-api and email-api. No user routes, no erase endpoint.

## Data, events
- Tables: `metrics` (`name`, `day`, `value`, PK `(name, day)`), `counted_events` (`event_id` PK), `integration_settings` (`provider`, `quota`, `token_enc`, `token_hint`), `integration_usage` (PK `(provider, month)`, `used`). Latest `0003_integrations.sql`. Env: `RABBITMQ_URL`, `HUNTER_API_KEY`, `APOLLO_API_KEY` (balance only).
- Consumes (queue `statistics-api.events`): `user.registered`, `email.sent`, `email.failed`, `email.opened`, `jobs.matched`, `subscription.created`, `payment.completed`, `cv.first_imported`, `preferences.first_saved`, `application.first_sent`, `application.replied`.
- Metrics: `user_registered`, `email_sent`, `email_failed`, `email_opened`, `jobs_matched`, `subscription_created`, `payment_completed`, `first_cv`, `first_preferences`, `first_application`, `replies`, `revenue_millimes`.

## Domain rules
- Count once: consumer puts the message id in `_event_id`; `counted_events` claims it in the same tx (`WithTransactor`); replays add nothing.
- Credit tracker: catalog in `services/integrations.go` (`gemini`, `openai` = tokens, `llamaindex` = documents, `hunter`, `apollo` = credits; add a provider there). Usage is per UTC month, quota 0 = none. Tokens are sealed with AES-GCM (`utils.Seal`), never returned by an admin route, only by the internal token route. ai-api reports gemini/openai/llamaindex and email-api reports hunter/apollo (Recruiter Radar); Mailgun and Ably are not tracked. Live balance (`pkg/balance`, factory map `Fetchers`): Hunter `GET /v2/account` (credits pool, else searches), Apollo `POST /api/v1/usage_stats/credit_usage_stats` (`lead_credit`; needs a master key or that endpoint in the key scope); both free. Admin token wins over the env key; key sent in a header only; answer or failure cached 5 min per provider (dropped when the token changes); failure = `balance: null`, logged with the provider only. The balance covers the whole provider account (JobAutopilot uses the same keys).
- Revenue = sum of `amount_millimes` (1/1000 TND) from `payment.completed`. No "active subscriptions" metric, only creations.
- Funnel (registered -> CV -> preferences -> 1st application -> paid) counts events per period, not a cohort. Days are UTC.

## Gotchas
- `pkg/account`, `pkg/payments` and `core` `LogPublisher` are unused leftovers (no peer URLs in config).
