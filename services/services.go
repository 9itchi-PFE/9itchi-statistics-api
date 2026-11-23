package services

import (
	"9itchi-statistics-api/core"
	"9itchi-statistics-api/repositories"
)

// Services holds service implementations and the peer clients they call.
type Services struct {
	Statistics StatisticsService
	Consumer   core.EventConsumer
}

func InitServices(repos *repositories.Repositories) *Services {
	return &Services{
		Statistics: NewStatisticsService(repos.Statistics),
		Consumer:   core.NoopEventConsumer{},
	}
}
