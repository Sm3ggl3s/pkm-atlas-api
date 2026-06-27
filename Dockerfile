# syntax=docker/dockerfile:1

# ---------------------------------------------------------------------------
# Base stage: shared by dev and the production builder.
# Downloads modules first so the layer is cached until go.mod/go.sum change.
# ---------------------------------------------------------------------------
FROM golang:1.25-alpine AS base
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

# ---------------------------------------------------------------------------
# Dev stage: hot reload with Air. Source is bind-mounted in docker-compose,
# so we don't COPY the code here.
# ---------------------------------------------------------------------------
FROM base AS dev
RUN go install github.com/air-verse/air@latest
EXPOSE 8080
CMD ["air", "-c", ".air.toml"]

# ---------------------------------------------------------------------------
# Builder stage: compile a small static binary for production.
# ---------------------------------------------------------------------------
FROM base AS builder
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/api ./cmd/api

# ---------------------------------------------------------------------------
# Production stage: minimal runtime image with just the binary.
# ---------------------------------------------------------------------------
FROM alpine:3.20 AS prod
WORKDIR /app
RUN adduser -D appuser
COPY --from=builder /bin/api /app/api
USER appuser
EXPOSE 8080
CMD ["/app/api"]
