# Multi-stage build: Bun builds the React client, Go builds a static server
# binary, and a minimal runtime image ships both plus the dataset.

# --- Stage 1: frontend build ---------------------------------------------
FROM oven/bun:1 AS frontend
WORKDIR /app
COPY frontend/package.json frontend/bun.lock* ./
RUN bun install
COPY frontend/ ./
# Vite output lands in /app/dist; the Go server serves it via -static.
RUN bun run build

# --- Stage 2: Go server binary --------------------------------------------
FROM golang:1.22-alpine AS backend
WORKDIR /src
COPY backend_go/go.mod backend_go/go.sum ./
RUN go mod download
COPY backend_go/ ./
RUN CGO_ENABLED=0 GOFLAGS=-trimpath go build -ldflags="-s -w" -o /out/football-sim ./cmd/server

# --- Stage 3: runtime ------------------------------------------------------
FROM alpine:3.20
RUN adduser -D -u 10001 sim
WORKDIR /app
COPY --from=backend /out/football-sim /app/football-sim
COPY --from=frontend /app/dist /app/frontend/dist
COPY dataset.json /app/dataset.json
RUN mkdir -p /app/saves && chown -R sim /app
USER sim
EXPOSE 8000
ENV FOOTBALL_SIM_SAVE=/app/saves/career.json
ENTRYPOINT ["/app/football-sim"]
CMD ["-host", "0.0.0.0", "-port", "8000", "-dataset", "/app/dataset.json", "-static", "/app/frontend/dist", "-save", "/app/saves/career.json"]
