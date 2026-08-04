package island

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	domainisland "location-service/internal/domain/island"
)

type repositoryStub struct {
	page       domainisland.Page
	pageNumber int
	pageLimit  int
	findErr    error
}

func (s *repositoryStub) ListIslands(_ context.Context, _ string, page, limit int) (domainisland.Page, error) {
	s.pageNumber, s.pageLimit = page, limit
	return s.page, nil
}

func (s *repositoryStub) FindIslandByCode(context.Context, string) (domainisland.Island, error) {
	return domainisland.Island{}, s.findErr
}

func TestListIslandsDefaultsAndValidation(t *testing.T) {
	repo := &repositoryStub{page: domainisland.Page{Items: []domainisland.Island{}}}
	service := NewService(repo)

	if _, err := service.ListIslands(context.Background(), "11", "", ""); err != nil {
		t.Fatalf("ListIslands() error = %v", err)
	}
	if repo.pageNumber != DefaultPage || repo.pageLimit != DefaultLimit {
		t.Fatalf("pagination = %d/%d; want %d/%d", repo.pageNumber, repo.pageLimit, DefaultPage, DefaultLimit)
	}

	for name, values := range map[string][3]string{
		"province": {"1", "", ""},
		"page":     {"11", "0", ""},
		"limit":    {"11", "", "501"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := service.ListIslands(context.Background(), values[0], values[1], values[2])
			var validationErr *ValidationError
			if !errors.As(err, &validationErr) {
				t.Fatalf("error = %v; want ValidationError", err)
			}
		})
	}
}

func TestGetIslandMapsMissingRow(t *testing.T) {
	service := NewService(&repositoryStub{findErr: sql.ErrNoRows})
	_, err := service.GetIsland(context.Background(), "11.01.40001")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetIsland() error = %v; want ErrNotFound", err)
	}

	_, err = service.GetIsland(context.Background(), "11.01.bad")
	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("invalid code error = %v; want ValidationError", err)
	}
}
