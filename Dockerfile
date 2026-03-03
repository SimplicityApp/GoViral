# Stage 1: Build Go binary
FROM golang:1.24-bookworm AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -tags embedweb -o /goviral ./apps/server

# Stage 2: Runtime with Python + Chromium
FROM python:3.12-slim-bookworm

RUN apt-get update && apt-get install -y --no-install-recommends \
    chromium \
    ca-certificates \
    fonts-liberation \
    fonts-dejavu-core \
    gosu \
    && rm -rf /var/lib/apt/lists/*

ENV CHROMIUM_PATH=/usr/bin/chromium

# Create non-root user (entrypoint handles the user switch)
RUN useradd -m -s /bin/bash goviral

COPY --from=builder /goviral /usr/local/bin/goviral
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

EXPOSE 8080

# Runs as root initially to fix volume permissions, then drops to goviral
ENTRYPOINT ["/entrypoint.sh"]
