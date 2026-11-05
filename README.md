# Orbit – Self-hosted Docker Infrastructure, Management & Real-Time Monitoring Platform

Orbit ist eine moderne, schlanke und eigenständige Docker-Infrastruktur- und Echtzeitüberwachungs-Plattform, konzipiert für CasaOS- und Linux-Server.

Orbit läuft als einzelner Container direkt auf dem Host, verbindet sich mit dem Docker Engine Daemon (`/var/run/docker.sock`) und visualisiert Container, Netzwerktopologien, Socket-Verbindungen, CPU-, RAM-, Disk- und GPU-Telemetrie in einer responsiven High-End-Weboberfläche.

---

## Highlights

- **Vollständig eigenständig**: Ein einzelnes Go-Binary mit eingebetteter SQLite-Datenbank und eingebetteter React 19 SPA. Keine externen Datenbanken (PostgreSQL/Redis) erforderlich.
- **Automatische Erkennung**: Erkennt alle existierenden Container (CasaOS Apps, Docker Compose Stacks, CLI-Container) automatisch, ohne sie zu verändern oder neu zu starten.
- **Interaktiver Network Flow**: React-Flow-Netzwerktopologie (`Host -> Docker Networks -> Containers`) mit animierten Verkehrsintensitäten, Socket-Snapshot-Tabellen und Port-basierter Protokollklassifikation.
- **Echte Metriken**: Exakte Übereinstimmung mit `docker stats` Berechnungsformeln (CFS-Deltas, cgroup v1/v2 Inactive-File Subtraktion). Unabhängige Erfassung der Host-Auslastung über `/proc` und `/sys`.
- **Keine Fake-Daten**: Jede Metrik besitzt einen expliziten Status (`ok`, `pending`, `unavailable`, `unsupported`, `no_permission`). Orbit rendert niemals `0 %` für nicht erfassbare Werte.
- **GPU-Telemetrie**: Automatische Erkennung von NVIDIA GPUs über NVML sowie AMD/Intel über DRM. Wenn keine GPU vorhanden ist, schaltet Orbit auf einen sauberen Hinweis ohne Fehlermeldungen um.
- **Sicherheitsarchitektur**:
  - Kein Standardpasswort: Einmaliger Setup-Token beim ersten Start im Container-Log
  - Argon2id Passwort-Hashing
  - CSRF-Schutz & Rate-Limiting
  - Docker Gateway mit Allowlist (kein ungeschützter API-Pass-Through)
  - Standardmäßig read-only (`ORBIT_ENABLE_ACTIONS=false`)

---

## Schnellstart mit Docker Compose

```yaml
services:
  orbit:
    image: ghcr.io/phili/orbit:latest
    container_name: orbit
    restart: unless-stopped
    ports:
      - "8787:8787"
    environment:
      - ORBIT_LISTEN=:8787
      - ORBIT_DATA_DIR=/data
      - ORBIT_ENABLE_ACTIONS=false       # auf true setzen für Container-Verwaltung
      - ORBIT_ENABLE_DESTRUCTIVE=false   # auf true setzen für Löschaktionen
      - ORBIT_GPU=auto
      - ORBIT_EBPF=auto
    volumes:
      - /DATA/AppData/orbit/data:/data
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - /proc:/host/proc:ro
      - /sys:/host/sys:ro
      - /etc/os-release:/host/etc/os-release:ro
    security_opt:
      - no-new-privileges:true
```

1. Starten mit `docker compose up -d`.
2. Setup-Token auslesen mit `docker logs orbit`.
3. Orbit öffnen unter `http://SERVER-IP:8787` und Admin-Konto anlegen.

---

## Dokumentation

- [Installationsanleitung CasaOS](docs/INSTALLATION.md)
- [Sicherheitskonzept & Best Practices](docs/SECURITY.md)
- [Systemarchitektur & Metriken](docs/ARCHITECTURE.md)
