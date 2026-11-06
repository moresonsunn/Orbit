Prepare Orbit for production server deployment by fixing the collapsed sidebar header collision, performing a comprehensive UI and browser audit across all 13 views, creating complete GitHub Actions CI/CD workflows with multi-arch Docker image publishing to Docker Hub and GHCR modeled on Lynx, providing production docker-compose configurations, and updating documentation.

Working directory: c:\Users\phili\Documents\GitHub\Orbit
Integrity mode: development

Reference material: https://github.com/moresonsunn/Lynx

## Requirements

### R1. Collapsed Sidebar Header Fix
Fix the visual collision in `frontend/src/components/layout/Sidebar.tsx` when the sidebar is in collapsed state (`w-16`). Ensure the logo is properly centered and the expand toggle (`>`) is cleanly positioned without overlapping or colliding with the logo icon.

### R2. Comprehensive UI and Browser Audit Across All Views
Conduct a thorough audit and resolve any layout defects, text clipping, responsiveness issues, and dark theme inconsistencies across all 13 application views:
- Dashboard (`/`)
- Containers (`/containers`)
- Container Detail (`/containers/:id`)
- Network Flow (`/network-flow`)
- Networks (`/networks`)
- Images (`/images`)
- Volumes (`/volumes`)
- GPU Monitoring (`/gpu`)
- Events (`/events`)
- Logs (`/logs`)
- Settings (`/settings`)
- Login (`/login`)
- Setup (`/setup`)

### R3. GitHub Actions CI/CD Workflows
Create GitHub Actions workflows in `.github/workflows/` modeled after Lynx (https://github.com/moresonsunn/Lynx):
- `ci.yml`: Automated CI workflow running on push and pull requests to main/master, executing Go backend unit tests (`go test ./...`) and Frontend typecheck and build (`npm run build`).
- `docker-publish.yml` (or `release.yml`): Multi-architecture build workflow for `linux/amd64` and `linux/arm64` using `docker/setup-buildx-action` and `docker/build-push-action`, publishing images to Docker Hub (`moresonsunn/orbit`) using `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN` secrets and GitHub Packages (`ghcr.io/${{ github.repository }}`) on release tags and main branch pushes.

### R4. Production Docker Deployment Configurations
Create production-ready `docker-compose.yml` and `docker-compose.prod.yml` files at the project root configured for server deployment, including:
- Host Docker daemon socket mounting (`/var/run/docker.sock:/var/run/docker.sock`)
- Persistent data directory mounting (`/data`)
- Standard restart policies (`unless-stopped` or `always`)
- Necessary environment variable configurations and port mappings (`8080:8080` or configurable port)

### R5. Documentation & Badges
Update `README.md` to production standard with:
- Architecture overview and feature breakdown
- One-line quickstart instructions using `docker run` and `docker compose`
- Complete environment variable and configuration reference
- GitHub status badges for Build (CI), Docker pulls/image, License, and GitHub Release

## Acceptance Criteria

### Sidebar Layout
- [ ] In `frontend/src/components/layout/Sidebar.tsx`, when collapsed (`w-16`), the logo and expand toggle button do not overlap or collide.
- [ ] In both collapsed and expanded states, header alignment and toggle transitions render smoothly and cleanly.

### UI & Dark Theme Consistency
- [ ] All 13 listed views (Dashboard, Containers, Container Detail, Network Flow, Networks, Images, Volumes, GPU Monitoring, Events, Logs, Settings, Login, Setup) render cleanly without text truncation, misaligned elements, or dark theme styling breaks.
- [ ] Responsive layouts adjust properly without horizontal overflow or clipped controls on standard desktop and laptop viewports.

### GitHub Actions Workflows
- [ ] `.github/workflows/ci.yml` successfully defines steps for Go backend tests (`go test ./...`) and frontend build (`npm run build`).
- [ ] Multi-arch Docker publish workflow in `.github/workflows/` targets `linux/amd64` and `linux/arm64` using `docker/build-push-action`.
- [ ] Docker workflow configures authentication and tags for both Docker Hub (`moresonsunn/orbit`) and GitHub Container Registry (`ghcr.io/${{ github.repository }}`).

### Production Compose & Documentation
- [ ] `docker-compose.yml` and `docker-compose.prod.yml` validate against Compose specification and define `/var/run/docker.sock` and `/data` volume mounts.
- [ ] `README.md` includes modern quickstart commands, architecture overview, configuration documentation, and status badges.

### Verification
- [ ] Go backend tests pass cleanly (`go test ./...`).
- [ ] Frontend builds cleanly with zero TypeScript errors or bundle failures (`npm run build`).
- [ ] Local web server or live container inspection confirms clean rendering across all 13 views.
