package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type HealthHandler struct {
	DB *pgxpool.Pool
}

type HealthResponse struct {
	Status   string `json:"status"`
	Database string `json:"database"`
}

func (h *HealthHandler) Check(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	response := HealthResponse{
		Status:   "ok",
		Database: "ok",
	}

	if err := h.DB.Ping(ctx); err != nil {
		response.Status = "error"
		response.Database = "unavailable"
		writeHealthJSON(w, http.StatusServiceUnavailable, response)
		return
	}

	writeHealthJSON(w, http.StatusOK, response)
}

func writeHealthJSON(w http.ResponseWriter, status int, value HealthResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
