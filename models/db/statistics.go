package db

import "time"

// Metric maps table metrics.
type Metric struct {
	ID         string            `db:"id" json:"id"`
	Key        string            `db:"key" json:"key"`
	Date       time.Time         `db:"date" json:"date"`
	Value      float64           `db:"value" json:"value"`
	Dimensions map[string]string `db:"dimensions" json:"dimensions"`
	CreatedAt  time.Time         `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time         `db:"updated_at" json:"updated_at"`
}
