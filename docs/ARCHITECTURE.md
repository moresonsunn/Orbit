# Orbit – Systemarchitektur & Datenquellen

Orbit ist als Single-Container-Anwendung mit eingebetteter SQLite-Datenbank konzipiert. Es werden weder ein externer Datenbankserver (wie PostgreSQL) noch ein Cache-Dienst (wie Redis) benötigt.

---

## 1. Komponenten-Diagramm

```
+--------------------------------------------------------------------------+
| Browser / Web Client (React 19 + TypeScript + Vite + Tailwind + Flow)    |
+--------------------------------------------------------------------------+
       |                                              ^
  REST API /api/v1                               WebSocket /api/v1/ws
  (Session Cookie + CSRF)                     (Topics: host, containers,
       |                                       gpu, netflow, events, alerts)
       v                                              |
+--------------------------------------------------------------------------+
| Go Backend Binary (cmd/orbit)                                            |
|                                                                          |
|  +--------------------------------------------------------------------+  |
|  | Chi Router + Security Middleware (Argon2id, CSRF, RBAC, Rate-Limit)|  |
|  +--------------------------------------------------------------------+  |
|                                                                          |
|  +-------------------+  +--------------------+  +---------------------+  |
|  | Realtime Hub      |  | Alert Engine       |  | Metrics History     |  |
|  | (Backpressure WS) |  | (Threshold Rules)  |  | (Rollups 5m / 1h)   |  |
|  +-------------------+  +--------------------+  +---------------------+  |
|                                                                          |
|  +-------------------+  +--------------------+  +---------------------+  |
|  | Docker Gateway    |  | Host Collector     |  | Netflow Builder     |  |
|  | (Strict Allowlist)|  | (/proc & /sys)     |  | (Topology & Sockets)|  |
|  +-------------------+  +--------------------+  +---------------------+  |
|           |                       |                        |             |
|  +-------------------+  +--------------------+             |             |
|  | Docker Inventory  |  | GPU Manager        |             |             |
|  | (Event-driven)    |  | (NVML / DRM)       |             |             |
|  +-------------------+  +--------------------+             |             |
+--------------------------------------------------------------------------+
       |                       |                             |
 /var/run/docker.sock     /host/proc & /host/sys      Socket-Tabellen
 (Docker Engine API)      (Linux Host-Metriken)       & eBPF-Option
```

---

## 2. Metriken und Berechnungslogik

### A. Container CPU-Auslastung (`calculateCPUPercentUnix`)

Die Berechnung entspricht exakt der offiziellen Docker CLI (`docker stats`):

$$\Delta \text{CPU} = \text{TotalUsage}_{t} - \text{TotalUsage}_{t-1}$$
$$\Delta \text{System} = \text{SystemUsage}_{t} - \text{SystemUsage}_{t-1}$$
$$\text{CPU \%} = \frac{\Delta \text{CPU}}{\Delta \text{System}} \times \text{OnlineCPUs} \times 100$$

- **100 %** bedeutet: Ein CPU-Kern ist voll ausgelastet.
- Bei einem 4-Kern-System liegt das Maximum bei 400 %.
- Zusätzlich berechnet Orbit `cpuPercentOfLimit` (normalisiert auf das konfigurierte CPU-Limit des Containers).

### B. Container RAM-Verbrauch (`calculateMemUsageUnixNoCache`)

Orbit zieht – wie Docker CLI – den inaktiven Dateicache (`inactive_file`) vom gemeldeten Speicherverbrauch ab:

- **cgroup v2**: `MemoryUsage = Usage - inactive_file`
- **cgroup v1**: `MemoryUsage = Usage - total_inactive_file`

Dadurch wird vermieden, dass unverfänglicher Puffer-Cache als tatsächlicher Speicherverbrauch dargestellt wird.

### C. Host-System-Metriken

Die Host-Auslastung wird **niemals** durch Aufsummieren der Container-Metriken geschätzt, sondern direkt und unabhängig erfasst:
- **CPU**: `/host/proc/stat` (Deltas aus user, nice, system, idle, iowait)
- **RAM**: `/host/proc/meminfo` (`MemAvailable` vs. `MemTotal`)
- **Disk I/O**: `/host/proc/diskstats` (Sektor-Deltas mit 512 Byte Multiplikator)
- **Netzwerk**: `/host/proc/net/dev`
- **Dateisysteme**: `statfs` auf gemounteten Host-Partitionen

### D. Status-Kennzeichnung aller Messwerte

Jeder Messwert besitzt einen expliziten Status:
- `ok`: Gültiger Messwert liegt vor.
- `pending`: Erster Messwert erfasst, zweiter Messwert zur Differenzberechnung wird erwartet.
- `unavailable`: Messwert vom Kernel oder Treiber nicht gemeldet.
- `unsupported`: Hardware oder cgroup-Konfiguration unterstützt dieses Feature nicht.
- `no_permission`: Zugriff verweigert.

Das Frontend stellt niemals `0 %` dar, wenn ein Wert in Wahrheit nicht erfasst werden konnte.
