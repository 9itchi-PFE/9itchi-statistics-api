package core

import (
	"context"

	"9itchi-statistics-api/utils"

	"go.uber.org/zap"
)

// Event names exchanged over RabbitMQ (exchange "9itchi.events"). Never rename a published name.
const (
	EventUserRegistered      = "user.registered"
	EventEmailSent           = "email.sent"
	EventEmailFailed         = "email.failed"
	EventEmailOpened         = "email.opened"
	EventJobsMatched         = "jobs.matched"
	EventSubscriptionCreated = "subscription.created"
	EventPaymentCompleted    = "payment.completed"
	// The first time a student does something (each service publishes it once per student): the steps between
	// signing up and subscribing. No user is stored here, only how many did it each day.
	EventCVFirstImported       = "cv.first_imported"
	EventPreferencesFirstSaved = "preferences.first_saved"
	EventApplicationFirstSent  = "application.first_sent"
)

// Publisher sends domain events. Swap LogPublisher for a RabbitMQ implementation without touching services.
type Publisher interface {
	Publish(ctx context.Context, event string, payload any) error
}

// LogPublisher logs the event name only (never the payload). Default until RabbitMQ is wired.
type LogPublisher struct{}

func (LogPublisher) Publish(_ context.Context, event string, _ any) error {
	utils.Logger.Info("event", zap.String("name", event))
	return nil
}
