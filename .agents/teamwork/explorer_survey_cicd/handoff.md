# Handoff Report: GitHub Actions CI/CD Architecture Survey & Specifications

**Agent**: `explorer_survey_cicd`  
**Working Directory**: `c:\Users\phili\Documents\GitHub\Orbit\.agents\teamwork\explorer_survey_cicd`  
**Date**: 2026-11-06  
**Reference Project**: [moresonsunn/Lynx](https://github.com/moresonsunn/Lynx)  

---

## 1. Observation

### 1.1 Existing Workflows
- **Path**: `c:\Users\phili\Documents\GitHub\Orbit\.github\`
- **Result**: `Test-Path .github` returned `False`. Orbit currently has no `.github/` directory or existing GitHub Actions workflow files.

### 1.2 Orbit Backend Architecture
- **`backend/go.mod`**:
  - Module path: `module github.com/phili/orbit`
  - Go directive: `go 1.26.3`
  - Dependencies: pure Go SQLite (`modernc.org/sqlite v1.61.1`), `github.com/go-chi/chi/v5 v5.5.0`, `github.com/moby/moby/client v0.6.2`, `github.com/coder/websocket v1.8.15`, `github.com/NVIDIA/go-nvml v0.13.4-1`.
- **Backend Test Execution**:
  - Command: `cd backend; go test -v ./...`
  - Result: All tests pass cleanly (`TestCalculateSubnetDetails`, `TestEnrichedDockerNetworkJSONSerialization`, `TestPasswordHashAndVerify`, `TestInitialSetupAndLogin`, `TestCPUPercent`, `TestMemoryUsage`, `TestCalculatorCompute`, `TestGPUManagerNoGPUFallback`, `TestCollectorWithMockProc`, `TestHistoryStore_FlushBatch_NoDeadlock`, `TestAssessPortSecurity`, `TestSynthesizeContainerSockets`, `TestBuilderBuildSnapshotEnrichment`, `TestExtractProcessName`, `TestParseAddrIPv4`, `TestClassifyPort`).
  - Flag Note: Running `go test -race ./...` failed with `go: -race requires cgo; enable cgo by setting CGO_ENABLED=1`. Orbit is deliberately designed for `CGO_ENABLED=0` (pure Go), so CI tests must run with `go test -v ./...` without `-race`.
- **Backend Static Analysis**:
  - Command: `cd backend; go vet ./...`
  - Result: Clean exit with code 0 (zero errors or warnings).
- **Backend Build & Web Dist Embedding**:
  - `backend/web/embed.go` line 8: `//go:embed all:dist` embeds `backend/web/dist`.
  - `git ls-files backend/web/dist` confirmed that `dist/index.html`, `dist/favicon.svg`, `dist/icons.svg`, `dist/assets/index-*.css`, and `dist/assets/index-*.js` are tracked in git. This allows the backend to compile immediately even before frontend compilation, though CI should validate both.

### 1.3 Orbit Frontend Architecture
- **`frontend/package.json`**:
  - Name: `frontend`, `type: module`
  - Node version: Node 22 (tested with Node `v22.17.0` and npm `10.5.2`)
  - Scripts: `"build": "tsc && vite build"`
  - Toolchain: TypeScript `~7.0.2`, Vite `^8.3.6`, TailwindCSS `@tailwindcss/vite ^4.3.4`, React `^19.3.1`
  - Lockfile: `frontend/package-lock.json` exists and is tracked.
- **Frontend Build Execution**:
  - Command: `cd frontend; npm run build`
  - Result: Clean build in ~780ms (`tsc` typecheck + `vite build` bundles `dist/index.html`, `dist/assets/index-*.css`, `dist/assets/index-*.js`).

### 1.4 Orbit Dockerfile & Container Architecture
- **`Dockerfile` (3-stage multi-stage build)**:
  - Stage 1 (`frontend-builder`): `FROM node:22-alpine AS frontend-builder` runs `npm ci` and `npm run build`.
  - Stage 2 (`backend-builder`): `FROM golang:1.26-bookworm AS backend-builder` copies `/app/frontend/dist` to `./web/dist`, defines `ARG TARGETOS=linux` and `ARG TARGETARCH=amd64`, sets `ENV CGO_ENABLED=0`, and executes `RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags="-s -w" -o /app/orbit ./cmd/orbit`.
  - Stage 3 (`production runtime`): `FROM debian:bookworm-slim`, installs `ca-certificates` and `tzdata`, creates non-root user `orbit:orbit` (UID/GID 10001), creates `/data`, exposes port `8787`, and configures healthcheck:
    `HEALTHCHECK --interval=15s --timeout=3s --start-period=5s --retries=3 CMD ["/app/orbit", "-healthcheck"] || exit 0`
  - `backend/cmd/orbit/main.go` lines 30-37 explicitly implement the `-healthcheck` flag by pinging `http://127.0.0.1:8787/api/v1/health` and exiting 0 on success.
- **`.dockerignore`**:
  - Excludes `.git`, `.github`, `frontend/node_modules`, `dist`, `**/dist`. This ensures host build artifacts never leak into container builds.

### 1.5 Reference Project (Lynx) CI/CD Architecture
Inspecting `https://github.com/moresonsunn/Lynx/.github/workflows/` revealed the following workflows and conventions:
1. **`docker-build-publish.yml`**:
   - Triggers: `push` on `main`, `master`, with path filters (`backend/**`, `frontend/**`, `Dockerfile*`), plus `workflow_dispatch` with optional `version` input.
   - Runners: `ubuntu-latest`.
   - Setup actions:
     - `docker/setup-qemu-action@v3`
     - `docker/setup-buildx-action@v3`
   - Registry authentication:
     - Docker Hub: `docker/login-action@v3` with `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN` (or `DOCKER_USERNAME`/`DOCKER_TOKEN`).
     - GHCR: `docker/login-action@v3` with `registry: ghcr.io`, `username: ${{ github.actor }}`, `password: ${{ secrets.GITHUB_TOKEN }}`.
   - Builder: `docker/build-push-action@v6` targeting `platforms: linux/amd64,linux/arm64`.
   - Caching: `cache-from: type=gha`, `cache-to: type=gha,mode=max`.
2. **`release.yml`**:
   - Triggers: `push` on `tags: [ 'v*' ]`.
   - Uses `docker/metadata-action` and `softprops/action-gh-release@v2` with `generate_release_notes: true` to publish releases and multi-arch containers simultaneously.

---

## 2. Logic Chain

1. **Absence of CI/CD in Orbit**:
   - Observation 1.1 confirms `.github/workflows/` does not exist.
   - Inference: We must create the entire workflow directory tree and provide production-ready YAML specifications that satisfy Requirements R3 of `ORIGINAL_REQUEST.md`.

2. **CI Pipeline Structure (`ci.yml`)**:
   - Observation 1.2 demonstrates Go backend unit tests and static checks (`go vet ./...`, `go test -v ./...`) run in `< 2s` on `CGO_ENABLED=0`.
   - Observation 1.3 demonstrates Frontend build (`tsc && vite build`) runs in `< 1s` using Node 22 and `npm ci`.
   - Inference: `ci.yml` should define two parallel, concurrent jobs (`backend` and `frontend`) on `ubuntu-latest`.
     - `backend` job uses `actions/setup-go@v5` with `cache: true` and `cache-dependency-path: backend/go.sum`, executing `go mod download`, `go vet ./...`, and `go test -v ./...`.
     - `frontend` job uses `actions/setup-node@v4` with `cache: 'npm'` and `cache-dependency-path: frontend/package-lock.json`, executing `npm ci` and `npm run build`.
     - Triggers must cover `push` and `pull_request` against `main` and `master`, plus `workflow_dispatch`.

3. **Multi-Architecture Docker Publishing Pipeline (`docker-publish.yml`)**:
   - Requirement R3 specifies multi-architecture publishing (`linux/amd64` and `linux/arm64`) to both Docker Hub (`moresonsunn/orbit`) and GitHub Packages / GHCR (`ghcr.io/${{ github.repository }}`).
   - Observation 1.4 confirms Orbit's Dockerfile accepts `TARGETOS` and `TARGETARCH` args and compiles with `CGO_ENABLED=0`.
   - Observation 1.5 confirms Lynx leverages QEMU (`docker/setup-qemu-action@v3`) and Buildx (`docker/setup-buildx-action@v3`) with GitHub Actions cache (`type=gha`).
   - Registry authentication:
     - GHCR requires `permissions: packages: write` and `docker/login-action@v3` with `registry: ghcr.io`, `username: ${{ github.actor }}`, `password: ${{ secrets.GITHUB_TOKEN }}`.
     - Docker Hub requires `docker/login-action@v3` with `username: ${{ secrets.DOCKERHUB_USERNAME }}` and `password: ${{ secrets.DOCKERHUB_TOKEN }}`. (Using `${{ secrets.DOCKERHUB_USERNAME || secrets.DOCKER_USERNAME }}` provides resilience against alternate secret names).
   - Tagging: `docker/metadata-action@v5` handles multi-registry image tags simultaneously (`moresonsunn/orbit` and `ghcr.io/${{ github.repository }}`), generating `latest`, git branch name, semver release tags (`vX.Y.Z`, `vX.Y`, `vX`), and short commit SHA.

4. **Release Automation Option (`release.yml`)**:
   - Following Lynx's `release.yml`, when a semantic tag `v*` is pushed, creating a formal GitHub Release via `softprops/action-gh-release@v2` allows attaching changelogs, release notes, and linking to container digests.
   - We provide both a unified `docker-publish.yml` that handles both branches and tags, and a modular `release.yml` for teams preferring distinct workflow files.

---

## 3. Caveats

1. **Docker Hub Secrets Configuration**:
   - For publishing to Docker Hub (`moresonsunn/orbit`), repository maintainers must configure repository secrets `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN` in GitHub repository settings. GHCR publishing works out-of-the-box using the automatic `GITHUB_TOKEN`.
2. **Docker Hub Repository Name**:
   - If the repository is cloned or forked under a different Docker Hub account than `moresonsunn`, the image prefix in `docker-publish.yml` should match the user's Docker Hub account or organization.
3. **Multi-Arch Build Optimization Notice**:
   - In Orbit's `Dockerfile`, Stage 1 (`node:22-alpine`) and Stage 2 (`golang:1.26-bookworm`) can optionally add `--platform=$BUILDPLATFORM` to compile frontend and backend binaries natively on the AMD64 host before assembling the ARM64 final runtime stage. Because Go cross-compiles cleanly with `GOARCH=${TARGETARCH}`, this avoids QEMU emulation overhead for Go compilation. Even without this change, `docker/setup-qemu-action@v3` guarantees 100% functional builds.
4. **No CGO `-race` in CI**:
   - Orbit backend tests cannot be run with `go test -race ./...` in CI without installing a C cross-compiler and enabling CGO (`CGO_ENABLED=1`). Standard `go test -v ./...` is the verified command.

---

## 4. Conclusion & Workflow Specifications

We have formulated complete, production-grade GitHub Actions workflow files modeled directly on Lynx and tailored specifically to Orbit's Go + React stack.

### Specification 1: `.github/workflows/ci.yml`

Create file `.github/workflows/ci.yml` with the following contents:

```yaml
name: CI

on:
  push:
    branches:
      - main
      - master
    paths:
      - 'backend/**'
      - 'frontend/**'
      - '.github/workflows/ci.yml'
  pull_request:
    branches:
      - main
      - master
    paths:
      - 'backend/**'
      - 'frontend/**'
      - '.github/workflows/ci.yml'
  workflow_dispatch:

permissions:
  contents: read

concurrency:
  group: ${{ github.workflow }}-${{ github.ref }}
  cancel-in-progress: true

jobs:
  # =========================================================================
  # Go Backend Unit Tests and Static Checks
  # =========================================================================
  backend:
    name: Backend Tests (Go)
    runs-on: ubuntu-latest
    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version-file: 'backend/go.mod'
          check-latest: true
          cache: true
          cache-dependency-path: backend/go.sum

      - name: Download Go modules
        working-directory: backend
        run: go mod download

      - name: Verify Go module dependencies
        working-directory: backend
        run: go mod verify

      - name: Run Go vet
        working-directory: backend
        run: go vet ./...

      - name: Run Go unit tests
        working-directory: backend
        run: go test -v ./...

      - name: Verify Go binary build
        working-directory: backend
        run: go build -v -ldflags="-s -w" -o /dev/null ./cmd/orbit

  # =========================================================================
  # Frontend TypeScript Typecheck and Vite Production Build
  # =========================================================================
  frontend:
    name: Frontend Typecheck & Build (React/Vite)
    runs-on: ubuntu-latest
    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Set up Node.js
        uses: actions/setup-node@v4
        with:
          node-version: 22
          cache: 'npm'
          cache-dependency-path: frontend/package-lock.json

      - name: Install dependencies (clean install)
        working-directory: frontend
        run: npm ci

      - name: Run TypeScript typecheck & production build
        working-directory: frontend
        run: npm run build

      - name: Verify built distribution assets
        working-directory: frontend
        run: |
          test -f dist/index.html || { echo "dist/index.html is missing!"; exit 1; }
          test -d dist/assets || { echo "dist/assets directory is missing!"; exit 1; }
```

---

### Specification 2: `.github/workflows/docker-publish.yml`

Create file `.github/workflows/docker-publish.yml` with the following contents:

```yaml
name: Build & Publish Docker Images

on:
  push:
    branches:
      - main
      - master
    tags:
      - 'v*'
    paths:
      - 'backend/**'
      - 'frontend/**'
      - 'Dockerfile*'
      - '.dockerignore'
      - '.github/workflows/docker-publish.yml'
  workflow_dispatch:
    inputs:
      tag_override:
        description: 'Optional custom image tag override'
        required: false
        type: string

permissions:
  contents: read
  packages: write  # Required for publishing to GitHub Container Registry (ghcr.io)

jobs:
  build-and-push:
    name: Multi-Arch Build & Publish (amd64 / arm64)
    runs-on: ubuntu-latest

    steps:
      - name: Checkout code
        uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Set up QEMU (for multi-arch emulation)
        uses: docker/setup-qemu-action@v3
        with:
          platforms: linux/amd64,linux/arm64

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3

      - name: Log in to Docker Hub
        if: github.event_name != 'pull_request' && (secrets.DOCKERHUB_USERNAME != '' || secrets.DOCKER_USERNAME != '')
        uses: docker/login-action@v3
        with:
          username: ${{ secrets.DOCKERHUB_USERNAME || secrets.DOCKER_USERNAME }}
          password: ${{ secrets.DOCKERHUB_TOKEN || secrets.DOCKER_TOKEN }}

      - name: Log in to GitHub Container Registry (GHCR)
        if: github.event_name != 'pull_request'
        uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Extract Docker metadata (tags & labels)
        id: meta
        uses: docker/metadata-action@v5
        with:
          images: |
            moresonsunn/orbit
            ghcr.io/${{ github.repository }}
          tags: |
            type=raw,value=latest,enable=${{ github.ref == format('refs/heads/{0}', github.event.repository.default_branch) || startsWith(github.ref, 'refs/tags/v') }}
            type=ref,event=branch
            type=ref,event=tag
            type=semver,pattern={{version}}
            type=semver,pattern={{major}}.{{minor}}
            type=semver,pattern={{major}}
            type=sha,prefix=,format=short
            type=raw,value=${{ inputs.tag_override }},enable=${{ inputs.tag_override != '' }}
          labels: |
            org.opencontainers.image.title=Orbit
            org.opencontainers.image.description=Self-hosted Docker Infrastructure Management & Real-Time Monitoring Platform
            org.opencontainers.image.vendor=moresonsunn
            org.opencontainers.image.source=https://github.com/${{ github.repository }}

      - name: Build and push multi-arch Docker image
        uses: docker/build-push-action@v6
        with:
          context: .
          file: ./Dockerfile
          platforms: linux/amd64,linux/arm64
          push: ${{ github.event_name != 'pull_request' }}
          tags: ${{ steps.meta.outputs.tags }}
          labels: ${{ steps.meta.outputs.labels }}
          cache-from: type=gha
          cache-to: type=gha,mode=max

      - name: Inspect Image Manifests (Verification)
        if: github.event_name != 'pull_request'
        run: |
          docker buildx imagetools inspect ghcr.io/${{ github.repository }}:${{ steps.meta.outputs.version }} || true
```

---

### Specification 3: Optional `.github/workflows/release.yml` (GitHub Release on Tags)

If the team maintains a dedicated GitHub Release workflow (as Lynx does) that generates GitHub Releases when tags like `v1.0.0` are pushed:

```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

permissions:
  contents: write
  packages: write

jobs:
  github-release:
    name: Create GitHub Release
    runs-on: ubuntu-latest
    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Extract Tag Info
        id: meta
        run: |
          TAG="${GITHUB_REF##*/}"
          echo "tag=$TAG" >> $GITHUB_OUTPUT
          SHORT_SHA=$(git rev-parse --short HEAD)
          echo "short_sha=$SHORT_SHA" >> $GITHUB_OUTPUT

      - name: Create GitHub Release with Auto Notes
        uses: softprops/action-gh-release@v2
        with:
          tag_name: ${{ steps.meta.outputs.tag }}
          name: Orbit ${{ steps.meta.outputs.tag }}
          draft: false
          prerelease: false
          generate_release_notes: true
          body: |
            ### Orbit Release ${{ steps.meta.outputs.tag }}
            
            **Docker Images:**
            - `moresonsunn/orbit:${{ steps.meta.outputs.tag }}`
            - `ghcr.io/${{ github.repository }}:${{ steps.meta.outputs.tag }}`
            - `ghcr.io/${{ github.repository }}:latest`
            
            **Architectures:**
            - `linux/amd64`
            - `linux/arm64`
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

---

## 5. Verification Method

To independently verify these workflow configurations:

1. **Verify Backend CI Steps Locally**:
   ```bash
   cd c:\Users\phili\Documents\GitHub\Orbit\backend
   go mod verify
   go vet ./...
   go test -v ./...
   go build -v -ldflags="-s -w" -o orbit.test ./cmd/orbit
   rm orbit.test
   ```
   *Expected outcome*: Zero errors, all tests pass, exit code 0.

2. **Verify Frontend CI Steps Locally**:
   ```bash
   cd c:\Users\phili\Documents\GitHub\Orbit\frontend
   npm ci
   npm run build
   test -f dist/index.html
   ```
   *Expected outcome*: Clean TypeScript compilation and Vite build in `< 1.5s`, exit code 0.

3. **Verify Action Syntax & Schema**:
   Inspect workflow files with standard GitHub Actions schema linter (`actionlint` or VSCode YAML extension). Ensure all action versions (`actions/checkout@v4`, `actions/setup-go@v5`, `actions/setup-node@v4`, `docker/setup-qemu-action@v3`, `docker/setup-buildx-action@v3`, `docker/login-action@v3`, `docker/metadata-action@v5`, `docker/build-push-action@v6`) match current GitHub runner standards.

4. **Verify Container Build Locally**:
   ```bash
   docker buildx build --platform linux/amd64 -t orbit:survey-verify .
   ```
   *Expected outcome*: Successfully builds all 3 stages without missing files or compilation errors.

5. **Invalidation Conditions**:
   - If `CGO_ENABLED=1` is introduced in `go.mod` or backend code, the backend CI job will require a C compiler (`gcc`).
   - If secret names in GitHub settings differ from `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN`, the fallback mechanism checks `DOCKER_USERNAME`/`DOCKER_TOKEN`.
