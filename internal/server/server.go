package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/iyosayi/linkstash/internal/link"
)

type Server struct {
	store  link.LinkStore
	logger *slog.Logger
}

func NewServer(store link.LinkStore, logger *slog.Logger) *Server {
	return &Server{
		store:  store,
		logger: logger,
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", s.getHealth)
	mux.HandleFunc("POST /links", s.createLinks)
	mux.HandleFunc("GET /links/{code}", s.getLinkByCode)
	mux.HandleFunc("GET /{code}", s.redirectLink)
	mux.HandleFunc("GET /links/{code}/stats", s.getLinkStats)

	return recoveryMiddleware(s.logger, requestIDMiddlware(
		loggingMiddleware(s.logger, mux),
	))
}

func isValidURL(rawURL string) bool {
	u, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return false
	}
	return u.Scheme == "http" || u.Scheme == "https"
}

func (s *Server) getHealth(w http.ResponseWriter, r *http.Request) {
	resp := link.HealthResponse{
		Status: "ok",
	}

	if err := writeJSON(w, http.StatusOK, resp); err != nil {
		s.logger.Error("write response", "error", err)
	}
}

func (s *Server) createLinks(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var req link.CreateLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	if req.URL == "" {
		writeError(w, http.StatusBadRequest, "url is required")
		return
	}

	if !isValidURL(req.URL) {
		writeError(w, http.StatusBadRequest, "url must be valid http or https URL")
		return
	}

	ctx := r.Context()
	resp, err := s.store.Create(ctx, req.URL)

	if err != nil {
		writeError(w, http.StatusInternalServerError, "error occured creating link")
		return
	}

	if err := writeJSON(w, http.StatusCreated, resp); err != nil {
		s.logger.Error("write response", "error", err)
	}
}

func (s *Server) getLinkByCode(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	if code == "" {
		writeError(w, http.StatusBadRequest, "code is required")
		return
	}

	ctx := r.Context()
	link, ok, err := s.store.Get(ctx, code)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "error occured retrieving link")
		return
	}

	if !ok {
		writeError(w, http.StatusNotFound, "link not found")
		return
	}

	if err := writeJSON(w, http.StatusOK, link); err != nil {
		s.logger.Error("write response", "error", err)
	}
}

func (s *Server) redirectLink(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")

	if code == "" {
		writeError(w, http.StatusBadRequest, "code is required")
		return
	}

	ctx := r.Context()
	link, ok, err := s.store.Get(ctx, code)

	if err != nil {
		writeError(w, http.StatusInternalServerError, "an error occured")
		return
	}

	if !ok {
		writeError(w, http.StatusNotFound, "link not found")
		return
	}

	if err := s.store.IncrementStats(ctx, code); err != nil {
		s.logger.Error("increment stats", "error", err)
	}

	http.Redirect(w, r, link.URL, http.StatusFound)
}

func (s *Server) getLinkStats(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")

	if code == "" {
		writeError(w, http.StatusBadRequest, "code is required")
		return
	}

	ctx := r.Context()
	stats, ok, err := s.store.GetStats(ctx, code)

	if err != nil {
		writeError(w, http.StatusInternalServerError, "error retrieving stats")
		return
	}

	if !ok {
		writeError(w, http.StatusNotFound, "no stats for this link")
		return
	}
	if err := writeJSON(w, http.StatusOK, stats); err != nil {
		s.logger.Error("write response", "error", err)
	}
}
