# Orbit – Installationsanleitung / Installation Guide

Orbit ist eine moderne, schlanke Docker-Infrastruktur- und Echtzeitüberwachungs-Plattform für Homelabs, Linux-Server und CasaOS.
Orbit is an ultra-fast, lightweight Docker infrastructure and real-time monitoring platform for homelabs, Linux servers, and CasaOS / ZimaOS.

---

## 1. Systemanforderungen / System Requirements

- **Betriebssystem / OS**: CasaOS (Debian/Ubuntu/ZimaOS) oder beliebiges Linux-System mit Docker Engine (20.10+)
- **Architekturen / Architectures**: 
  - `linux/amd64` (x86_64, Intel/AMD PC & Server)
  - `linux/arm64` (Raspberry Pi 4/5, Orange Pi, Rockchip, Apple Silicon Server)
- **Ressourcenbedarf / Resource Usage**: ca. 30–50 MB RAM, < 0.5 % CPU im Leerlauf
- **Docker Socket**: `/var/run/docker.sock` (für Container- und Host-Metriken)
- **Netzwerk / Network**: Standardport `8787` (Web UI und HTTP REST / WebSocket API)

---

## 2. Installation auf CasaOS / Installation on CasaOS

Orbit bietet zwei bequeme grafische Wege zur Installation auf CasaOS sowie die klassische Installation via Terminal:

### Methode 1: 1-Click Custom Install (UI Compose Import) — *Empfohlen für Einzelinstallation*

1. Öffne dein **CasaOS Dashboard** im Browser (`http://<deine-server-ip>`).
2. Klicke auf das Symbol **App Store**.
3. Klicke oben rechts auf **„Benutzerdefinierte App installieren“** (**Custom Install**).
4. Klicke oben rechts im Fenster auf das Icon **„Importieren“** (**Import**).
5. Füge die offizielle CasaOS-Compose-Konfiguration aus `Apps/Orbit/docker-compose.yml` ein:

```yaml
name: orbit

services:
  orbit:
    image: moresonsun/orbit:latest
    container_name: orbit
    restart: unless-stopped
    ports:
      - "8787:8787"
    environment:
      ORBIT_LISTEN: ":8787"
      ORBIT_DATA_DIR: "/data"
      ORBIT_ENABLE_ACTIONS: "false"
      ORBIT_ENABLE_DESTRUCTIVE: "false"
      ORBIT_GPU: "auto"
      ORBIT_EBPF: "auto"
    volumes:
      - /DATA/AppData/$AppID/data:/data
      - /var/run/docker.sock:/var/run/docker.sock
      - /proc:/host/proc:ro
      - /sys:/host/sys:ro
      - /etc/os-release:/host/etc/os-release:ro
    security_opt:
      - no-new-privileges:true
    x-casaos:
      ports:
        - container: "8787"
          description:
            en_US: "Orbit Web UI and API Port"
      volumes:
        - container: /data
          description:
            en_US: "Orbit persistent SQLite database and application data"
        - container: /var/run/docker.sock
          description:
            en_US: "Host Docker daemon socket for container management and monitoring"
        - container: /host/proc
          description:
            en_US: "Host /proc filesystem for real-time CPU and memory metrics"
        - container: /host/sys
          description:
            en_US: "Host /sys filesystem for kernel and hardware telemetry"
        - container: /host/etc/os-release
          description:
            en_US: "Host operating system release identification"

x-casaos:
  id: dev.orbit.orbit
  main: orbit
  port_map: "8787"
  scheme: http
  index: /
  category: Developer
  architectures:
    - amd64
    - arm64
  title:
    en_US: "Orbit"
  tagline:
    en_US: "Lightweight Docker Container Management & Monitoring"
  description:
    en_US: "Orbit is an ultra-fast, self-hosted Docker infrastructure, management and real-time monitoring platform. Built with Go and modern web technologies, Orbit delivers instantaneous visibility into Docker containers, network flows, IPAM subnets, host telemetry, and container lifecycle operations through modern WebSocket streaming."
  developer: moresonsunn
  author: Orbit Team
  icon: https://raw.githubusercontent.com/moresonsunn/Orbit/main/Apps/Orbit/icon.png
  thumbnail: https://raw.githubusercontent.com/moresonsunn/Orbit/main/Apps/Orbit/icon.png
  screenshot_link:
    - https://raw.githubusercontent.com/moresonsunn/Orbit/main/Apps/Orbit/screenshot-1.png
    - https://raw.githubusercontent.com/moresonsunn/Orbit/main/Apps/Orbit/screenshot-2.png
    - https://raw.githubusercontent.com/moresonsunn/Orbit/main/Apps/Orbit/screenshot-3.png
```

6. Klicke auf **„Übermitteln“** (**Submit**). CasaOS liest alle Metadaten automatisch ein:
   - Name: **Orbit**
   - Web UI Port: **8787**
   - Icon und Beschreibungen werden automatisch konfiguriert.
7. Klicke auf **„Installieren“** (**Install**).
8. Nach dem Download erscheint das Orbit-Icon direkt auf deinem CasaOS Desktop.

---

### Methode 2: Custom AppStore Repository Setup — *Empfohlen für App Store Katalog*

Durch Hinzufügen des Orbit GitHub-Repositorys als Third-Party App Store Quelle erscheint Orbit direkt im grafischen App Store Katalog:

1. Öffne das **CasaOS Dashboard** und klicke auf den **App Store**.
2. Klicke oben auf **„Weitere Apps“** bzw. das Zahnrad-/Quellen-Symbol (**Manage App Stores** / **+ Add Source**).
3. Gib im Eingabefeld für die Store-URL die offizielle GitHub Archive ZIP-URL ein:
   ```text
   https://github.com/moresonsunn/Orbit/archive/refs/heads/main.zip
   ```
4. Klicke auf **„Hinzufügen“** (**Add**).
5. CasaOS lädt das Archiv herunter und indexiert die `Apps/`-Struktur.
6. Orbit ist nun permanent im CasaOS App Store unter der Kategorie **„Developer“** gelistet.
7. Klicke bei Orbit einfach auf **„Installieren“** (**Install**).
8. Zukünftige Updates werden von CasaOS automatisch erkannt und als Update-Hinweis auf dem Dashboard angezeigt.

---

### Methode 3: Manuelle Installation via Terminal (Docker CLI & Compose)

Für reine Linux-Server oder SSH-Administratoren:

```bash
# 1. Verzeichnis für persistente Daten anlegen
mkdir -p /DATA/AppData/orbit/data
cd /DATA/AppData/orbit

# 2. Docker Compose Datei erstellen
cat << 'EOF' > docker-compose.yml
services:
  orbit:
    image: moresonsun/orbit:latest
    container_name: orbit
    restart: unless-stopped
    ports:
      - "8787:8787"
    environment:
      - ORBIT_LISTEN=:8787
      - ORBIT_DATA_DIR=/data
      - ORBIT_ENABLE_ACTIONS=false
      - ORBIT_ENABLE_DESTRUCTIVE=false
      - ORBIT_GPU=auto
      - ORBIT_EBPF=auto
    volumes:
      - /DATA/AppData/orbit/data:/data
      - /var/run/docker.sock:/var/run/docker.sock
      - /proc:/host/proc:ro
      - /sys:/host/sys:ro
      - /etc/os-release:/host/etc/os-release:ro
    security_opt:
      - no-new-privileges:true
EOF

# 3. Orbit starten
docker compose up -d
```

---

## 3. Ersteinrichtung (Setup-Token) / First-Time Setup

Aus Sicherheitsgründen besitzt Orbit **kein Standardpasswort**. Beim allerersten Start wird ein kryptografischer Einmal-Token generiert:

1. Rufe das Container-Log auf:
   ```bash
   docker logs orbit | grep "SETUP TOKEN"
   ```
   Ausgabe:
   ```text
   ==========================================================================
     ORBIT SETUP ERFORDERLICH: Kein Administrator vorhanden!
     Einmaliger Setup-Token für Ersteinrichtung:
     SETUP TOKEN: a1b2c3d4e5f6...
     Verwende diesen Token in der Weboberfläche, um den ersten Admin anzulegen.
   ==========================================================================
   ```
2. Öffne Orbit im Browser unter:
   ```text
   http://<deine-server-ip>:8787
   ```
3. Gib den Setup-Token ein und erstelle deinen Administrator-Benutzernamen und ein sicheres Passwort.
4. Nach dem Speichern wird der Setup-Token unwiderruflich entwertet.

---

## 4. Container-Verwaltungsaktionen freischalten / Enabling Container Actions

Im Standardzustand läuft Orbit im sicheren **Nur-Lese-Modus** (`ORBIT_ENABLE_ACTIONS=false`).

Wenn du Container über die Orbit-Weboberfläche starten, stoppen, neustarten oder löschen möchtest:
1. In CasaOS: Klicke auf die drei Punkte der Orbit-Kachel -> **Einstellungen** (**Settings**).
2. Setze die Umgebungsvariablen:
   - `ORBIT_ENABLE_ACTIONS=true` (Erlaubt Start, Stop, Restart, Pause)
   - `ORBIT_ENABLE_DESTRUCTIVE=true` (Optional: Erlaubt Löschen von Containern, Pruning von Images und Volumes)
3. Speichere die Einstellungen. CasaOS startet den Container automatisch neu.

---

## 5. Hardware- & GPU-Telemetrie / GPU Telemetry

Orbit erkennt installierte GPUs automatisch:
- **NVIDIA**: Unterstützt via NVML. Erfordert das [NVIDIA Container Toolkit](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/latest/install-guide.html) auf dem Host. In der `docker-compose.yml` unter dem Service `orbit` ergänzen:
  ```yaml
      deploy:
        resources:
          reservations:
            devices:
              - driver: nvidia
                count: all
                capabilities: [gpu, utility]
  ```
- **AMD & Intel**: Automatische Telemetrie via Linux DRM Subsystem (`/sys/class/drm`). Keine zusätzlichen Treiber-Container nötig.
