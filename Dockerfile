# One image serves both the API and the web UI (deployed as a single Render service).
#
# Stage 1 builds the frontend from its own public repository. The ADD below
# fetches the latest commit of the frontend branch, so Docker's layer cache is
# invalidated whenever the frontend changes.
FROM node:22 AS web
ARG FRONTEND_REPO=https://github.com/SunatSS/aura-spaceapps-frontend.git
ARG FRONTEND_API=https://api.github.com/repos/SunatSS/aura-spaceapps-frontend/commits/main
ARG FRONTEND_REF=main
ADD ${FRONTEND_API} /tmp/frontend-version.json
RUN git clone --depth 1 --branch "${FRONTEND_REF}" "${FRONTEND_REPO}" /web
WORKDIR /web
RUN npm ci && npm run build

# Stage 2 builds the Go server.
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/aura ./cmd/aura && mkdir -p /out/data

# Stage 3: minimal runtime.
FROM gcr.io/distroless/static-debian12
COPY --from=build /out/aura /aura
COPY --from=build /out/data /data
COPY --from=web /web/dist /web
# Render injects PORT; the DB lives in the container and is re-seeded on each start.
ENV AURA_DB_PATH=/data/aura.db \
    AURA_STATIC_DIR=/web
EXPOSE 8080
ENTRYPOINT ["/aura"]
