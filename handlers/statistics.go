package handlers

import (
	"net/http"

	"9itchi-statistics-api/models"
	"9itchi-statistics-api/services"

	"github.com/gin-gonic/gin"
)

// Overview: GET /admin/statistics/overview (admin) headline numbers.
func Overview(svc services.StatisticsService) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, apiErr := svc.Overview(c.Request.Context(), c.GetString("user_id"))
		if apiErr != nil {
			models.FormatError(c, apiErr)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

// Registrations: GET /admin/statistics/registrations (admin) new users over time.
func Registrations(svc services.StatisticsService) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, apiErr := svc.Registrations(c.Request.Context(), c.GetString("user_id"))
		if apiErr != nil {
			models.FormatError(c, apiErr)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

// Emails: GET /admin/statistics/emails (admin) sent vs opened.
func Emails(svc services.StatisticsService) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, apiErr := svc.Emails(c.Request.Context(), c.GetString("user_id"))
		if apiErr != nil {
			models.FormatError(c, apiErr)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

// Subscriptions: GET /admin/statistics/subscriptions (admin) active subscriptions by plan.
func Subscriptions(svc services.StatisticsService) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, apiErr := svc.Subscriptions(c.Request.Context(), c.GetString("user_id"))
		if apiErr != nil {
			models.FormatError(c, apiErr)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

// Revenue: GET /admin/statistics/revenue (admin) revenue over time.
func Revenue(svc services.StatisticsService) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, apiErr := svc.Revenue(c.Request.Context(), c.GetString("user_id"))
		if apiErr != nil {
			models.FormatError(c, apiErr)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}
