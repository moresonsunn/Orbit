# Orbit Network UX & Telemetry Survey Research
**Author:** Orbit Read-Only Network UX Researcher  
**Target Date:** 2026-11-06  
**Context:** Revamp of Orbit's Networks interface (`frontend/src/pages/Networks.tsx`) and the lower section of Network Flows (`frontend/src/pages/NetworkFlow.tsx`).

---

## 1. Executive Summary & Research Benchmark

Modern infrastructure dashboards have evolved beyond raw Docker CLI dumps. Prosumer and homelab operators today expect the network clarity of **UniFi Network** (IPAM allocation bars, gateway mapping, deep client device insights, Traffic Inspector flows), the endpoint security diagnostics of **GlassWire** (visual risk flags on public sockets, unencrypted protocols), the real-time granular container bandwidth attribution of **Netdata**, and the refined, distraction-free console ergonomics of **Tailscale** (`#111111` dark theme, high-density monospace tables, subtle copy actions).

Orbit currently possesses strong foundational primitives (Docker container inventory, `/proc/[pid]/net/*` socket inspection, React Flow topology visualization, and container streaming statistics). However, its current UI surfaces suffer from two primary limitations:
1. **`Networks.tsx`** renders minimal 3-column cards with raw JSON fields (`Subnetz`, `Gateway`, `Scope`), omitting CIDR allocation mathematics, IP utilization bars, connected container client tables, MAC address mapping, and driver security flags (MTU, ICC, internal isolation).
2. **`NetworkFlow.tsx` (Lower Section)** features a static 6-box protocol counter and an unfilterable socket table without live container bandwidth meters, traffic volume attribution, process tracing, or port security risk indicators (e.g., flagging unencrypted databases exposed to `0.0.0.0`).

This specification provides the industry benchmark analysis, UI/UX architecture, component hierarchy, design tokens, and exact TypeScript/Go data models required to implement an enterprise-grade experience in Orbit.

---

## 2. Industry UX Patterns & Docker Network Adaptations

### 2.1 Subnet & IPAM Inspection Patterns (UniFi Network & Netipam)

In UniFi Network 7.x/8.x and community tools like *getIPAM* / *Netipam*, network operators rely on instant visual cues rather than manually computing CIDR offsets.

#### A. Visual CIDR Breakdown
For every Docker subnet (e.g., `172.20.0.0/16` or `192.168.144.0/20`), users need:
- **Subnet CIDR**: E.g., `172.20.0.0/16` (displayed with a copyable badge).
- **Network Mask**: E.g., `255.255.0.0` (/16) or `255.255.240.0` (/20).
- **Usable Host Range**: Calculated as `[NetworkIP + 1]` to `[BroadcastIP - 1]`. For example: `172.20.0.1` → `172.20.255.254`.
- **Broadcast Address**: E.g., `172.20.255.255`.
- **Total Address Capacity**: $2^{(32 - \text{prefix})} - 2$ usable host addresses (e.g., 254 for `/24`, 4,094 for `/20`, 65,534 for `/16`).

#### B. Gateway Identification & Bridge Device
- Clearly mark the Docker Gateway address (e.g., `172.20.0.1`).
- Display the underlying Linux bridge interface when applicable (e.g., `docker0` or `br-3a4b9c1d8e0f`).

#### C. IP Pool Allocation Range & Utilization Bar
- In UniFi, every network features an **IP Pool Utilization Bar** showing how much of the DHCP / IP pool is currently occupied.
- **Docker Equivalent**:
  - **Allocated / Used IPs**: Number of containers currently assigned an IP on this network + 1 (reserved for gateway).
  - **Available / Free IPs**: Total usable IPs minus Allocated IPs.
  - **Utilization Percentage**: $\frac{\text{Used IPs}}{\text{Total Usable IPs}} \times 100\%$.
  - **Visual Bar Representation**:
    - High-density progress bar (`h-1.5` rounded-full bg `#262626`).
    - Color ramp:
      - `< 70%`: Emerald (`#10b981`) or Subtle Blue (`#3b82f6`)
      - `70% - 90%`: Amber (`#f59e0b`)
      - `> 90%`: Rose (`#f43f5e`) for pool exhaustion warning
    - Metric label: `14 / 254 IPs belegt (5.5%)` with remaining free count `240 frei`.

---

### 2.2 Container Client Table (Connected Endpoints)

UniFi's **Client Devices** tab lists all endpoints connected to a specific VLAN/subnet. Portainer also provides a "Connected Containers" matrix in its network inspection view. Orbit adapts this into a native container client table nested inside or expandable from each network view.

#### Required Data Fields per Connected Container
| Field | Type | UX Presentation & Behavior |
|---|---|---|
| **Container** | Name & ID | Container icon, Name in bold white (`font-medium text-white`), Short ID in muted mono (`text-neutral-500 font-mono text-[11px]`). Click navigates to container detail. |
| **State** | Status | Orbit `StatusBadge` (Running = green dot, Exited = grey dot, Paused = amber dot). |
| **IPv4 Address** | String (IP) | JetBrains Mono tabular IP (e.g. `172.20.0.4`), with a one-click copy icon. |
| **IPv6 Address** | String (optional) | Monospace IPv6 or `—` if IPv6 disabled. |
| **MAC Address** | String | Virtual MAC address (e.g., `02:42:ac:14:00:04`) in mono `text-[11px] text-neutral-400`. |
| **Live Bandwidth** | Rx / Tx | Current throughput from container stats (e.g., `↓ 24.2 kB/s  ↑ 1.8 kB/s`). |
| **Endpoint ID** | String | Truncated 12-char hex endpoint ID (`font-mono text-neutral-500`). |

---

### 2.3 Network Driver & Configuration Badges

Docker networks have critical security and routing flags that operators need to see at a glance:
- **Driver Badges**:
  - `bridge`: Default local bridge (Blue badge: `bg-blue-950/40 text-blue-400 border-blue-800/40`).
  - `macvlan` / `ipvlan`: Direct host NIC assignment (Purple badge: `bg-purple-950/40 text-purple-400 border-purple-800/40`).
  - `overlay`: Swarm multi-host overlay (Amber badge: `bg-amber-950/40 text-amber-400 border-amber-800/40`).
  - `host`: Direct host networking (Cyan badge: `bg-cyan-950/40 text-cyan-400 border-cyan-800/40`).
  - `none`: Isolated container loopback only (Neutral badge: `bg-neutral-800 text-neutral-400 border-neutral-700`).
- **Security & Scope Badges**:
  - **Internal Only (`internal: true`)**: Green shield pill `🛡️ Internes Netz (Kein WAN/Gateway)`. Indicates zero external internet egress, ideal for database/backend networks.
  - **Inter-Container Communication (`enable_icc: false`)**: Security pill `🔒 ICC Deaktiviert` (containers on this network cannot talk directly to each other without explicit links).
  - **MTU Size**: Badge showing MTU (e.g., `MTU 1500` or `MTU 9000 Jumbo`).
  - **Attachable (`attachable: true`)**: Manual container attachment enabled.
  - **IP Masquerade (`enable_ip_masquerade: true`)**: NAT enabled to host interface.

---

## 3. Network Flows Lower Section: UniFi Traffic Inspector, Netdata & GlassWire Patterns

Orbit's `frontend/src/pages/NetworkFlow.tsx` upper half holds a React Flow interactive topology graph. The lower half must be revamped to provide real-time traffic inspection, protocol detection, and active socket telemetry with security risk attribution.

```
+-------------------------------------------------------------------------------------------------------+
| TOP: React Flow Interactive Network Topology (Retained & Polished)                                    |
| [ Host Node ]  ==>  [ Docker Bridge Networks ]  ==>  [ Running Containers ]                           |
+-------------------------------------------------------------------------------------------------------+
| LOWER SECTION: Traffic Inspector, Bandwidth Meters & Socket Security Diagnostics                     |
|                                                                                                       |
| [1. Live Telemetry Summary Cards: Total Rx/Tx Throughput, Active Sockets, Listeners, Security Alerts]  |
|                                                                                                       |
| +---------------------------------------------------+-----------------------------------------------+ |
| | A. Container Bandwidth Ranking (UniFi/Netdata)     | B. Application Protocol Distribution (DPI)    | |
| | - web-proxy:  4.2 MB/s  ████████████████ (62%)     | - HTTPS (443):  18 flows  | 4.1 MB/s          | |
| | - database:   1.8 MB/s  ███████          (27%)     | - PostgreSQL:   8 flows   | 1.8 MB/s          | |
| | - redis:      450 kB/s  ██               (7%)      | - DNS (53):     12 flows  | 42 kB/s           | |
| +---------------------------------------------------+-----------------------------------------------+ |
|                                                                                                       |
| [2. Active Sockets & Port Security Inspector (GlassWire & Tailscale)]                                 |
| Filter Controls: [ Search IP/Port/Name ] [ All | Listen | Established ] [ All Risks | Critical | Safe ] |
|                                                                                                       |
| Table: Risk Badge | Protocol | Local Socket | Remote Socket | State | Container | Process | Volume/Rate|
| 🔴 CRITICAL     | TCP      | 0.0.0.0:6379 | 0.0.0.0:*     | LISTEN| redis     | redis-server | 0 B/s      |
| 🟢 SAFE         | TCP      | 127.0.0.1:80 | 0.0.0.0:*     | LISTEN| nginx     | nginx: master| 12 kB/s    |
| 🔵 INTERNAL     | TCP      | 172.18.0.3:5432| 172.18.0.2:48912| ESTAB | postgres  | postgres     | 85 kB/s   |
+-------------------------------------------------------------------------------------------------------+
```

### 3.1 Live Traffic Breakdown & Bandwidth Rate Gauges

Inspired by UniFi Traffic Inspector and Netdata cgroups telemetry:
1. **Aggregated Header Metric Cards (4-column grid)**:
   - **Download (Rx) Throughput**: Current aggregate received bandwidth across all containers (formatted via `formatBps`, e.g. `12.4 MB/s`), with a micro sparkline or activity indicator.
   - **Upload (Tx) Throughput**: Current aggregate transmitted bandwidth across all containers (e.g. `2.1 MB/s`).
   - **Active Sockets / Flows**: Total live `ESTABLISHED` connections count.
   - **Port Security Alerts**: Counter of detected public exposure warnings (e.g. `2 Warnungen` in amber/rose).

2. **Container Bandwidth Ranking**:
   - Ranked list showing containers consuming the highest network bandwidth.
   - For each container: Container Name, Rx Rate, Tx Rate, total volume transferred, and a proportional horizontal fill bar.

3. **Protocol Categorization & Detection**:
   - Deep classification beyond raw port numbers:
     - `Web & Secure Web`: HTTPS (443), HTTP (80, 8080)
     - `Databases`: PostgreSQL (5432), MySQL (3306), Redis (6379), MongoDB (27017)
     - `Infrastructure & DNS`: DNS (53), NTP (123), DHCP (67/68)
     - `Remote Access & Shell`: SSH (22), Telnet (23), RDP (3389)
     - `Management`: Portainer (9000/9443), Orbit (8787), Docker Daemon (2375/2376)
     - `Message Queues & RPC`: gRPC, NATS (4222), RabbitMQ (5672)

---

### 3.2 Port Security Indicators & Threat Risk Badges (GlassWire & Tailscale)

Exposing internal ports on `0.0.0.0` is one of the most common configuration errors in Docker, frequently leading to crypto-jacking or data leaks (e.g., exposed unauthenticated Redis instances). Orbit provides an automated security posture classification for every socket:

#### Security Risk Classification Rules
| Risk Tier | Badge Styling | Condition / Heuristics | Real-World Context & Warning |
|---|---|---|---|
| 🚨 **CRITICAL** | `bg-rose-950/40 text-rose-400 border-rose-800/40` | `LocalIP == 0.0.0.0` or `::` **AND** port belongs to an unencrypted protocol (FTP 21, Telnet 23, plain HTTP 80 without TLS proxy) OR raw internal database (Redis 6379, MySQL 3306, Postgres 5432, Mongo 27017, Elasticsearch 9200). | **Extreme Danger**: Database or unencrypted service is publicly reachable from the entire WAN/LAN without encryption. |
| ⚠️ **CAUTION** | `bg-amber-950/40 text-amber-400 border-amber-800/40` | `LocalIP == 0.0.0.0` or `::` with standard encrypted ingress (HTTPS 443/8443, SSH 22, WireGuard 51820) or unknown exposed ports. | **Public Ingress**: Service is open to all interfaces. Ensure firewall and strong auth are configured. |
| 🛡️ **SAFE / LOCAL** | `bg-emerald-950/40 text-emerald-400 border-emerald-800/40` | `LocalIP == 127.0.0.1` or `::1`. | **Host-Local Only**: Bound exclusively to loopback interface. Inaccessible from external network; safe for reverse-proxy backends. |
| 🔒 **INTERNAL MESH** | `bg-blue-950/40 text-blue-400 border-blue-800/40` | `LocalIP` is within private RFC1918 container subnet (`172.16.0.0/12`, `10.0.0.0/8`, `192.168.0.0/16`) without host publishing. | **Container-to-Container**: Traffic stays inside Docker virtual bridge or overlay network. |

---

### 3.3 Active Socket Table UX

Tailscale-inspired high-density data table:
- **Header**: Tabular column titles in uppercase tracking-wider font (`text-[11px] font-semibold text-neutral-400 uppercase tracking-wider`).
- **Row Hover**: Subtle highlight `hover:bg-white/[0.02]`.
- **Monospace Addresses**: IP and port formatted as `172.18.0.4:8080` in `font-mono tabular-nums text-neutral-200`. Ports styled with subtle accent.
- **Copy on Click**: Hovering an IP displays a copy tooltip / icon for quick terminal pasting.
- **Socket States**:
  - `ESTABLISHED`: Active bidirectional communication (Emerald pill).
  - `LISTEN`: Daemon waiting for incoming connection (Slate/neutral pill).
  - `TIME_WAIT`: Socket waiting for past packets to terminate (Amber pill).
  - `CLOSE_WAIT`: Remote closed, local waiting to close (Zinc pill).
  - `SYN_SENT`: Connection initiation (Blue pulse pill).

---

## 4. Orbit Design System & UX Hierarchy

Orbit uses a strict dark console aesthetic inspired by Tailscale:

### 4.1 Palette Tokens & Style Guide
- **Canvas Base**: `#111111` (`bg-[#111111]`)
- **Card / Surface Background**: `#181818` (`bg-[#181818]`)
- **Inner Panel / Table Container**: `#141414` or `#121212`
- **Border Default**: `#282828` (`border-[#282828]`)
- **Border Hover**: `#383838` (`hover:border-[#383838]`)
- **Dividers & Table Borders**: `#242424` or `#1f1f1f`
- **Typography**:
  - Body / Headings: `font-sans` (`Inter`, system UI fallback)
  - Code, IPs, MACs, Ports, Throughput: `font-mono` (`JetBrains Mono`, monospace fallback)
- **Status & Risk Accents**:
  - Emerald: `text-emerald-400`, `bg-emerald-950/40`, `border-emerald-800/40`, `bg-emerald-500`
  - Amber: `text-amber-400`, `bg-amber-950/40`, `border-amber-800/40`, `bg-amber-500`
  - Rose: `text-rose-400`, `bg-rose-950/40`, `border-rose-800/40`, `bg-rose-500`
  - Blue: `text-blue-400`, `bg-blue-950/40`, `border-blue-800/40`, `bg-blue-500`
  - Cyan: `text-cyan-400`, `bg-cyan-950/40`, `border-cyan-800/40`
  - Purple: `text-purple-400`, `bg-purple-950/40`, `border-purple-800/40`

---

### 4.2 Layout Hierarchy: `Networks.tsx`

```
Networks Page
│
├── Header & Controls
│   ├── Title: "Networks"
│   ├── Subtitle: "Docker-Netzwerktreiber, IPAM-Subnetze und Container-Zuweisungen"
│   ├── Search bar (filter by Network Name, Driver, CIDR)
│   └── Driver filter pills: [ All (6) | Bridge (4) | Macvlan (1) | Host (1) ]
│
├── Subnet IPAM Overview Cards (1-col or 2-col responsive grid)
│   └── Per Network Card:
│       ├── Header:
│       │   ├── Network Name & Short ID
│       │   ├── Driver Badge (bridge, overlay, etc.)
│       │   ├── Scope Badge (local / swarm)
│       │   └── Destructive Action: Delete button (with confirmation modal)
│       │
│       ├── IPAM Inspection Section:
│       │   ├── CIDR badge with copy button
│       │   ├── Gateway IP with interface icon
│       │   ├── Usable Host Range: startIP — endIP
│       │   └── IP Utilization Progress Bar:
│       │       ├── Bar: filled portion (emerald/blue) vs free portion (neutral)
│       │       └── Metric text: "X / Y IPs belegt (Z%) — N frei"
│       │
│       ├── Driver & Security Options:
│       │   ├── Internal only (🛡️ Keine WAN-Verbindung)
│       │   ├── Inter-Container Communication (ICC: An/Aus)
│       │   ├── MTU (1500)
│       │   └── IP Masquerade (NAT)
│       │
│       └── Connected Containers Client Section:
│           ├── Header: "Verbundene Container (N)" + expand/collapse toggle
│           └── Client Table:
│               ├── Container Name & StatusBadge (running/exited)
│               ├── Assigned IPv4 & IPv6
│               ├── Virtual MAC Address
│               ├── Live Throughput (Rx / Tx kB/s)
│               └── Disconnect / Action shortcut
```

---

### 4.3 Layout Hierarchy: `NetworkFlow.tsx` (Lower Section)

```
Network Flows Page
│
├── Top Section: Interactive React Flow Topology Graph
│   └── Host -> Networks -> Containers (Retained)
│
└── Lower Section: Traffic Inspector & Socket Telemetry
    │
    ├── 1. Live Bandwidth & Flow Metric Cards (4 cards)
    │   ├── Rx Rate (e.g. "4.2 MB/s") + sparkline
    │   ├── Tx Rate (e.g. "820 kB/s") + sparkline
    │   ├── Active Sockets count (e.g. "34")
    │   └── Security Risk Alerts count (e.g. "2 kritisch")
    │
    ├── 2. Traffic Inspector Visual Distribution (2 side-by-side cards)
    │   ├── Left: Container Bandwidth Ranking
    │   │   └── Top 5 containers with Rx/Tx rates and percentage bar
    │   └── Right: Protocol Classification Breakdown
    │       └── HTTPS, HTTP, PostgreSQL, Redis, DNS with flow counts & badge colors
    │
    └── 3. Active Sockets & Port Security Table
        ├── Filter & Search Bar:
        │   ├── Search input: "Filter nach IP, Port, Container..."
        │   ├── State filter: [ Alle | ESTABLISHED | LISTEN | TIME_WAIT ]
        │   └── Security filter: [ Alle | 🔴 Kritisch | ⚠️ Öffentlich | 🛡️ Lokal ]
        │
        └── Table View:
            ├── Column 1: Security Risk Badge (Critical / Public / Localhost / Private)
            ├── Column 2: Protocol (TCP / UDP) & App Detection (Redis, HTTPS, etc.)
            ├── Column 3: Local Socket (IP:Port)
            ├── Column 4: Remote Socket (IP:Port)
            ├── Column 5: State (ESTABLISHED, LISTEN, TIME_WAIT)
            ├── Column 6: Container Attribution (Name + ID)
            ├── Column 7: Process Attribution (Process Name / Command)
            └── Column 8: Volume / Speed (kB/s or Bytes)
```

---

## 5. Input & Output Data Models Specification

### 5.1 Frontend TypeScript Data Models (`frontend/src/types/index.ts`)

```typescript
// ==========================================
// 1. IPAM & Network Types
// ==========================================

export interface IPAMSubnetConfig {
  subnet: string;              // e.g. "172.20.0.0/16"
  gateway?: string;            // e.g. "172.20.0.1"
  ipRange?: string;            // e.g. "172.20.10.0/24" (optional pool limit)
  networkAddress: string;      // e.g. "172.20.0.0"
  broadcastAddress: string;    // e.g. "172.20.255.255"
  startUsableIP: string;       // e.g. "172.20.0.1"
  endUsableIP: string;         // e.g. "172.20.255.254"
  totalIPs: number;            // e.g. 65534
  usedIPs: number;             // e.g. 12
  freeIPs: number;             // e.g. 65522
  utilizationPercent: number;  // e.g. 0.02 (%)
}

export interface NetworkContainerMember {
  id: string;                  // Container ID
  shortId: string;             // 12-char ID
  name: string;                // Container name
  state: string;               // "running" | "exited" | "paused"
  ipv4Address: string;         // e.g. "172.20.0.3"
  ipv6Address?: string;        // e.g. "fe80::1"
  macAddress: string;          // e.g. "02:42:ac:14:00:03"
  endpointId: string;          // Endpoint ID
  rxBps?: number;              // Current Rx rate in bytes/sec
  txBps?: number;              // Current Tx rate in bytes/sec
}

export interface NetworkDriverOptions {
  mtu?: string;
  enableICC?: boolean;
  enableIPMasquerade?: boolean;
  bridgeName?: string;
  parentInterface?: string;
  rawOptions?: Record<string, string>;
}

export interface EnrichedDockerNetwork {
  id: string;
  name: string;
  created: string;
  scope: string;               // "local" | "swarm" | "global"
  driver: string;              // "bridge" | "macvlan" | "overlay" | "host" | "none"
  enableIPv4: boolean;
  enableIPv6: boolean;
  internal: boolean;           // Isolated without WAN
  attachable: boolean;
  ingress: boolean;
  ipam: {
    driver: string;
    subnets: IPAMSubnetConfig[];
  };
  options: NetworkDriverOptions;
  containers: NetworkContainerMember[];
  containersCount: number;
}

// ==========================================
// 2. Netflow & Socket Telemetry Types
// ==========================================

export type PortSecurityRisk = 'critical' | 'warning' | 'safe' | 'internal';

export interface EnrichedSocketConnection {
  protocol: 'tcp' | 'tcp6' | 'udp' | 'udp6';
  localIp: string;
  localPort: number;
  remoteIp: string;
  remotePort: number;
  state: 'LISTEN' | 'ESTABLISHED' | 'TIME_WAIT' | 'CLOSE_WAIT' | 'SYN_SENT' | 'ACTIVE';
  containerId?: string;
  containerName?: string;
  processName?: string;
  processPid?: number;
  appProtocol: string;         // "HTTPS", "HTTP", "DNS", "Redis", "PostgreSQL", etc.
  classification: 'port_based' | 'ebpf_observed';
  detectionMode: 'socket_snapshot' | 'ebpf';
  securityRisk: PortSecurityRisk;
  riskReason: string;          // Explanation for badge tooltip
  isPublicBind: boolean;       // Bound to 0.0.0.0 or [::]
  isInsecureProtocol: boolean; // Plaintext or raw DB
}

export interface ContainerTrafficSummary {
  containerId: string;
  containerName: string;
  rxBps: number;
  txBps: number;
  totalBps: number;
  rxBytes: number;
  txBytes: number;
  activeSockets: number;
  bandwidthSharePercent: number; // Share of total container bandwidth
}

export interface ProtocolTrafficStats {
  protocol: string;            // "HTTPS", "HTTP", "PostgreSQL", "Redis", etc.
  category: 'web' | 'database' | 'system' | 'remote' | 'other';
  configuredPortsCount: number;
  observedConnectionsCount: number;
  activeFlows: number;
  rxBps?: number;
  txBps?: number;
}

export interface EnrichedNetflowSnapshot {
  timestamp: string;
  ebpfAvailable: boolean;
  ebpfStatus: string;
  trafficTotals: {
    totalRxBps: number;
    totalTxBps: number;
    totalRxBytes: number;
    totalTxBytes: number;
    activeConnectionsCount: number;
    listeningSocketsCount: number;
    securityAlertsCount: number;
  };
  topology: {
    nodes: any[];
    edges: any[];
  };
  containerTraffic: ContainerTrafficSummary[];
  protocolStats: ProtocolTrafficStats[];
  connections: EnrichedSocketConnection[];
}
```

---

### 5.2 Backend Go Data Models (`backend/internal/netflow/` & `docker/`)

```go
package netflow

import (
	"time"
	"github.com/phili/orbit/internal/model"
)

type PortSecurityRisk string

const (
	RiskCritical PortSecurityRisk = "critical"
	RiskWarning  PortSecurityRisk = "warning"
	RiskSafe     PortSecurityRisk = "safe"
	RiskInternal PortSecurityRisk = "internal"
)

type EnrichedSocketConnection struct {
	Protocol           string           `json:"protocol"`
	LocalIP            string           `json:"localIp"`
	LocalPort          uint16           `json:"localPort"`
	RemoteIP           string           `json:"remoteIp"`
	RemotePort         uint16           `json:"remotePort"`
	State              string           `json:"state"`
	ContainerID        string           `json:"containerId,omitempty"`
	ContainerName      string           `json:"containerName,omitempty"`
	ProcessName        string           `json:"processName,omitempty"`
	ProcessPID         int              `json:"processPid,omitempty"`
	AppProtocol        string           `json:"appProtocol"`
	Classification     string           `json:"classification"`
	DetectionMode      string           `json:"detectionMode"`
	SecurityRisk       PortSecurityRisk `json:"securityRisk"`
	RiskReason         string           `json:"riskReason"`
	IsPublicBind       bool             `json:"isPublicBind"`
	IsInsecureProtocol bool             `json:"isInsecureProtocol"`
}

type ContainerTrafficSummary struct {
	ContainerID           string  `json:"containerId"`
	ContainerName         string  `json:"containerName"`
	RxBps                 int64   `json:"rxBps"`
	TxBps                 int64   `json:"txBps"`
	TotalBps              int64   `json:"totalBps"`
	RxBytes               int64   `json:"rxBytes"`
	TxBytes               int64   `json:"txBytes"`
	ActiveSockets         int     `json:"activeSockets"`
	BandwidthSharePercent float64 `json:"bandwidthSharePercent"`
}

type TrafficTotals struct {
	TotalRxBps             int64 `json:"totalRxBps"`
	TotalTxBps             int64 `json:"totalTxBps"`
	TotalRxBytes           int64 `json:"totalRxBytes"`
	TotalTxBytes           int64 `json:"totalTxBytes"`
	ActiveConnectionsCount int   `json:"activeConnectionsCount"`
	ListeningSocketsCount  int   `json:"listeningSocketsCount"`
	SecurityAlertsCount    int   `json:"securityAlertsCount"`
}

type ProtocolTrafficStats struct {
	Protocol    string `json:"protocol"`
	Category    string `json:"category"`
	Configured  int    `json:"configuredPortsCount"`
	Observed    int    `json:"observedConnectionsCount"`
	ActiveFlows int    `json:"activeFlows"`
	RxBps       int64  `json:"rxBps,omitempty"`
	TxBps       int64  `json:"txBps,omitempty"`
}

type EnrichedNetflowSnapshot struct {
	Timestamp        time.Time                  `json:"timestamp"`
	EBPFAvailable    bool                       `json:"ebpfAvailable"`
	EBPFStatus       string                     `json:"ebpfStatus"`
	TrafficTotals    TrafficTotals              `json:"trafficTotals"`
	Topology         Topology                   `json:"topology"`
	ContainerTraffic []ContainerTrafficSummary  `json:"containerTraffic"`
	ProtocolStats    []ProtocolTrafficStats     `json:"protocolStats"`
	Connections      []EnrichedSocketConnection `json:"connections"`
}
```

```go
package docker

type SubnetIPAMInfo struct {
	Subnet             string  `json:"subnet"`
	Gateway            string  `json:"gateway,omitempty"`
	IPRange            string  `json:"ipRange,omitempty"`
	NetworkAddress     string  `json:"networkAddress"`
	BroadcastAddress   string  `json:"broadcastAddress"`
	StartUsableIP      string  `json:"startUsableIP"`
	EndUsableIP        string  `json:"endUsableIP"`
	TotalIPs           int     `json:"totalIPs"`
	UsedIPs            int     `json:"usedIPs"`
	FreeIPs            int     `json:"freeIPs"`
	UtilizationPercent float64 `json:"utilizationPercent"`
}

type NetworkContainerMember struct {
	ID          string `json:"id"`
	ShortID     string `json:"shortId"`
	Name        string `json:"name"`
	State       string `json:"state"`
	IPv4Address string `json:"ipv4Address"`
	IPv6Address string `json:"ipv6Address,omitempty"`
	MacAddress  string `json:"macAddress,omitempty"`
	EndpointID  string `json:"endpointId"`
	RxBps       int64  `json:"rxBps,omitempty"`
	TxBps       int64  `json:"txBps,omitempty"`
}

type NetworkDriverOptions struct {
	MTU                string            `json:"mtu,omitempty"`
	EnableICC          bool              `json:"enableICC"`
	EnableIPMasquerade bool              `json:"enableIPMasquerade"`
	BridgeName         string            `json:"bridgeName,omitempty"`
	ParentInterface    string            `json:"parentInterface,omitempty"`
	RawOptions         map[string]string `json:"rawOptions,omitempty"`
}

type EnrichedNetwork struct {
	ID              string                   `json:"id"`
	Name            string                   `json:"name"`
	Created         string                   `json:"created"`
	Scope           string                   `json:"scope"`
	Driver          string                   `json:"driver"`
	EnableIPv4      bool                     `json:"enableIPv4"`
	EnableIPv6      bool                     `json:"enableIPv6"`
	Internal        bool                     `json:"internal"`
	Attachable      bool                     `json:"attachable"`
	Ingress         bool                     `json:"ingress"`
	IPAMDriver      string                   `json:"ipamDriver"`
	Subnets         []SubnetIPAMInfo         `json:"subnets"`
	Options         NetworkDriverOptions     `json:"options"`
	Containers      []NetworkContainerMember `json:"containers"`
	ContainersCount int                      `json:"containersCount"`
}
```

---

## 6. Implementation Architecture Plan

### 6.1 Backend Telemetry Pipeline
1. **Network Enrichment in `backend/internal/api/handlers.go` & `docker/inventory.go`**:
   - Enhance `HandleListNetworks` to return `[]EnrichedNetwork` rather than raw `network.Summary`.
   - Calculate CIDR properties for every subnet using standard Go `net` package IP bit arithmetic:
     - Parse CIDR (`net.ParseCIDR`).
     - Compute Network address, Broadcast address, total address count ($2^{32-\text{ones}} - 2$).
     - Match containers from `inv.Containers()` belonging to this network (`c.Networks[].NetworkID == net.ID`).
     - Populate `usedIPs` = count of assigned containers + (1 if gateway is defined).
     - Populate `freeIPs` = totalIPs - usedIPs.
     - Calculate `utilizationPercent` = `(usedIPs / totalIPs) * 100`.
   - Extract driver options (`mtu`, `enable_icc`, `com.docker.network.bridge.name`).
   - Extract container telemetry from `streamer.GetAllLatest()` to attach real-time `rxBps`/`txBps` to each container member.

2. **Netflow Pipeline in `backend/internal/netflow/builder.go` & `socket_reader.go`**:
   - In `SocketReader`, enrich each `SocketConnection` with security analysis:
     - Check `isPublicBind = (localIP == "0.0.0.0" || localIP == "::")`.
     - Check if port is sensitive/unencrypted:
       - Raw database: 6379 (Redis), 3306 (MySQL), 5432 (Postgres), 27017 (Mongo), 9200 (Elastic).
       - Plaintext protocol: 21 (FTP), 23 (Telnet), 80 (HTTP).
     - Assign `SecurityRisk`:
       - If `isPublicBind` and sensitive/insecure $\rightarrow$ `RiskCritical` (Reason: "Öffentlich unverschlüsselte Schnittstelle auf 0.0.0.0").
       - If `isPublicBind` and encrypted (443, 22) $\rightarrow$ `RiskWarning` (Reason: "Öffentlicher Ingress-Port").
       - If `localIP == "127.0.0.1" || localIP == "::1"` $\rightarrow$ `RiskSafe` (Reason: "Host-interner Loopback-Bind").
       - Otherwise (e.g. `172.x.x.x`) $\rightarrow$ `RiskInternal` (Reason: "Privates Docker-Subnetz").
   - Aggregate container bandwidth summaries:
     - Query `streamer.GetAllLatest()`.
     - Compute `TrafficTotals` (total Rx rate, total Tx rate).
     - Compute `ContainerTrafficSummary` for all containers with non-zero traffic or active sockets.
     - Calculate each container's share of total traffic (`BandwidthSharePercent`).

### 6.2 Frontend Component Architecture
1. **`frontend/src/pages/Networks.tsx`**:
   - Re-architect from a basic 3-card grid into:
     - Top filter/search bar (Search network, filter by driver: all, bridge, macvlan, host, overlay).
     - Subnet Cards displaying:
       - Header: Name, ID, Driver pill, Scope pill, Delete button (if destructive allowed).
       - CIDR & IPAM block with copyable badge, Gateway, and Range.
       - Visual IP Utilization Bar with percentage and used/free counts.
       - Driver options badges (Internal, ICC, MTU, Masquerade).
       - Connected Containers drawer or expandable table displaying Container Name, Status badge, IPv4 with copy, MAC address, and current throughput.
2. **`frontend/src/pages/NetworkFlow.tsx` (Lower Section)**:
   - Keep React Flow graph in upper section.
   - Replace old protocol boxes and unfilterable table with:
     - 4 metric cards: Total Rx rate, Total Tx rate, Active flows, Security risk alerts.
     - Dual-column Traffic Inspector panel:
       - Left: Top Bandwidth Consumers (by container) with horizontal bar gauges.
       - Right: Protocol Distribution (HTTPS, HTTP, Postgres, Redis, DNS) with flow counts.
     - Active Socket Table with:
       - Search bar & risk/state filter pills.
       - Visual risk badges (Critical, Caution, Safe, Internal).
       - Monospace IP:Port formatting with copy actions.
       - Container and process attribution.
       - State badges.

---

## 7. Verification & Acceptance Testing Protocol

To ensure no regressions and verify complete fidelity:
1. **Backend Verification**:
   - `go test ./...` in `backend/` must pass.
   - `go build ./...` must succeed with zero compile errors.
   - Unit tests covering CIDR calculations (IPv4 /24, /16, /20), edge cases (gateway at start vs end), and port security classification heuristics.
2. **Frontend Verification**:
   - `npm run build` in `frontend/` must complete with zero TypeScript (`tsc`) and Vite build errors.
   - Verify dark console theme contrast ratios (Tailscale palette `#111111`, `#181818`, `#282828`).
3. **End-to-End Application Verification**:
   - In running Docker container `orbit-live`:
     - Access `/networks`: Verify all Docker networks load with full CIDR breakdown, IP utilization bars, driver badges, and connected containers table.
     - Access `/network-flow`: Verify interactive React Flow topology is rendered at top, and the lower section displays live bandwidth gauges, top container bandwidth distribution, protocol breakdown, and the active socket table with port security risk badges.
