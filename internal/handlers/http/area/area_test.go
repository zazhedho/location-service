package area

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	domainarea "location-service/internal/domain/area"
	interfacearea "location-service/internal/interfaces/area"
)

type serviceStub struct {
	item domainarea.Area
	err  error
}

func (s serviceStub) Area(context.Context, string) (domainarea.Area, error) {
	return s.item, s.err
}

var _ interfacearea.Service = serviceStub{}

func TestAreaUsesResponseEnvelope(t *testing.T) {
	handler := NewHandler(serviceStub{item: domainarea.Area{Code: "32.73", Name: "Kota Bandung", AreaKM2: 166.593}})
	request := httptest.NewRequest(http.MethodGet, "/api/locations/32.73/area", nil)
	request.SetPathValue("code", "32.73")
	recorder := httptest.NewRecorder()

	handler.Area(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200", recorder.Code)
	}
	var envelope map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope["status"] != true || envelope["message"] != "Success" || envelope["data"] == nil {
		t.Fatalf("envelope = %v", envelope)
	}
}

func TestAreaNotFoundUses404Envelope(t *testing.T) {
	handler := NewHandler(serviceStub{err: domainarea.ErrNotFound})
	request := httptest.NewRequest(http.MethodGet, "/api/locations/32.73/area", nil)
	request.SetPathValue("code", "32.73")
	recorder := httptest.NewRecorder()

	handler.Area(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d; want 404", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), `"status":false`) {
		t.Fatal("response envelope missing failure status")
	}
}

func TestAreaValidationUses400Envelope(t *testing.T) {
	handler := NewHandler(serviceStub{err: domainarea.ErrCodeInvalid})
	request := httptest.NewRequest(http.MethodGet, "/api/locations/not-a-code/area", nil)
	request.SetPathValue("code", "not-a-code")
	recorder := httptest.NewRecorder()

	handler.Area(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d; want 400", recorder.Code)
	}
}
