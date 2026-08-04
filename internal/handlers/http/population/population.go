package population

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	domainpopulation "location-service/internal/domain/population"
	interfacepopulation "location-service/internal/interfaces/population"
	populationservice "location-service/internal/services/population"
	"location-service/pkg/messages"
	"location-service/pkg/response"
	"location-service/utils"
)

type Handler struct {
	service interfacepopulation.Service
}

func NewHandler(service interfacepopulation.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.GetPopulation(r.Context(), r.PathValue("code"))
	respond(w, r, item, err)
}

func respond(w http.ResponseWriter, r *http.Request, data any, err error) {
	logID := utils.LogID(r)
	if err == nil {
		writeJSON(w, http.StatusOK, response.Response(http.StatusOK, messages.MsgSuccess, logID, data))
		return
	}

	var validationErr *populationservice.ValidationError
	if errors.As(err, &validationErr) {
		writeJSON(w, http.StatusBadRequest, response.ErrorResponse(http.StatusBadRequest, http.StatusText(http.StatusBadRequest), logID, validationErr.Error()))
		return
	}
	if errors.Is(err, domainpopulation.ErrNotFound) {
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
