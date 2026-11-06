# Project: Orbit Docker Socket Permission & Dashboard Warning Banner

## Architecture
Orbit is a lightweight Docker management and observability dashboard with a Go backend and React (TypeScript + Vite + Tailwind CSS) frontend.
The container interacts with the host Docker Engine via `/var/run/docker.sock`.
The backend provides `/api/v1/system` returning `dockerConnected` (boolean) and `dockerError` (string).
The frontend displays host metrics, container tables, and telemetry on `Dashboard.tsx`.

## Feature Inventory
| # | Feature | Description | Milestone | Source |
|---|---------|-------------|-----------|--------|
| 1 | Dockerfile Root Execution | Remove `USER orbit` so container runs as root (UID 0: GID 0) by default to access `/var/run/docker.sock` | M1 | R1, Survey 1 |
| 2 | Compose Socket & User Config | Ensure `Apps/Orbit/docker-compose.yml` and `docker-compose.prod.yml` specify `user: "0:0"` and clean socket mounts | M1 | R1, Survey 1 |
| 3 | Dashboard UI Disconnection Warning Banner | Display prominent warning banner when `dockerConnected` is false explaining socket/daemon issues | M1 | R2, Survey 2 |
| 4 | Dashboard AST Compliance | Ensure banner does not add any `<table>` tag, preserving `totalTables == 1` invariant in tests | M1 | R2, Survey 2 & 3 |
| 5 | Full Test Suite Verification | Verify `backend` Go tests, `frontend` build, and Python `tests/run_all_tests.py` | M2 | R3, Survey 3 |
| 6 | Git Commit and Push to Main | Commit changes and push to `origin main` to trigger GitHub Actions publishing | M2 | R4, Survey 3 |

## Milestones
| # | Name | Scope | Dependencies | Status |
|---|------|-------|-------------|--------|
| 1 | Implementation | Dockerfile, compose files, and Dashboard UI warning banner | Survey | DONE |
| 2 | Verification & Publishing | Run test suites, commit changes, push to origin main | M1 | IN_PROGRESS |

### Milestone 1 Key Outputs:
- `Dockerfile`: Removed `USER orbit`; container runs as root by default.
- `Apps/Orbit/docker-compose.yml`: Added `user: "0:0"`. Verified `/var/run/docker.sock` mount. Validated CasaOS package.
- `docker-compose.prod.yml`: Added `user: "0:0"`. Verified `/var/run/docker.sock` mount. Retained `group_add`.
- `frontend/src/pages/Dashboard.tsx`: Added prominent warning banner for `!sysInfo.dockerConnected` with `dockerError` display and retry handler. Preserved `totalTables == 1`.
- Gate Verdicts: Worker DONE, Reviewer 1 APPROVE, Reviewer 2 APPROVE, Challenger 1 APPROVE, Challenger 2 APPROVE, Auditor 1 CLEAN. Gate: PASS.

## Interface Contracts
### Backend System API (`/api/v1/system`) ↔ Frontend Dashboard (`Dashboard.tsx`)
- Method: `GET /api/v1/system`
- Response Payload Schema:
  ```json
  {
    "dockerConnected": boolean,
    "dockerError": string | null,
    "docker": { ... },
    "host": { ... },
    "inventoryCounts": { "running": number, "stopped": number, "unhealthy": number, "total": number }
  }
  ```
- Contract Rule: If `dockerConnected === false`, `Dashboard.tsx` MUST display a prominent warning banner explaining the disconnection status and permission/daemon issue, rather than silently showing 0 containers.
- DOM Rule: `Dashboard.tsx` MUST NOT add any `<table>` tags (must maintain exactly 1 `<table>` tag in the file).

## Code Layout
- `Dockerfile` - Container build definition
- `Apps/Orbit/docker-compose.yml` - CasaOS App Store compose template
- `docker-compose.prod.yml` - Production Docker Compose configuration
- `frontend/src/pages/Dashboard.tsx` - React dashboard page component
- `tests/run_all_tests.py` - Master Python test runner
- `backend/` - Go backend source code and unit tests
- `tests/e2e/` - E2E Go tests
