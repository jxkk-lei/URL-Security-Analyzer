package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"url-security-analyzer/backend/internal/service"
)

type ChecksHandler struct {
	Service *service.ChecksService
}

type CreateCheckRequest struct {
	URL string `json:"url"`
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

	ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
	defer cancel()

	check, err := h.Service.CheckURL(ctx, request.URL)
	if err != nil {
		if errors.Is(err, service.ErrInvalidURL) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		http.Error(w, "Failed to save check result", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, check)
}

func (h *ChecksHandler) List(w http.ResponseWriter, r *http.Request) {
	limit := 20

	if value := r.URL.Query().Get("limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 100 {
			http.Error(w, "limit must be between 1 and 100", http.StatusBadRequest)
			return
		}
		limit = parsed
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	checks, err := h.Service.List(ctx, limit)
	if err != nil {
		http.Error(w, "Failed to retrieve check history", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, checks)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(value); err != nil {
		// После WriteHeader изменить HTTP-статус уже нельзя.
		// Ошибки записи здесь можно дополнительно логировать.
		return
	}
}
