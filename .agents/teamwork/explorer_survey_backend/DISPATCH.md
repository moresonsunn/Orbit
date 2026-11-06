## 2026-11-05T23:26:18Z
You are a Read-Only Backend Codebase Explorer for Orbit.

Your working directory is: c:\Users\phili\Documents\GitHub\Orbit\.agents\teamwork\explorer_survey_backend\
You MUST first view and read the original user request at:
c:\Users\phili\Documents\GitHub\Orbit\.agents\teamwork\ORIGINAL_REQUEST.md

Objective:
Investigate Orbit's Go backend thoroughly regarding Docker network handling, telemetry, socket observation, container network stats, and API routing.

Specifically investigate:
1. Current network APIs in Orbit (e.g. routes in cmd/ or internal/api/, controllers/handlers for networks, containers, flows).
2. Existing Docker client usage (how Orbit interacts with the Docker engine API to inspect networks, IPAM configs, subnets, gateways, driver options, connected containers, endpoints).
3. Existing network flow / telemetry / socket observation mechanisms (how network flows are currently populated, what data structures exist, whether socket connection tracking or container bandwidth rate metrics currently exist or where/how they should be added).
4. Container stats / telemetry collection loops, background collectors, WebSocket or REST endpoints.
5. Port security / exposed ports metadata: how container ports are mapped, inspected, and categorized.
6. Build and test commands for the backend (`go test ./...`, `go build ./...`).
7. Docker live environment setup: how `orbit-live` container is run, environment variables, mounts (e.g. docker.sock, /proc).

Scope & Rules:
- Read-only exploration! DO NOT modify any code or files outside your working directory.
- Document exact file paths, line numbers, function signatures, structs, and recommendations for backend changes to satisfy R2, R3, R4.
- Write your comprehensive findings to:
  `c:\Users\phili\Documents\GitHub\Orbit\.agents\teamwork\explorer_survey_backend\survey_backend.md`
- Write your final handoff to:
  `c:\Users\phili\Documents\GitHub\Orbit\.agents\teamwork\explorer_survey_backend\handoff.md`
- When complete, send a message to parent with summary and file paths.
