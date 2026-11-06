# Project: Orbit Networks & Network Flows Revamp

## Architecture
Orbit delivers container management and flow telemetry with a Tailscale-inspired dark console aesthetic (`#111111` canvas, `#181818` surfaces, `#282828` borders). This revamp upgrades Orbit's network management and flow telemetry with UniFi Network-inspired IPAM subnet inspection and Traffic Inspector observability.

### System Diagram
```
                     ┌──────────────────────────────────────────────┐
                     │              Orbit React Frontend            │
                     │  - /networks: UniFi IPAM & Container Clients │
                     │  - /network-flow: Topology + Traffic & Sockets│
                     └──────────────────────┬───────────────────────┘
                                            │ HTTP / WebSocket
                     ┌──────────────────────▼───────────────────────┐
                     │              Orbit Go Backend                │
                     │  - Enriched Networks API (/api/v1/networks)  │
                     │  - Netflow Snapshot Engine (/api/v1/netflow) │
                     │    * StatsStreamer bandwidth calculations    │
                     │    * Hybrid Procfs + Docker socket collector │
                     │    * Port security risk classifier           │
                     └──────────────────────┬───────────────────────┘
                                            │ Docker Socket / Procfs
                     ┌──────────────────────▼───────────────────────┐
                     │     Docker Engine & Container Infrastructure │
                     │  - Network IPAM configurations & endpoints   │
                     │  - Container network attachments & stats     │
                     └──────────────────────────────────────────────┘
```

## Feature Inventory
| # | Feature | Description | Milestone | Source |
|---|---------|-------------|-----------|--------|
| 1 | Backend Enriched Network API | Serve network configurations with IPAM subnet details, allocated IP ranges, usable capacity, and connected container attachments | M1 | R2, R4, survey_backend |
| 2 | Backend Container Bandwidth Telemetry | Expose per-container live bandwidth rates (Rx/Tx Bps, pps, bytes) inside `NetflowSnapshot` | M1 | R3, R4, survey_backend |
| 3 | Hybrid Socket Collector & Port Security Classifier | Synthesize socket connections with container/process attribution and port security risk classification (Critical, Caution, Safe, Internal) | M1 | R3, R4, survey_backend, survey_research |
| 4 | Frontend Enriched Types & API Helpers | TypeScript interfaces for IPAM subnets, container members, container traffic, sockets, and port security | M2 | R2, R3, survey_frontend |
| 5 | UniFi-Style IPAM Subnet Inspection | Visual CIDR breakdown, gateway, network/broadcast IPs, usable host range, IP pool allocation progress bar & capacity | M2 | R2, survey_research, survey_frontend |
| 6 | Connected Container Client Insights | Connected container table with status badges, IPv4/IPv6, MAC address, and network endpoint details | M2 | R2, survey_research, survey_frontend |
| 7 | Network Driver Options & Scope Inspection | Driver badges (bridge, overlay, macvlan, host), scope, MTU, internal isolation, ICC, and compose labels | M2 | R2, survey_research |
| 8 | Retain Interactive Topology Graph | Maintain `@xyflow/react` node/link graph canvas at top of `/network-flow` intact | M3 | R3, survey_frontend |
| 9 | Live Bandwidth Rate Meters | Aggregate RX/TX rate meters, active flows count, and security alerts summary | M3 | R3, survey_research, survey_frontend |
| 10 | Live Traffic Breakdown by Container & Protocol | Top container bandwidth ranking bar chart and protocol traffic distribution chart using Recharts | M3 | R3, survey_research, survey_frontend |
| 11 | Active Sockets Table with Security Attribution | Comprehensive socket table with container badge, process command, protocol, local/remote addresses, state, and port security risk badges | M3 | R3, survey_research, survey_frontend |
| 12 | Dashboard JSX Pre-existing Build Fix | Fix JSX nesting syntax in `Dashboard.tsx:422-506` so `npm run build` compiles with zero errors | M4 | survey_frontend, Acceptance Criteria |
| 13 | End-to-End Build & Live Docker Validation | Verify backend tests (`go test ./...`), frontend build (`npm run build`), and live Docker container (`orbit-live`) route loading | M4 | Acceptance Criteria |

## Milestones
| # | Name | Scope | Dependencies | Status |
|---|------|-------|-------------|--------|
| M1 | Backend Telemetry & Enriched Endpoints | Implement enriched network responses with IPAM & containers, container traffic metrics in `NetflowSnapshot`, hybrid socket collection, and port security classification | none | DONE |
| M2 | Networks Page Overhaul (`Networks.tsx`) | Redesign `/networks` with UniFi IPAM subnet inspector, allocation pool progress bar, connected container table, and driver options | M1 | DONE |
| M3 | Network Flows Lower Section Redesign (`NetworkFlow.tsx`) | Retain top topology graph; implement live bandwidth meters, Recharts container & protocol breakdown, and active sockets table with security indicators | M1 | DONE |
| M4 | Integration, Build Fix & Live Docker Validation | Fix pre-existing build issue in `Dashboard.tsx`, run full test suites (`go test ./...`, `npm run build`), and validate `/networks` and `/network-flow` in `orbit-live` | M2, M3 | DONE |

## Interface Contracts

### Backend ↔ Frontend Network API
`GET /api/v1/networks` and `GET /api/v1/networks/{id}`:
```typescript
interface EnrichedDockerNetwork {
  Id: string;
  Name: string;
  Driver: string;
  Scope: string;
  Internal: boolean;
  EnableIPv6: boolean;
  IPAM: {
    Driver: string;
    Config: Array<{
      Subnet: string;
      Gateway: string;
      IPRange?: string;
    }>;
  };
  SubnetDetails?: {
    Subnet: string;
    Gateway: string;
    NetworkIP: string;
    BroadcastIP: string;
    UsableRange: string;
    TotalIPs: number;
    UsedIPs: number;
    FreeIPs: number;
    UtilizationPercent: number;
  };
  Containers: Record<string, {
    Name: string;
    EndpointID: string;
    MacAddress: string;
    IPv4Address: string;
    IPv6Address: string;
    State?: string;
  }>;
  Options?: Record<string, string>;
  Labels?: Record<string, string>;
}
```

### Backend ↔ Frontend Netflow API
`GET /api/v1/netflow` and WebSocket `netflow` event:
```typescript
interface EnrichedNetflowSnapshot {
  timestamp: string;
  topology: {
    nodes: Array<TopologyNode>;
    edges: Array<TopologyEdge>;
  };
  containerTraffic: Array<{
    containerId: string;
    containerName: string;
    rxBps: number;
    txBps: number;
    rxBytes: number;
    txBytes: number;
    rxPps: number;
    txPps: number;
  }>;
  connections: Array<{
    localAddr: string;
    localPort: number;
    remoteAddr: string;
    remotePort: number;
    protocol: string;
    state: string;
    containerId: string;
    containerName: string;
    processName?: string;
    classification: string;
    detectedProto: string;
    securityLevel: 'critical' | 'caution' | 'safe' | 'internal';
    securityBadge: string;
    securityDesc: string;
  }>;
  protocolStats: Array<{
    protocol: string;
    configured: number;
    observed: number;
    active: number;
  }>;
  totals: {
    totalRxBps: number;
    totalTxBps: number;
    activeConnections: number;
    securityAlertsCount: number;
  };
}
```

## Code Layout
- `backend/internal/api/handlers.go`: `HandleListNetworks` and `HandleGetNetwork` enrichment.
- `backend/internal/netflow/types.go`: Extended `NetflowSnapshot`, `ContainerTrafficMetric`, `SocketConnection` security fields.
- `backend/internal/netflow/builder.go`: Population of `ContainerTraffic` from `streamer.GetAllLatest()`, and totals calculation.
- `backend/internal/netflow/socket_reader.go`: Hybrid procfs + Docker container port synthesis.
- `backend/internal/netflow/protocols.go`: Port security classifier (`AssessPortSecurity`).
- `frontend/src/types/index.ts`: Shared TypeScript interfaces.
- `frontend/src/lib/api.ts`: API methods for enriched networks and netflow.
- `frontend/src/pages/Networks.tsx`: Overhauled UniFi-inspired IPAM & container interface.
- `frontend/src/pages/NetworkFlow.tsx`: Decoupled lower section with bandwidth meters, Recharts visualizations, and security-aware active sockets table.
- `frontend/src/pages/Dashboard.tsx`: Fix pre-existing JSX syntax formatting.
- `tests/e2e/`: E2E test harness verifying network IPAM, flow telemetry, and socket metrics.
