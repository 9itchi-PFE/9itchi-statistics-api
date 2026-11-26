package handlers

import (
	"net/http"

	"9itchi-statistics-api/models"
	"9itchi-statistics-api/models/api"
	"9itchi-statistics-api/services"

	"github.com/gin-gonic/gin"
)

// ListIntegrations: GET /admin/integrations (admin) monthly consumption and quota of every AI provider and plugin.
func ListIntegrations(svc services.IntegrationService) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, apiErr := svc.List(c.Request.Context())
		if apiErr != nil {
			models.FormatError(c, apiErr)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

// UpdateIntegration: PUT /admin/integrations/:provider (admin) sets the monthly quota and/or replaces the API token.
func UpdateIntegration(svc services.IntegrationService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req api.UpdateIntegrationRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			models.FormatError(c, &models.ApiError{Code: http.StatusBadRequest, Error: err})
			return
		}
		result, apiErr := svc.Update(c.Request.Context(), c.Param("provider"), req)
		if apiErr != nil {
			models.FormatError(c, apiErr)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

// RecordIntegrationUsage: POST /internal/integrations/:provider/usage adds what a call consumed.
func RecordIntegrationUsage(svc services.IntegrationService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req api.UsageRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			models.FormatError(c, &models.ApiError{Code: http.StatusBadRequest, Error: err})
			return
		}
		if apiErr := svc.RecordUsage(c.Request.Context(), c.Param("provider"), req.Amount); apiErr != nil {
			models.FormatError(c, apiErr)
			return
		}
		c.Status(http.StatusAccepted)
	}
}

// IntegrationToken: GET /internal/integrations/:provider/token gives a service the token the admin set.
func IntegrationToken(svc services.IntegrationService) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, apiErr := svc.Token(c.Request.Context(), c.Param("provider"))
		if apiErr != nil {
			models.FormatError(c, apiErr)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}
