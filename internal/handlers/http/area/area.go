package area

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	domainarea "location-service/internal/domain/area"
	interfacearea "location-service/internal/interfaces/area"
	"location-service/pkg/messages"
	"location-service/pkg/response"
	"location-service/utils"
)

type Handler struct {
	service interfacearea.Service
}

func NewHandler(service interfacearea.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Area(w http.ResponseWriter, r *http.Request) {
	data, err := h.service.Area(r.Context(), r.PathValue("code"))
	if errors.Is(err, domainarea.ErrNotFound) {
		w.Header().Set("Cache-Control", "public, max-age=60")
	} else {
		w.Header().Set("Cache-Control", "public, max-age=86400, stale-while-revalidate=86400")
	}
	respond(w, r, data, err)
}

func respond(w http.ResponseWriter, r *http.Request, data domainarea.Area, err error) {
	logID := utils.LogID(r)
	if err == nil {
		writeJSON(w, http.StatusOK, response.Response(http.StatusOK, messages.MsgSuccess, logID, data))
		return
	}
	if errors.Is(err, domainarea.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, response.ErrorResponse(http.StatusNotFound, messages.MsgNotFound, logID, err.Error()))
		return
	}
	if errors.Is(err, domainarea.ErrCodeRequired) || errors.Is(err, domainarea.ErrCodeInvalid) {
		writeJSON(w, http.StatusBadRequest, response.ErrorResponse(http.StatusBadRequest, http.StatusText(http.StatusBadRequest), logID, err.Error()))
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
