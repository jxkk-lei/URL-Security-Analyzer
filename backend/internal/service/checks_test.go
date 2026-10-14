package service

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"url-security-analyzer/backend/internal/model"
)

type fakeRepository struct {
	created model.Check
	createCalls int
}

func (r *fakeRepository) Create(
	_ context.Context,
	check model.Check,
) (model.Check, error) {
	r.createCalls++
	r.created = check
	check.ID = 1
	check.CheckedAt = time.Now()
	return check, nil
}

func (r *fakeRepository) List(
	_ context.Context,
	_ int,
) ([]model.Check, error) {
	return []model.Check{}, nil
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(
	req *http.Request,
) (*http.Response, error) {
	return f(req)
}

func newTestService(
	repo *fakeRepository,
	statusCode int,
	requestErr error,
) *ChecksService {
	client := &http.Client{
		Transport: roundTripFunc(func(
			req *http.Request,
		) (*http.Response, error) {
			if requestErr != nil {
				return nil, requestErr
			}

			return &http.Response{
				StatusCode: statusCode,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("")),
				Request:    req,
			}, nil
		}),
	}

	return &ChecksService{
		Repository: repo,
		Client:     client,
	}
}

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "valid HTTPS URL",
			input: "https://example.com",
			want:  "https://example.com",
		},
		{
			name:  "trim spaces and normalize host",
			input: "  https://EXAMPLE.COM/path#section  ",
			want:  "https://example.com/path",
		},
		{
			name:    "reject FTP",
			input:   "ftp://example.com",
			wantErr: true,
		},
		{
			name:    "reject missing host",
			input:   "https://",
			wantErr: true,
		},
		{
			name:    "reject localhost",
			input:   "http://localhost",
			wantErr: true,
		},
		{
			name:    "reject loopback IP",
			input:   "http://127.0.0.1",
			wantErr: true,
		},
		{
			name:    "reject private IP",
			input:   "http://192.168.1.10",
			wantErr: true,
		},
		{
			name:    "reject unsupported port",
			input:   "https://example.com:8443",
			wantErr: true,
		},
		{
			name:    "reject URL credentials",
			input:   "https://user:pass@example.com",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeURL(tt.input)

			if (err != nil) != tt.wantErr {
				t.Fatalf("normalizeURL() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err == nil && got != tt.want {
				t.Errorf("normalizeURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCheckURLHTTPStatus(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		available bool
	}{
		{
			name:      "200 is available",
			status:    http.StatusOK,
			available: true,
		},
		{
			name:      "302 is available",
			status:    http.StatusFound,
			available: true,
		},
		{
			name:      "404 is unavailable",
			status:    http.StatusNotFound,
			available: false,
		},
		{
			name:      "500 is unavailable",
			status:    http.StatusInternalServerError,
			available: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepository{}
			svc := newTestService(repo, tt.status, nil)

			got, err := svc.CheckURL(
				context.Background(),
				"https://example.com",
			)
			if err != nil {
				t.Fatalf("CheckURL() error = %v", err)
			}

			if got.StatusCode != tt.status {
				t.Errorf("StatusCode = %d, want %d", got.StatusCode, tt.status)
			}

			if got.Available != tt.available {
				t.Errorf("Available = %v, want %v", got.Available, tt.available)
			}

			if repo.createCalls != 1 {
				t.Errorf("Create calls = %d, want 1", repo.createCalls)
			}

			if repo.created.URL != "https://example.com" {
				t.Errorf("saved URL = %q", repo.created.URL)
			}
		})
	}
}

func TestCheckURLNetworkError(t *testing.T) {
	repo := &fakeRepository{}
	svc := newTestService(
		repo,
		0,
		errors.New("connection refused"),
	)

	got, err := svc.CheckURL(
		context.Background(),
		"https://example.com",
	)
	if err != nil {
		t.Fatalf("CheckURL() error = %v", err)
	}

	if got.Available {
		t.Error("Available = true, want false")
	}

	if got.ErrorMessage == "" {
		t.Error("expected network error message")
	}

	if repo.createCalls != 1 {
		t.Errorf("Create calls = %d, want 1", repo.createCalls)
	}
}

func TestCheckURLRejectsInvalidURL(t *testing.T) {
	repo := &fakeRepository{}
	svc := newTestService(repo, http.StatusOK, nil)

	_, err := svc.CheckURL(
		context.Background(),
		"http://127.0.0.1",
	)

	if !errors.Is(err, ErrInvalidURL) {
		t.Fatalf("error = %v, want ErrInvalidURL", err)
	}

	if repo.createCalls != 0 {
		t.Errorf("Create calls = %d, want 0", repo.createCalls)
	}
}