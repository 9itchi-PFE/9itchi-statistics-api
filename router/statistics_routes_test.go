package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"9itchi-statistics-api/models"
	"9itchi-statistics-api/models/api"
	"9itchi-statistics-api/services"
	"9itchi-statistics-api/services/mocks"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestInternalEventRouteRequiresAPIKey(t *testing.T) {
	t.Setenv("INTERNAL_API_KEY", "internal-secret")
	t.Setenv("INTERNAL_API_KEY_REQUIRED", "true")
	gin.SetMode(gin.TestMode)
	svc := &mocks.StatisticsService{
		RecordEventFunc: func(_ context.Context, req api.EventRequest) (*api.EventResponse, *models.ApiError) {
			return &api.EventResponse{Name: req.Type, Day: "2026-01-01", Value: 1}, nil
		},
	}
	r := InitializeRouter(&services.Services{Statistics: svc})

	request := httptest.NewRequest(http.MethodPost, "/internal/events", strings.NewReader(`{"type":"email.sent"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusUnauthorized, recorder.Code)

	request = httptest.NewRequest(http.MethodPost, "/internal/events", strings.NewReader(`{"type":"email.sent"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("x-api-key", "internal-secret")
	recorder = httptest.NewRecorder()
	r.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusAccepted, recorder.Code)
}
