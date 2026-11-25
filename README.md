# 9itchi-statistics-api

PostgreSQL-backed daily counters for registrations, emails, job matches, subscription creations, and completed payments. Dashboard routes return event counts, not customer or monetary aggregates.

Port `8090`, health `/health`. Daily counters are stored in PostgreSQL. Dashboard date range query parameters are `from` and `to` (`YYYY-MM-DD`), defaulting to the last 30 days with a maximum span of 366 days. Event ingestion is internal-key protected.

## Endpoints

| Method | Path | Access | Handler |
|---|---|---|---|
| GET | `/admin/statistics/overview` | admin | Overview |
| GET | `/admin/statistics/registrations` | admin | Registrations |
| GET | `/admin/statistics/emails` | admin | Emails |
| GET | `/admin/statistics/subscriptions` | admin | Subscriptions |
| GET | `/admin/statistics/revenue` | admin | Revenue |
| POST | `/internal/events` | internal (`x-api-key`) | Increment a supported event counter |

Dashboard responses contain totals and per-day series for each metric. The subscriptions route counts creation events; revenue returns `payment_completed` (how many payments) and `revenue_millimes` (what they brought, in millimes: the `amount_millimes` of each `payment.completed` read from RabbitMQ; a payment recorded through `POST /internal/events` carries no amount and only counts). Supported event types: `user.registered`, `email.sent`, `email.failed`, `email.opened`, `jobs.matched`, `subscription.created`, `payment.completed`, and the three « first time » events below.

**From signing up to subscribing.** `cv.first_imported` (resume-api), `preferences.first_saved` (job-api) and `application.first_sent` (email-api) are published once per student, the first time they do it. They are counted as `first_cv`, `first_preferences` and `first_application`: how many students took that step each day. No user is stored here; the event's `user_id` is not read.

## Events (RabbitMQ)

Optional. With `RABBITMQ_URL` set, the service reads the durable queue `statistics-api.events`, bound on the topic exchange `9itchi.events` to the supported event types (routing key = event name), and counts each one for its day (`payload.ts`, else today). Without the variable nothing is consumed and `POST /internal/events` is the only way in.

A handler error asks the broker for one more delivery; a second failure drops the message. The consumer reconnects on its own and stops with the HTTP server. Events published while the service is down wait in the queue. Counting is not deduplicated: a message whose acknowledgement is lost is counted twice. Broker tests: `RABBITMQ_TEST_URL=amqp://guest:guest@localhost:5672/ go test -tags integration ./core/` (skipped without the variable).

## Run

```
cp .env.example .env
go run main.go
```
