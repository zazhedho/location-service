package area

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	domainarea "location-service/internal/domain/area"
)

type repositoryStub struct {
	item domainarea.Area
	err  error
	code string
}

func (s *repositoryStub) FindAreaByCode(_ context.Context, code string) (domainarea.Area, error) {
	s.code = code
	return s.item, s.err
}

func TestAreaValidatesCodeAndMapsMissing(t *testing.T) {
	repo := &repositoryStub{item: domainarea.Area{Code: "11", AreaKM2: 56835.019}}
	service := NewService(repo)

	item, err := service.Area(context.Background(), " 11 ")
	if err != nil || item.AreaKM2 != 56835.019 || repo.code != "11" {
		t.Fatalf("Area() = %+v, code=%q, err=%v", item, repo.code, err)
	}

	for _, code := range []string{"", "11.1", "not-a-code"} {
		t.Run(code, func(t *testing.T) {
			if _, err := service.Area(context.Background(), code); !errors.Is(err, domainarea.ErrCodeRequired) && !errors.Is(err, domainarea.ErrCodeInvalid) {
				t.Fatalf("Area(%q) error = %v; want validation error", code, err)
			}
		})
	}

	service = NewService(&repositoryStub{err: sql.ErrNoRows})
	if _, err := service.Area(context.Background(), "11"); !errors.Is(err, domainarea.ErrNotFound) {
		t.Fatalf("missing area error = %v; want ErrNotFound", err)
	}
}
