package location

import (
	"context"
	"strings"
	"testing"

	domainlocation "location-service/internal/domain/location"
)

type postalRepositoryStub struct {
	items []domainlocation.PostalLocation
	calls int
}

func (*postalRepositoryStub) CountStats(context.Context, domainlocation.StatsScope) (domainlocation.Stats, error) {
	return domainlocation.Stats{}, nil
}
func (*postalRepositoryStub) ListProvinces(context.Context) ([]domainlocation.Item, error) {
	return nil, nil
}
func (*postalRepositoryStub) ListRegencies(context.Context, string, string) ([]domainlocation.Item, error) {
	return nil, nil
}
func (*postalRepositoryStub) ListDistricts(context.Context, string, string) ([]domainlocation.Item, error) {
	return nil, nil
}
func (*postalRepositoryStub) ListVillages(context.Context, string, string) ([]domainlocation.Item, error) {
	return nil, nil
}
func (*postalRepositoryStub) Search(context.Context, string, int) ([]domainlocation.Item, error) {
	return nil, nil
}
func (*postalRepositoryStub) GetDetail(context.Context, string) (domainlocation.Detail, error) {
	return domainlocation.Detail{}, nil
}
func (*postalRepositoryStub) GetBoundary(context.Context, string) (domainlocation.Boundary, error) {
	return domainlocation.Boundary{}, nil
}
func (s *postalRepositoryStub) SearchByPostalCode(context.Context, string) ([]domainlocation.PostalLocation, error) {
	s.calls++
	return s.items, nil
}

func TestPostalCodesValidatesAndReturnsCandidates(t *testing.T) {
	repo := &postalRepositoryStub{items: []domainlocation.PostalLocation{{PostalCode: "23773"}}}
	service := NewService(repo)

	items, err := service.PostalCodes(context.Background(), "23773")
	if err != nil || len(items) != 1 || repo.calls != 1 {
		t.Fatalf("items=%+v calls=%d err=%v", items, repo.calls, err)
	}
	if _, err := service.PostalCodes(context.Background(), "2377"); err == nil || err.Error() != "postal_code is invalid" {
		t.Fatalf("invalid postal code error = %v", err)
	}
}

func TestPostalCodesReturnsEmptyArrayWhenNoMatches(t *testing.T) {
	service := NewService(&postalRepositoryStub{})
	items, err := service.PostalCodes(context.Background(), "99999")
	if err != nil {
		t.Fatal(err)
	}
	if items == nil || len(items) != 0 {
		t.Fatalf("items=%#v, want non-nil empty array", items)
	}
}

func TestSearchRejectsOverlongQuery(t *testing.T) {
	service := NewService(&postalRepositoryStub{})

	_, err := service.Search(context.Background(), strings.Repeat("a", 101), "")
	if err == nil || err.Error() != "q must not exceed 100 characters" {
		t.Fatalf("overlong query error = %v; want length validation", err)
	}
}
