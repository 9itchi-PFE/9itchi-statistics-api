package services

import (
	"9itchi-statistics-api/core"
	"9itchi-statistics-api/repositories"
	"9itchi-statistics-api/utils"
)

// EventsQueue is the durable queue the service reads its events from.
const EventsQueue = "statistics-api.events"

// CountedEvents are the events the dashboard counts: the routing keys of the queue.
var CountedEvents = []string{
	core.EventUserRegistered, core.EventEmailSent, core.EventEmailFailed, core.EventEmailOpened,
	core.EventJobsMatched, core.EventSubscriptionCreated, core.EventPaymentCompleted,
	core.EventCVFirstImported, core.EventPreferencesFirstSaved, core.EventApplicationFirstSent,
}

// Services holds service implementations and the peer clients they call.
type Services struct {
	Statistics StatisticsService
	Consumer   core.EventConsumer
}

func InitServices(config utils.AppConfig, repos *repositories.Repositories) *Services {
	return &Services{
		Statistics: NewStatisticsService(repos.Statistics),
		Consumer:   core.NewEventConsumer(config.RabbitMQURL, EventsQueue, CountedEvents),
	}
}
