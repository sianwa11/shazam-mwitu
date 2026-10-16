# syntax=docker/dockerfile:1

# Keep this in step with the `go` line in go.mod.
ARG GO_VERSION=1.25

FROM golang:${GO_VERSION}-bookworm AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
# mattn/go-sqlite3 needs CGo. The golang image already ships gcc.
RUN CGO_ENABLED=1 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api


FROM debian:bookworm-slim
# ffmpeg: audio conversion. ca-certificates: HTTPS calls to the iTunes API.
RUN apt-get update \
    && apt-get install -y --no-install-recommends ffmpeg ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && useradd --system --uid 10001 --no-create-home app

WORKDIR /app
COPY --from=build /out/api /app/api
# Prebuilt fingerprint database. CI downloads it into the build context first.
COPY songs.db /app/songs.db

# The API only reads the database, so open it read-only and immutable.
ENV DB_DSN="file:/app/songs.db?mode=ro&immutable=1"

USER app
EXPOSE 8080
CMD ["/app/api"]