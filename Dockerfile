FROM node:22-alpine AS css

WORKDIR /app

COPY tailwind.config.js static/css/input.css ./
COPY pkg/views/ ./pkg/views/

RUN npm install tailwindcss@3 && npx tailwindcss -i input.css -o style.css --minify

FROM golang:1.24.4-alpine AS builder

ARG VERSION=${VERSION}

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o omnidex -ldflags "-X main.version=$VERSION -X main.name=omnidex" ./cmd/omnidex/main.go

FROM alpine:3.21

# Create non-root user.
RUN addgroup -S omnidex && adduser -S omnidex -G omnidex

# Create data directories.
RUN mkdir -p /data/docs /data/search && chown -R omnidex:omnidex /data

COPY --from=builder /app/omnidex /usr/local/bin/omnidex
COPY --from=builder /app/static /static
COPY --from=css /app/style.css /static/css/style.css
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

USER omnidex

EXPOSE 8080

ENTRYPOINT ["omnidex"]
# Empty --config disables config file loading; the container is configured
# entirely via environment variables (see .env.example).
CMD ["serve", "--config", ""]
