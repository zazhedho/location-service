package population

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	domainpopulation "location-service/internal/domain/population"
	populationservice "location-service/internal/services/population"
)

type serviceStub struct {
	item domainpopulation.Population
	err  error
}

func (s *serviceStub) GetPopulation(context.Context, string) (domainpopulation.Population, error) {
	return s.item, s.err
}

func TestGetUsesResponseEnvelope(t *testing.T) {
	handler := NewHandler(&serviceStub{item: domainpopulation.Population{Code: "11.01", Total: 239629}})
	request := httptest.NewRequest(http.MethodGet, "/api/locations/11.01/population", nil)
	request.SetPathValue("code", "11.01")
	recorder := httptest.NewRecorder()

	handler.Get(recorder, request)

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

func TestGetMapsValidationAndNotFound(t *testing.T) {
	for name, serviceErr := range map[string]error{
		"validation": &populationservice.ValidationError{Message: "code is invalid"},
		"not found":  domainpopulation.ErrNotFound,
	} {
		t.Run(name, func(t *testing.T) {
			handler := NewHandler(&serviceStub{err: serviceErr})
			request := httptest.NewRequest(http.MethodGet, "/api/locations/11.01/population", nil)
			request.SetPathValue("code", "11.01")
			recorder := httptest.NewRecorder()

			handler.Get(recorder, request)

			want := http.StatusBadRequest
			if errors.Is(serviceErr, domainpopulation.ErrNotFound) {
				want = http.StatusNotFound
			}
			if recorder.Code != want {
				t.Fatalf("status = %d; want %d", recorder.Code, want)
			}
		})
	}
}
