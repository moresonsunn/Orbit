<p align="center">
  <h1 align="center">Orbit</h1>
  <p align="center">Self-hosted Docker infrastructure, management and real-time telemetry platform</p>
</p>

<p align="center">
  <a href="https://github.com/moresonsunn/orbit/actions/workflows/ci.yml"><img src="https://github.com/moresonsunn/orbit/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://hub.docker.com/r/moresonsunn/orbit"><img src="https://img.shields.io/docker/pulls/moresonsunn/orbit" alt="Docker Pulls"></a>
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
  moresonsunn/orbit:latest
```

### One-Line Docker Compose

```bash
curl -fsSL https://raw.githubusercontent.com/moresonsunn/orbit/main/docker-compose.prod.yml -o docker-compose.yml && docker compose up -d
```

### CasaOS Installation

Orbit includes full native support and packaging for **CasaOS** and **ZimaOS**:

#### Method 1: 1-Click Custom Install (CasaOS Web UI)
1. Open the **App Store** on your CasaOS dashboard.
2. Click **Custom Install** (top right) and then click **Import**.
3. Copy & paste the contents of [`Apps/Orbit/docker-compose.yml`](Apps/Orbit/docker-compose.yml).
4. Click **Submit** and then **Install**. The Orbit tile will be created on your dashboard.

#### Method 2: Custom AppStore Repository Setup
1. In CasaOS, open the **App Store** and click **Manage App Stores / Add Source**.
2. Enter the repository ZIP archive URL:
   ```text
   https://github.com/moresonsunn/Orbit/archive/refs/heads/main.zip
   ```
3. Click **Add**. Orbit will appear directly inside the App Store catalog under the **Developer** category with 1-click install and automated update notifications.

See the complete [CasaOS & Linux Installation Guide](docs/INSTALLATION.md) for full configuration options and details.

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
| `ORBIT_KEEP_ROOT` | `false` | Run container process as root without dropping privileges. |
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
