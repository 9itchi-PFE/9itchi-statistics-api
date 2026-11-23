package db

import "time"

// Metric maps table metrics.
type Metric struct {
	Name  string    `db:"name"`
	Day   time.Time `db:"day"`
	Value int64     `db:"value"`
}
