# Handoff Report: Go Backend Codebase Survey for Orbit

**Date:** 2026-11-06  
**From:** `explorer_survey_backend`  
**To:** `orchestrator` / `parent` (`4572cb91-25c6-48e4-bf11-6d7c5a635dd0`)  
**Type:** Hard Handoff (Investigation & Survey Complete)

---

## 1. Observation

1. **Network API Routing & Handlers:**
   - In `backend/internal/api/router.go`:
     - Lines 62–63: `authGroup.Get("/networks", server.HandleListNetworks)` and `authGroup.Get("/networks/{id}", server.HandleGetNetwork)`.
     - Line 68: `authGroup.Get("/netflow", server.HandleNetflow)`.
     - Line 35: `api.Get("/ws", hub.HandleWebSocket)`.
   - In `backend/internal/api/handlers.go`:
     - Line 375: `HandleListNetworks` executes `nets, err := s.inv.Networks(r.Context(), 10*time.Second)`. It serializes `[]network.Summary` directly to JSON.
     - Line 385: `HandleGetNetwork` calls `s.gw.InspectNetwork(r.Context(), id)`. It returns full `network.Inspect`.
     - When inspecting `fixtures_default` via `docker network inspect` and `GET /api/v1/networks/5edf60f8cc45`, `network.Inspect` returns `Containers` (5 connected containers with `IPv4Address`, `MacAddress`, `EndpointID`) and `Status.IPAM.Subnets` (`IPsInUse: 8`, `DynamicIPsAvailable: 65528`).
     - In `network.Summary` (from `GET /api/v1/networks`), `Containers` is empty/omitted by Docker daemon.

2. **Docker Client & Inventory:**
   - In `backend/internal/docker/gateway.go`:
     - Line 70: Docker client initialized with `client.WithHost(host)`, `client.WithAPIVersionNegotiation()`.
     - Line 268: `ListNetworks` runs `cli.NetworkList`.
     - Line 276: `InspectNetwork` runs `cli.NetworkInspect`.
   - In `backend/internal/docker/inventory.go`:
     - Line 208: `Inventory.Containers()` maintains all containers in memory.
     - Line 246: `Inventory.ByIP()` maps all container IPv4/IPv6 addresses to `docker.Container`.
     - In `docker.Container` (`backend/internal/docker/types.go:41,60`), each container has `Networks []ContainerNet` (`Name`, `NetworkID`, `IPAddress`, `Gateway`, `MacAddress`).

3. **Netflow & Socket Observation:**
   - In `backend/internal/netflow/builder.go`:
     - Line 30: `BuildSnapshot` builds `NetflowSnapshot` with `Topology` (`Nodes`, `Edges`), `Connections`, and `ProtocolStats`.
     - Line 87: It retrieves all latest stats via `b.streamer.GetAllLatest()`.
     - Lines 100–101: `TopologyNode` for containers receives `st.NetRxBps` and `st.NetTxBps`.
     - Lines 146–147: Calls `b.socketReader.ReadSockets()`.
   - In `backend/internal/netflow/socket_reader.go`:
     - Lines 55–61: Attempts to read `/proc/<pid>/net/tcp`, `tcp6`, `udp`, `udp6` using `filepath.Join(sr.cfg.HostProc, fmt.Sprintf("%d", c.Pid), "net")` falling back to `/proc/<pid>/net`.
   - In `backend/cmd/orbit/main.go`:
     - Lines 230–245: `runNetflowMonitoringLoop` calls `bld.BuildSnapshot(ctx)` and `hub.Broadcast("netflow", snapshot)` on a ticker (`sm.Get().Intervals.NetflowMs`).

4. **Container Stats & Streaming Calculations:**
   - In `backend/internal/docker/calc.go`:
     - Lines 201–239: `Calculator.Compute` calculates cumulative `NetRxBytes`, `NetTxBytes`, per-second rates `NetRxBps`, `NetTxBps`, and packet rates `NetRxPps`, `NetTxPps`.
     - Lines 48–63: `IfaceStats` tracks per-interface `RxBps`, `TxBps`, `RxPackets`, `TxPackets`, `RxErrors`, `RxDropped`.
   - In `backend/internal/docker/stats_stream.go`:
     - Lines 56–75: `StatsStreamer` caches `latestStats[id]` and ring buffer of last 60 samples in `recentHistory[id]`.

5. **Port Security & Classification:**
   - In `backend/internal/netflow/protocols.go`:
     - Line 3: `wellKnownPorts` maps 17 port numbers to strings.
     - Line 26: `ClassifyPort` returns port name or `"UNKNOWN"`.
   - In `backend/internal/docker/types.go`:
     - Lines 53–58: `Port` struct has `PrivatePort uint16`, `PublicPort uint16`, `Protocol string`, `HostIP string`.
     - Line 39: `ExposedPorts []string`.

6. **Backend Build and Tests:**
   - Running `go test ./...` in `backend/` returns exit code 0 (`PASS` across all packages).
   - Running `go build ./...` in `backend/` returns exit code 0 without warnings.

7. **Docker Live Container `orbit-live` Environment:**
   - `docker ps` shows container `orbit-live` (`9cb372a6977d`) running on port `8787`.
   - `docker inspect orbit-live` shows mounts:
     - `/var/run/docker.sock:/var/run/docker.sock:ro`
     - `orbit-data:/data`
   - Mounts for `/proc` or `/sys` are **not present**.
   - Executing `docker exec orbit-live ls -la /host/proc` returns `/host/proc not found`.
   - Inside `orbit-live`, `/proc` contains only the container's private PID namespace (PIDs 1, 1087, 1093). Host container PIDs (e.g. `c.Pid` of `test-orbit-traffic`) do not exist.
   - Calling `GET /api/v1/netflow` on `orbit-live` currently returns only port 8787 listening on host/orbit. Container socket connections are absent because procfs files cannot be reached.

---

## 2. Logic Chain

1. **R2 (Networks Page Overhaul):**
   - Observation 1 shows `GET /api/v1/networks` returns `network.Summary` which omits connected containers and IPAM subnet allocation statistics (`IPsInUse`, `DynamicIPsAvailable`).
   - However, Observation 1 and 2 show that `network.Inspect` already contains `Containers` and `Status.IPAM.Subnets`, AND `Inventory.Containers()` already holds all containers in memory with their network attachments (`c.Networks`).
   - Therefore, by creating an `EnrichedNetwork` response in `HandleListNetworks` (attaching connected containers and IPAM stats), the frontend can render IPAM subnet inspection, allocated IP ranges, and connected container status badges in a single, high-speed request without N+1 HTTP calls.

2. **R3 (Network Flows Lower Section) & R4 (Backend Telemetry):**
   - Observation 3 and 4 show that Orbit's `docker.ContainerStats` already accurately calculates `NetRxBps`, `NetTxBps`, `NetRxBytes`, `NetTxBytes`, and `IfaceStats`.
   - However, `NetflowSnapshot` currently only places these values in `TopologyNode` and does not provide a structured `ContainerTraffic` list for the lower section.
   - Observation 7 proves that when `/host/proc` is missing (as in `orbit-live`), `SocketReader` cannot read container sockets via procfs.
   - Observation 5 shows that `Inventory` already has container port configurations (`c.Ports`, `c.ExposedPorts`), container states, IPs, commands, and image names.
   - Therefore, by enhancing `netflow.Builder` and `netflow.SocketReader` with:
     a) A structured `ContainerTraffic []ContainerTrafficMetric` in `NetflowSnapshot`,
     b) A hybrid socket collection mechanism that synthesizes container listening endpoints from Docker inspection data when procfs is unavailable,
     c) Process attribution (`ProcessName`) and port security indicators (`SecurityLevel`, `SecurityBadge`, `SecurityDesc`),
   Orbit will deliver a comprehensive, live traffic breakdown and active socket connection table across all environments.

---

## 3. Caveats

1. **Host Procfs Mount in Docker Desktop:**
   - Under Docker Desktop (Windows/WSL2), `/proc` is isolated to the WSL2 virtual machine. Deployments with only `/var/run/docker.sock` will rely on the synthesized Docker socket reader fallback. On Linux systems using `deploy/docker-compose.yml` (`- /proc:/host/proc:ro`), both procfs connection tracking and Docker port synthesis work harmoniously.
2. **Deep Packet Inspection vs Port-Based Classification:**
   - In accordance with Orbit design principles, protocol detection is port-based and metadata-driven, not deep packet inspection (eBPF is optional). The security indicators reflect exposure risk and protocol encryption status (e.g. HTTP vs HTTPS, open Redis/MongoDB on 0.0.0.0).
3. **Frontend Compilation Issue:**
   - During verification tooling checks, `frontend/src/pages/Dashboard.tsx` had a pre-existing JSX formatting issue (lines 422–424). This is documented for the frontend team to address.

---

## 4. Conclusion

The Go backend architecture is robust, clean, and well-positioned for the R2, R3, and R4 enhancements:
1. All underlying Docker engine APIs and streaming statistics pipelines (`StatsStreamer`, `Inventory`, `Gateway`) are operational.
2. The exact modifications required have been detailed in `survey_backend.md`:
   - `backend/internal/api/handlers.go`: Enrich `HandleListNetworks` with connected containers and IPAM status.
   - `backend/internal/netflow/types.go`: Add `ContainerTraffic` to `NetflowSnapshot`, add `ProcessName` and `SecurityLevel`/`SecurityBadge`/`SecurityDesc` to `SocketConnection`.
   - `backend/internal/netflow/builder.go`: Populate `ContainerTraffic` from `streamer.GetAllLatest()` and `inv.Containers()`.
   - `backend/internal/netflow/socket_reader.go`: Implement hybrid socket collection (procfs + Docker port mapping synthesis fallback) so sockets are always populated.
   - `backend/internal/netflow/protocols.go`: Expand protocol definitions and add `AssessPortSecurity()`.

---

## 5. Verification Method

To independently verify these findings:

1. **Verify Backend Tests:**
   ```powershell
   cd c:\Users\phili\Documents\GitHub\Orbit\backend
   go test ./...
   ```
   *Expected:* All packages pass cleanly.

2. **Verify Backend Build:**
   ```powershell
   cd c:\Users\phili\Documents\GitHub\Orbit\backend
   go build ./...
   ```
   *Expected:* Compiles without errors.

3. **Verify Network Inspect via API:**
   ```powershell
   $s = New-Object Microsoft.PowerShell.Commands.WebRequestSession
   $login = Invoke-RestMethod -Uri 'http://127.0.0.1:8787/api/v1/auth/login' -Method Post -Body (@{username='admin'; password='OrbitAdminSecret2026!'} | ConvertTo-Json) -ContentType 'application/json' -WebSession $s
   $headers = @{'X-CSRF-Token' = $login.csrfToken}
   Invoke-RestMethod -Uri 'http://127.0.0.1:8787/api/v1/networks/5edf60f8cc45e80442d3ee8cc586c119b01d076861c22d65a669e8df25f80684' -Headers $headers -WebSession $s
   ```
   *Expected:* Returns `Containers` and `Status.IPAM.Subnets`.

4. **Verify Live Container `orbit-live` Procfs State:**
   ```powershell
   docker exec orbit-live sh -c "ls -la /host/proc 2>/dev/null || echo '/host/proc not found'"
   ```
   *Expected:* Outputs `/host/proc not found`.
