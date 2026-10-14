package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ChecksHandler struct {
	DB *pgxpool.Pool
}

type CreateCheckRequest struct {
	URL string `json:"url"`
}

type CreateCheckResponse struct {
	ID             int64  `json:"id"`
	URL            string `json:"url"`
	Available      bool   `json:"available"`
	StatusCode     int    `json:"status_code"`
	ResponseTimeMS int64  `json:"response_time_ms"`
	ErrorMessage   string `json:"error_message,omitempty"`
	CheckedAt      string `json:"checked_at"`
}

func (h *ChecksHandler) Create(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 10*1024)

	var request CreateCheckRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(request.URL) == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	target, err := normalizeURL(request.URL)
	if err != nil {
		http.Error(w, "Invalid URL: provide a valid http or https URL", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	start := time.Now()

	// MVP: check availability with a GET request.
	// Automatic redirects are limited to avoid redirect loops.
	client := &http.Client{
		Timeout: 8 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return errors.New("too many redirects")
			}
			return nil
		},
	}

	checkRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusBadRequest)
		return
	}

	checkRequest.Header.Set("User-Agent", "URL-Security-Analyzer-MVP/1.0")

	response, requestErr := client.Do(checkRequest)
	responseTime := time.Since(start).Milliseconds()

	statusCode := 0
	available := false
	errorMessage := ""

	if requestErr != nil {
		errorMessage = requestErr.Error()
	} else {
		statusCode = response.StatusCode
		available = statusCode >= 200 && statusCode < 400
		response.Body.Close()
	}

	var id int64
	var checkedAt time.Time

	err = h.DB.QueryRow(
		ctx,
		`INSERT INTO url_checks
			(url, available, status_code, response_time_ms, error_message)
		 VALUES ($1, $2, NULLIF($3, 0), $4, NULLIF($5, ''))
		 RETURNING id, checked_at`,
		target,
		available,
		statusCode,
		responseTime,
		errorMessage,
	).Scan(&id, &checkedAt)

	if err != nil {
		http.Error(w, "Failed to save check result", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(CreateCheckResponse{
		ID:             id,
		URL:            target,
		Available:      available,
		StatusCode:     statusCode,
		ResponseTimeMS: responseTime,
		ErrorMessage:   errorMessage,
		CheckedAt:      checkedAt.UTC().Format(time.RFC3339),
	})
}

func normalizeURL(rawURL string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", errors.New("unsupported URL scheme")
	}

	if parsed.Hostname() == "" || parsed.User != nil {
		return "", errors.New("invalid URL host")
	}

	if parsed.Port() != "" {
		port := parsed.Port()
		if port != "80" && port != "443" {
			return "", errors.New("unsupported port")
		}
	}

	host := strings.ToLower(parsed.Hostname())

	// Avoid allowing the service to request localhost or private
	// network addresses. This is a basic MVP safeguard.
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return "", errors.New("local addresses are not allowed")
	}

	if ip := net.ParseIP(host); ip != nil {
		if !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
			return "", errors.New("private IP addresses are not allowed")
		}
	}

	parsed.Host = strings.ToLower(parsed.Host)
	parsed.Fragment = ""

	return parsed.String(), nil
}
