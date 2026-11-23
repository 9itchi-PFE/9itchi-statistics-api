package mapping

import (
	"time"

	"9itchi-statistics-api/models/api"
	"9itchi-statistics-api/models/db"
)

func MetricToEventResponse(metric *db.Metric) *api.EventResponse {
	return &api.EventResponse{
		Name:  metric.Name,
		Day:   metric.Day.Format(time.DateOnly),
		Value: metric.Value,
	}
}
