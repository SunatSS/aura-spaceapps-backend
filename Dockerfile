# One image serves both the API and the web UI (deployed as a single Render service).
# Build from the repository root: docker build -t aura .

# Stage 1 builds the frontend.
FROM node:22-slim AS web
WORKDIR /web
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Stage 2 builds the Go server.
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/aura ./cmd/aura && mkdir -p /out/data

# Stage 3: minimal runtime.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/aura /aura
COPY --from=build --chown=65532:65532 /out/data /data
COPY --from=web /web/dist /web
# Render injects PORT. Without a volume the DB lives in the container and is re-seeded on each start.
ENV AURA_DB_PATH=/data/aura.db \
    AURA_STATIC_DIR=/web
EXPOSE 8080
ENTRYPOINT ["/aura"]
