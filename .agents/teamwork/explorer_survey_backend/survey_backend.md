# Backend Codebase Survey: Docker Networking, Telemetry, Socket Observation & API Routing

**Survey Date:** 2026-11-06  
**Target Project:** Orbit (`github.com/phili/orbit`)  
**Scope:** Go Backend (`backend/cmd/orbit`, `backend/internal/...`), Docker Live Environment (`orbit-live`), and verification tooling.

---

## 1. Executive Summary

This survey provides a comprehensive audit of Orbit's Go backend with respect to Docker networking, IPAM inspection, network flow telemetry, socket observation, container rate metrics, and port security.

Orbit has a well-structured, high-performance Go backend built on:
- **Router:** `go-chi/chi/v5`
- **Docker Client:** `github.com/moby/moby/client` (v28.0.1)
- **Realtime Streaming:** `github.com/coder/websocket`
- **Model & Architecture:** Event-driven Docker inventory with in-memory caching, per-container background stats streaming via Docker Engine streaming API, and ticker-driven netflow and host monitoring loops.

### Key Discoveries for R2, R3, and R4:
1. **Network APIs (R2):**  
   `GET /api/v1/networks` currently returns raw `[]network.Summary` from Docker daemon, which lacks connected container resources and IPAM pool statistics. However, `GET /api/v1/networks/{id}` calls `client.NetworkInspect` and returns full `network.Inspect` (including `Containers` map with IPs/MACs and `Status.IPAM.Subnets` with `IPsInUse` and `DynamicIPsAvailable`). Furthermore, `Inventory.Containers()` already maintains all containers in memory with their network attachments. Enriching `GET /api/v1/networks` with connected containers and IPAM statistics enables the UniFi-style Networks page without requiring N+1 HTTP round trips.

2. **Network Flow & Bandwidth Meters (R3, R4):**  
   Orbit already tracks per-second network transfer rates (`NetRxBps`, `NetTxBps`, `NetRxPps`, `NetTxPps`) and per-interface metrics (`IfaceStats`) inside `docker.ContainerStats`. Currently, these rates are only surfaced inside `TopologyNode` in `NetflowSnapshot`. Adding a structured `ContainerTraffic` list to `NetflowSnapshot` will directly feed the live bandwidth meters and traffic breakdown required by R3.

3. **Socket Observation & Procfs Environment Caveat (R3, R4):**  
   Orbit's `SocketReader` attempts to read `/proc/<pid>/net/tcp` via `filepath.Join(cfg.HostProc, pid, "net")` or `/proc/<pid>/net`. In the current live container `orbit-live`, `/host/proc` is **not mounted** (only `/var/run/docker.sock` and `/data` volume are bound). Because `orbit-live` runs in an isolated PID namespace, host PIDs are not present in the container's own `/proc`. As a result, `SocketReader` only sees the host/orbit socket (`8787`).  
   **Recommendation:** Orbit must implement a hybrid socket collector:
   - Procfs socket reading when `/host/proc` is available (e.g. Linux Compose / CasaOS).
   - Container port-mapping synthesis fallback from Docker inspection (`c.Ports` and `c.ExposedPorts` from `Inventory`) so that active listening sockets and port security indicators are always fully populated in all containerized environments (including Docker Desktop on Windows/macOS).
   - In addition, add `ProcessName` (from `c.Command`) and `PortSecurity` indicators (risk level, exposure scope, protocol safety) to `SocketConnection`.

---

## 2. Current Network & Resource APIs

### 2.1 Route Registry
**File:** `backend/internal/api/router.go` (lines 44–106)

All authenticated API routes are registered under `/api/v1` with authentication and CSRF protection middlewares:

```go
// router.go: lines 53–70
// Containers Read
authGroup.Get("/containers", server.HandleListContainers)
authGroup.Get("/containers/{id}", server.HandleGetContainer)
authGroup.Get("/containers/{id}/stats", server.HandleContainerStats)
authGroup.Get("/containers/{id}/stats/history", server.HandleContainerStatsHistory)
authGroup.Get("/containers/{id}/logs", server.HandleContainerLogs)
authGroup.Get("/containers/{id}/top", server.HandleContainerTop)

// Docker Resources Read
authGroup.Get("/networks", server.HandleListNetworks)
authGroup.Get("/networks/{id}", server.HandleGetNetwork)
authGroup.Get("/images", server.HandleListImages)
authGroup.Get("/volumes", server.HandleListVolumes)

// Network Flow & GPU
authGroup.Get("/netflow", server.HandleNetflow)
authGroup.Get("/gpu", server.HandleGPU)

// Admin-Only Mutating Operations
admin.Delete("/networks/{id}", server.HandleNetworkRemove)
```

WebSocket endpoint:
- `GET /api/v1/ws` -> `hub.HandleWebSocket` (`backend/internal/realtime/hub.go:152`)
- Supports subprotocol `"orbit.v1"`.
- Topics streamed: `"containers"`, `"container:{id}"`, `"netflow"`, `"events"`, `"alerts"`, `"host"`, `"gpu"`.

### 2.2 Network Handlers
**File:** `backend/internal/api/handlers.go` (lines 374–392)

```go
// Line 374: HandleListNetworks
func (s *Server) HandleListNetworks(w http.ResponseWriter, r *http.Request) {
    nets, err := s.inv.Networks(r.Context(), 10*time.Second)
    if err != nil {
        writeError(w, http.StatusInternalServerError, "Netzwerke konnten nicht geladen werden: "+err.Error())
        return
    }
    writeJSON(w, http.StatusOK, nets)
}

// Line 384: HandleGetNetwork
func (s *Server) HandleGetNetwork(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    netInspect, err := s.gw.InspectNetwork(r.Context(), id)
    if err != nil {
        writeError(w, http.StatusNotFound, "Netzwerk nicht gefunden: "+err.Error())
        return
    }
    writeJSON(w, http.StatusOK, netInspect)
}
```

### 2.3 Container & Stats Handlers
**File:** `backend/internal/api/handlers.go` (lines 259–333)

- `HandleListContainers`: Returns `[]EnrichedContainer`. If live stats exist in `s.streamer.GetLatest(c.ID)`, they are attached directly into `c.Stats`.
- `HandleGetContainer`: Returns `EnrichedContainer` and full inspect data (with environment variables masked for non-admin users).
- `HandleContainerStats`: Returns the latest live `docker.ContainerStats`.
- `HandleContainerStatsHistory`: Returns up to 60 recent metric samples (ring buffer from `StatsStreamer.recentHistory`).

### 2.4 Netflow Handler
**File:** `backend/internal/api/handlers.go` (lines 441–444)

```go
func (s *Server) HandleNetflow(w http.ResponseWriter, r *http.Request) {
    snapshot := s.netflowBld.BuildSnapshot(r.Context())
    writeJSON(w, http.StatusOK, snapshot)
}
```

---

## 3. Docker Client Usage & Network Inspection

### 3.1 Docker Gateway
**File:** `backend/internal/docker/gateway.go`

Orbit connects to the Docker daemon via `client.New` with API version negotiation and user-agent `"orbit"`:
- `Gateway.ListNetworks(ctx)` calls `g.cli.NetworkList(ctx, client.NetworkListOptions{})`.
- `Gateway.InspectNetwork(ctx, id)` calls `g.cli.NetworkInspect(ctx, id, client.NetworkInspectOptions{})`.
- `Gateway.RemoveNetwork(ctx, id)` calls `g.cli.NetworkRemove(ctx, id, client.NetworkRemoveOptions{})`.
- `Gateway.StatsStream(ctx, id)` calls `g.cli.ContainerStats(ctx, id, client.ContainerStatsOptions{Stream: true})`.
- `Gateway.Top(ctx, id)` calls `g.cli.ContainerTop(ctx, id, client.ContainerTopOptions{})`.

### 3.2 Moby Network Data Types
Orbit uses `github.com/moby/moby/api/types/network`:

1. **`network.Summary`** (returned by `NetworkList` / `/api/v1/networks`):
   - `Name`: string
   - `Id`: string
   - `Created`: timestamp
   - `Scope`: string (`local`, `swarm`, `global`)
   - `Driver`: string (`bridge`, `host`, `null`, `macvlan`, `overlay`)
   - `EnableIPv4`, `EnableIPv6`: bool
   - `IPAM`: `network.IPAM` (`Driver`, `Options`, `Config []network.IPAMConfig`)
     - `IPAMConfig`: `Subnet`, `Gateway`, `IPRange`, `AuxAddress`
   - `Internal`: bool (isolated from external traffic)
   - `Attachable`: bool
   - `Ingress`: bool
   - `Options`: map[string]string (e.g. `com.docker.network.bridge.default_bridge`, `com.docker.network.bridge.enable_icc`, `com.docker.network.bridge.enable_ip_masquerade`, `com.docker.network.driver.mtu`)
   - `Labels`: map[string]string (e.g. compose project and network labels)
   - *Note:* In `network.Summary`, Docker does **not** populate `Containers`.

2. **`network.Inspect`** (returned by `NetworkInspect` / `/api/v1/networks/{id}`):
   - Contains all `network.Summary` fields.
   - **`Containers`**: `map[string]network.EndpointResource`
     ```go
     type EndpointResource struct {
         Name        string // Container name
         EndpointID  string
         MacAddress  string
         IPv4Address string // CIDR format, e.g. "172.18.0.3/16"
         IPv6Address string
     }
     ```
   - **`Status`**: `network.Status`
     ```go
     type IPAMStatus struct {
         Subnets map[string]SubnetStatus
     }
     type SubnetStatus struct {
         IPsInUse            int // e.g. 8
         DynamicIPsAvailable int // e.g. 65528
     }
     ```

### 3.3 Resource In-Memory Caching in `docker.Inventory`
**File:** `backend/internal/docker/inventory.go` (lines 357–400)

- `Inventory.Networks(ctx, maxAge)`: Serves cached `[]network.Summary`, reloading via `gw.ListNetworks(ctx)` if older than `maxAge` (10s in handler, 30s in netflow).
- `Inventory.Containers()`: Thread-safe in-memory map of all containers refreshed on Docker events and background sync (every 60s).
- `Inventory.ByIP()`: Returns `map[string]Container` indexed by both IPv4 and IPv6 addresses.
- Each `Container` has `Networks []ContainerNet` (`Name`, `NetworkID`, `EndpointID`, `IPAddress`, `PrefixLen`, `IPv6`, `Gateway`, `MacAddress`, `Aliases`).

---

## 4. Network Flow, Telemetry & Socket Observation

### 4.1 Data Models
**File:** `backend/internal/netflow/types.go`

```go
type NetflowSnapshot struct {
    Timestamp      time.Time          `json:"timestamp"`
    EBPFAvailable  bool               `json:"ebpfAvailable"`
    EBPFStatus     string             `json:"ebpfStatus"`
    Topology       Topology           `json:"topology"`
    Connections    []SocketConnection `json:"connections"`
    ProtocolStats  []ProtocolSummary  `json:"protocolStats"`
}

type TopologyNode struct {
    ID        string                 `json:"id"`
    Type      NodeType               `json:"type"`   // host, network, container, gateway, external
    Label     string                 `json:"label"`
    Subnet    string                 `json:"subnet,omitempty"`
    IP        string                 `json:"ip,omitempty"`
    Status    string                 `json:"status,omitempty"` // running, exited, etc.
    RxBps     model.Gauge            `json:"rxBps"`
    TxBps     model.Gauge            `json:"txBps"`
    Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type TopologyEdge struct {
    ID       string      `json:"id"`
    Source   string      `json:"source"`
    Target   string      `json:"target"`
    Type     EdgeType    `json:"type"`   // member, traffic, gateway
    Active   bool        `json:"active"`
    RxBps    model.Gauge `json:"rxBps"`
    TxBps    model.Gauge `json:"txBps"`
    Protocol string      `json:"protocol,omitempty"`
}

type SocketConnection struct {
    Protocol       string `json:"protocol"` // tcp, tcp6, udp, udp6
    LocalIP        string `json:"localIp"`
    LocalPort      uint16 `json:"localPort"`
    RemoteIP       string `json:"remoteIp"`
    RemotePort     uint16 `json:"remotePort"`
    State          string `json:"state"` // LISTEN, ESTABLISHED, TIME_WAIT, etc.
    ContainerID    string `json:"containerId,omitempty"`
    ContainerName  string `json:"containerName,omitempty"`
    AppProtocol    string `json:"appProtocol,omitempty"`
    Classification string `json:"classification"` // "port_based" or "ebpf_observed"
    DetectionMode  string `json:"detectionMode"`  // "socket_snapshot" or "ebpf"
}

type ProtocolSummary struct {
    Protocol    string `json:"protocol"`
    Configured  int    `json:"configuredPortsCount"`
    Observed    int    `json:"observedConnectionsCount"`
    ActiveFlows int    `json:"activeFlows"`
}
```

### 4.2 Snapshot Builder
**File:** `backend/internal/netflow/builder.go`

`Builder.BuildSnapshot(ctx)` executes the following pipeline:
1. Creates Host node (`"host"`).
2. Creates Network nodes (`"net-<id>"`) from `inv.Networks()`, and connects `host` -> `net-<id>` with `EdgeGateway`.
3. Gets container stats from `streamer.GetAllLatest()`.
4. Creates Container nodes (`"container-<shortId>"`) with `RxBps` and `TxBps` gauges from `ContainerStats.NetRxBps` and `NetTxBps`.
5. Connects `net-<id>` -> `container-<shortId>` with `EdgeMember`. Edge is marked `active: true` if `RxBps > 100` or `TxBps > 100`.
6. Invokes `socketReader.ReadSockets()` to populate socket table.
7. Computes `ProtocolStats`: counts configured container ports vs observed socket connections and active flows (`ESTABLISHED` / `ACTIVE`).

### 4.3 Socket Reader Implementation & Limitations
**File:** `backend/internal/netflow/socket_reader.go`

- Reads `/proc/<pid>/net/tcp`, `tcp6`, `udp`, `udp6`.
- For each container:
  ```go
  pidNetDir := filepath.Join(sr.cfg.HostProc, fmt.Sprintf("%d", c.Pid), "net")
  if fi, err := os.Stat(pidNetDir); err != nil || !fi.IsDir() {
      pidNetDir = filepath.Join("/proc", fmt.Sprintf("%d", c.Pid), "net")
      if fi, err := os.Stat(pidNetDir); err != nil || !fi.IsDir() {
          continue
      }
  }
  ```
- Parses hex addresses via `parseAddr`:
  - IPv4: 8-digit hex, little-endian byte order (`net.IPv4(b[3], b[2], b[1], b[0])`).
  - IPv6: 32-digit hex, 4-byte word swapped for Linux little-endian representation.
- Translates TCP state codes: `01` -> `ESTABLISHED`, `0A` -> `LISTEN`, `06` -> `TIME_WAIT`, etc.
- Maps socket IP to container name via `byIP := inv.ByIP()`.
- Maps port to application protocol via `ClassifyPort()`.

**Critical Limitation Identified:**  
In container environments where host `/proc` is not bound to `/host/proc`, `sr.cfg.HostProc` does not exist, and `/proc/<pid>/net` fails because container PID namespaces isolate PIDs. Therefore, `ReadSockets()` discovers only host/orbit sockets. Adding container port listening fallback guarantees active socket visibility in all deployment modes.

---

## 5. Container Stats & Telemetry Collection Loops

### 5.1 Stats Streamer
**File:** `backend/internal/docker/stats_stream.go`

- Reconciles every 5s (`ss.reconcile(ctx)`): starts a streaming worker for every running container (up to `MaxStatsStreams`, default 250).
- Subscribes to `inv.OnChange()` to immediately start workers for new containers or stop workers on container exit/removal.
- Calls `gw.StatsStream(ctx, id)` which opens an HTTP persistent connection to Docker's `/containers/{id}/stats?stream=true`.
- Decodes continuous `container.StatsResponse` JSON frames.

### 5.2 Rate Calculation
**File:** `backend/internal/docker/calc.go`

`Calculator.Compute(id, resp, cpuLimit)` derives rates across consecutive samples:
- `dt := s.Read.Sub(c.prev.Read).Seconds()` (minimum threshold `dt > 0.05s`).
- Rate formula: `rate(cur, prev, dt) = round2(float64(cur - prev) / dt)`.
- Calculates:
  - `NetRxBps`, `NetTxBps` (bytes/sec)
  - `NetRxPps`, `NetTxPps` (packets/sec)
  - `Interfaces []IfaceStats` per-interface rates (`eth0`, etc.)
  - `CPUPercent`, `CPUPercentOfLimit`, `CPUThrottledPct`
  - `MemUsage`, `MemLimit`, `MemPercent`, `MemCache`, `MemSwap`
  - `BlkReadBps`, `BlkWriteBps`

### 5.3 Realtime Streaming Pipeline
**File:** `backend/cmd/orbit/main.go` (lines 135–246)

1. `streamer.Subscribe(func(st docker.ContainerStats) { ... })`:
   - Broadcasts to WebSocket: `hub.Broadcast("containers", st)` and `hub.Broadcast(fmt.Sprintf("container:%s", st.ID), st)`.
   - Records into metric history store: `historyStore.RecordContainer(st)`.
   - Evaluates alerts: `alertEngine.CheckContainer(ctx, c, st)`.
2. `runNetflowMonitoringLoop`:
   - Fires every `Intervals.NetflowMs` (configurable in settings, default 3000ms–5000ms).
   - Generates `snapshot := bld.BuildSnapshot(ctx)`.
   - Broadcasts to WebSocket: `hub.Broadcast("netflow", snapshot)`.

---

## 6. Port Security & Exposed Ports Metadata

### 6.1 Docker Port Structures
**File:** `backend/internal/docker/types.go` (lines 53–59, 194–206)

```go
type Port struct {
    PrivatePort uint16 `json:"privatePort"`
    PublicPort  uint16 `json:"publicPort,omitempty"`
    Protocol    string `json:"protocol"` // "tcp" | "udp"
    HostIP      string `json:"hostIp,omitempty"`
}
```

In `FromInspect(in container.InspectResponse)`:
- Maps `in.NetworkSettings.Ports`:
  - `PrivatePort = p.Num()`
  - `PublicPort = parsePort(b.HostPort)`
  - `Protocol = string(p.Proto())`
  - `HostIP = b.HostIP.String()`
- Extracts `in.Config.ExposedPorts` into `c.ExposedPorts []string` (e.g. `["80/tcp", "8787/tcp"]`).

### 6.2 Port Classification
**File:** `backend/internal/netflow/protocols.go`

Current port table has 17 entries: `FTP` (21), `SSH` (22), `SMTP` (25), `DNS` (53), `DHCP` (67, 68), `HTTP` (80, 8080), `NTP` (123), `HTTPS` (443, 8443), `MySQL` (3306), `PostgreSQL` (5432), `Redis` (6379), `Orbit` (8787), `Portainer` (9000), `MongoDB` (27017).

### 6.3 Port Security Assessment Matrix
UniFi Network, Netdata, and security scanners evaluate ports according to:

| Category | Conditions | Risk Level | UniFi Badge | Security Action / Warning |
|---|---|---|---|---|
| **Public Critical** | Exposed to `0.0.0.0` or `::` AND port is an administrative or unauthenticated database/daemon (e.g. 2375 Docker daemon, 6379 Redis, 27017 MongoDB, 9200 Elasticsearch, 11211 Memcached) | `CRITICAL` | `Critical Risk` | High risk of unauthorized host or data access without authentication. |
| **Public Insecure** | Exposed to `0.0.0.0` or `::` AND cleartext protocol (e.g. 80 HTTP, 21 FTP, 23 Telnet, 25 SMTP) | `WARNING` | `Unencrypted` | Traffic transmitted in plaintext without TLS encryption. |
| **Public Secure** | Exposed to `0.0.0.0` or `::` AND encrypted protocol (e.g. 443/8443 HTTPS, 22 SSH) | `INFO` | `Public Secure` | Publicly reachable but protected with TLS/SSH. |
| **Localhost Only** | Bound strictly to `127.0.0.1` or `::1` | `SECURE` | `Local Only` | Not reachable from LAN or WAN; bound to loopback. |
| **Internal Only** | Not published to host (`PublicPort == 0`); accessible only within Docker network | `SECURE` | `Isolated Net` | Protected by Docker bridge/network isolation. |

---

## 7. Build and Test Tooling Assessment

### 7.1 Backend Test Execution
- Command: `cd backend; go test ./...`
- Status: **PASSED** (0 failures, cached/clean execution across all packages):
  - `internal/auth`: ok
  - `internal/docker`: ok
  - `internal/gpu`: ok
  - `internal/host`: ok
  - `internal/metrics`: ok
  - `internal/netflow`: ok
  - `internal/alerts`: ok
  - `internal/settings`: ok

### 7.2 Backend Build Execution
- Command: `cd backend; go build ./...`
- Status: **PASSED** (clean compilation with zero warnings or errors).

### 7.3 Frontend Compilation Status (Informational)
- Command: `cd frontend; npm run build`
- Observation: Failed due to a pre-existing JSX formatting issue in `frontend/src/pages/Dashboard.tsx` (lines 422–424: an object comment `{/* ... */}` placed directly inside JSX parentheses `{prefs.showCharts && ( ... )}`).  
  *Note:* As backend explorers, we did not touch frontend files, but this observation is highlighted for the frontend explorer/developer and orchestrator.

---

## 8. Docker Live Environment Setup (`orbit-live`)

Inspection of container `orbit-live` (`9cb372a6977d`):

| Parameter | Current Value in `orbit-live` | Notes & Impact |
|---|---|---|
| **Image** | `orbit:test` | Built from repo Dockerfile |
| **Status** | `Up (healthy)` | Running on port `8787` |
| **Network Mode** | `bridge` | IP `172.17.0.2`, Gateway `172.17.0.1` |
| **Port Bindings** | `0.0.0.0:8787 -> 8787/tcp` | Reached at `http://127.0.0.1:8787` |
| **Mounts** | `/var/run/docker.sock` (ro)<br>`orbit-data` -> `/data` (rw) | Docker Engine API is fully accessible.<br>Host filesystem (`/proc`, `/sys`) is **not** mounted. |
| **Environment** | `ORBIT_ENABLE_ACTIONS=true`<br>`ORBIT_ENABLE_DESTRUCTIVE=true`<br>`ORBIT_LISTEN=:8787`<br>`ORBIT_DATA_DIR=/data`<br>`ORBIT_DOCKER_HOST=unix:///var/run/docker.sock`<br>`ORBIT_HOST_PROC=/host/proc`<br>`ORBIT_HOST_SYS=/host/sys`<br>`ORBIT_HOST_ETC=/host/etc`<br>`ORBIT_HOST_ROOTFS=/host/rootfs` | Configuration expects host mounts at `/host/proc` etc. |
| **Procfs State** | Inside container: `/proc` contains only container PIDs (PID 1, self). `/host/proc` is absent. | Container PID isolation prevents reading other container sockets via `/proc/<pid>/net`. |

In `deploy/docker-compose.yml`, `/proc:/host/proc:ro` and `/sys:/host/sys:ro` are included. However, standalone Docker runs or Docker Desktop environments often omit `/proc`. Therefore, Orbit backend **must be resilient to missing `/host/proc`** by augmenting procfs data with Docker-derived socket endpoints.

---

## 9. Concrete Recommendations for Backend Changes (R2, R3, R4)

### 9.1 For R2: Networks Page Overhaul (`frontend/src/pages/Networks.tsx`)
1. **Enrich `HandleListNetworks` (`backend/internal/api/handlers.go`):**  
   Instead of returning bare `[]network.Summary`, return `[]EnrichedNetwork`:
   ```go
   type EnrichedNetwork struct {
       network.Summary
       Containers     []NetworkConnectedContainer `json:"containers"`
       IPAMUsage      *NetworkIPAMUsage           `json:"ipamUsage,omitempty"`
       TotalIPs       int                         `json:"totalIps"`
       UsedIPs        int                         `json:"usedIps"`
       AvailableIPs   int                         `json:"availableIps"`
   }

   type NetworkConnectedContainer struct {
       ID          string   `json:"id"`
       ShortID     string   `json:"shortId"`
       Name        string   `json:"name"`
       State       string   `json:"state"`
       Health      string   `json:"health"`
       IPv4Address string   `json:"ipv4Address"`
       IPv6Address string   `json:"ipv6Address"`
       MacAddress  string   `json:"macAddress"`
       Aliases     []string `json:"aliases,omitempty"`
   }

   type NetworkIPAMUsage struct {
       Subnet              string `json:"subnet"`
       Gateway             string `json:"gateway"`
       IPsInUse            int    `json:"ipsInUse"`
       DynamicIPsAvailable int    `json:"dynamicIpsAvailable"`
   }
   ```
   *Implementation Note:* For each network in `s.inv.Networks()`, Orbit can populate `Containers` in memory directly from `s.inv.Containers()` where `c.Networks` references that network ID, AND extract IPAM usage either from `s.gw.InspectNetwork` or calculate CIDR capacity using `netip.Prefix`. This gives the frontend instant access to connected containers and IPAM subnet allocation pools.

2. **Add Network Driver Details & Options:**  
   Docker driver options (e.g. `com.docker.network.bridge.name`, `com.docker.network.bridge.enable_icc`, `com.docker.network.bridge.enable_ip_masquerade`, `mtu`) are already present in `n.Options` map in `network.Summary`. Ensure they are cleanly presented.

### 9.2 For R3 & R4: Network Flow Lower Section & Flow Telemetry
1. **Add `ContainerTraffic` Breakdown to `NetflowSnapshot` (`backend/internal/netflow/types.go`):**
   ```go
   type ContainerTrafficMetric struct {
       ContainerID   string             `json:"containerId"`
       ContainerName string             `json:"containerName"`
       Status        string             `json:"status"`
       RxBps         model.Gauge        `json:"rxBps"`
       TxBps         model.Gauge        `json:"txBps"`
       RxBytes       model.Gauge        `json:"rxBytes"`
       TxBytes       model.Gauge        `json:"txBytes"`
       RxPps         model.Gauge        `json:"rxPps"`
       TxPps         model.Gauge        `json:"txPps"`
       Interfaces    []docker.IfaceStats `json:"interfaces,omitempty"`
   }
   ```
   In `Builder.BuildSnapshot()`, populate `snapshot.ContainerTraffic` from `streamer.GetAllLatest()` and `inv.Containers()`.

2. **Enhance `SocketConnection` with Process Attribution and Port Security:**
   ```go
   type SocketConnection struct {
       Protocol       string `json:"protocol"` // tcp, tcp6, udp, udp6
       LocalIP        string `json:"localIp"`
       LocalPort      uint16 `json:"localPort"`
       RemoteIP       string `json:"remoteIp"`
       RemotePort     uint16 `json:"remotePort"`
       State          string `json:"state"` // LISTEN, ESTABLISHED, TIME_WAIT, etc.
       ContainerID    string `json:"containerId,omitempty"`
       ContainerName  string `json:"containerName,omitempty"`
       ProcessName    string `json:"processName,omitempty"`    // e.g. "nginx", "redis-server", "sh"
       AppProtocol    string `json:"appProtocol,omitempty"`    // e.g. "HTTP", "HTTPS", "DNS"
       Classification string `json:"classification"`           // "port_based", "docker_inspection", "ebpf_observed"
       DetectionMode  string `json:"detectionMode"`            // "procfs", "docker_engine", "ebpf"
       SecurityLevel  string `json:"securityLevel"`            // "critical", "warning", "info", "secure", "local"
       SecurityBadge  string `json:"securityBadge"`            // "Critical Risk", "Unencrypted", "Public Secure", "Local Only", "Internal"
       SecurityDesc   string `json:"securityDesc,omitempty"`   // Human explanation of exposure
   }
   ```

3. **Hybrid Socket Reader with Docker Port Synthesis (`backend/internal/netflow/socket_reader.go`):**
   - Step A: Read available `/proc` sockets if accessible.
   - Step B: Iterate over all running containers from `sr.inv.Containers()`. For each container:
     - For each port in `c.Ports`:
       - Synthesize a `LISTEN` socket connection with `LocalIP = c.IP` (or `0.0.0.0` if bound to host), `LocalPort = p.PrivatePort`, `ProcessName = c.Command`, and evaluate `AssessPortSecurity(p.PrivatePort, p.HostIP, true)`.
       - If `p.PublicPort > 0`, also synthesize the host-bound listening endpoint (`LocalIP = p.HostIP`, `LocalPort = p.PublicPort`).
     - For each exposed port in `c.ExposedPorts` not already mapped, synthesize internal listening socket.
   - Deduplicate connections.
   - This guarantees that the Active Sockets table is **always fully populated** with container badges, process names, and security indicators regardless of host mount configuration.

4. **Expand Protocol & Security Classification (`backend/internal/netflow/protocols.go`):**
   - Add comprehensive well-known port mapping (databases, web, caches, message brokers, dev servers).
   - Implement `AssessPortSecurity(port uint16, hostIP string, isListen bool) (level string, badge string, desc string)`.

---

## 10. Summary File Reference Map

| File Path | Component | Role / Purpose |
|---|---|---|
| `backend/internal/api/router.go` | API Router | Defines HTTP & WebSocket routing (`/networks`, `/netflow`, `/containers`, `/ws`). |
| `backend/internal/api/handlers.go` | API Handlers | Dispatches `HandleListNetworks`, `HandleGetNetwork`, `HandleNetflow`. |
| `backend/internal/docker/gateway.go` | Docker Gateway | Safe, strictly gated Docker client wrapper. |
| `backend/internal/docker/inventory.go` | Docker Inventory | In-memory container & resource cache with event listener. |
| `backend/internal/docker/stats_stream.go`| Stats Streamer | Background streaming workers reading Docker `/containers/{id}/stats`. |
| `backend/internal/docker/calc.go` | Metric Calculator | Computes CPU, RAM, and network Bps/Pps deltas. |
| `backend/internal/docker/types.go` | Docker Types | Container, Port, ContainerNet, Mount structs. |
| `backend/internal/netflow/types.go` | Netflow Types | Topology, SocketConnection, NetflowSnapshot data models. |
| `backend/internal/netflow/builder.go` | Netflow Builder | Assembles topology nodes, edges, connections, and protocol stats. |
| `backend/internal/netflow/socket_reader.go`| Socket Reader | Reads procfs net files and resolves IPs to containers. |
| `backend/internal/netflow/protocols.go` | Protocols | Well-known port dictionary and classifier. |
| `backend/internal/realtime/hub.go` | Realtime Hub | WebSocket client manager and topic broadcaster. |
| `backend/cmd/orbit/main.go` | Main Service Entry | Initializes all subsystems and launches background monitoring tickers. |
