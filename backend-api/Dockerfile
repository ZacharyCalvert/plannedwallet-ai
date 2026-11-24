# Build stage
FROM golang:1.27.4-alpine AS builder

WORKDIR /app

# Copy dependency manifests and download modules
COPY go.mod go.sum ./
RUN go mod download

# Copy source code and compile statically linked binary
COPY . .
ARG GITHUB_RUN=local
ARG BUILD_DATE=unknown
ARG GIT_SHA=unknown
ARG TAG=untagged
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s -X github.com/plannedwallet/internal/version.GitHubRun=${GITHUB_RUN} -X github.com/plannedwallet/internal/version.BuildDate=${BUILD_DATE} -X github.com/plannedwallet/internal/version.GitSHA=${GIT_SHA} -X github.com/plannedwallet/internal/version.Tag=${TAG}" \
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
