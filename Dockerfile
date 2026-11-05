# ==========================================
# Stage 1: Build React Frontend
# ==========================================
FROM node:22-alpine AS frontend-builder
WORKDIR /app/frontend

COPY frontend/package*.json ./
RUN npm ci

COPY frontend/ ./
RUN npm run build

# ==========================================
# Stage 2: Build Go Backend
# ==========================================
FROM golang:1.26-bookworm AS backend-builder
WORKDIR /app/backend

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./
# Copy built frontend into backend/web/dist for go:embed
COPY --from=frontend-builder /app/frontend/dist ./web/dist

ARG TARGETOS=linux
ARG TARGETARCH=amd64
ENV CGO_ENABLED=0

RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags="-s -w" -o /app/orbit ./cmd/orbit

# ==========================================
# Stage 3: Production Runtime
# ==========================================
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    tzdata \
    && rm -rf /var/lib/apt/lists/*

# Create non-root orbit user
RUN groupadd -g 10001 orbit && \
    useradd -u 10001 -g orbit -m -s /bin/false orbit

WORKDIR /app
COPY --from=backend-builder /app/orbit /app/orbit

# Data directory
RUN mkdir -p /data && chown -R orbit:orbit /data

ENV ORBIT_LISTEN=":8787" \
    ORBIT_DATA_DIR="/data" \
    ORBIT_DOCKER_HOST="unix:///var/run/docker.sock" \
    ORBIT_HOST_PROC="/host/proc" \
    ORBIT_HOST_SYS="/host/sys" \
    ORBIT_HOST_ETC="/host/etc" \
    ORBIT_HOST_ROOTFS="/host/rootfs"

EXPOSE 8787

HEALTHCHECK --interval=15s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/app/orbit", "-healthcheck"] || exit 0

USER orbit

ENTRYPOINT ["/app/orbit"]

