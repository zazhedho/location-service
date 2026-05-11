package router

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	locationhandler "location-service/internal/handlers/http/location"
	locationrepo "location-service/internal/repositories/location"
	locationservice "location-service/internal/services/location"
	"location-service/pkg/messages"
	"location-service/pkg/response"
	"location-service/utils"
)

func New(db *sql.DB) http.Handler {
	repo := locationrepo.NewRepository(db)
	service := locationservice.NewService(repo)
	handler := locationhandler.NewHandler(service)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", health(db))
	mux.HandleFunc("GET /api/locations/provinces", handler.Provinces)
	mux.HandleFunc("GET /api/locations/regencies", handler.Regencies)
	mux.HandleFunc("GET /api/locations/districts", handler.Districts)
	mux.HandleFunc("GET /api/locations/villages", handler.Villages)
	mux.HandleFunc("GET /api/locations/search", handler.Search)
	return utils.WithRequestID(mux)
}

func health(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		logID := utils.LogID(r)
		if err := db.PingContext(ctx); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, response.ErrorResponse(http.StatusServiceUnavailable, http.StatusText(http.StatusServiceUnavailable), logID, err.Error()))
			return
		}
		writeJSON(w, http.StatusOK, response.Response(http.StatusOK, messages.MsgSuccess, logID, map[string]string{"status": "ok"}))
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
