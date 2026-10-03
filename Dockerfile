# Build Stage
FROM golang:1.22-alpine AS builder

# Set the working directory
WORKDIR /app

# Copy go.mod and go.sum first to cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build both the API and the Worker binaries
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/api ./cmd/api/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/worker ./cmd/worker/main.go

# Production Stage
FROM alpine:latest

WORKDIR /app

# Copy the compiled binaries from the builder stage
COPY --from=builder /bin/api ./api
COPY --from=builder /bin/worker ./worker

# Expose the API port
EXPOSE 8080

# By default, start the API (this can be overridden by the cloud provider for the worker)
CMD ["./api"]
