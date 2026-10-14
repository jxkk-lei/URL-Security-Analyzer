package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"url-security-analyzer/backend/internal/model"
	"url-security-analyzer/backend/internal/service"
)

type testRepository struct {
	checks    []model.Check
	createErr error
	listErr   error
}

func (r *testRepository) Create(
	_ context.Context,
	check model.Check,
) (model.Check, error) {
	if r.createErr != nil {
		return model.Check{}, r.createErr
	}

	check.ID = 1
	r.checks = append(r.checks, check)
	return check, nil
}

func (r *testRepository) List(
	_ context.Context,
	limit int,
) ([]model.Check, error) {
	if r.listErr != nil {
		return nil, r.listErr
	}

	if limit < len(r.checks) {
		return r.checks[:limit], nil
	}

	return r.checks, nil
}

type testRoundTripper struct{}

func (testRoundTripper) RoundTrip(
	req *http.Request,
) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader("")),
		Request:    req,
	}, nil
}

func newTestChecksHandler(
	repo *testRepository,
) *ChecksHandler {
	svc := &service.ChecksService{
		Repository: repo,
		Client: &http.Client{
			Transport: testRoundTripper{},
		},
	}

	return &ChecksHandler{Service: svc}
}

func TestCreateCheck(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "valid request",
			body:       `{"url":"https://example.com"}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "empty URL",
			body:       `{"url":""}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid JSON",
			body:       `{`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "unsupported scheme",
			body:       `{"url":"ftp://example.com"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "unknown field",
			body:       `{"url":"https://example.com","extra":true}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "local IP",
			body:       `{"url":"http://127.0.0.1"}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &testRepository{}
			h := newTestChecksHandler(repo)

			req := httptest.NewRequest(
				http.MethodPost,
				"/api/v1/checks",
				strings.NewReader(tt.body),
			)
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()
			h.Create(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf(
					"status = %d, want %d; body: %s",
					rec.Code,
					tt.wantStatus,
					rec.Body.String(),
				)
			}

			if tt.wantStatus == http.StatusCreated {
				if !strings.Contains(
					rec.Header().Get("Content-Type"),
					"application/json",
				) {
					t.Error("expected JSON Content-Type")
				}

				var result model.Check
				if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
					t.Fatalf("invalid JSON response: %v", err)
				}

				if result.URL != "https://example.com" {
					t.Errorf("URL = %q", result.URL)
				}

				if result.StatusCode != http.StatusOK {
					t.Errorf("StatusCode = %d", result.StatusCode)
				}

				if len(repo.checks) != 1 {
					t.Errorf("saved checks = %d, want 1", len(repo.checks))
				}
			}
		})
	}
}

func TestCreateCheckRepositoryError(t *testing.T) {
	repo := &testRepository{
		createErr: errors.New("database unavailable"),
	}
	h := newTestChecksHandler(repo)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/checks",
		strings.NewReader(`{"url":"https://example.com"}`),
	)
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}

func TestListChecks(t *testing.T) {
	repo := &testRepository{
		checks: []model.Check{
			{ID: 1, URL: "https://example.com", Available: true},
			{ID: 2, URL: "https://example.org", Available: true},
		},
	}
	h := newTestChecksHandler(repo)

	tests := []struct {
		name       string
		target     string
		wantStatus int
		wantCount  int
	}{
		{
			name:       "default limit",
			target:     "/api/v1/checks",
			wantStatus: http.StatusOK,
			wantCount:  2,
		},
		{
			name:       "limit one",
			target:     "/api/v1/checks?limit=1",
			wantStatus: http.StatusOK,
			wantCount:  1,
		},
		{
			name:       "limit zero",
			target:     "/api/v1/checks?limit=0",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "limit over maximum",
			target:     "/api/v1/checks?limit=101",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "non-numeric limit",
			target:     "/api/v1/checks?limit=abc",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.target, nil)
			rec := httptest.NewRecorder()

			h.List(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf(
					"status = %d, want %d; body: %s",
					rec.Code,
					tt.wantStatus,
					rec.Body.String(),
				)
			}

			if tt.wantStatus != http.StatusOK {
				return
			}

			var result []model.Check
			if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
				t.Fatalf("invalid JSON response: %v", err)
			}

			if len(result) != tt.wantCount {
				t.Errorf("result count = %d, want %d", len(result), tt.wantCount)
			}
		})
	}
}

func TestListChecksRepositoryError(t *testing.T) {
	repo := &testRepository{
		listErr: errors.New("database unavailable"),
	}
	h := newTestChecksHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/checks", nil)
	rec := httptest.NewRecorder()

	h.List(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}
