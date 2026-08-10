package location

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	domainlocation "location-service/internal/domain/location"
	locationservice "location-service/internal/services/location"
)

type handlerRepository struct{}

func (handlerRepository) CountStats(context.Context, domainlocation.StatsScope) (domainlocation.Stats, error) {
	return domainlocation.Stats{}, nil
}
func (handlerRepository) ListProvinces(context.Context) ([]domainlocation.Item, error) {
	return nil, nil
}
func (handlerRepository) ListRegencies(context.Context, string, string) ([]domainlocation.Item, error) {
	return nil, nil
}
func (handlerRepository) ListDistricts(context.Context, string, string) ([]domainlocation.Item, error) {
	return nil, nil
}
func (handlerRepository) ListVillages(context.Context, string, string) ([]domainlocation.Item, error) {
	return nil, nil
}
func (handlerRepository) Search(context.Context, string, int) ([]domainlocation.Item, error) {
	return nil, nil
}
func (handlerRepository) SearchByPostalCode(context.Context, string) ([]domainlocation.PostalLocation, error) {
	return []domainlocation.PostalLocation{{PostalCode: "23773"}}, nil
}
func (handlerRepository) GetDetail(context.Context, string) (domainlocation.Detail, error) {
	return domainlocation.Detail{
		Code:        "11.01",
		FullCode:    "11.01",
		Name:        "Kabupaten Aceh Selatan",
		Level:       "regency",
		ParentCode:  "11",
		Coordinates: &domainlocation.Coordinates{Latitude: 3.1, Longitude: 97.4},
		HasBoundary: true,
	}, nil
}
func (handlerRepository) GetBoundary(_ context.Context, code string) (domainlocation.Boundary, error) {
	if code == "11.02" {
		return domainlocation.Boundary{}, domainlocation.ErrBoundaryNotFound
	}
	return domainlocation.Boundary{
		Code:      code,
		Name:      "Kabupaten Aceh Selatan",
		Latitude:  3.1,
		Longitude: 97.4,
		LeafletPath: []byte(
			`[[[1,2],[3,4],[1,2]]]`,
		),
	}, nil
}

func TestBoundaryResponseAndCacheControl(t *testing.T) {
	handler := NewHandler(locationservice.NewService(handlerRepository{}))
	request := httptest.NewRequest(http.MethodGet, "/api/locations/11.01/boundary", nil)
	request.SetPathValue("code", "11.01")
	recorder := httptest.NewRecorder()

	handler.Boundary(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if got := recorder.Header().Get("Cache-Control"); got != "public, max-age=86400, stale-while-revalidate=86400" {
		t.Fatalf("cache control = %q", got)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `"leaflet_path":[[[1,2],[3,4],[1,2]]]`) {
		t.Fatalf("response omitted leaflet path: %s", body)
	}
}

func TestBoundaryNotFoundUsesShortCache(t *testing.T) {
	handler := NewHandler(locationservice.NewService(handlerRepository{}))
	request := httptest.NewRequest(http.MethodGet, "/api/locations/11.02/boundary", nil)
	request.SetPathValue("code", "11.02")
	recorder := httptest.NewRecorder()

	handler.Boundary(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", recorder.Code)
	}
	if got := recorder.Header().Get("Cache-Control"); got != "public, max-age=60" {
		t.Fatalf("cache control = %q", got)
	}
	if !strings.Contains(recorder.Body.String(), `"status":false`) {
		t.Fatal("response envelope missing failure status")
	}
}

func TestDetailResponseIncludesCoordinates(t *testing.T) {
	handler := NewHandler(locationservice.NewService(handlerRepository{}))
	request := httptest.NewRequest(http.MethodGet, "/api/locations/11.01", nil)
	request.SetPathValue("code", "11.01")
	recorder := httptest.NewRecorder()

	handler.Detail(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `"coordinates":{"latitude":3.1,"longitude":97.4}`) || !strings.Contains(body, `"has_boundary":true`) {
		t.Fatalf("detail fields missing: %s", body)
	}
}

func TestPostalCodesResponse(t *testing.T) {
	handler := NewHandler(locationservice.NewService(handlerRepository{}))
	request := httptest.NewRequest(http.MethodGet, "/api/postal-codes/23773", nil)
	request.SetPathValue("postal_code", "23773")
	recorder := httptest.NewRecorder()

	handler.PostalCodes(recorder, request)

	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"postal_code":"23773"`) {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestPostalCodesRejectsInvalidFormat(t *testing.T) {
	handler := NewHandler(locationservice.NewService(handlerRepository{}))
	request := httptest.NewRequest(http.MethodGet, "/api/postal-codes/2377", nil)
	request.SetPathValue("postal_code", "2377")
	recorder := httptest.NewRecorder()

	handler.PostalCodes(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}
