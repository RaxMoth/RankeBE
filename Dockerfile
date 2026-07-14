# Build stage — Go version must match go.mod (1.25).
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Cache module downloads in a separate layer so source edits don't re-download.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Static binary so the distroless/scratch final image works.
RUN CGO_ENABLED=0 GOOS=linux go build \
      -trimpath \
      -ldflags="-s -w" \
      -o /out/server ./cmd/server

# Final stage — minimal image, non-root user, no shell.
FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata \
  && addgroup -S app && adduser -S -G app app

WORKDIR /app
COPY --from=builder /out/server .
# Migrations are compiled into the binary via embed.FS (internal/db/migrations),
# so there's nothing to copy alongside it.

USER app
EXPOSE 8080
ENTRYPOINT ["/app/server"]
