package location

import (
	"context"
	domainlocation "location-service/internal/domain/location"
	"strings"
)

func codeExpression(codeFormat string) string {
	if codeFormat == "short" {
		return "short_code"
	}
	return "code"
}

func isTransient(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "connection reset by peer") ||
		strings.Contains(msg, "broken pipe") ||
		strings.Contains(msg, "EOF") ||
		strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "bad connection")
}

func (r *repository) queryLocations(ctx context.Context, query string, args ...any) ([]domainlocation.Item, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil && isTransient(err) {
		rows, err = r.db.QueryContext(ctx, query, args...)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]domainlocation.Item, 0)
	for rows.Next() {
		var item domainlocation.Item
		if err := rows.Scan(&item.Code, &item.FullCode, &item.Name, &item.Level, &item.ParentCode); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
