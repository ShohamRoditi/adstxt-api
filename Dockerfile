FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy only go.mod first for better layer caching
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build with optimizations for smaller binary
RUN CGO_ENABLED=0 GOOS=linux go build \
    -a -installsuffix cgo \
    -ldflags="-w -s" \
    -o main ./cmd/server

FROM scratch

# Copy CA certs from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY --from=builder /app/main .

EXPOSE 8080

ENTRYPOINT ["./main"]
