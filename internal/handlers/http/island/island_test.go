package island

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	domainisland "location-service/internal/domain/island"
	interfacelocation "location-service/internal/interfaces/island"
	islandservice "location-service/internal/services/island"
)

type serviceStub struct {
	listData domainisland.Page
	listErr  error
	getData  domainisland.Island
	getErr   error
	code     string
}

func (s *serviceStub) ListIslands(context.Context, string, string, string) (domainisland.Page, error) {
	return s.listData, s.listErr
}

func (s *serviceStub) GetIsland(_ context.Context, code string) (domainisland.Island, error) {
	s.code = code
	return s.getData, s.getErr
}

var _ interfacelocation.Service = (*serviceStub)(nil)

func TestListUsesResponseEnvelope(t *testing.T) {
	handler := NewHandler(&serviceStub{listData: domainisland.Page{
		Items:      []domainisland.Island{{Code: "11.01.40001", Name: "Pulau"}},
		Pagination: domainisland.Pagination{Page: 1, Limit: 50, Total: 1, TotalPages: 1},
	}})
	recorder := httptest.NewRecorder()
	handler.List(recorder, httptest.NewRequest(http.MethodGet, "/api/islands?page=1", nil))

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

func TestDetailReadsPathValueFallback(t *testing.T) {
	service := &serviceStub{getErr: islandservice.ErrNotFound}
	handler := NewHandler(service)
	recorder := httptest.NewRecorder()
	handler.Detail(recorder, httptest.NewRequest(http.MethodGet, "/api/islands/11.01.40001", nil))

	if service.code != "11.01.40001" {
		t.Fatalf("code = %q; want 11.01.40001", service.code)
	}
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d; want 404", recorder.Code)
	}
}
