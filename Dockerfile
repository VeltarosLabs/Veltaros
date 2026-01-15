# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /src

RUN apk add --no-cache git ca-certificates && update-ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build a static-ish binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/veltaros-node ./cmd/veltaros-node

# Runtime stage
FROM alpine:3.20

RUN apk add --no-cache ca-certificates && update-ca-certificates

WORKDIR /app
COPY --from=builder /out/veltaros-node /app/veltaros-node

# Default ports (override at runtime)
EXPOSE 8080 30303

# Default command (override flags in platform config)
ENTRYPOINT ["/app/veltaros-node"]
