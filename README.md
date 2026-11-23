# 9itchi-statistics-api

Admin dashboard statistics: clients, registrations, emails sent/opened, active subscriptions, revenue. Consumes events, caches in Redis.

Port `8090`, health `/health`. Skeleton only: every endpoint returns `501 not implemented` until built.

## Endpoints

| Method | Path | Access | Handler |
|---|---|---|---|
| GET | `/admin/statistics/overview` | admin | Overview |
| GET | `/admin/statistics/registrations` | admin | Registrations |
| GET | `/admin/statistics/emails` | admin | Emails |
| GET | `/admin/statistics/subscriptions` | admin | Subscriptions |
| GET | `/admin/statistics/revenue` | admin | Revenue |

## Run

```
cp .env.example .env
go run main.go
```
