package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"9itchi-statistics-api/models"
	"9itchi-statistics-api/models/api"
	"9itchi-statistics-api/services/mocks"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRecordEventHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mocks.StatisticsService{
		RecordEventFunc: func(_ context.Context, req api.EventRequest) (*api.EventResponse, *models.ApiError) {
			return &api.EventResponse{Name: req.Type, Day: "2026-01-01", Value: 1}, nil
		},
	}
	r := gin.New()
	r.POST("/internal/events", RecordEvent(svc))
	req := httptest.NewRequest(http.MethodPost, "/internal/events", strings.NewReader(`{"type":"email.sent"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusAccepted, rec.Code)
	require.Contains(t, rec.Body.String(), `"name":"email.sent"`)

	req = httptest.NewRequest(http.MethodPost, "/internal/events", strings.NewReader(`{"type":`))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.JSONEq(t, `{"error":"unexpected EOF"}`, rec.Body.String())
}
