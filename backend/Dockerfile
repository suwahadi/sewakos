# Same image serves both cmd/api and cmd/worker (SPEC-GO.md §3); pick which to
# run via the container's command, e.g. `docker run <image> api`.

FROM golang:1.27-bookworm AS build
RUN apt-get update && apt-get install -y --no-install-recommends \
    libvips-dev \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ENV CGO_ENABLED=1
RUN go build -o /out/api ./cmd/api
RUN go build -o /out/worker ./cmd/worker
RUN go build -o /out/seed ./cmd/seed
RUN go build -o /out/admin ./cmd/admin
RUN go build -o /out/demoseed ./cmd/demoseed
RUN GOBIN=/out go install -tags "no_clickhouse no_libsql no_mssql no_mysql no_sqlite3 no_vertica no_ydb no_duckdb no_starrocks" \
    github.com/pressly/goose/v3/cmd/goose@v3.30.0

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
    libvips \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*
RUN useradd --system --no-create-home app
COPY --from=build /out/api /out/worker /out/seed /out/admin /out/demoseed /out/goose /usr/local/bin/
COPY db/migrations /migrations
USER app
CMD ["api"]
