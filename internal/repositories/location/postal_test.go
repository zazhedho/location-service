package location

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"location-service/infrastructure/database"
)

func TestSearchByPostalCodeAgainstConfiguredDatabase(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if dsn == "" {
		t.Skip("DATABASE_URL is not configured")
	}

	db, err := database.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	items, err := NewRepository(db).SearchByPostalCode(ctx, "23773")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) == 0 {
		t.Fatal("postal-code lookup returned no rows")
	}
	for _, item := range items {
		if item.PostalCode != "23773" || item.Village.Code == "" || item.District.Code == "" || item.Regency.Code == "" || item.Province.Code == "" {
			t.Fatalf("incomplete postal location: %+v", item)
		}
	}
}
