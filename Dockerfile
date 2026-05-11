# Stage 1: Build the Go application
FROM golang:1.26-alpine3.23 AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates

# Copy module files first so Docker can cache dependencies.
COPY go.mod go.sum ./
RUN go mod download

# Copy source code and build a static binary.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/location-service .

# Stage 2: Create the final runtime image
FROM alpine:3.23

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /out/location-service /app/location-service
COPY migrations /app/migrations
COPY entrypoint.sh /app/entrypoint.sh

RUN chmod +x /app/entrypoint.sh

ENV APP_ENV=production
ENV PORT=8088
ENV PATH_MIGRATE=migrations/000001_init.sql

EXPOSE 8088

ENTRYPOINT ["/app/entrypoint.sh"]
CMD ["serve"]
