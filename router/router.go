package router

import (
	"9itchi-statistics-api/handlers"
	"9itchi-statistics-api/services"

	"github.com/gin-gonic/gin"
)

func InitializeRouter(services *services.Services) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	initializeMiddlewares(r)
	initializeRoutes(r, services)

	return r
}

func initializeRoutes(r *gin.Engine, services *services.Services) {
	// Health endpoints
	r.GET("/", handlers.GetHealth)
	r.GET("/health", handlers.GetHealth)

	// admin endpoints
	adminGroup := r.Group("/")
	adminGroup.Use(AuthMiddleware(), RequireRole("admin"))
	{
		adminGroup.GET("/admin/statistics/overview", handlers.Overview(services.Statistics))
		adminGroup.GET("/admin/statistics/registrations", handlers.Registrations(services.Statistics))
		adminGroup.GET("/admin/statistics/emails", handlers.Emails(services.Statistics))
		adminGroup.GET("/admin/statistics/subscriptions", handlers.Subscriptions(services.Statistics))
		adminGroup.GET("/admin/statistics/revenue", handlers.Revenue(services.Statistics))
	}

	// Fallback
	r.NoRoute(handlers.NoRoute)
}
