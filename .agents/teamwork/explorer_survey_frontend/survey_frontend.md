# Frontend UI Architecture & Investigation Survey: Networks & Network Flow

**Investigator**: Read-Only Frontend UI Explorer  
**Target Project**: Orbit (CasaOS / Docker Telemetry & Management Platform)  
**Date**: 2026-11-05 / 2026-11-06  
**Scope**: `frontend/src/pages/Networks.tsx`, `frontend/src/pages/NetworkFlow.tsx`, Theme Tokens, UI Components, Visualization Libraries, API & WebSocket Clients, Build Setup.

---

## Executive Summary

Orbit's frontend is a modern React 19 SPA built with Vite 8, TypeScript (~7.0 / ES2022), Tailwind CSS v4, and `@xyflow/react` (React Flow v12) alongside `recharts` v3.10. It adheres to a dark console aesthetic inspired by Tailscale (`#111111` canvas, `#181818` card surfaces, `#282828` borders, `Inter` and `JetBrains Mono` typography).

Currently:
1. **`Networks.tsx` (123 lines)** is an overly simplistic card grid that only renders basic network driver badges and first-element IPAM strings (`Subnet`, `Gateway`). It completely omits connected container assignments, IP allocation pool calculation/visualization (UniFi-style IP pools), driver options/flags, and master-detail or expandable inspection.
2. **`NetworkFlow.tsx` (265 lines)** features an interactive 3-column topology graph powered by `@xyflow/react` (Host $\to$ Networks $\to$ Containers), but its lower section only displays static protocol summary cards and an unpaginated 25-row socket table without live container/protocol traffic breakdowns, bandwidth rate meters, process attribution, or port security analysis.
3. **Backend Readiness**: The Go backend endpoint `/api/v1/networks/{id}` already returns complete Docker inspect data (including `Containers`, `Options`, `Labels`, and Docker v25 `Status.IPAM.Subnets` with `IPsInUse` and `DynamicIPsAvailable`), and `api.getNetwork(id)` already exists in `frontend/src/lib/api.ts`.
4. **Existing Build Issue**: `npm run build` currently fails due to a JSX syntax error in `frontend/src/pages/Dashboard.tsx:422-506` where sibling elements are wrapped in parentheses without a fragment `<>...</>`. This must be addressed during frontend implementation.

---

## 1. Deep Dive: `frontend/src/pages/Networks.tsx`

### 1.1 Current Architecture & File Inspection
- **Path**: `c:\Users\phili\Documents\GitHub\Orbit\frontend\src\pages\Networks.tsx`
- **Lines of Code**: 123 lines
- **Imports**: `React`, `useState`, `useEffect`, `Globe`, `Trash2`, `ArrowRight`, `DockerNetwork`, `AuthStatus`, `api` (`../lib/api`), `ActionConfirmModal`.
- **State Management**:
  - `networks: DockerNetwork[]` (state initialized empty)
  - `loading: boolean` (state initialized `true`)
  - `selectedNetwork: DockerNetwork | null` (declared on line 14, **never utilized anywhere in the component!**)
  - `deleteModal: DockerNetwork | null` (modal state for network removal)
- **Data Fetching**:
  ```tsx
  const loadNetworks = async () => {
    try {
      const data = await api.getNetworks();
      setNetworks(data);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };
  ```
- **Layout & Render**:
  - A simple header banner with title "Networks" and subtitle.
  - A 3-column CSS grid (`grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4`).
  - Each card renders:
    - `net.Name`, 12-char ID (`net.Id.slice(0, 12)`), `net.Driver` badge.
    - Delete button (`Trash2`) if `enableDestructive` is active and network name is not `bridge`, `host`, or `none`.
    - `Subnet: net.IPAM?.Config?.[0]?.Subnet || '—'` (only checks the 0th item!).
    - `Gateway: net.IPAM?.Config?.[0]?.Gateway || '—'`.
    - `Scope: net.Scope`.
    - `Intern: net.Internal ? 'Ja' : 'Nein'`.

### 1.2 Deficiencies & Gaps vs. UniFi Network & Modern Container Tools
1. **IPAM Subnet Inspection & IP Pools**:
   - **No CIDR Calculation**: Does not calculate usable IP addresses, total capacity, prefix length, broadcast, or network boundaries (e.g. `172.18.0.0/16` provides 65,534 usable IPs; `172.20.0.0/24` provides 254 usable IPs).
   - **No Visual IP Allocation Pool / Utilization Meter**: UniFi Network displays a horizontal segmented bar representing allocated vs. available IP addresses with utilization percentages. Orbit currently has no visual bar or metric for IP pool utilization.
   - **No Support for `IPRange` or `AuxiliaryAddresses`**: Docker IPAM allows setting an `IPRange` (e.g. `172.18.1.0/24`) and reserved auxiliary IPs; `Networks.tsx` completely ignores these fields.
   - **No IPv6 IPAM Support**: Dual-stack Docker networks have multiple entries in `net.IPAM.Config` (IPv4 and IPv6). Hardcoding `net.IPAM.Config[0]` drops IPv6 subnets and gateways.
   - **Missing Dynamic IP Status**: Docker Engine v25+ reports `Status.IPAM.Subnets[subnet] = { IPsInUse, DynamicIPsAvailable }` directly in `NetworkInspect`. Orbit does not surface this data.

2. **Connected Containers**:
   - **Zero Container Visibility**: Currently, `Networks.tsx` does **not display any connected containers**. Users cannot see which containers are attached to which Docker networks!
   - **Docker Inspect Payload**: Docker's `NetworkInspect` endpoint (`GET /api/v1/networks/{id}`) returns:
     ```json
     "Containers": {
       "0690d0622806dd576fcac61be96e6e4d12a9a08cc5082c434d18d238ea1ae369": {
         "Name": "test-orbit-cpu-load",
         "EndpointID": "40a24698f777143bdb8861525a90ba5f30676427a9a3333b8371ee5f51ab3c09",
         "MacAddress": "3e:ac:cc:bb:e6:7a",
         "IPv4Address": "172.18.0.3/16",
         "IPv6Address": ""
       }
     }
     ```
   - **Container Attribution & Status**: By cross-referencing `net.Containers` with container data from `api.getContainers()`, Orbit can render container status badges (Running `emerald-500`, Stopped `neutral-500`, Unhealthy `rose-500`), container short IDs, assigned IPv4/IPv6 addresses, and MAC addresses.

3. **Driver Options, Flags & Scope**:
   - Docker networks store crucial driver configurations in `net.Options`, such as:
     - `com.docker.network.bridge.enable_icc` (inter-container communication)
     - `com.docker.network.bridge.enable_ip_masquerade` (NAT masquerading)
     - `com.docker.network.bridge.host_binding_ipv4`
     - `com.docker.network.driver.mtu` (MTU size)
   - Important flags are uninspected: `Attachable`, `Ingress`, `EnableIPv4`, `EnableIPv6`, `ConfigOnly`.
   - Docker Compose labels (`com.docker.compose.project`, `com.docker.compose.network`) are hidden.

4. **Layout & UX**:
   - The current 3-column card grid is static and lacks depth.
   - Missing master-detail or expandable drawer / accordion pattern to inspect deep IPAM configurations without cluttering the overview.
   - Missing top KPI summary bar (Total Networks, Total Subnets Managed, Active Endpoints, Bridge vs. Host vs. Overlay breakdown).
   - Missing search and filter bar (filter by driver, search by network name, CIDR, or container IP).

---

## 2. Deep Dive: `frontend/src/pages/NetworkFlow.tsx`

### 2.1 Current Architecture & File Inspection
- **Path**: `c:\Users\phili\Documents\GitHub\Orbit\frontend\src\pages\NetworkFlow.tsx`
- **Lines of Code**: 265 lines
- **Core Technology**:
  - React Flow via `@xyflow/react` v12.12.1 (`ReactFlow`, `Background`, `Controls`, `MiniMap`, `Node`, `Edge`, `MarkerType`).
  - Styling overrides in `frontend/src/index.css:58-99`.
- **Top Section (Lines 132-165)**:
  - Top banner with title ("Interaktive Netzwerktopologie") and eBPF status badge (`snapshot.ebpfStatus`).
  - Interactive React Flow canvas (`h-[460px] w-full rounded-lg overflow-hidden border border-[#282828] bg-[#121212]`).
  - Layout generation (lines 58-127):
    - Column 1 ($x=50$): Host node (`#181818`, border `#3b82f6`).
    - Column 2 ($x=300$): Docker Network nodes (`#181818`, border `#282828`, labels with subnets).
    - Column 3 ($x=620$): Container nodes (emerald border if `status === 'running'`, otherwise `#282828`).
    - Edges: Directed arrows (`MarkerType.ArrowClosed`), stroke `#3b82f6` with `animated: true` when `e.active === true` (`rxBps > 100` or `txBps > 100`).
- **Lower Section (Lines 167-260)**:
  - Lines 167-195: "Netzwerk-Protokoll-Statistik" — A grid of 6 protocol boxes displaying configured ports vs. observed connections vs. active flows.
  - Lines 197-260: "Beobachtete Verbindungen" — A basic table rendering up to 25 items (`snapshot.connections.slice(0, 25)`).

### 2.2 Preservation & Redesign Strategy for R3
- **Preserve the Top Interactive Topology Graph**:
  - The top section (banner, React Flow graph canvas, controls, minimap, background, and node-positioning math) is well implemented and must remain fully intact.
  - We can optionally enhance node interactivity (e.g. clicking a container node in the graph filters the lower socket table to that specific container).
- **Overhaul the Lower Section**:
  Replace lines 167-260 with 3 cohesive, feature-rich modules inspired by UniFi Network Traffic Inspector, Netdata, and GlassWire:

  #### Module A: Live Traffic Rate Meters & Telemetry KPIs
  - **Bandwidth Meters**: Dual live rate meters for Ingress (RX) and Egress (TX) showing current throughput formatted via `formatBps()` (e.g. `↓ 1.4 MB/s` · `↑ 320 KB/s`), dynamic rate bar/gauge, and total observed throughput.
  - **Connection KPI Pills**: Active Flows count, Listening Sockets count, External/WAN Endpoints count, eBPF / Socket telemetry mode indicator.

  #### Module B: Live Traffic Breakdown by Container & Protocol
  - **Container Traffic Breakdown**:
    - Ranked bar chart or horizontal bar meter (using Recharts or styled Tailwind progress bars) showing top containers by network bandwidth (RX/TX rate).
    - Status indicators (running dot), container short ID, and percentage share of total container network traffic.
  - **Protocol Breakdown**:
    - Donut/Pie chart (Recharts `PieChart`, `Pie`, `Cell`, `Tooltip`) or segmented horizontal bar showing traffic distribution by protocol (`HTTP`, `HTTPS`, `DNS`, `SSH`, `Database`, `Orbit`, etc.).
    - Legend showing configured vs. observed socket counts.

  #### Module C: Active Sockets & Flow Telemetry Inspector Table
  - **Full-Featured Filtering Toolbar**:
    - Global search box: Filter across container name, local/remote IP, port, and protocol.
    - Protocol selector: All, TCP, UDP, HTTPS, HTTP, Database.
    - State filter: All, ESTABLISHED, LISTEN, TIME_WAIT, CLOSE_WAIT.
    - Container filter dropdown.
    - Security alert toggle: Filter to insecure or exposed ports.
  - **Enhanced Table Columns**:
    1. **Protocol & App**: Protocol badge (`TCP`/`UDP`) + Classified App tag (`HTTP`, `HTTPS`, `DNS`, `Orbit`, `Redis`, `PostgreSQL`).
    2. **Security Status Indicator**:
       - 🛡️ *Green Shield / Lock*: Encrypted/Secure (HTTPS :443, SSH :22, TLS :8443).
       - ⚠️ *Amber Warning*: Cleartext traffic on internal/local networks (HTTP :80, Redis internal).
       - 🚨 *Red Alert*: Insecure/Sensitive port exposed on `0.0.0.0` or public interface (Cleartext Database, Telnet, Unauthenticated API).
    3. **Local Socket**: `IP:Port` (e.g. `172.17.0.2:8787`) with copy-to-clipboard button.
    4. **Remote Socket**: `IP:Port` with WAN (Public) vs. LAN (RFC1918 Private) badge.
    5. **Connection State**: Tailscale-style badge (`ESTABLISHED` emerald, `LISTEN` neutral, `TIME_WAIT` amber).
    6. **Container Attribution**: Container name with live running dot, short ID badge, and link/click to container detail.
    7. **Process Attribution**: Process name (e.g. `orbit`, `wget`, `postgres`, `alpine`) and PID where available.
    8. **Telemetry Mode**: `eBPF` or `port_based`.
  - **Pagination / Virtual List**: Eliminate the hardcoded `slice(0, 25)` limit by adding pagination controls or a scrollable container with item count badge.

---

## 3. Design System & Theme Analysis

### 3.1 Color Palette & Theme Tokens
Orbit implements a Tailscale-inspired dark theme defined in `frontend/src/index.css`:
- **Canvas / Root Background**: `#111111`
- **Cards & Surfaces**:
  - Primary Card Background: `#181818`
  - Secondary / Nested Surface: `#141414` or `#121212`
  - Elevated Popovers / Modals: `#1a1a1a` or `#202020`
- **Borders & Dividers**:
  - Primary Border: `#282828`
  - Hover / Active Border: `#383838`
  - Subtle / Inner Border: `#222222` or `#242424`
- **Typography & Font Tokens**:
  - Sans-serif: `var(--font-sans)` (`'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif`)
  - Monospace: `var(--font-mono)` (`'JetBrains Mono', ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace`)
  - Numeric values: `font-mono tabular-nums`
- **Semantic Accent Colors**:
  - **Primary / Brand Blue**: `#3b82f6` (`blue-500`), hover `#2563eb`, background tint `bg-blue-950/20`
  - **Emerald / Success / Running**: `#10b981` (`emerald-500`), text `text-emerald-400`, badge `bg-emerald-950/40 border-emerald-800/40`
  - **Amber / Warning / Paused**: `#f59e0b` (`amber-500`), text `text-amber-400`, badge `bg-amber-950/40 border-amber-800/40`
  - **Rose / Danger / Unhealthy**: `#f43f5e` (`rose-500`), text `text-rose-400`, badge `bg-rose-950/20 border-rose-800/80`
  - **Purple / Metrics / RAM**: `#a855f7` (`purple-500`)
  - **Neutral / Text**:
    - Primary White: `#ffffff` / `#f5f5f5`
    - Muted White: `text-neutral-200` (`#e5e5e5`) / `text-neutral-300` (`#d4d4d4`)
    - Subtitle / Label: `text-neutral-400` (`#a3a3a3`)
    - Micro / Inactive: `text-neutral-500` (`#737373`)
    - Deep Muted: `text-neutral-600` (`#525252`)

### 3.2 Icon Library & Existing Reusable Components
- **Icon Library**: `lucide-react` v1.69.0.
  - Networking & Telemetry: `Network`, `Globe`, `Activity`, `Wifi`, `Radio`, `ArrowDown`, `ArrowUp`, `ExternalLink`, `Layers`.
  - Security & Status: `Shield`, `ShieldCheck`, `ShieldAlert`, `Lock`, `Unlock`, `AlertCircle`, `AlertTriangle`.
  - UI Controls: `Search`, `Filter`, `SlidersHorizontal`, `RefreshCw`, `Copy`, `Check`, `Trash2`, `ChevronDown`, `ChevronUp`, `ChevronRight`.
- **Existing Common Components**:
  - `MetricCard` (`frontend/src/components/common/MetricCard.tsx`): Supports `title`, `gauge`, `formattedValue`, `unit`, `subtitle`, `icon`, `progressPercent`, and `alertThresholdExceeded`.
  - `StatusBadge` (`frontend/src/components/common/StatusBadge.tsx`): Renders running, exited/stopped, unhealthy, healthy, paused, restarting badges with a color-coded dot.
  - `ActionConfirmModal` (`frontend/src/components/modals/ActionConfirmModal.tsx`): Standard modal for destructive or confirming actions.
  - `Header` (`frontend/src/components/layout/Header.tsx`): Page title, subtitle, WebSocket status indicator, alert bell, container counter.
  - `Sidebar` (`frontend/src/components/layout/Sidebar.tsx`): Navigation sidebar.
- **Utility Functions (`frontend/src/lib/utils.ts`)**:
  - `cn(...inputs)`: Tailwind class merging via `clsx` and `tailwind-merge`.
  - `formatBytes(bytes, decimals)`: Converts raw bytes to `B`, `KB`, `MB`, `GB`, `TB`.
  - `formatBps(bytesPerSec)`: Formats bandwidth rate as `X/s`.
  - `formatPercent(value, decimals)`: Formats percentages.
  - `formatUptime(seconds)`: Formats seconds into `Xd Xh Xm Xs`.
  - `formatDate(dateString)`: Formats ISO timestamps.

---

## 4. Visualization & Charting Libraries

### 4.1 Recharts (v3.10.1 Installed)
- Fully supported and active in `Dashboard.tsx` and `ContainerDetail.tsx`.
- Available components:
  - `ResponsiveContainer`, `AreaChart`, `Area`, `BarChart`, `Bar`, `PieChart`, `Pie`, `Cell`, `XAxis`, `YAxis`, `Tooltip`, `Legend`.
- Tailscale Dark Theme configuration pattern:
  ```tsx
  <Tooltip
    contentStyle={{
      backgroundColor: '#141414',
      borderColor: '#282828',
      borderRadius: '0.375rem',
      fontSize: '11px',
      color: '#f5f5f5',
    }}
  />
  ```
- **Application in R3**:
  - Container Bandwidth Bar Chart: Horizontal `BarChart` or ranked bars comparing container RX and TX rates.
  - Protocol Breakdown: Donut `PieChart` with color-coded slices for HTTP (`#3b82f6`), HTTPS (`#10b981`), DNS (`#f59e0b`), SSH (`#8b5cf6`), Orbit (`#ec4899`), Databases (`#06b6d4`).

### 4.2 @xyflow/react (v12.12.1 Installed)
- Powers the interactive network topology graph in `NetworkFlow.tsx`.
- Dark theme controls and minimap CSS overrides already defined in `frontend/src/index.css:58-99`.
- Top interactive graph retains full functionality, zoom, pan, minimap, and animated edges.

### 4.3 Custom SVG & HTML5 Visualization
- Orbit uses sleek custom SVG components for:
  - Mini rate sparklines
  - UniFi-style horizontal IP pool allocation bars
  - Status indicators and micro-gauges

---

## 5. Frontend API, State Management, Types & Real-time

### 5.1 API Client (`frontend/src/lib/api.ts`)
- Features CSRF token injection on mutating HTTP methods (`POST`, `PUT`, `DELETE`, `PATCH`).
- Relevant existing endpoints:
  - `api.getNetworks()` $\to$ `GET /api/v1/networks` (returns `DockerNetwork[]`)
  - `api.getNetwork(id)` $\to$ `GET /api/v1/networks/{id}` (returns complete inspect object)
  - `api.removeNetwork(id)` $\to$ `DELETE /api/v1/networks/{id}`
  - `api.getNetflow()` $\to$ `GET /api/v1/netflow` (returns `NetflowSnapshot`)
  - `api.getContainers()` $\to$ `GET /api/v1/containers` (returns `Container[]`)
  - `api.getContainer(id)` $\to$ `GET /api/v1/containers/{id}`
  - `api.getContainerTop(id)` $\to$ `GET /api/v1/containers/{id}/top`

### 5.2 Real-time WebSocket Client (`frontend/src/lib/ws.ts`)
- Connects to `/api/v1/ws` with exponential backoff auto-reconnect.
- Topic subscriptions:
  - `wsClient.subscribe('netflow', (snapshot: NetflowSnapshot) => ...)`: Real-time netflow streaming!
  - `wsClient.subscribe('containers', (stats: ContainerStats) => ...)`: Real-time container metrics.
  - `wsClient.subscribe('host', (metrics: HostMetrics) => ...)`: Real-time host metrics.

### 5.3 TypeScript Interfaces (`frontend/src/types/index.ts`)
#### Current `DockerNetwork` Interface (Lines 371-389):
```ts
export interface DockerNetwork {
  Name: string;
  Id: string;
  Created: string;
  Scope: string;
  Driver: string;
  EnableIPv4: boolean;
  EnableIPv6: boolean;
  Internal: boolean;
  IPAM: {
    Driver: string;
    Config?: Array<{
      Subnet?: string;
      Gateway?: string;
    }>;
  };
  Containers?: Record<string, any>;
}
```

#### Proposed Enhanced `DockerNetwork` & IPAM Types:
```ts
export interface NetworkContainerEndpoint {
  Name: string;
  EndpointID: string;
  MacAddress: string;
  IPv4Address: string;
  IPv6Address: string;
}

export interface IPAMConfig {
  Subnet?: string;
  Gateway?: string;
  IPRange?: string;
  AuxiliaryAddresses?: Record<string, string>;
}

export interface IPAMStatus {
  Subnets?: Record<
    string,
    {
      IPsInUse: number;
      DynamicIPsAvailable: number;
    }
  >;
}

export interface DockerNetwork {
  Name: string;
  Id: string;
  Created: string;
  Scope: string;
  Driver: string;
  EnableIPv4: boolean;
  EnableIPv6: boolean;
  Internal: boolean;
  Attachable?: boolean;
  Ingress?: boolean;
  ConfigOnly?: boolean;
  IPAM: {
    Driver: string;
    Options?: Record<string, string> | null;
    Config?: IPAMConfig[];
  };
  Containers?: Record<string, NetworkContainerEndpoint>;
  Options?: Record<string, string>;
  Labels?: Record<string, string>;
  Status?: {
    IPAM?: IPAMStatus;
  };
}
```

#### Proposed Enhanced `SocketConnection` Interface (Lines 255-267):
```ts
export interface SocketConnection {
  protocol: string;
  localIp: string;
  localPort: number;
  remoteIp: string;
  remotePort: number;
  state: string;
  containerId?: string;
  containerName?: string;
  processName?: string;
  pid?: number;
  appProtocol?: string;
  classification: string;
  detectionMode: string;
  rxBps?: number;
  txBps?: number;
  securityStatus?: 'secure' | 'insecure' | 'warning' | 'info';
  securityNotice?: string;
}
```

---

## 6. Build Setup & Verification Commands

### 6.1 Package Scripts & Configuration
- `frontend/package.json`:
  - `"build": "tsc && vite build"`
  - `"dev": "vite"`
  - `"preview": "vite preview"`
- `frontend/tsconfig.json`:
  - Target: `ES2022`, Module: `ESNext`, ModuleResolution: `bundler`, JSX: `react-jsx`.
  - Non-strict compilation (`"strict": false`).

### 6.2 Pre-Existing Build Failure Uncovered
When running `npm run build` in `frontend/`, TypeScript compilation currently terminates with errors:
```
src/pages/Dashboard.tsx(424,14): error TS1005: ')' expected.
src/pages/Dashboard.tsx(424,87): error TS1382: Unexpected token. Did you mean `{'>'}` or `&gt;`?
src/pages/Dashboard.tsx(464,9): error TS1005: ')' expected.
...
```
- **Root Cause**: In `frontend/src/pages/Dashboard.tsx` lines 422-506, `{prefs.showCharts && (` contains two sibling `<div>` elements and curly-brace comments `{/* ... */}` directly inside parentheses without an enclosing fragment `<>...</>` or container `<div>...</div>`.
- **Fix Recommendation for Implementer**: Wrap the contents inside `{prefs.showCharts && (<div className="space-y-4"> ... </div>)}` or `<> ... </>`.

---

## 7. Concrete UI Architecture Recommendations

### 7.1 Architecture for R2: Networks Page (`frontend/src/pages/Networks.tsx`)
1. **IPAM Calculation Utility (`frontend/src/lib/ipam.ts`)**:
   - Create a dedicated CIDR helper:
     - `parseCIDR(cidr: string)`: Extracts IP, prefix length, calculates total usable hosts ($2^{32 - \text{prefix}} - 2$ for IPv4), network address, and broadcast address.
     - `calculateUtilization(total: number, used: number)`: Percentage calculation with safety bounds.
2. **Layout Structure**:
   - **Header & Summary Stats Bar**:
     - Metric cards: Total Networks, Total Connected Endpoints, Managed Subnets, Custom Bridge Networks.
   - **Search & Filter Bar**:
     - Search input (filter by network name, CIDR, container name, IP).
     - Driver filter pills (All, Bridge, Overlay, Macvlan, Host, None).
   - **Master-Detail / Expandable Network Cards**:
     - Collapsed Card View: Network name, driver badge, scope, internal badge, primary subnet CIDR, gateway, connected container count pill, and IP pool allocation bar.
     - Expanded / Detailed View (or Drawer):
       - **IPAM Deep Dive**: Full subnet CIDR, Gateway, IPRange, Usable IP range (Start $\to$ End), IP Allocation Bar with allocated count vs. available count and percentage badge.
       - **Connected Containers Table**: Container status badge, container name (clickable to ContainerDetail), IPv4 address, IPv6 address, MAC address, and Endpoint ID.
       - **Driver Options & Metadata**: Key-value grid of `net.Options` (ICC, masquerading, MTU, etc.) and `net.Labels` (Compose project, version).
       - **Network Actions**: Delete button for custom networks (with confirmation modal).

### 7.2 Architecture for R3: Network Flows Lower Section (`frontend/src/pages/NetworkFlow.tsx`)
1. **Retain Top Interactive Graph**:
   - Leave `ReactFlow` canvas, `@xyflow/react` node/edge generators, and automatic layout intact.
2. **Lower Section Component Breakdown**:
   - **Section 1: Live Rate Meters & Telemetry KPIs**:
     - Ingress / Egress throughput rate gauges (`formatBps(totalRxBps)`, `formatBps(totalTxBps)`).
     - Active socket count, listening ports count, remote public endpoints count.
   - **Section 2: Traffic Breakdown (Container & Protocol)**:
     - Left: Container Traffic Ranking bar chart (top containers by network bandwidth).
     - Right: Protocol Distribution Donut / Segmented Bar (`Recharts` `PieChart` or segmented progress bar).
   - **Section 3: Traffic Inspector Table**:
     - Filter toolbar (search query, protocol pills, state pills, security alert toggle).
     - Enhanced socket table with:
       - Protocol & App classification.
       - Port security status indicator (green lock, amber caution, red warning for cleartext exposed ports).
       - Local IP:Port with copy button.
       - Remote IP:Port with LAN vs. WAN badge.
       - Connection state badge (`ESTABLISHED`, `LISTEN`, `TIME_WAIT`).
       - Container attribution with status dot.
       - Process attribution and PID (when available).
     - Pagination controls (`Showing 1-25 of N connections`).

---

## Conclusion
Orbit's frontend codebase possesses the necessary dependencies (`recharts`, `@xyflow/react`, `lucide-react`, Tailwind v4) and clean styling tokens to support both the UniFi-inspired IPAM overhaul in `Networks.tsx` and the live telemetry inspector in `NetworkFlow.tsx`. The Go backend's `/api/v1/networks/{id}` and `/api/v1/netflow` endpoints already provide rich inspect data that the frontend can immediately leverage.
