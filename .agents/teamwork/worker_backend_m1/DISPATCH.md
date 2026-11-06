## 2026-11-05T23:35:29Z
You are the Backend Implementation Worker for Milestone M1: Backend Telemetry & Enriched Endpoints.

Your working directory is: c:\Users\phili\Documents\GitHub\Orbit\.agents\teamwork\worker_backend_m1\

You MUST read:
1. c:\Users\phili\Documents\GitHub\Orbit\.agents\teamwork\ORIGINAL_REQUEST.md
2. c:\Users\phili\Documents\GitHub\Orbit\PROJECT.md
3. c:\Users\phili\Documents\GitHub\Orbit\.agents\teamwork\explorer_survey_backend\survey_backend.md
4. c:\Users\phili\Documents\GitHub\Orbit\.agents\teamwork\explorer_survey_backend\handoff.md

MANDATORY INTEGRITY WARNING:
DO NOT CHEAT. All implementations must be genuine. DO NOT hardcode test results, create dummy/facade implementations, or circumvent the intended task. A teamwork_auditor will independently verify your work. Integrity violations WILL be detected and your work WILL be rejected.

Exclusive Write Ownership:
- `backend/internal/api/handlers.go`
- `backend/internal/api/handlers_test.go`
- `backend/internal/netflow/types.go`
- `backend/internal/netflow/builder.go`
- `backend/internal/netflow/socket_reader.go`
- `backend/internal/netflow/protocols.go`
- `backend/internal/netflow/netflow_test.go` (and any new test files under `backend/internal/netflow/` or `backend/internal/api/`)

Implementation Requirements:
1. Networks API Enrichment (`backend/internal/api/handlers.go`):
   - In `HandleListNetworks` and `HandleGetNetwork`, deliver enriched network responses matching `PROJECT.md § Interface Contracts`:
     - Compute/populate IPAM subnet details: Subnet, Gateway, NetworkIP, BroadcastIP, UsableRange, TotalIPs, UsedIPs, FreeIPs, UtilizationPercent.
     - Include connected containers from `s.inv.Containers()` (and/or inspected network containers) with Name, EndpointID, MacAddress, IPv4Address, IPv6Address, State.
   - Ensure backward compatibility so existing Docker network fields remain accessible.
2. Netflow Data Types (`backend/internal/netflow/types.go`):
   - Add `ContainerTraffic []ContainerTrafficMetric` to `NetflowSnapshot`.
   - Add `Totals NetflowTotals` to `NetflowSnapshot` (totalRxBps, totalTxBps, activeConnections, securityAlertsCount).
   - In `SocketConnection`, add `ProcessName`, `SecurityLevel` ("critical" | "caution" | "safe" | "internal"), `SecurityBadge`, and `SecurityDesc`.
3. Netflow Builder (`backend/internal/netflow/builder.go`):
   - In `BuildSnapshot`, populate `ContainerTraffic` from `b.streamer.GetAllLatest()` and container metadata from `b.inv.Containers()`.
   - Aggregate totals into `Totals`.
4. Hybrid Socket Reader Fallback (`backend/internal/netflow/socket_reader.go`):
   - Keep procfs socket collection. When `/proc` returns 0 container sockets or is unavailable (such as inside container `orbit-live` where `/host/proc` is missing), synthesize container listening endpoints from Docker container data (`c.Ports`, `c.ExposedPorts`), correctly attributing container name, container ID, process/command name, and port metadata.
5. Port Security Classifier (`backend/internal/netflow/protocols.go`):
   - Implement `AssessPortSecurity(port uint16, hostIP string, proto string, detectedProto string) (level string, badge string, desc string)`.
   - Critical: Unencrypted databases/caches (e.g. Redis 6379, MongoDB 27017, Postgres 5432, MySQL 3306) bound to public `0.0.0.0` or `::`.
   - Caution: Unencrypted HTTP (80, 8080) or unclassified ports bound to public `0.0.0.0`.
   - Safe: Encrypted services (HTTPS 443, SSH 22) or services bound to localhost `127.0.0.1`.
   - Internal: Container-to-container private network bindings.
6. Automated Tests & Build Verification:
   - Add comprehensive unit tests in `backend/internal/netflow/` and `backend/internal/api/` covering enriched network calculation, container traffic extraction, hybrid socket fallback, and port security classification.
   - Run tests: `cd backend; go test ./...`
   - Run build: `cd backend; go build ./...`
   - Ensure all tests pass and build succeeds cleanly.
7. Deliverable:
   - Write `c:\Users\phili\Documents\GitHub\Orbit\.agents\teamwork\worker_backend_m1\handoff.md` with Observation, Logic Chain, Caveats, Conclusion, and Verification Method.
   - Send completion message to parent.
