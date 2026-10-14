package model

import "time"

type Check struct {
	ID             int64     `json:"id"`
	URL            string    `json:"url"`
	Available      bool      `json:"available"`
	StatusCode     int       `json:"status_code,omitempty"`
	ResponseTimeMS int64     `json:"response_time_ms"`
	ErrorMessage   string    `json:"error_message,omitempty"`
	CheckedAt      time.Time `json:"checked_at"`
}
