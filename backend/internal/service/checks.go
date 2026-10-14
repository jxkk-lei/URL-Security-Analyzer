package service

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"url-security-analyzer/backend/internal/model"
)

var ErrInvalidURL = errors.New("invalid URL")

type CheckRepository interface {
	Create(
		ctx context.Context,
		check model.Check,
	) (model.Check, error)

	List(
		ctx context.Context,
		limit int,
	) ([]model.Check, error)
}

type ChecksService struct {
	Repository CheckRepository
	Client     *http.Client
}

func NewChecksService(repo CheckRepository) *ChecksService {
	transport := &http.Transport{
		// Не используем системный HTTP-прокси: он обошёл бы нашу
		// проверку адреса целевого сервера.
		Proxy:                 nil,
		DialContext:           safeDialContext,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 5 * time.Second,
		IdleConnTimeout:       30 * time.Second,
	}

	return &ChecksService{
		Repository: repo,
		Client: &http.Client{
			Transport: transport,
			Timeout:   8 * time.Second,
			CheckRedirect: func(
				req *http.Request,
				via []*http.Request,
			) error {
				if len(via) >= 5 {
					return errors.New("too many redirects")
				}

				if _, err := normalizeURL(req.URL.String()); err != nil {
					return fmt.Errorf("unsafe redirect: %w", err)
				}

				return nil
			},
		},
	}
}

func (s *ChecksService) CheckURL(
	ctx context.Context,
	rawURL string,
) (model.Check, error) {
	target, err := normalizeURL(rawURL)
	if err != nil {
		return model.Check{}, err
	}

	checkCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	check := model.Check{URL: target}
	start := time.Now()

	req, err := http.NewRequestWithContext(
		checkCtx,
		http.MethodGet,
		target,
		nil,
	)
	if err != nil {
		return model.Check{}, fmt.Errorf("%w: cannot create request", ErrInvalidURL)
	}

	req.Header.Set("User-Agent", "URL-Security-Analyzer-MVP/1.0")

	response, requestErr := s.Client.Do(req)
	check.ResponseTimeMS = time.Since(start).Milliseconds()

	if requestErr != nil {
		// Сетевая ошибка — результат проверки, а не ошибка API.
		check.ErrorMessage = requestErr.Error()
	} else {
		check.StatusCode = response.StatusCode
		check.Available = response.StatusCode >= 200 &&
			response.StatusCode < 400
		_ = response.Body.Close()
	}

	return s.Repository.Create(checkCtx, check)
}

func (s *ChecksService) List(
	ctx context.Context,
	limit int,
) ([]model.Check, error) {
	if limit < 1 || limit > 100 {
		return nil, fmt.Errorf("limit must be between 1 and 100")
	}

	return s.Repository.List(ctx, limit)
}

func normalizeURL(rawURL string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("%w: malformed URL", ErrInvalidURL)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("%w: only HTTP and HTTPS are supported", ErrInvalidURL)
	}

	if parsed.Hostname() == "" || parsed.User != nil {
		return "", fmt.Errorf("%w: invalid host", ErrInvalidURL)
	}

	if parsed.Port() != "" &&
		parsed.Port() != "80" &&
		parsed.Port() != "443" {
		return "", fmt.Errorf("%w: unsupported port", ErrInvalidURL)
	}

	host := strings.ToLower(parsed.Hostname())

	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return "", fmt.Errorf("%w: local addresses are not allowed", ErrInvalidURL)
	}

	if ip := net.ParseIP(host); ip != nil {
		if !ip.IsGlobalUnicast() ||
			ip.IsPrivate() ||
			ip.IsLoopback() ||
			ip.IsLinkLocalUnicast() {
			return "", fmt.Errorf("%w: private IP addresses are not allowed", ErrInvalidURL)
		}
	}

	parsed.Host = strings.ToLower(parsed.Host)
	parsed.Fragment = ""

	return parsed.String(), nil
}
