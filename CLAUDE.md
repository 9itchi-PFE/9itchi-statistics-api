# 9itchi Statistics API

Admin dashboard statistics: clients, registrations, emails sent/opened, active subscriptions, revenue. Consumes events and keeps the figures in PostgreSQL (there is no Redis).

## Stack
Go, Gin, PostgreSQL (db `statistics`, pgx + goose migrations), Viper config, Zap logger. Peers over HTTP (`pkg/`): account, payments. Events over RabbitMQ (names in `core/events.go`).

## Layout
`main.go`, `app/` (init order: config, repositories, services, router), `router/` (routes + middleware), `handlers/` (thin) -> `services/` (logic) -> `repositories/` (Postgres only), `core/` (interfaces/factories), `models/{api,db,mapping}`, `pkg/` (peer clients), `utils/` (config, logger).

Entities: `Metric`.

## Rules
- Every endpoint: validate inputs -> authorize caller -> execute via service -> respond JSON. No business logic in handlers.
- Services return `(T, *models.ApiError)`; repos return plain errors; services wrap with a status code. Error body is always `{"error": "message"}`.
- Users are referenced by their UUID `id` (JWT claim `id`). JWT claims: `id`, `email`, `role`, `typ`, `exp`. Header `Authorization: Bearer <AES-GCM encrypted HS256 JWT>`.
- Every DB call uses a context with a deadline (`repositories.QueryTimeout`). Schema and indexes live in `migrations/*.sql` (goose, embedded, run at startup). Multi-table writes use `Store.WithTx`. Never select blob columns in list queries.
- Nil-check optional nested fields. Never log passwords, tokens, Gmail tokens or PII. Secrets and URLs only from env.
- `AuthMiddleware` validates the encrypted JWT (access tokens only); internal routes need `x-api-key`.
- Tests: `{resource}_test.go`, testify, mock dependencies, cover success and error paths. Write the test with the implementation.

## Commands
`go build ./... && go test ./...`, `golangci-lint run`, `go run main.go` (port 8090), `docker build -t 9itchi-statistics-api .`.
