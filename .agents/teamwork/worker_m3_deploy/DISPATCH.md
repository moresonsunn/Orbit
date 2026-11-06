## 2026-11-06T01:21:10Z
You are worker_m3_deploy, an implementation worker agent.
Your working directory is: c:\Users\phili\Documents\GitHub\Orbit\.agents\teamwork\worker_m3_deploy

MANDATORY FIRST STEP:
Read c:\Users\phili\Documents\GitHub\Orbit\.agents\teamwork\ORIGINAL_REQUEST.md before starting any work.

MANDATORY INTEGRITY WARNING:
DO NOT CHEAT. All implementations must be genuine. DO NOT hardcode test results, create dummy/facade implementations, or circumvent the intended task. A teamwork_auditor will independently verify your work. Integrity violations WILL be detected and your work WILL be rejected.

REFERENCE FILES:
- `PROJECT.md` at repository root
- Explorer survey and specifications: `c:\Users\phili\Documents\GitHub\Orbit\.agents\teamwork\explorer_survey_deploy\handoff.md`

WRITE OWNERSHIP:
You exclusively own:
- `docker-compose.yml` (at project root)
- `docker-compose.prod.yml` (at project root)
- `backend/cmd/orbit/main.go` (healthcheck port fix)
- `README.md` (at project root)
You MUST NOT edit any frontend source files or `.github/` workflow files.

TASKS FOR MILESTONE M3:
Implement production Docker deployment configurations, dynamic healthcheck port resolution, and modern production documentation using specifications in `c:\Users\phili\Documents\GitHub\Orbit\.agents\teamwork\explorer_survey_deploy\handoff.md`:
1. Create `docker-compose.yml` at repository root (for standalone/development deployment):
   - Service `orbit`, image `moresonsun/orbit:latest`, build context `.`, container_name `orbit`, `restart: unless-stopped`, ports `"${ORBIT_PORT:-8080}:8080"`, environment variables, volume mounts (`orbit_data:/data`, `/var/run/docker.sock:/var/run/docker.sock`, `/proc:/host/proc:ro`, `/sys:/host/sys:ro`, `/etc/os-release:/host/etc/os-release:ro`), `no-new-privileges: true`, `group_add: ["${DOCKER_GID:-999}"]`, healthcheck, json-file log rotation.
2. Create `docker-compose.prod.yml` at repository root (for server deployment):
   - Pre-built image `moresonsun/orbit:latest`, container_name `orbit`, `restart: unless-stopped`, ports `"8080:8080"`, environment (`ORBIT_LISTEN=:8080`, `ORBIT_DATA_DIR=/data`, `ORBIT_TRUST_PROXY=true`), socket mount `/var/run/docker.sock`, persistent data `/data`, `no-new-privileges: true`, `group_add: ["${DOCKER_GID:-999}"]`, healthcheck, json-file log rotation (`max-size: 20m`, `max-file: 5`).
3. Patch `backend/cmd/orbit/main.go` line 30:
   - Make `/app/orbit -healthcheck` dynamically extract the port from `os.Getenv("ORBIT_LISTEN")` (defaulting to `:8787`), ensuring healthcheck succeeds when running with `ORBIT_LISTEN=:8080`.
4. Rewrite `README.md` to production standard modeled after Lynx:
   - Centered title and description.
   - 5 centered status badges: CI (`https://github.com/moresonsunn/orbit/actions/workflows/ci.yml/badge.svg`), Docker pulls (`https://img.shields.io/docker/pulls/moresonsun/orbit`), Arch (`amd64 | arm64`), GitHub Release (`https://img.shields.io/github/v/release/moresonsunn/orbit`), License (MIT).
   - 10-feature breakdown (Containers, Networks, IPAM, Network Flow, GPU, Images, Volumes, Events, Logs, Auth).
   - Quick Start section: One-line `docker run` command and one-line `docker compose` command.
   - First-Time Setup instructions for retrieving the single-use setup token from `docker logs orbit`.
   - ASCII Architecture block diagram and technology table.
   - Comprehensive 22-variable configuration reference table (Server & Storage, Security, Telemetry, Permissions).
   - Development commands and documentation table.

VERIFICATION:
- Verify Go backend tests pass: `cd backend && go test ./...`
- Validate Compose files: `docker compose -f docker-compose.yml config` and `docker compose -f docker-compose.prod.yml config`
- Verify E2E test suites for deployment, healthcheck, and docs:
  `cd tests/e2e && go test -v -run "TestTier1_DeployUI_FeatureCoverage/(D04|D05|D06)" ./...`
Document commands and results in `c:\Users\phili\Documents\GitHub\Orbit\.agents\teamwork\worker_m3_deploy\handoff.md` and send a completion message to the orchestrator.
