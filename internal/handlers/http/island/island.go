package island

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	interfaceisland "location-service/internal/interfaces/island"
	islandservice "location-service/internal/services/island"
	"location-service/pkg/messages"
	"location-service/pkg/response"
	"location-service/utils"
)

type Handler struct {
	service interfaceisland.Service
}

func NewHandler(service interfaceisland.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	page, limit := query.Get("page"), query.Get("limit")
	data, err := h.service.ListIslands(r.Context(), query.Get("province_code"), page, limit)
	respond(w, r, data, err)
}

func (h *Handler) Detail(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	if code == "" {
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) > 0 {
			code = parts[len(parts)-1]
		}
	}
	data, err := h.service.GetIsland(r.Context(), code)
	respond(w, r, data, err)
}

func respond(w http.ResponseWriter, r *http.Request, data any, err error) {
	logID := utils.LogID(r)
	if err == nil {
		writeJSON(w, http.StatusOK, response.Response(http.StatusOK, messages.MsgSuccess, logID, data))
		return
	}

	var validationErr *islandservice.ValidationError
	if errors.As(err, &validationErr) {
		writeJSON(w, http.StatusBadRequest, response.ErrorResponse(http.StatusBadRequest, http.StatusText(http.StatusBadRequest), logID, validationErr.Error()))
		return
	}
	if errors.Is(err, islandservice.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, response.ErrorResponse(http.StatusNotFound, messages.MsgNotFound, logID, messages.MsgNotFound))
		return
	}

	log.Printf("internal error [%s]: %v", logID, err)
	writeJSON(w, http.StatusInternalServerError, response.InternalServerError(logID))
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
