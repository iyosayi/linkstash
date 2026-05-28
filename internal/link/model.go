package link

import "time"

type HealthResponse struct {
	Status string `json:"status"`
}

type CreateLinkRequest struct {
	URL string `json:"url"`
}

type LinkResponse struct {
	Code string `json:"code"`
	URL  string `json:"url"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type LinkStatResponse struct {
	Code           string     `json:"code"`
	URL            string     `json:"url"`
	ClickCount     int64      `json:"click_count"`
	CreatedAt      time.Time  `json:"created_at"`
	LastAccessedAt *time.Time `json:"last_accessed_at,omitempty"`
}
