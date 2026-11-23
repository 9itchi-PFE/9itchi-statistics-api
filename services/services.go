package services

import (
	"9itchi-statistics-api/pkg/account"
	"9itchi-statistics-api/pkg/payments"
	"9itchi-statistics-api/repositories"
	"9itchi-statistics-api/utils"
)

// Services holds service implementations and the peer clients they call.
type Services struct {
	Statistics StatisticsService
	Clients    *Clients
}

// Clients holds one HTTP client per peer service (singletons, 50s timeout).
type Clients struct {
	Account  *account.Client
	Payments *payments.Client
}

func InitServices(config utils.AppConfig, repos *repositories.Repositories) *Services {
	return &Services{
		Statistics: NewStatisticsService(repos.Statistics),
		Clients: &Clients{
			Account:  account.New(config.AccountAPIURL, config.InternalAPIKey),
			Payments: payments.New(config.PaymentsAPIURL, config.InternalAPIKey),
		},
	}
}
