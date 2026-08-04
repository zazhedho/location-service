package population

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	domainpopulation "location-service/internal/domain/population"
)

type repositoryStub struct {
	item domainpopulation.Population
	err  error
	code string
}

func (s *repositoryStub) FindPopulationByCode(_ context.Context, code string) (domainpopulation.Population, error) {
	s.code = code
	return s.item, s.err
}

func TestGetPopulationValidatesCodeAndReturnsData(t *testing.T) {
	repo := &repositoryStub{item: domainpopulation.Population{
		Code:          "11.01",
		Name:          "Kabupaten Aceh Selatan",
		Male:          120041,
		Female:        119588,
		Total:         239629,
		Source:        domainpopulation.Source,
		ReferenceDate: domainpopulation.ReferenceDate,
		ImportedAt:    time.Date(2026, time.August, 4, 0, 0, 0, 0, time.UTC),
	}}
	service := NewService(repo)

	item, err := service.GetPopulation(context.Background(), "11.01")
	if err != nil {
		t.Fatalf("GetPopulation() error = %v", err)
	}
	if item.Total != 239629 || repo.code != "11.01" {
		t.Fatalf("item = %+v, code = %q", item, repo.code)
	}

	for _, code := range []string{"11.1", "11.01/anything", "", " 11.01"} {
		_, err := service.GetPopulation(context.Background(), code)
		var validationErr *ValidationError
		if !errors.As(err, &validationErr) {
			t.Fatalf("code %q error = %v; want ValidationError", code, err)
		}
	}
}

func TestGetPopulationMapsMissingRow(t *testing.T) {
	service := NewService(&repositoryStub{err: sql.ErrNoRows})
	_, err := service.GetPopulation(context.Background(), "11.01")
	if !errors.Is(err, domainpopulation.ErrNotFound) {
		t.Fatalf("GetPopulation() error = %v; want ErrNotFound", err)
	}
}
