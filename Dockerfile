FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Cache Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w -X main.Version=${VERSION}" -o /bin/terrascore-api ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/terrascore-migrate ./cmd/migrate

# Runtime stage
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /bin/terrascore-api /bin/terrascore-api
COPY --from=builder /bin/terrascore-migrate /bin/terrascore-migrate
COPY db/migrations /app/db/migrations

EXPOSE 8080

ENTRYPOINT ["/bin/terrascore-api"]
