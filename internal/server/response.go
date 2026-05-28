package server

import (
	"encoding/json"
	"github.com/iyosayi/linkstash/internal/link"
	"log"
	"net/http"
)

func writeError(w http.ResponseWriter, status int, message string) {
	if err := writeJSON(w, status, link.ErrorResponse{Error: message}); err != nil {
		log.Printf("write error response: %v", err)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}
