# 9itchi-statistics-api

PostgreSQL-backed daily counters for registrations, emails, job matches, subscription creations, and completed payments. Dashboard routes return event counts, not customer or monetary aggregates. RabbitMQ is not wired; the default consumer is a no-op.

Port `8090`, health `/health`. Daily counters are stored in PostgreSQL. Dashboard date range query parameters are `from` and `to` (`YYYY-MM-DD`), defaulting to the last 30 days with a maximum span of 366 days. Event ingestion is internal-key protected. RabbitMQ is not wired; the default consumer is a no-op.

## Endpoints

| Method | Path | Access | Handler |
|---|---|---|---|
| GET | `/admin/statistics/overview` | admin | Overview |
| GET | `/admin/statistics/registrations` | admin | Registrations |
| GET | `/admin/statistics/emails` | admin | Emails |
| GET | `/admin/statistics/subscriptions` | admin | Subscriptions |
| GET | `/admin/statistics/revenue` | admin | Revenue |
| POST | `/internal/events` | internal (`x-api-key`) | Increment a supported event counter |

Dashboard responses contain totals and per-day series for each metric. The subscriptions route counts creation events; revenue represents completed payment count, not monetary amounts. Supported event types: `user.registered`, `email.sent`, `email.failed`, `email.opened`, `jobs.matched`, `subscription.created`, `payment.completed`.

## Run

```
cp .env.example .env
go run main.go
```
