package location

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	domainlocation "location-service/internal/domain/location"
	interfacelocation "location-service/internal/interfaces/location"
	"location-service/pkg/messages"
	"location-service/pkg/response"
	"location-service/utils"
)

type Handler struct {
	service interfacelocation.Service
}

func NewHandler(service interfacelocation.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Stats(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	stats, err := h.service.Stats(r.Context(), query.Get("province_code"), query.Get("regency_code"), query.Get("district_code"))
	respond(w, r, stats, err)
}

func (h *Handler) Provinces(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListProvinces(r.Context())
	respond(w, r, items, err)
}

func (h *Handler) Regencies(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	items, err := h.service.ListRegencies(r.Context(), query.Get("province_code"), query.Get("code_format"))
	respond(w, r, items, err)
}

func (h *Handler) Districts(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	items, err := h.service.ListDistricts(r.Context(), query.Get("province_code"), query.Get("regency_code"), query.Get("code_format"))
	respond(w, r, items, err)
}

func (h *Handler) Villages(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	items, err := h.service.ListVillages(r.Context(), query.Get("province_code"), query.Get("regency_code"), query.Get("district_code"), query.Get("code_format"))
	respond(w, r, items, err)
}

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	items, err := h.service.Search(r.Context(), query.Get("q"), query.Get("limit"))
	respond(w, r, items, err)
}

func (h *Handler) PostalCodes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "public, max-age=86400, stale-while-revalidate=86400")
	items, err := h.service.PostalCodes(r.Context(), r.PathValue("postal_code"))
	respond(w, r, items, err)
}

func (h *Handler) Detail(w http.ResponseWriter, r *http.Request) {
	detail, err := h.service.Detail(r.Context(), r.PathValue("code"))
	respond(w, r, detail, err)
}

func (h *Handler) Boundary(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "public, max-age=86400, stale-while-revalidate=86400")
	boundary, err := h.service.Boundary(r.Context(), r.PathValue("code"))
	if errors.Is(err, domainlocation.ErrBoundaryNotFound) {
		w.Header().Set("Cache-Control", "public, max-age=60")
	}
	respond(w, r, boundary, err)
}

func respond(w http.ResponseWriter, r *http.Request, data any, err error) {
	logID := utils.LogID(r)
	if err != nil {
		if errors.Is(err, domainlocation.ErrNotFound) || errors.Is(err, domainlocation.ErrBoundaryNotFound) {
			writeJSON(w, http.StatusNotFound, response.ErrorResponse(http.StatusNotFound, messages.MsgNotFound, logID, err.Error()))
			return
		}
		if isClientError(err.Error()) {
			writeJSON(w, http.StatusBadRequest, response.ErrorResponse(http.StatusBadRequest, http.StatusText(http.StatusBadRequest), logID, err.Error()))
			return
		}
		log.Printf("internal error [%s]: %v", logID, err)
		writeJSON(w, http.StatusInternalServerError, response.InternalServerError(logID))
		return
	}
	writeJSON(w, http.StatusOK, response.Response(http.StatusOK, messages.MsgSuccess, logID, data))
}

func isClientError(message string) bool {
	switch message {
	case "province_code is required",
		"regency_code is required",
		"district_code is required",
		"q is required",
		"q must not exceed 100 characters",
		"limit must be a number between 1 and 500",
		"postal_code is invalid",
		"code is required",
		"code is invalid",
		"province_code is required when regency_code is short",
		"province_code is required when district_code is short":
		return true
	default:
		return false
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
