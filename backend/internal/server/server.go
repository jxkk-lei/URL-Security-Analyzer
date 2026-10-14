package server

import (
	"net/http"
	"time"

	"url-security-analyzer/backend/internal/handler"
)

func New(
	addr string,
	checksHandler *handler.ChecksHandler,
	healthHandler *handler.HealthHandler,
) *http.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/v1/checks", checksHandler.Create)
	mux.HandleFunc("GET /api/v1/checks", checksHandler.List)
	mux.HandleFunc("GET /api/v1/health", healthHandler.Check)

	return &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
}
