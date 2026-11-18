# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy dependency manifests and download modules
COPY go.mod go.sum ./
RUN go mod download

# Copy source code and compile statically linked binary
COPY . .
ARG GITHUB_RUN=local
ARG BUILD_DATE=unknown
ARG GIT_SHA=unknown
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s -X github.com/plannedwallet/internal/version.GitHubRun=${GITHUB_RUN} -X github.com/plannedwallet/internal/version.BuildDate=${BUILD_DATE} -X github.com/plannedwallet/internal/version.GitSHA=${GIT_SHA}" \
    -o /app/server .

# Deployment stage
FROM scratch

WORKDIR /

# Copy SSL certificates from builder for HTTPS support
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy binary from builder
COPY --from=builder /app/server /server

EXPOSE 8080

ENTRYPOINT ["/server"]
