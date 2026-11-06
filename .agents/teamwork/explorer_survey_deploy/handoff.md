# Deployment & Documentation Survey Report

## 1. Observation

### 1.1 Existing Orbit Root & Deployment Structure
- **Root Directory Files**:
  - `docker-compose.yml`: **Missing** from repository root.
  - `docker-compose.prod.yml`: **Missing** from repository root.
  - `Dockerfile` (exists at `Dockerfile`): 3-stage multi-stage build:
    - Stage 1 (`node:22-alpine`): builds React SPA via `npm ci && npm run build` (`Dockerfile:4-11`).
    - Stage 2 (`golang:1.26-bookworm`): embeds frontend into `backend/web/dist` and compiles static binary `CGO_ENABLED=0 go build -ldflags="-s -w" -o /app/orbit ./cmd/orbit` (`Dockerfile:16-31`).
    - Stage 3 (`debian:bookworm-slim`): creates non-root user `orbit` (UID 10001, GID 10001), creates `/data` directory with ownership `orbit:orbit`, exposes port `8787`, specifies `HEALTHCHECK --interval=15s --timeout=3s --start-period=5s --retries=3 CMD ["/app/orbit", "-healthcheck"] || exit 0`, runs as `USER orbit`, entrypoint `["/app/orbit"]` (`Dockerfile:35-67`).
  - `README.md` (exists at `README.md`): Currently a short 65-line German-language document focused primarily on CasaOS without CI badges, without `docker run` quickstart, without comprehensive environment variable tables, and without feature breakdowns for the 10 core capability areas.
- **Existing Compose Artifacts in `deploy/`**:
  - `deploy/docker-compose.yml`:
    - Image: `ghcr.io/phili/orbit:latest`
    - Ports: `"8787:8787"`
    - Volumes: `/DATA/AppData/orbit/data:/data`, `/var/run/docker.sock:/var/run/docker.sock:ro`, `/proc:/host/proc:ro`, `/sys:/host/sys:ro`, `/etc/os-release:/host/etc/os-release:ro`
    - Group add: `"${DOCKER_GID:-999}"`
    - Environment defaults: `ORBIT_LISTEN=:8787`, `ORBIT_DATA_DIR=/data`, `ORBIT_ENABLE_ACTIONS=false`, `ORBIT_ENABLE_DESTRUCTIVE=false`, `ORBIT_GPU=auto`, `ORBIT_EBPF=auto`, `ORBIT_LOG_LEVEL=info`
  - `deploy/docker-compose.gpu.yml`:
    - Nvidia reservation override: `driver: nvidia`, `count: all`, `capabilities: [gpu, utility]`.
  - `deploy/casaos/docker-compose.yml`:
    - CasaOS AppStore metadata labels.

### 1.2 Backend Configuration & Port Binding Analysis
- In `backend/internal/config/config.go:54-102`:
  - `Listen: env("ORBIT_LISTEN", ":8787")`
  - `DataDir: env("ORBIT_DATA_DIR", "/data")`
  - `DockerHost: env("ORBIT_DOCKER_HOST", env("DOCKER_HOST", "unix:///var/run/docker.sock"))`
  - `HostProc: env("ORBIT_HOST_PROC", "/host/proc")`
  - `HostSys: env("ORBIT_HOST_SYS", "/host/sys")`
  - `HostEtc: env("ORBIT_HOST_ETC", "/host/etc")`
  - `HostRootFS: env("ORBIT_HOST_ROOTFS", "/host/rootfs")`
  - `EnableActions: envBool("ORBIT_ENABLE_ACTIONS", false)`
  - `EnableDestructive: envBool("ORBIT_ENABLE_DESTRUCTIVE", false)`
  - `EBPFMode: strings.ToLower(env("ORBIT_EBPF", "auto"))`
  - `GPUMode: strings.ToLower(env("ORBIT_GPU", "auto"))`
  - `TrustProxy: envBool("ORBIT_TRUST_PROXY", false)`
  - `SecureCookies: strings.ToLower(env("ORBIT_SECURE_COOKIES", "auto"))`
  - `RunAsUID: envInt("ORBIT_UID", 10001)`
  - `RunAsGID: envInt("ORBIT_GID", 10001)`
  - `KeepRoot: envBool("ORBIT_KEEP_ROOT", false)`
  - `MaxStatsStreams: envInt("ORBIT_MAX_STATS_STREAMS", 250)`
  - `LogLevel: strings.ToLower(env("ORBIT_LOG_LEVEL", "info"))`
  - `DevMode: envBool("ORBIT_DEV", false)`
  - `TrustedOrigins: strings.Split(env("ORBIT_TRUSTED_ORIGINS", ""), ",")`
  - `SessionTTL: envDuration("ORBIT_SESSION_TTL", 7*24*time.Hour)`
  - `SessionIdle: envDuration("ORBIT_SESSION_IDLE", 12*time.Hour)`
- In `backend/cmd/orbit/main.go:30-37`:
  ```go
  if len(os.Args) > 1 && os.Args[1] == "-healthcheck" {
      client := http.Client{Timeout: 2 * time.Second}
      resp, err := client.Get("http://127.0.0.1:8787/api/v1/health")
      if err != nil || resp.StatusCode != http.StatusOK {
          os.Exit(1)
      }
      os.Exit(0)
  }
  ```
  - Crucial observation: `main.go:32` hardcodes `http://127.0.0.1:8787/api/v1/health`.
  - The runtime container base `debian:bookworm-slim` does **not** install `curl` or `wget`.
  - Consequently, if a user specifies `ORBIT_LISTEN=:8080`, the internal healthcheck will query `:8787`, get `connection refused`, and fail.
  - To support arbitrary listen ports (including `:8080`), `main.go:32` should dynamically resolve the port from `os.Getenv("ORBIT_LISTEN")`.
  - Alternatively, in Docker compose port mappings, mapping host port 8080 to container port 8787 (`"8080:8787"`) or allowing configurable host port `"${ORBIT_PORT:-8080}:8080"` with `ORBIT_LISTEN=:8080` (once patched) solves this completely.

### 1.3 Lynx Reference Repository (`https://github.com/moresonsunn/Lynx`)
Direct inspection of Lynx repository on GitHub revealed:
- **Lynx `README.md` Structure**:
  - Centered title and description header (`<p align="center">...`).
  - Centered status badges:
    - CI Workflow: `https://github.com/moresonsunn/Lynx/actions/workflows/ci.yml/badge.svg`
    - Docker Hub pulls: `https://img.shields.io/docker/pulls/moresonsun/lynx`
    - Architecture: `https://img.shields.io/badge/arch-amd64%20%7C%20arm64-blue`
    - License: `https://img.shields.io/badge/license-proprietary-lightgrey`
  - High-level feature breakdown with bold category tags.
  - Quick Start section:
    - Requirements
    - One-line install (curl for Linux/macOS, irm for Windows)
    - Manual install with Docker Compose (`curl docker-compose.yml && docker compose up -d`)
    - Post-install setup instructions (default password warning, logs retrieval)
  - ASCII Architecture block diagram + Tech stack table.
  - Development instructions (backend / frontend startup commands).
  - Clean documentation table referencing `docs/` files.
- **Lynx `docker-compose.yml` Pattern**:
  - Service definition with `image`, `build` context/args, `container_name`.
  - Restart policy: `unless-stopped`.
  - Port mapping: `"8000:8000"`.
  - Volume definitions: named volumes for app data (`servers_data:/data/servers`, `sqlite_data:/data/sqlite`) and socket mount (`/var/run/docker.sock:/var/run/docker.sock`).
  - Structured json-file logging: `max-size: "10m"`, `max-file: "3"`.
  - Healthcheck block.
  - CasaOS labels: `io.casaos.app`, `io.casaos.managed`.
- **Lynx CI/CD (`docker-build-publish.yml`) Pattern**:
  - Multi-arch `docker/setup-qemu-action@v3` + `docker/setup-buildx-action@v3`.
  - Authentication against Docker Hub using `secrets.DOCKER_USERNAME` / `secrets.DOCKER_TOKEN` (or `DOCKERHUB_USERNAME` / `DOCKERHUB_TOKEN`).
  - Dual publishing to Docker Hub and GHCR (`ghcr.io/${{ github.repository }}`).

---

## 2. Logic Chain

1. **Root Compose File Absence**: Orbit currently lacks `docker-compose.yml` and `docker-compose.prod.yml` at the project root (`C:\Users\phili\Documents\GitHub\Orbit\`). Having root-level Compose configurations is standard convention for developer ergonomics, container management tools (Dockge, Portainer, CasaOS), and one-line curl deployment scripts.
2. **Development vs. Production Separation**:
   - `docker-compose.yml` should support both local building from source (`build: { context: ., dockerfile: Dockerfile }`) and pulling published images (`image: moresonsun/orbit:latest`), allowing developers to run `docker compose up -d --build` or users to spin up Orbit instantly.
   - `docker-compose.prod.yml` is streamlined strictly for production deployments on Linux servers. It pulls the pre-built multi-arch image (`image: moresonsun/orbit:latest` or `ghcr.io/phili/orbit:latest`), omits build contexts, mounts `/var/run/docker.sock:/var/run/docker.sock`, mounts persistent data `/data`, configures `restart: unless-stopped`, sets `no-new-privileges: true`, configures json-file log rotation (`max-size: 20m`, `max-file: 5`), and enables reverse-proxy trust (`ORBIT_TRUST_PROXY=true`).
3. **Port Strategy**:
   - The user specification mandates: "port mappings (`8080:8080` or configurable port)".
   - Orbit's internal web server defaults to `:8787`.
   - If container port is set to `8080` (`ORBIT_LISTEN=:8080`), `backend/cmd/orbit/main.go:32` must be made aware of `ORBIT_LISTEN` for its `-healthcheck` flag.
   - Therefore, the recommended production pattern is:
     - Configurable host port: `"${ORBIT_PORT:-8080}:8080"` with `ORBIT_LISTEN=:8080` and the 5-line `main.go` healthcheck fix.
     - Alternatively, port mapping `"8080:8787"` works out-of-the-box with zero Go code changes.
     - Providing both configurations ensures total robustness.
4. **README Production Standard**:
   - The current `README.md` is in German, short, and CasaOS-centric.
   - Lynx sets the benchmark with English, clean typography, badge layout, ASCII architecture diagrams, one-line quickstarts for both `docker run` and `docker compose`, detailed feature breakdowns, and comprehensive tables.
   - The rewritten README should preserve all architectural truths (CFS delta CPU math, cgroup v1/v2 inactive_file memory math, procfs host metrics, eBPF/DRM/NVML telemetry, setup token security) while adopting Lynx's aesthetic and structure.

---

## 3. Detailed Specifications for Implementation

### 3.1 Specification: `docker-compose.yml` (Development / Standalone)
Path: `c:\Users\phili\Documents\GitHub\Orbit\docker-compose.yml`

```yaml
# Orbit - Self-hosted Docker Infrastructure, Management & Real-Time Monitoring Platform
# Standalone & Development Deployment

services:
  orbit:
    image: moresonsun/orbit:latest
    build:
      context: .
      dockerfile: Dockerfile
    container_name: orbit
    restart: unless-stopped
    ports:
      - "${ORBIT_PORT:-8080}:8080"
    environment:
      - ORBIT_LISTEN=:8080
      - ORBIT_DATA_DIR=/data
      - ORBIT_ENABLE_ACTIONS=${ORBIT_ENABLE_ACTIONS:-false}
      - ORBIT_ENABLE_DESTRUCTIVE=${ORBIT_ENABLE_DESTRUCTIVE:-false}
      - ORBIT_GPU=${ORBIT_GPU:-auto}
      - ORBIT_EBPF=${ORBIT_EBPF:-auto}
      - ORBIT_LOG_LEVEL=${ORBIT_LOG_LEVEL:-info}
    volumes:
      - orbit_data:/data
      - /var/run/docker.sock:/var/run/docker.sock
      - /proc:/host/proc:ro
      - /sys:/host/sys:ro
      - /etc/os-release:/host/etc/os-release:ro
    security_opt:
      - no-new-privileges:true
    group_add:
      - "${DOCKER_GID:-999}"
    healthcheck:
      test: ["CMD", "/app/orbit", "-healthcheck"]
      interval: 15s
      timeout: 5s
      retries: 3
      start_period: 10s
    logging:
      driver: json-file
      options:
        max-size: "10m"
        max-file: "3"

volumes:
  orbit_data:
```

### 3.2 Specification: `docker-compose.prod.yml` (Production Server Deployment)
Path: `c:\Users\phili\Documents\GitHub\Orbit\docker-compose.prod.yml`

```yaml
# Orbit - Production Server Deployment
# Usage: docker compose -f docker-compose.prod.yml up -d

services:
  orbit:
    image: moresonsun/orbit:latest
    container_name: orbit
    restart: unless-stopped
    ports:
      - "8080:8080"
    environment:
      - ORBIT_LISTEN=:8080
      - ORBIT_DATA_DIR=/data
      - ORBIT_ENABLE_ACTIONS=${ORBIT_ENABLE_ACTIONS:-false}
      - ORBIT_ENABLE_DESTRUCTIVE=${ORBIT_ENABLE_DESTRUCTIVE:-false}
      - ORBIT_GPU=${ORBIT_GPU:-auto}
      - ORBIT_EBPF=${ORBIT_EBPF:-auto}
      - ORBIT_LOG_LEVEL=${ORBIT_LOG_LEVEL:-info}
      - ORBIT_TRUST_PROXY=true
      - ORBIT_TRUSTED_ORIGINS=${ORBIT_TRUSTED_ORIGINS:-}
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - orbit_data:/data
      - /proc:/host/proc:ro
      - /sys:/host/sys:ro
      - /etc/os-release:/host/etc/os-release:ro
    security_opt:
      - no-new-privileges:true
    group_add:
      - "${DOCKER_GID:-999}"
    healthcheck:
      test: ["CMD", "/app/orbit", "-healthcheck"]
      interval: 15s
      timeout: 5s
      retries: 3
      start_period: 10s
    logging:
      driver: json-file
      options:
        max-size: "20m"
        max-file: "5"

volumes:
  orbit_data:
    name: orbit_data
```

### 3.3 Proposed Patch for `backend/cmd/orbit/main.go` (Healthcheck Port Resolution)
To make `/app/orbit -healthcheck` dynamically adapt to any `ORBIT_LISTEN` port (e.g. `:8080`):
```go
// In backend/cmd/orbit/main.go line 30:
if len(os.Args) > 1 && os.Args[1] == "-healthcheck" {
    port := os.Getenv("ORBIT_LISTEN")
    if port == "" {
        port = ":8787"
    }
    if idx := strings.LastIndex(port, ":"); idx != -1 {
        port = port[idx:]
    } else {
        port = ":" + port
    }
    client := http.Client{Timeout: 2 * time.Second}
    resp, err := client.Get("http://127.0.0.1" + port + "/api/v1/health")
    if err != nil || resp.StatusCode != http.StatusOK {
        os.Exit(1)
    }
    os.Exit(0)
}
```

### 3.4 Specification: `README.md` Layout & Content

```markdown
<p align="center">
  <h1 align="center">Orbit</h1>
  <p align="center">Self-hosted Docker infrastructure, management and real-time telemetry platform</p>
</p>

<p align="center">
  <a href="https://github.com/moresonsunn/orbit/actions/workflows/ci.yml"><img src="https://github.com/moresonsunn/orbit/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://hub.docker.com/r/moresonsun/orbit"><img src="https://img.shields.io/docker/pulls/moresonsun/orbit" alt="Docker Pulls"></a>
  <img src="https://img.shields.io/badge/arch-amd64%20%7C%20arm64-blue" alt="Architecture">
  <a href="https://github.com/moresonsunn/orbit/releases"><img src="https://img.shields.io/github/v/release/moresonsunn/orbit?color=brightgreen" alt="Release"></a>
  <img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License">
</p>

---

Orbit is a modern, standalone Docker infrastructure management and real-time monitoring platform designed for homelabs, Linux servers, and CasaOS. Orbit runs as a single lightweight container, connects to the Docker Engine daemon (`/var/run/docker.sock`), and visualizes containers, UniFi-style network IPAM, interactive network flows, socket traffic, and GPU telemetry in an executive dark-themed interface.

## Features

- **Containers** — Real-time state tracking and lifecycle control (start, stop, restart, pause, unpause, delete). High-fidelity metrics matching `docker stats` exact formulas (CFS deltas, cgroup v1/v2 inactive_file memory subtraction, block I/O, network rates).
- **Networks** — Multi-driver inspection (bridge, host, overlay, macvlan, ipvlan). Scope, internal isolation status, IPv6 enablement, IPAM driver, driver options, and live connected container attachments.
- **IPAM (IP Address Management)** — UniFi-inspired subnet inspector. Visual CIDR breakdown, gateway, network/broadcast IPs, usable host range, and live IP allocation capacity progress bars.
- **Network Flow** — Interactive topology graph (`@xyflow/react`) mapping Host -> Networks -> Containers with animated traffic intensity pulses. Live RX/TX rate meters, protocol traffic distribution, and an active socket table attributing connections to containers with automated port security risk classification (Critical, Caution, Safe, Internal).
- **GPU Telemetry** — Out-of-the-box telemetry for NVIDIA GPUs (NVML) and AMD/Intel GPUs (DRM). Real-time utilization %, VRAM used/total, temperature, fan speed, power draw, and container process attribution. Graceful clean fallback when no GPU is detected.
- **Images** — Comprehensive image catalog with tags, digests, layer breakdown, creation history, pull from registries, tagging, and dangling image prune.
- **Volumes** — Volume catalog with driver, mountpoints, size estimation, in-use container detection (preventing accidental deletion), and unused volume pruning.
- **Events** — Real-time Docker daemon event streaming over WebSocket. Filter by event type, action, actor ID, and time range.
- **Logs** — High-performance live log streaming with client-side backpressure protection, ANSI color parsing, search, and regex filters.
- **Auth & Security** — Zero default passwords. Single-use cryptographically generated setup token on first launch. Argon2id password hashing, synchronized CSRF tokens, origin validation, rate limiting, and read-only mode by default (`ORBIT_ENABLE_ACTIONS=false`).

## Quick Start

### One-Line Docker Run

```bash
docker run -d \
  --name orbit \
  --restart unless-stopped \
  -p 8080:8080 \
  -e ORBIT_LISTEN=:8080 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v orbit_data:/data \
  -v /proc:/host/proc:ro \
  -v /sys:/host/sys:ro \
  -v /etc/os-release:/host/etc/os-release:ro \
  moresonsun/orbit:latest
```

### One-Line Docker Compose

```bash
curl -fsSL https://raw.githubusercontent.com/moresonsunn/orbit/main/docker-compose.prod.yml -o docker-compose.yml && docker compose up -d
```

### First-Time Setup
1. Retrieve the single-use setup token from container logs:
   ```bash
   docker logs orbit | grep "SETUP TOKEN"
   ```
2. Open `http://<your-server-ip>:8080` in your browser.
3. Paste the setup token and create your administrator credentials. The token is permanently invalidated once the admin user is created.

## Architecture

```
┌────────────────────────────────────────────────────────────────────────┐
│             Web Client (React 19 + TypeScript + Tailwind CSS)           │
│         Tailscale Dark Aesthetic (#111111 canvas, #181818 surface)     │
└───────────────────────────────────┬────────────────────────────────────┘
                                    │ HTTP REST API / WebSocket
                                    ▼
┌────────────────────────────────────────────────────────────────────────┐
│                     Orbit Single Binary (Go 1.26)                      │
│                                                                        │
│   Chi Router ──► Security Middleware (Argon2id, CSRF, RBAC, Rate-Limit)│
│                                                                        │
│   ┌────────────────────┬────────────────────┬──────────────────────┐   │
│   │ Realtime Hub (WS)  │ Alert Engine       │ Metrics History      │   │
│   ├────────────────────┼────────────────────┼──────────────────────┤   │
│   │ Docker Gateway     │ Host Collector     │ Netflow & IPAM       │   │
│   │ (Strict Allowlist) │ (/proc & /sys)     │ (Procfs + Sockets)   │   │
│   ├────────────────────┼────────────────────┼──────────────────────┤   │
│   │ Container Inventory│ GPU Manager        │ Embedded SQLite      │   │
│   │ (Events & Stats)   │ (NVML / DRM)       │ (Pure Go Driver)     │   │
│   └────────────────────┴────────────────────┴──────────────────────┘   │
└───────────────────────────────────┬────────────────────────────────────┘
                                    │ Mounts & Engine Socket
                                    ▼
┌────────────────────────────────────────────────────────────────────────┐
│                       Host & Docker Infrastructure                     │
│   /var/run/docker.sock       /proc & /sys          Persistent /data    │
│   (Docker Engine API)    (Host Telemetry)       (SQLite DB & Config)   │
└────────────────────────────────────────────────────────────────────────┘
```

| Layer | Technologies |
|---|---|
| Frontend | React 19, TypeScript, Tailwind CSS v4, Vite, @xyflow/react, Lucide Icons, Recharts |
| Backend | Go 1.26, Chi v5 HTTP router, Coder WebSocket, pure-Go SQLite |
| Metrics Engine | Real-time cgroup v1/v2 parser, Linux procfs collector, NVIDIA NVML, Linux DRM |
| Security | Argon2id password hashing, synchronized CSRF tokens, strict Docker Gateway allowlist, non-root container |
| Packaging | Multi-arch Docker (`linux/amd64`, `linux/arm64`), single standalone binary, embedded SPA & SQLite |

## Configuration & Environment Variables

Orbit is configured via environment variables in `docker-compose.yml` or your `.env` file:

### Server & Storage

| Variable | Default | Description |
|---|---|---|
| `ORBIT_LISTEN` | `:8787` (or `:8080`) | Listen address and port for web UI and API. |
| `ORBIT_DATA_DIR` | `/data` | Path inside container where SQLite database and configs are stored. |
| `ORBIT_DOCKER_HOST` | `unix:///var/run/docker.sock` | Docker Engine API endpoint (Unix socket or TCP socket proxy). |
| `ORBIT_LOG_LEVEL` | `info` | Logging verbosity (`debug`, `info`, `warn`, `error`). |

### Security & Feature Gates

| Variable | Default | Description |
|---|---|---|
| `ORBIT_ENABLE_ACTIONS` | `false` | Enable container management actions (start, stop, restart, pause). Defaults to read-only. |
| `ORBIT_ENABLE_DESTRUCTIVE` | `false` | Enable container deletion, image pruning, and volume removal. Requires `ORBIT_ENABLE_ACTIONS=true`. |
| `ORBIT_TRUST_PROXY` | `false` | Trust `X-Forwarded-For` and `X-Real-IP` headers from reverse proxies (Traefik, Nginx, Caddy). |
| `ORBIT_TRUSTED_ORIGINS` | `""` | Comma-separated list of allowed origins for WebSocket connections and CORS. |
| `ORBIT_SECURE_COOKIES` | `auto` | Enforce `Secure` attribute on cookies (`auto`, `true`, `false`). |
| `ORBIT_SESSION_TTL` | `168h` | Session lifetime before re-login is required (e.g. `168h` = 7 days). |
| `ORBIT_SESSION_IDLE` | `12h` | Maximum idle inactivity before session expires. |

### Telemetry & Hardware

| Variable | Default | Description |
|---|---|---|
| `ORBIT_GPU` | `auto` | GPU telemetry mode (`auto` detects NVIDIA NVML and AMD/Intel DRM, `off` disables). |
| `ORBIT_EBPF` | `auto` | eBPF socket monitoring mode (`auto`, `on`, `off`). |
| `ORBIT_MAX_STATS_STREAMS` | `250` | Maximum concurrent `docker stats` streams multiplexed by backend. |
| `ORBIT_HOST_PROC` | `/host/proc` | Host `/proc` filesystem mount point. |
| `ORBIT_HOST_SYS` | `/host/sys` | Host `/sys` filesystem mount point. |
| `ORBIT_HOST_ETC` | `/host/etc` | Host `/etc` mount point (for `/etc/os-release`). |
| `ORBIT_HOST_ROOTFS` | `/host/rootfs` | Host root filesystem mount point for disk stats. |

### Permissions & System

| Variable | Default | Description |
|---|---|---|
| `DOCKER_GID` | `999` | Host docker group ID to grant non-root container user access to `/var/run/docker.sock`. |
| `ORBIT_UID` | `10001` | Non-root user ID inside container. |
| `ORBIT_GID` | `10001` | Non-root group ID inside container. |
| `ORBIT_DEV` | `false` | Development mode flag enabling debug endpoints. |

## Development

```bash
# Backend development
cd backend
go run ./cmd/orbit

# Frontend development
cd frontend
npm install
npm run dev
```

## Documentation

| Document | Description |
|---|---|
| [System Architecture](docs/ARCHITECTURE.md) | Architectural diagram, metric calculation formulas (CFS deltas, cgroup v1/v2), and telemetry |
| [CasaOS & Linux Installation](docs/INSTALLATION.md) | Detailed installation instructions for CasaOS, ZimaOS, and bare Linux servers |
| [Security Concept](docs/SECURITY.md) | Defense-in-depth model, Docker socket protection, Argon2id, CSRF, and non-root execution |

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
```

---

## 4. Caveats
1. **Healthcheck Port Dependency**:
   - `Dockerfile` defines `HEALTHCHECK CMD ["/app/orbit", "-healthcheck"] || exit 0`.
   - `backend/cmd/orbit/main.go:32` contains hardcoded `8787`.
   - If `ORBIT_LISTEN=:8080` is deployed, updating `main.go:30-37` to read `os.Getenv("ORBIT_LISTEN")` is necessary for `-healthcheck` to succeed without relying on the fallback `|| exit 0`.
2. **Docker Socket GID on Host**:
   - On Linux distributions, the `docker` group GID is often `998` or `999` or `1001`. Using `group_add: ["${DOCKER_GID:-999}"]` in docker-compose allows users to override `DOCKER_GID=$(getent group docker | cut -d: -f3)` if their host GID differs.
3. **Repository Namespace**:
   - In GitHub Actions workflows and image references, `moresonsun/orbit` or `moresonsunn/orbit` is referenced. The compose files should use `moresonsun/orbit:latest` with optional fallback to `ghcr.io/phili/orbit:latest` or `ghcr.io/${{ github.repository }}`.

---

## 5. Conclusion
1. **Root Compose Files Required**:
   - Create `docker-compose.yml` for developer / standalone workflow (with build stanza + image).
   - Create `docker-compose.prod.yml` for production server deployments (pre-built image, `/var/run/docker.sock:/var/run/docker.sock`, `/data` mount, `unless-stopped`, `no-new-privileges:true`, log rotation, port `8080:8080`).
2. **README Overhaul**:
   - Completely replace the German CasaOS-only `README.md` with a comprehensive English production README modeled on Lynx.
   - Include 5 centered status badges, 10 feature breakdown items, `docker run` and `docker compose` one-liners, first-time setup token retrieval guide, ASCII architecture diagram + tech table, complete 22-variable configuration reference, and documentation links.
3. **Healthcheck Alignment**:
   - Apply the lightweight patch to `backend/cmd/orbit/main.go` so `-healthcheck` respects `ORBIT_LISTEN`.

---

## 6. Verification Method

1. **Compose Configuration Validation**:
   ```powershell
   docker compose -f docker-compose.yml config
   docker compose -f docker-compose.prod.yml config
   ```
   Both commands must exit code 0 with valid parsed YAML syntax without schema warnings.
2. **README Rendering & Link Integrity**:
   Verify markdown syntax, valid shield badge URLs, code fence blocks, and relative links to `docs/ARCHITECTURE.md`, `docs/INSTALLATION.md`, and `docs/SECURITY.md`.
3. **Backend Build & Tests**:
   ```powershell
   cd backend
   go test ./...
   ```
   Ensure all Go tests pass cleanly.

