# Handoff Report: Network UX & Telemetry Survey Research

**Author:** Read-Only Network UX Researcher  
**Target:** Parent Orchestrator (`4572cb91-25c6-48e4-bf11-6d7c5a635dd0`)  
**Date:** 2026-11-06  
**Artifact:** `c:\Users\phili\Documents\GitHub\Orbit\.agents\teamwork\explorer_survey_research\survey_research.md`

---

## 1. Observation

1. **Current Networks Page Implementation (`frontend/src/pages/Networks.tsx:56-107`)**:
   - Lines 57-60: Extracts only basic subnet and gateway strings:
     ```tsx
     const subnet = net.IPAM?.Config?.[0]?.Subnet || '—';
     const gateway = net.IPAM?.Config?.[0]?.Gateway || '—';
     ```
   - Lines 87-104: Renders 4 simple text lines in a mono font: `Subnetz`, `Gateway`, `Scope`, and `Intern`.
   - Lacks CIDR network/broadcast breakdown, usable host range calculation, IP pool allocation range, IP utilization progress bar, connected container table, MAC address attribution, and network driver security options (MTU, ICC, internal isolation badges).

2. **Current Network Flows Implementation (`frontend/src/pages/NetworkFlow.tsx:147-260`)**:
   - Lines 147-165: Renders a React Flow canvas with host, network, and container nodes.
   - Lines 167-195: Renders a static grid of 6 protocol boxes displaying `Configured`, `Observed`, and `Active` counts.
   - Lines 198-259: Renders a 7-column table (`Protokoll`, `Lokale Adresse:Port`, `Remote Adresse:Port`, `Status`, `Container`, `Erkanntes Protokoll`, `Klassifikation`) without live bandwidth rate meters, traffic volume meters, process command attribution, or port security risk badges.

3. **Existing Backend Capabilities (`backend/internal/`)**:
   - `backend/internal/netflow/builder.go:93-143`: Already maps container network attachments and collects container stats (`NetRxBps`, `NetTxBps`).
   - `backend/internal/netflow/socket_reader.go:44-75`: Scans `/proc/[pid]/net/tcp`, `tcp6`, `udp`, `udp6` for container PIDs and host network namespace.
   - `backend/internal/docker/inventory.go:250-260`: `ByIP()` maps every container IP to its container entity with full details (`c.Networks` holds `IPAddress`, `Gateway`, `MacAddress`, `EndpointID`, `PrefixLen`).
   - `backend/internal/docker/gateway.go:268-283`: `ListNetworks` and `InspectNetwork` fetch Docker networks including options, IPAM configuration, and endpoint resources.

4. **Orbit Dark Theme Tokens (`frontend/src/index.css:33-40` & `MetricCard.tsx:36-42`)**:
   - Body canvas: `#111111`
   - Card surface: `#181818` with border `#282828`
   - Typography: Font Sans Inter (`--font-sans`), Font Mono JetBrains Mono (`--font-mono`)
   - Accent classes: Tailwind emerald, amber, rose, blue with `/40` translucent backgrounds.

---

## 2. Logic Chain

1. **Subnet & IPAM UX Gap**:
   - *Observation 1* shows that `Networks.tsx` displays only raw subnet strings without computing total addresses, usable ranges, or utilization percentages.
   - *Observation 3* proves the backend already has all container network assignments (`c.Networks`) and Docker IPAM configs in memory.
   - *Inference*: By adding standard CIDR bit calculations in the backend or frontend, Orbit can calculate Total IPs ($2^{32-\text{prefix}} - 2$), Used IPs (containers + gateway), Free IPs, and Utilization %, rendering a UniFi-style visual IP utilization bar and connected container client table with MAC addresses and live container throughput.

2. **Network Flows Lower Section Gap**:
   - *Observation 2* shows `NetworkFlow.tsx` only lists raw socket snapshots without live traffic breakdown or security risk indicators.
   - *Inference*: UniFi Traffic Inspector, Netdata, and GlassWire patterns demonstrate that users need:
     - Real-time aggregate bandwidth meters (Rx/Tx kB/s or MB/s).
     - Top container bandwidth ranking (identifying bandwidth hogs).
     - Application protocol breakdown (HTTPS, Postgres, Redis, DNS, SSH, etc.).
     - Port security indicators flagging dangerous public exposures (e.g. unencrypted Redis or Postgres bound to `0.0.0.0` vs safe `127.0.0.1` binds).

3. **System Cohesion**:
   - *Observation 4* shows Orbit's Tailscale aesthetic uses `#111111` canvas, `#181818` surfaces, and `#282828` borders.
   - The proposed UI components and data models in `survey_research.md` cleanly follow this exact design language without introducing extraneous CSS dependencies.

---

## 3. Caveats

- **eBPF vs Proc Socket Inspection**: Socket inspection in containerized environments relies on `/proc/[pid]/net/tcp` when eBPF is disabled or unavailable. As noted in the codebase, socket attribution to processes inside non-root containers requires host PID namespace or matching `/proc`. The fallback to container ID and port classification is already robust and documented.
- **Dynamic Port Ephemeral Ranges**: Sockets with high ephemeral remote ports (>32768) represent outbound client connections; security risk classification specifically distinguishes between listening server sockets (`LISTEN`) and established outbound clients (`ESTABLISHED`).

---

## 4. Conclusion

The comprehensive research report, UI hierarchy specification, and exact data models have been synthesized and documented in:
`c:\Users\phili\Documents\GitHub\Orbit\.agents\teamwork\explorer_survey_research\survey_research.md`

Key Deliverables Specified:
1. **Networks Page Revamp**:
   - CIDR visual breakdown (Subnet mask, Network IP, Usable Range, Broadcast IP, Total capacity).
   - Gateway identification and bridge device mapping.
   - IP pool allocation range and visual utilization bar (`Used / Total IPs` with warning thresholds).
   - Container client table (Container Name, StatusBadge, IPv4/IPv6, MAC address, live Rx/Tx throughput).
   - Driver options badges (Bridge, Overlay, Macvlan, Host, Internal isolation, ICC, MTU, NAT).
2. **Network Flows Lower Section Revamp**:
   - Aggregated bandwidth rate meters (Total Rx/Tx MB/s, Active flows, Security alerts).
   - Container traffic ranking bar (top bandwidth consumers).
   - Protocol categorization (Web, Databases, System, Remote).
   - Active sockets table with automated security risk badges (Critical, Caution, Safe, Internal), state pills, and monospace endpoints.
3. **Formalized Data Contracts**:
   - Complete TypeScript interfaces (`EnrichedDockerNetwork`, `IPAMSubnetConfig`, `NetworkContainerMember`, `EnrichedSocketConnection`, `EnrichedNetflowSnapshot`).
   - Complete Go struct specifications (`EnrichedNetwork`, `SubnetIPAMInfo`, `NetworkContainerMember`, `EnrichedSocketConnection`, `EnrichedNetflowSnapshot`).

---

## 5. Verification Method

1. **Document Verification**:
   - Inspect `survey_research.md` to verify all required sections from `ORIGINAL_REQUEST.md` (IPAM inspection, client table, traffic inspector, active sockets table, port security indicators, and Tailscale theme tokens) are fully defined.
2. **Project Code & Build Commands (for downstream implementers)**:
   - Go backend verification:
     ```powershell
     cd c:\Users\phili\Documents\GitHub\Orbit\backend
     go test ./...
     go build ./...
     ```
   - Frontend build verification:
     ```powershell
     cd c:\Users\phili\Documents\GitHub\Orbit\frontend
     npm run build
     ```
   - Live Docker verification:
     - Navigate to `http://localhost:8787/networks` and `http://localhost:8787/network-flow` to verify visual rendering and absence of console errors.
