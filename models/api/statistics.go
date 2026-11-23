package api

import "time"

type EventRequest struct {
	Type string     `json:"type" binding:"required"`
	TS   *time.Time `json:"ts"`
}

type EventResponse struct {
	Name  string `json:"name"`
	Day   string `json:"day"`
	Value int64  `json:"value"`
}

type DateRangeQuery struct {
	From string
	To   string
}

type MetricPoint struct {
	Day   string `json:"day"`
	Value int64  `json:"value"`
}

type MetricSummary struct {
	Total  int64         `json:"total"`
	Series []MetricPoint `json:"series"`
}

type DashboardResponse struct {
	From    string                   `json:"from"`
	To      string                   `json:"to"`
	Metrics map[string]MetricSummary `json:"metrics"`
}
