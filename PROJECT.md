# Project: Orbit Production Server Deployment Readiness & UI Audit

## Architecture
Orbit delivers container management and infrastructure telemetry with an executive dark console aesthetic (`#111111` canvas, `#181818` surfaces, `#282828` borders). This project readies Orbit for production server deployment by resolving UI collisions, auditing all 13 application views, implementing Lynx-modeled GitHub Actions CI/CD workflows for multi-architecture Docker image publishing, authoring production Docker Compose configurations, and elevating documentation to production standards.

### System Diagram
```
                             GitHub Repository (moresonsunn/orbit)
                                │                      │
                 Push/PR to main│                      │ Tag v*
                                ▼                      ▼
                     ┌──────────────────────┐  ┌──────────────────────┐
                     │ .github/workflows/   │  │ .github/workflows/   │
                     │ ci.yml               │  │ docker-publish.yml   │
                     │ - Go unit tests      │  │ - Buildx multi-arch  │
                     │ - Vite build/typechk │  │ - Docker Hub & GHCR  │
                     └──────────────────────┘  └──────────┬───────────┘
                                                          │
                                                          ▼
                                      ┌──────────────────────────────────────┐
                                      │ Container Registries                 │
                                      │ - Docker Hub (moresonsun/orbit)      │
                                      │ - GitHub Packages (ghcr.io)          │
                                      └──────────────────┬───────────────────┘
                                                         │
                                        docker compose   │ docker run
                                                         ▼
                                      ┌──────────────────────────────────────┐
                                      │ Production Host Deployment           │
                                      │ docker-compose.prod.yml              │
                                      │ - /var/run/docker.sock               │
                                      │ - /data persistent storage           │
                                      │ - 8080:8080 port mapping             │
                                      │ - unless-stopped restart policy      │
                                      └──────────────────┬───────────────────┘
                                                         │
                                                         ▼
                                      ┌──────────────────────────────────────┐
                                      │ Web Client (React 19 / Vite)         │
                                      │ - Collapsed Sidebar (no collision)   │
                                      │ - 13 Audited Executive Views         │
                                      │ - Restored Customize Modal           │
                                      └──────────────────────────────────────┘
```

## Feature Inventory
Every feature identified during survey is inventoried below with its milestone assignment:

| # | Feature | Description | Milestone | Source |
|---|---------|-------------|-----------|--------|
| 1 | Collapsed Sidebar Header Fix | Fix visual collision between logo box and expand toggle (`>`) in `Sidebar.tsx` when collapsed (`w-16`). Center logo at 32px with floating edge toggle on border seam. Center navigation icons at 32px. | M1 | R1, survey_ui |
| 2 | Dashboard Customize Modal Restoration | Implement missing `showConfigModal` dialog JSX in `Dashboard.tsx` with widget toggles and preset buttons; bind `prefs.showContainers` to the active containers table. | M1 | R2, survey_ui |
| 3 | 13-View UI & Browser Audit Polish | Audit and fix minor defects across all 13 views: fix `py-0.2` typos in `NetworkFlow.tsx` and `Networks.tsx`, add `overflow-x-auto` to `ContainerDetail.tsx` tabs, standardize backdrop blur to `backdrop-blur-sm`, standardize text grays, and polish localization. | M1 | R2, survey_ui |
| 4 | GitHub Actions CI Workflow | Create `.github/workflows/ci.yml` running parallel Go backend tests (`go mod verify`, `go vet ./...`, `go test -v ./...`, `go build`) and Frontend typecheck & build (`npm ci`, `npm run build`, asset check) on push/PR to main/master. | M2 | R3, survey_cicd |
| 5 | Multi-Arch Docker Publish Workflow | Create `.github/workflows/docker-publish.yml` and `.github/workflows/release.yml` with QEMU, Buildx, and GHA cache, publishing multi-arch images (`linux/amd64`, `linux/arm64`) to Docker Hub (`moresonsunn/orbit`) and GHCR (`ghcr.io/${{ github.repository }}`). | M2 | R3, survey_cicd |
| 6 | Production Docker Compose Configurations | Create `docker-compose.yml` (development/standalone) and `docker-compose.prod.yml` (production server) with `/var/run/docker.sock` and persistent `/data` volume mounts, `unless-stopped` restart policy, `8080:8080` port mapping, and log rotation. | M3 | R4, survey_deploy |
| 7 | Dynamic Healthcheck Port Resolution | Update `backend/cmd/orbit/main.go` line 30 so `/app/orbit -healthcheck` dynamically queries the port specified by `ORBIT_LISTEN` (handling `:8080`, `:8787`, etc.). | M3 | R4, survey_deploy |
| 8 | Production README & Status Badges | Rewrite `README.md` modeled after Lynx: 5 centered status badges (CI, Docker pulls, Arch, Release, License), 10 feature breakdown items, `docker run` and `docker compose` one-liners, setup token retrieval guide, ASCII architecture diagram & tech table, 22-variable config reference. | M3 | R5, survey_deploy |
| 9 | Requirement-Driven Verification & Audit | Opaque-box verification across all features and tiers followed by independent forensic integrity audit. | M4 | AC, orchestrator |

## Milestones

| # | Name | Scope | Dependencies | Status |
|---|------|-------|-------------|--------|
| M1 | UI Polish: Sidebar & 13 Views | Sidebar collapsed header collision fix, vertical icon alignment, Dashboard customize modal JSX, and view-level polish across all 13 views | none | DONE |
| M2 | CI/CD Automation Workflows | GitHub Actions workflows: `ci.yml`, `docker-publish.yml`, and `release.yml` modeled on Lynx with multi-arch Docker Hub & GHCR publishing | none | DONE |
| M3 | Production Deployment & Docs | `docker-compose.yml`, `docker-compose.prod.yml`, healthcheck dynamic port fix in `main.go`, and complete `README.md` rewrite with badges | none | DONE |
| M4 | Final Integration & Forensic Audit | Global build and test verification (`go test ./...`, `npm run build`), compose config validation, E2E validation, and independent Forensic Integrity Audit | M1, M2, M3 | DONE |

## Interface Contracts

### 1. Sidebar Layout Contract (`Sidebar.tsx`)
- Collapsed width: `w-16` (64px).
- Expanded width: `w-56` (224px).
- Logo: 28px (`w-7 h-7`), centered horizontally at x = 32px in collapsed mode.
- Expand toggle button (`>`): In collapsed mode, positioned as edge-mounted floating handle at `absolute -right-3 top-1/2 -translate-y-1/2 w-6 h-6 rounded-full bg-[#1c1c1c] border border-[#2e2e2e] shadow-md flex items-center justify-center text-neutral-400 hover:text-white z-40`. Zero collision with logo.
- In expanded mode: Logo on left, chevron toggle `<` on right inside `justify-between px-3.5`.
- Navigation icons: Centered horizontally at x = 32px when collapsed using `collapsed ? 'justify-center px-0 py-2' : 'px-2.5 py-1.5'`.

### 2. CI/CD Workflow Contract (`.github/workflows/`)
- Runners: `ubuntu-latest`.
- Go Version: Derived from `backend/go.mod` (Go 1.26).
- Node Version: 22.
- Architecture Platforms: `linux/amd64,linux/arm64`.
- Registries:
  - Docker Hub: `moresonsunn/orbit` (authentication via `DOCKERHUB_USERNAME` / `DOCKERHUB_TOKEN`).
  - GitHub Packages: `ghcr.io/${{ github.repository }}` (authentication via `GITHUB_TOKEN`).
- Caching: `cache-from: type=gha`, `cache-to: type=gha,mode=max`.

### 3. Docker Compose Production Contract (`docker-compose.prod.yml`)
- Service name: `orbit`.
- Image: `moresonsun/orbit:latest`.
- Restart policy: `unless-stopped`.
- Ports: `"8080:8080"`.
- Environment: `ORBIT_LISTEN=:8080`, `ORBIT_DATA_DIR=/data`, `ORBIT_TRUST_PROXY=true`.
- Volumes:
  - `/var/run/docker.sock:/var/run/docker.sock`
  - `orbit_data:/data`
  - `/proc:/host/proc:ro`
  - `/sys:/host/sys:ro`
  - `/etc/os-release:/host/etc/os-release:ro`
- Security: `no-new-privileges:true`.
- Group Add: `"${DOCKER_GID:-999}"`.
- Healthcheck: `["CMD", "/app/orbit", "-healthcheck"]`.

### 4. Backend Healthcheck Contract (`backend/cmd/orbit/main.go`)
- CLI flag: `-healthcheck`.
- Port: Derived from `os.Getenv("ORBIT_LISTEN")`, fallback to `:8787`.
- Endpoint: `http://127.0.0.1:<port>/api/v1/health`.
- Exit code: 0 on 200 OK, 1 on error or non-200.

## Code Layout
- `frontend/src/components/layout/Sidebar.tsx`: Sidebar header layout and navigation item alignment.
- `frontend/src/pages/Dashboard.tsx`: Customize modal JSX dialog and active containers preference toggle.
- `frontend/src/pages/NetworkFlow.tsx`: CSS class cleanup (`py-0.5`).
- `frontend/src/pages/Networks.tsx`: CSS class cleanup (`py-0.5`).
- `frontend/src/pages/ContainerDetail.tsx`: Tab container overflow fix.
- `frontend/src/pages/GPU.tsx` & `frontend/src/pages/Settings.tsx`: Neutral gray palette standardization.
- `frontend/src/pages/Containers.tsx`: Localization polish.
- `frontend/src/components/auth/LoginModal.tsx` & `SetupModal.tsx`: Backdrop blur standardization.
- `.github/workflows/ci.yml`: Automated CI testing pipeline for Go & Node.
- `.github/workflows/docker-publish.yml`: Multi-arch Docker build and push workflow.
- `.github/workflows/release.yml`: GitHub release automation on tags.
- `docker-compose.yml`: Root compose configuration for standalone/development deployment.
- `docker-compose.prod.yml`: Root compose configuration for production server deployment.
- `backend/cmd/orbit/main.go`: Healthcheck port resolution.
- `README.md`: Modern production documentation with status badges and quickstart.
