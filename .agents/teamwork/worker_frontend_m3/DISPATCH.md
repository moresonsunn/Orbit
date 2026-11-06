## 2026-11-06T00:03:19Z

You are the Frontend Implementation Worker for Milestone M3: Network Flows Lower Section Redesign (`frontend/src/pages/NetworkFlow.tsx`).

Your working directory is: c:\Users\phili\Documents\GitHub\Orbit\.agents\teamwork\worker_frontend_m3\

You MUST read:
1. c:\Users\phili\Documents\GitHub\Orbit\.agents\teamwork\ORIGINAL_REQUEST.md
2. c:\Users\phili\Documents\GitHub\Orbit\PROJECT.md
3. c:\Users\phili\Documents\GitHub\Orbit\.agents\teamwork\explorer_survey_frontend\survey_frontend.md
4. c:\Users\phili\Documents\GitHub\Orbit\.agents\teamwork\explorer_survey_research\survey_research.md
5. c:\Users\phili\Documents\GitHub\Orbit\.agents\teamwork\worker_backend_m1\handoff.md
6. c:\Users\phili\Documents\GitHub\Orbit\.agents\teamwork\worker_frontend_m2\handoff.md

MANDATORY INTEGRITY WARNING:
DO NOT CHEAT. All implementations must be genuine. DO NOT hardcode test results, create dummy/facade implementations, or circumvent the intended task. A teamwork_auditor will independently verify your work. Integrity violations WILL be detected and your work WILL be rejected.

Exclusive Write Ownership:
- `frontend/src/pages/NetworkFlow.tsx`

Implementation Requirements:
1. RETAIN TOP TOPOLOGY GRAPH INTACT:
   - The interactive React Flow topology graph canvas (`@xyflow/react`) at the top of the page MUST remain intact and fully functional (nodes for host, networks, containers; animated directed edges; traffic throughput indicators).
2. REDESIGN THE LOWER SECTION (inspired by UniFi Network Traffic Inspector, Tailscale logs, Netdata, and GlassWire):
   - Adhere strictly to Orbit's Tailscale-inspired dark console theme (`#111111` canvas, `#181818` card surfaces, `#282828` borders), typography, and `lucide-react` icons.
   - Live Bandwidth Rate Meters & KPI Cards:
     - Total Ingress rate (RX kB/s or MB/s formatted with `formatBps`)
     - Total Egress rate (TX kB/s or MB/s formatted with `formatBps`)
     - Active Socket Connections count
     - Port Security Alerts counter (flagging critical & caution items)
   - Live Traffic Breakdown by Container & Protocol:
     - Container Bandwidth Ranking: Horizontal bars or Recharts bar chart showing top bandwidth-consuming containers, container status badge, and RX / TX throughput.
     - Application Protocol Breakdown: Donut or PieChart (using installed `recharts` v3.10) or card grid showing traffic and flow distribution across protocols (Web/HTTPS, Databases/Postgres/Redis, System/DNS/SSH, etc.).
   - Active Socket Connections Table:
     - Search & Filter bar: Search by container name, process, port, or IP; protocol filter pills (All, TCP, UDP); state filter pills (All, ESTABLISHED, LISTEN, etc.); security level filter (All, Critical, Caution, Safe, Internal).
     - Columns:
       - Port Security Indicator: Badge with icon & color:
         - Critical (Rose badge + AlertTriangle icon: "Critical Risk: Unencrypted public service")
         - Caution (Amber badge + AlertCircle icon: "Caution: Unencrypted plain HTTP / public bind")
         - Safe (Emerald badge + ShieldCheck icon: "Safe: Encrypted or localhost bind")
         - Internal (Blue/Cyan badge + Lock icon: "Internal: Private container network")
       - Container Attribution: Container name badge with link/icon and process name (e.g. `redis-server`, `nginx`, `postgres`).
       - Local Address & Port (in `JetBrains Mono` with copy button).
       - Remote Address & Port (in `JetBrains Mono`).
       - Protocol (TCP / UDP).
       - Connection State (ESTABLISHED, LISTEN, TIME_WAIT with color-coded status pills).
       - Detected Protocol / Application Classification.
     - Pagination controls (10/25/50 per page) with total count indicator.
     - Clean empty state banner if no sockets match the search/filter criteria.
3. Verification:
   - Typecheck `NetworkFlow.tsx`:
     `cd frontend; npx tsc --noEmit --ignoreConfig --target ES2022 --moduleResolution bundler --jsx react-jsx src/pages/NetworkFlow.tsx src/types/index.ts src/lib/api.ts`
   - Zero runtime errors on empty/null arrays or missing fields.
4. Report & Deliverable:
   - Write your handoff report to `c:\Users\phili\Documents\GitHub\Orbit\.agents\teamwork\worker_frontend_m3\handoff.md`.
   - Send completion message to parent.
