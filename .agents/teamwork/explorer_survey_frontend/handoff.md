# Handoff Report: Frontend UI Architecture & Survey

**Agent**: Read-Only Frontend UI Explorer  
**Task**: Investigate Orbit's React / TypeScript frontend regarding `Networks.tsx`, `NetworkFlow.tsx`, existing styling, design tokens, components, data hooks, and build setup.  
**Destination**: `4572cb91-25c6-48e4-bf11-6d7c5a635dd0` (Parent Orchestrator)  
**Survey Report Path**: `c:\Users\phili\Documents\GitHub\Orbit\.agents\teamwork\explorer_survey_frontend\survey_frontend.md`

---

## 1. Observation

1. **`frontend/package.json`**:
   - Dependencies include:
     - React 19: `"react": "^19.3.1"`, `"react-dom": "^19.3.1"`
     - Flow Graph: `"@xyflow/react": "^12.12.1"`
     - Charts: `"recharts": "^3.10.1"`
     - Icons: `"lucide-react": "^1.69.0"`
     - CSS: `"tailwindcss": "^4.3.4"`, `"@tailwindcss/vite": "^4.3.4"`
     - Utilities: `"clsx": "^2.1.1"`, `"tailwind-merge": "^3.7.0"`, `"zustand": "^5.0.16"`
   - Scripts: `"build": "tsc && vite build"`, `"dev": "vite"`, `"preview": "vite preview"`.

2. **`frontend/src/pages/Networks.tsx`**:
   - Total lines: 123.
   - Line 14 declares `const [selectedNetwork, setSelectedNetwork] = useState<DockerNetwork | null>(null);` but `selectedNetwork` is never read or used anywhere in the file.
   - Lines 56-107 render a simple 3-column card grid (`grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4`).
   - Line 58 only inspects the first item: `net.IPAM?.Config?.[0]?.Subnet || '—'`.
   - Line 59 only inspects the first item: `net.IPAM?.Config?.[0]?.Gateway || '—'`.
   - Connected containers (`net.Containers`) are **100% absent** from rendering.
   - Driver options (`net.Options`), Labels, IPv6 flags, and allocation pools are completely omitted.

3. **`frontend/src/pages/NetworkFlow.tsx`**:
   - Total lines: 265.
   - Lines 147-165 contain the interactive `@xyflow/react` topology graph in an `h-[460px]` canvas, using automatic 3-column node calculation (Host $x=50$, Networks $x=300$, Containers $x=620$) and directed animated edges.
   - Lines 167-195 display a static 6-card grid of protocol summaries (`snapshot.protocolStats.map(...)`).
   - Lines 197-260 display a hardcoded 25-row slice (`snapshot.connections.slice(0, 25).map(...)`) with no search, filters, pagination, bandwidth meters, process attribution, or port security indicators.

4. **Live Docker & API Backend Testing**:
   - Running container `orbit-live` is active on port 8787 (`0.0.0.0:8787->8787/tcp`).
   - `api.getNetwork(id)` is already implemented in `frontend/src/lib/api.ts:124`.
   - Testing `GET /api/v1/networks/5edf60f8cc45e80442d3ee8cc586c119b01d076861c22d65a669e8df25f80684` returned full Docker inspect JSON containing:
     - `Containers` map with `{ Name, EndpointID, MacAddress, IPv4Address, IPv6Address }` for all connected containers (e.g. `test-orbit-cpu-load`, `test-orbit-traffic`).
     - `Options` map with `com.docker.network.enable_ipv4`, `enable_ipv6`, MTU, etc.
     - `Labels` map with compose project and hash.
     - `Status.IPAM.Subnets["172.18.0.0/16"]` containing `IPsInUse: 8` and `DynamicIPsAvailable: 65528`.
   - Testing `GET /api/v1/netflow` returned live topology nodes (with active container traffic e.g. `rxBps: 1313.37` on `test-orbit-traffic`), animated edges, active socket connections, and protocol summaries.

5. **Build Verification (`npm run build`)**:
   - Command: `cd frontend; npm run build`
   - Exit code: 1.
   - Verbatim error:
     ```
     src/pages/Dashboard.tsx(424,14): error TS1005: ')' expected.
     src/pages/Dashboard.tsx(424,87): error TS1382: Unexpected token. Did you mean `{'>'}` or `&gt;`?
     src/pages/Dashboard.tsx(464,9): error TS1005: ')' expected.
     src/pages/Dashboard.tsx(505,7): error TS1128: Declaration or statement expected.
     src/pages/Dashboard.tsx(508,12): error TS1005: ';' expected.
     src/pages/Dashboard.tsx(509,9): error TS2657: JSX expressions must have one parent element.
     ```
   - Inspection of `frontend/src/pages/Dashboard.tsx` lines 422-506 reveals that `{prefs.showCharts && (` wraps multiple sibling divs and JSX comments `{/* ... */}` directly inside parentheses without an enclosing fragment `<>...</>` or container `<div>...</div>`.

---

## 2. Logic Chain

1. **R2 Scope**: The user requires redesigning `frontend/src/pages/Networks.tsx` to provide clear IPAM subnet inspection (CIDR, gateway, IP allocation pools), connected container assignments with status badges, and network driver options/insights within Orbit's dark theme.
2. **Current State**: `Networks.tsx` has only 123 lines, lacks container rendering, lacks CIDR calculation, and displays minimal driver info.
3. **Data Availability**: The backend already exposes `GET /api/v1/networks/{id}` which contains complete container endpoints, options, labels, and IPAM allocation data. Furthermore, `frontend/src/types/index.ts` can be enriched with `NetworkContainerEndpoint` and `IPAMStatus` without breaking any existing interfaces.
4. **R3 Scope**: The user requires retaining the interactive topology graph in `frontend/src/pages/NetworkFlow.tsx` while overhauling the lower section to provide a live traffic breakdown by container and protocol, bandwidth rate meters, an active socket connection table with container/process attribution, and port security indicators.
5. **Component Separation**: In `NetworkFlow.tsx`, the top graph (lines 132-165) is cleanly decoupled from the lower section (lines 167-260). The top graph uses `@xyflow/react` and should be preserved in its entirety. The lower section can be replaced by three modular sections:
   - Bandwidth meters and live telemetry KPIs (using existing `formatBps` and rate gauges).
   - Container traffic ranking (using `recharts` BarChart or styled progress bars) and protocol breakdown (using `recharts` PieChart).
   - Active sockets table with search, protocol/state filter pills, port security indicators (lock/warning badges based on port security analysis), container badges, and pagination.
6. **Theme & Token Consistency**: Both redesigned views will leverage the existing Tailscale tokens (`#111111` canvas, `#181818` card surfaces, `#282828` borders), typography (`Inter` / `JetBrains Mono`), and `lucide-react` icons.
7. **Prerequisite Fix**: Before `npm run build` will succeed cleanly in the verification phase, the syntax error in `Dashboard.tsx:422-506` must be fixed (or wrapped in `<>...</>`).

---

## 3. Caveats

- **Backend Endpoints for List vs. Inspect**: While `GET /api/v1/networks/{id}` provides complete `Containers` and `Status` data, the list endpoint `GET /api/v1/networks` returns `network.Summary` which in standard Docker Engine does not always populate `Containers`. In the frontend, `Networks.tsx` should either fetch individual network details via `api.getNetwork(id)` on demand / expansion, or the backend explorer/implementer can enrich the list handler.
- **Process Attribution on Linux**: Process names in `SocketConnection` require `/proc` namespace visibility. In environments where container PID mapping or host `/proc` is restricted, process attribution will gracefully display the container name or `—`.

---

## 4. Conclusion

1. Orbit's frontend infrastructure is robust and well-equipped: React 19, Vite 8, Tailwind v4, `@xyflow/react` v12, `recharts` v3.10, and `lucide-react` are fully operational.
2. `Networks.tsx` requires a complete overhaul into a UniFi-inspired IPAM and container management interface with:
   - Summary metric cards at the top.
   - Filter and search bar.
   - Master-detail / expandable card layout.
   - CIDR / IP allocation pool visualizer with capacity and utilization percentage.
   - Connected containers table with status badges, IPs, and MAC addresses.
   - Driver options and Compose labels inspection.
3. `NetworkFlow.tsx` top topology graph can be retained without modification, while lines 167-260 should be replaced with live bandwidth rate meters, a container/protocol traffic breakdown (Recharts), and a security-aware active sockets table.
4. Detailed findings, code locations, type definitions, and architecture specifications have been written to `survey_frontend.md`.

---

## 5. Verification Method

To independently verify the frontend survey findings:

1. **Verify Package Dependencies**:
   ```powershell
   Get-Content frontend/package.json | ConvertFrom-Json | Select-Object -ExpandProperty dependencies
   ```
   Confirm `recharts`, `@xyflow/react`, and `lucide-react` are present.

2. **Verify Network Inspect API Payload**:
   ```powershell
   $session = New-Object Microsoft.PowerShell.Commands.WebRequestSession
   $login = Invoke-RestMethod -Uri "http://127.0.0.1:8787/api/v1/auth/login" -Method Post -Body (@{ username = "admin"; password = "OrbitAdminSecret2026!" } | ConvertTo-Json) -ContentType "application/json" -WebSession $session
   $net = Invoke-RestMethod -Uri "http://127.0.0.1:8787/api/v1/networks/5edf60f8cc45e80442d3ee8cc586c119b01d076861c22d65a669e8df25f80684" -WebSession $session
   $net.Containers | ConvertTo-Json
   ```
   Confirm `Containers` contains connected containers with `IPv4Address` and `MacAddress`.

3. **Verify Existing Build Failure**:
   ```powershell
   cd frontend; npm run build
   ```
   Confirm TypeScript flags the JSX error in `src/pages/Dashboard.tsx:424`.

4. **Review Survey Report**:
   Inspect `c:\Users\phili\Documents\GitHub\Orbit\.agents\teamwork\explorer_survey_frontend\survey_frontend.md`.
