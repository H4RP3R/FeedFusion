package api

import (
	"time"

	"gateway/pkg/models"
)

type Pagination struct {
	TotalPages  int `json:"total_pages"`
	CurrentPage int `json:"current_page"`
	Limit       int `json:"limit"`
}

type PostsResponse struct {
	Posts      []models.Post `json:"posts"`
	Pagination Pagination    `json:"pagination"`
}

type LogEntry struct {
	Timestamp  time.Time `json:"timestamp"`
	IP         string    `json:"ip"`
	StatusCode int       `json:"status_code"`
	RequestID  string    `json:"request_id"`
	Method     string    `json:"method"`
	Path       string    `json:"path"`
	Duration   float64   `json:"duration_sec"`
	Service    string    `json:"service"`
}
