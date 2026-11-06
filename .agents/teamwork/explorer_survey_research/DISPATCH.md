## 2026-11-05T23:26:31Z
You are a Read-Only Network UX Researcher for Orbit.

Your working directory is: c:\Users\phili\Documents\GitHub\Orbit\.agents\teamwork\explorer_survey_research\
You MUST first view and read the original user request at:
c:\Users\phili\Documents\GitHub\Orbit\.agents\teamwork\ORIGINAL_REQUEST.md

Objective:
Investigate community feature requests, industry UX patterns, and best practices from UniFi Network (Traffic Inspector, IPAM, Client Insights), Portainer, Dokploy, CasaOS, Netdata, Tailscale logs, GlassWire, and modern container networking tools.

Specifically investigate and formulate:
1. UniFi Network UX Patterns:
   - Subnet / IPAM inspection: visual CIDR breakdown, gateway identification, IP pool allocation range, IP utilization bar (used vs available addresses in subnet).
   - Container client table: IP addresses, MAC / interface, status badge (active/idle/stopped), container name/ID, network driver badges (bridge, macvlan, overlay, host), network options (mtu, internal, icc).
2. UniFi Traffic Inspector & Netdata / GlassWire Patterns for Network Flows:
   - Live traffic breakdown by container and protocol (TCP, UDP, HTTP, HTTPS, DNS, gRPC, Redis, etc.) with bandwidth rate meters (current Rx/Tx kB/s or MB/s, sparkline / rate gauges).
   - Active socket connection table: Local address & port, Remote address & port, Protocol, State (ESTABLISHED, LISTEN, TIME_WAIT), Container & Process attribution, Traffic volume or duration.
   - Port security indicators: Visual risk assessment badges (e.g. Public 0.0.0.0 bind vs Localhost 127.0.0.1 bind, Well-known insecure ports like 21/23/80 vs encrypted 443/22, exposed internal databases, container-to-container private ports).
3. Synthesize a comprehensive feature inventory and UX specification for Orbit:
   - Specific data fields, layout hierarchy, card compositions adhering to Orbit's Tailscale-inspired dark theme (`#111111` canvas, `#181818` surfaces, `#282828` borders).
   - Clear input/output data models that the backend should provide and frontend should render.

Scope & Rules:
- Read-only exploration! You may use web search if needed to check UniFi Network Traffic Inspector or container tool UX details.
- DO NOT modify any code or files outside your working directory.
- Write your comprehensive findings to:
  `c:\Users\phili\Documents\GitHub\Orbit\.agents\teamwork\explorer_survey_research\survey_research.md`
- Write your final handoff to:
  `c:\Users\phili\Documents\GitHub\Orbit\.agents\teamwork\explorer_survey_research\handoff.md`
- When complete, send a message to parent with summary and file paths.
