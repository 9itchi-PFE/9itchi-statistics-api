package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"9itchi-statistics-api/core"
	"9itchi-statistics-api/repositories"
	"9itchi-statistics-api/router"
	"9itchi-statistics-api/services"
	"9itchi-statistics-api/utils"

	"github.com/gin-gonic/gin"
)

type App struct {
	Config     utils.AppConfig
	Router     *gin.Engine
	Repos      *repositories.Repositories
	Consumer   core.EventConsumer
	Statistics services.StatisticsService
}

func New() *App {
	app := &App{}
	app.setup()
	return app
}

// setup order: config -> repositories (connect + migrate) -> services -> router.
func (app *App) setup() {
	config := utils.LoadConfig()
	if err := config.Validate(); err != nil {
		log.Fatalf("invalid configuration: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	repos, err := repositories.InitRepositories(ctx, config)
	if err != nil {
		log.Fatalf("repositories: %v", err)
	}

	servicesWrapper := services.InitServices(config, repos)

	app.Config = config
	app.Repos = repos
	app.Consumer = servicesWrapper.Consumer
	app.Statistics = servicesWrapper.Statistics
	app.Router = router.InitializeRouter(servicesWrapper)
}

// Run serves until SIGINT/SIGTERM, then drains requests and closes the database pool.
func (app *App) Run() {
	workerCtx, stopWorkers := context.WithCancel(context.Background())
	consumerDone := make(chan error, 1)
	go func() {
		consumerDone <- app.Consumer.Run(workerCtx, func(ctx context.Context, event string, payload map[string]any) error {
			return app.Statistics.ConsumeEvent(ctx, event, payload)
		})
	}()

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", app.Config.Port),
		Handler:           app.Router,
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	stopWorkers()
	_ = srv.Shutdown(ctx)
	select {
	case err := <-consumerDone:
		if err != nil {
			log.Printf("event consumer: %v", err)
		}
	case <-ctx.Done():
		log.Printf("event consumer shutdown timed out")
	}
	app.Repos.Close()
}
