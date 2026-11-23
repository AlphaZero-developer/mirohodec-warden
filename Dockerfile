# Backend-only image. The optional Mafia Mini App client is not bundled.
FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/warden ./cmd/warden

FROM alpine:3.21
RUN addgroup -S warden && adduser -S -G warden warden
WORKDIR /app
COPY --from=build /out/warden /usr/local/bin/warden
COPY config.example.yaml /app/config.yaml
RUN mkdir -p /app/data && chown -R warden:warden /app
USER warden
EXPOSE 8090
ENTRYPOINT ["warden", "-config", "/app/config.yaml"]
