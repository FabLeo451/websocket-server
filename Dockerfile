# ---------- BUILD STAGE ----------
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Dipendenze
COPY go.mod go.sum ./
RUN go mod download

# Codice sorgente
COPY . .

# Build binario statico
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-X 'ekhoes-server/config.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)'" -o ekhoes-server

# ---------- RUNTIME STAGE ----------
FROM alpine:3.20

WORKDIR /app

# Certificati (per HTTPS, DB, ecc.)
RUN apk add --no-cache ca-certificates

# Copia binario
COPY --from=builder /app/ekhoes-server .

# Porta (se il tuo server espone una porta)
EXPOSE 9876

# Comando di avvio
CMD ["./ekhoes-server", "start"]
