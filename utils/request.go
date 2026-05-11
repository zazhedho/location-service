package utils

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type requestIDKey struct{}

func WithRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logID, err := uuid.NewV7()
		if err != nil {
			logID = uuid.New()
		}
		ctx := context.WithValue(r.Context(), requestIDKey{}, logID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func LogID(r *http.Request) uuid.UUID {
	if r != nil {
		if logID, ok := r.Context().Value(requestIDKey{}).(uuid.UUID); ok {
			return logID
		}
	}
	logID, err := uuid.NewV7()
	if err != nil {
		return uuid.New()
	}
	return logID
}
