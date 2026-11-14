FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/aura ./cmd/aura

FROM gcr.io/distroless/static-debian12
COPY --from=build /out/aura /aura
ENV AURA_DB_PATH=/data/aura.db
VOLUME ["/data"]
EXPOSE 8080
ENTRYPOINT ["/aura"]
