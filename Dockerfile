# syntax=docker/dockerfile:1

# ---- Build stage ----
FROM golang:1.26 AS build
WORKDIR /src

# Cache dependencies first.
COPY go.mod go.sum ./
COPY vendor ./vendor
COPY . .

# Build a statically-linked binary.
ENV CGO_ENABLED=0 GOFLAGS=-mod=vendor
RUN go build -o /out/api ./cmd/api

# ---- Runtime stage ----
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app

COPY --from=build /out/api /app/api

EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/api"]
