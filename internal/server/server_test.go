package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/iyosayi/linkstash/internal/link"
)

type fakeStore struct {
	links map[string]link.LinkResponse
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		links: make(map[string]link.LinkResponse),
	}
}

func (f *fakeStore) Create(ctx context.Context, url string) (link.LinkResponse, error) {
	resp := link.LinkResponse{
		Code: "abc123",
		URL:  url,
	}

	f.links[resp.Code] = resp
	return resp, nil
}

func (f *fakeStore) Get(ctx context.Context, code string) (link.LinkResponse, bool, error) {
	resp, ok := f.links[code]
	return resp, ok, nil
}

func (f *fakeStore) GetStats(ctx context.Context, code string) (link.LinkStatResponse, bool, error) {
	resp, ok := f.links[code]
	if !ok {
		return link.LinkStatResponse{}, false, nil
	}

	return link.LinkStatResponse{
		Code:       resp.Code,
		URL:        resp.URL,
		ClickCount: 0,
	}, true, nil
}

func (f *fakeStore) IncrementStats(ctx context.Context, code string) error {
	return nil
}

func TestCreateLinksValidation(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantError  string
	}{
		{
			name:       "invalid json",
			body:       `{invalid json}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid json",
		},
		{
			name:       "empty url",
			body:       `{"url":""}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "url is required",
		},
		{
			name:       "invalid url",
			body:       `{"url":"hello"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "url must be valid http or https URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newFakeStore()
			logger := slog.Default()
			server := NewServer(store, logger)

			req := httptest.NewRequest(
				http.MethodPost,
				"/links",
				bytes.NewReader([]byte(tt.body)),
			)

			rec := httptest.NewRecorder()

			server.createLinks(rec, req)
			if rec.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}

			var resp link.ErrorResponse
			if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
				t.Fatalf("decode response: %v", err)
			}

			if resp.Error != tt.wantError {
				t.Fatalf("expected error %q, got %q", tt.wantError, resp.Error)
			}
		})
	}
}

func TestCreateLinks(t *testing.T) {
	store := newFakeStore()
	logger := slog.Default()
	server := NewServer(store, logger)

	body := []byte(`{"url":"https://example.com"}`)

	req := httptest.NewRequest(
		http.MethodPost,
		"/links",
		bytes.NewReader(body),
	)

	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()

	server.createLinks(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}

	var resp link.LinkResponse

	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.URL != "https://example.com" {
		t.Fatalf("expected URL to match")
	}

	if resp.Code == "" {
		t.Fatalf("expected generated code")
	}
}

func TestRedirectLink(t *testing.T) {
	store := newFakeStore()
	logger := slog.Default()
	server := NewServer(store, logger)
	rawURL := "https://example.com"

	link, err := store.Create(context.Background(), rawURL)

	if err != nil {
		t.Fatalf("create link: %v", err)
	}

	if link.Code == "" {
		t.Fatalf("expected generated code")
	}
	req := httptest.NewRequest(
		http.MethodGet,
		fmt.Sprintf("/%s", link.Code),
		nil,
	)

	req.SetPathValue("code", link.Code)

	rec := httptest.NewRecorder()

	server.redirectLink(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected status %d, got %d", http.StatusFound, rec.Code)
	}

	location := rec.Header().Get("Location")

	if location != rawURL {
		t.Fatalf("expected location to be %s, got %s", rawURL, location)
	}
}

func TestGetLinkStats(t *testing.T) {
	store := newFakeStore()
	logger := slog.Default()
	server := NewServer(store, logger)

	rawURL := "https://example.com"

	created, err := store.Create(context.Background(), rawURL)

	if err != nil {
		t.Fatalf("create link: %v", err)
	}

	if created.Code == "" {
		t.Fatalf("expected generated code")
	}

	req := httptest.NewRequest(
		http.MethodGet,
		fmt.Sprintf("/links/%s/stats", created.Code),
		nil,
	)

	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)

	server.getLinkStats(rec, req)

	var resp link.LinkStatResponse

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}

	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.URL != rawURL {
		t.Fatalf("expected url to be %s, got %s", rawURL, resp.URL)
	}

	if resp.Code != created.Code {
		t.Fatalf("expected code to be %s, got %s", created.Code, resp.Code)
	}

}

func TestRecoveryMiddleware(t *testing.T) {
	logger := slog.Default()

	handler := recoveryMiddleware(logger, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	}))

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestRequestIDMiddleware(t *testing.T) {
	handler := requestIDMiddlware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := requestIDFromContext(r.Context())
		if id == "" {
			t.Fatal("expected request id")
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Header().Get("X-Request-ID") == "" {
		t.Fatal("expected X-Request-ID header")
	}
}
