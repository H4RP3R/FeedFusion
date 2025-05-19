package api

import "time"

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
