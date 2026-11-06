## 2026-11-05T23:25:21Z

Revamp Orbit's Networks interface and the lower section of Network Flows, incorporating top features from UniFi Network and modern container networking tools to deliver comprehensive IPAM inspection, container topology insights, live traffic breakdowns, and socket metrics.

Working directory: c:\Users\phili\Documents\GitHub\Orbit
Integrity mode: development

## Requirements

### R1. Online Research & Implementation Plan
Investigate community feature requests and UX patterns from UniFi Network, Portainer, Dokploy, CasaOS, and Docker networking tools. Formulate a comprehensive implementation plan for Orbit's network management and flow telemetry before code modifications.

### R2. Networks Page Overhaul (`frontend/src/pages/Networks.tsx`)
Redesign the Networks page to provide clear IPAM subnet inspection (CIDR, gateway, IP allocation pools), connected container assignments with status badges, network driver options/insights, and an intuitive layout adhering to Orbit's Tailscale-inspired dark theme.

### R3. Network Flows Lower Section Redesign (`frontend/src/pages/NetworkFlow.tsx`)
Retain the interactive topology graph and redesign the lower section inspired by UniFi Network Traffic Inspector, Tailscale logs, Netdata, and GlassWire. Deliver a live traffic breakdown by container and protocol, bandwidth rate meters, an active socket connection table with container and process attribution, and port security indicators.

### R4. Backend Telemetry and Flow Endpoints
Extend Orbit's Go backend endpoints and collectors as necessary to capture and serve Docker network configurations, active socket connections, container traffic statistics, and port metadata required by the frontend views.

## Acceptance Criteria

### Planning & Research
- [ ] An implementation plan detailing UniFi-inspired features, UI layout changes, and backend telemetry endpoints is documented before making code modifications.

### Networks Page
- [ ] `frontend/src/pages/Networks.tsx` displays IPAM subnet configurations, allocated IP ranges, and connected containers with IP addresses.
- [ ] Network driver details, scope, and options are visible and inspectable for each Docker network.
- [ ] Visual design adheres to Orbit's dark console theme (`#111111` canvas, `#181818` surfaces, `#282828` borders).

### Network Flows Page
- [ ] The top interactive topology graph in `frontend/src/pages/NetworkFlow.tsx` remains intact and functional.
- [ ] The lower section features a live traffic breakdown per container and protocol with bandwidth rate meters.
- [ ] Active sockets table lists observed connections with container badges, local/remote addresses, protocols, and port security indicators.

### Verification
- [ ] Backend tests and builds pass cleanly (`go test ./...` and `go build ./...`).
- [ ] Frontend builds cleanly with no TypeScript or lint compilation errors (`npm run build`).
- [ ] The Orbit application running in Docker container `orbit-live` loads both `/networks` and `/network-flow` routes without console errors and correctly renders the new data.
