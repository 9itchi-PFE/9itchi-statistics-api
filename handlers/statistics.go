package handlers

import (
	"net/http"

	"9itchi-statistics-api/models"
	"9itchi-statistics-api/models/api"
	"9itchi-statistics-api/services"

	"github.com/gin-gonic/gin"
)

// Overview: GET /admin/statistics/overview (admin) headline numbers.
func Overview(svc services.StatisticsService) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, apiErr := svc.Overview(c.Request.Context(), dateRange(c))
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
		result, apiErr := svc.Registrations(c.Request.Context(), dateRange(c))
		if apiErr != nil {
			models.FormatError(c, apiErr)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

// Emails: GET /admin/statistics/emails (admin) sent, failed, and opened events.
func Emails(svc services.StatisticsService) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, apiErr := svc.Emails(c.Request.Context(), dateRange(c))
		if apiErr != nil {
			models.FormatError(c, apiErr)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

// Subscriptions: GET /admin/statistics/subscriptions (admin) subscription creation events.
func Subscriptions(svc services.StatisticsService) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, apiErr := svc.Subscriptions(c.Request.Context(), dateRange(c))
		if apiErr != nil {
			models.FormatError(c, apiErr)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

// Revenue: GET /admin/statistics/revenue (admin) completed payment count over time.
func Revenue(svc services.StatisticsService) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, apiErr := svc.Revenue(c.Request.Context(), dateRange(c))
		if apiErr != nil {
			models.FormatError(c, apiErr)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

func RecordEvent(svc services.StatisticsService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req api.EventRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			models.FormatError(c, &models.ApiError{Code: http.StatusBadRequest, Error: err})
			return
		}
		result, apiErr := svc.RecordEvent(c.Request.Context(), req)
		if apiErr != nil {
			models.FormatError(c, apiErr)
			return
		}
		c.JSON(http.StatusAccepted, result)
	}
}

func dateRange(c *gin.Context) api.DateRangeQuery {
	return api.DateRangeQuery{From: c.Query("from"), To: c.Query("to")}
}
