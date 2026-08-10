package router

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"

	areahandler "location-service/internal/handlers/http/area"
	islandhandler "location-service/internal/handlers/http/island"
	locationhandler "location-service/internal/handlers/http/location"
	populationhandler "location-service/internal/handlers/http/population"
	arearepo "location-service/internal/repositories/area"
	islandrepo "location-service/internal/repositories/island"
	locationrepo "location-service/internal/repositories/location"
	populationrepo "location-service/internal/repositories/population"
	areaservice "location-service/internal/services/area"
	islandservice "location-service/internal/services/island"
	locationservice "location-service/internal/services/location"
	populationservice "location-service/internal/services/population"
	"location-service/middlewares"
	"location-service/pkg/messages"
	"location-service/pkg/response"
	"location-service/pkg/storage"
	"location-service/utils"
)

func New(db *sql.DB, redisClient *redis.Client, providers ...storage.Provider) http.Handler {
	repo := locationrepo.NewRepository(db, providers...)
	service := locationservice.NewService(repo, redisClient)
	handler := locationhandler.NewHandler(service)
	islandHandler := islandhandler.NewHandler(islandservice.NewService(islandrepo.NewRepository(db), redisClient))
	populationHandler := populationhandler.NewHandler(populationservice.NewService(populationrepo.NewRepository(db), redisClient))
	areaHandler := areahandler.NewHandler(areaservice.NewService(arearepo.NewRepository(db), redisClient))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", health(db))
	mux.HandleFunc("GET /api/locations/stats", handler.Stats)
	mux.HandleFunc("GET /api/locations/provinces", handler.Provinces)
	mux.HandleFunc("GET /api/locations/regencies", handler.Regencies)
	mux.HandleFunc("GET /api/locations/districts", handler.Districts)
	mux.HandleFunc("GET /api/locations/villages", handler.Villages)
	mux.HandleFunc("GET /api/locations/search", handler.Search)
	mux.HandleFunc("GET /api/locations/{code}/boundary", handler.Boundary)
	mux.HandleFunc("GET /api/locations/{code}/population", populationHandler.Get)
	mux.HandleFunc("GET /api/locations/{code}/area", areaHandler.Area)
	mux.HandleFunc("GET /api/locations/{code}", handler.Detail)
	mux.HandleFunc("GET /api/islands", islandHandler.List)
	mux.HandleFunc("GET /api/islands/{code}", islandHandler.Detail)
	return middlewares.CORS(utils.WithRequestID(mux))
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
