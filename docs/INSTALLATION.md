# Orbit – Installationsanleitung für CasaOS & Linux Docker Server

Orbit ist eine moderne, schlanke Docker-Infrastruktur- und Echtzeitüberwachungs-Plattform. Sie wurde speziell dafür entwickelt, auf bestehenden CasaOS- und Linux-Servern als einzelner, ressourcenschonender Container betrieben zu werden.

---

## 1. Systemanforderungen

- **Betriebssystem**: CasaOS (Debian/Ubuntu/ZimaOS) oder beliebiges Linux-System mit Docker Engine (20.10+)
- **Architektur**: `linux/amd64` (x86_64) oder `linux/arm64` (Raspberry Pi 4/5, Orange Pi, Apple Silicon)
- **Ressourcenbedarf**: ca. 30–50 MB RAM, minimalste CPU-Nutzung (< 0.5 %)
- **Docker Socket**: `/var/run/docker.sock`

---

## 2. Standard-Installation auf CasaOS (Empfohlen)

### Methode A: Über das CasaOS Web-Dashboard (Benutzerdefinierte App)

1. Öffne dein CasaOS-Dashboard im Browser.
2. Klicke im App Store oben rechts auf **„Benutzerdefinierte App installieren“** (Custom Install).
3. Klicke oben auf **„Importieren“** und füge folgende Compose-Konfiguration ein:

```yaml
name: orbit
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
      - ORBIT_ENABLE_ACTIONS=false       # standardmäßig rein lesendes Monitoring
      - ORBIT_ENABLE_DESTRUCTIVE=false   # separat gesperrt
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

4. Klicke auf **„Installieren“**.

---

### Methode B: Über das Terminal (Docker Compose)

Verbinde dich per SSH mit deinem CasaOS-Server und führe aus:

```bash
mkdir -p /DATA/AppData/orbit/data
cd /DATA/AppData/orbit

# Docker-Compose Datei erstellen
cat << 'EOF' > docker-compose.yml
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
      - ORBIT_ENABLE_ACTIONS=false
      - ORBIT_ENABLE_DESTRUCTIVE=false
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
EOF

# Orbit starten
docker compose up -d
```

---

## 3. Ersteinrichtung (Setup-Token)

Aus Sicherheitsgründen besitzt Orbit **kein Standardpasswort**. Beim ersten Start generiert Orbit einen einmaligen, kryptografischen Setup-Token und schreibt ihn in das Container-Log:

1. Rufe das Container-Log auf:
   ```bash
   docker logs orbit
   ```
   Dort siehst du folgende Ausgabe:
   ```text
   ==========================================================================
     ORBIT SETUP ERFORDERLICH: Kein Administrator vorhanden!
     Einmaliger Setup-Token für Ersteinrichtung:
     SETUP TOKEN: a1b2c3d4e5f6...
     Verwende diesen Token in der Weboberfläche, um den ersten Admin anzulegen.
   ==========================================================================
   ```
2. Öffne Orbit im Browser unter `http://DEINE-SERVER-IP:8787`.
3. Die Weboberfläche zeigt automatisch den Ersteinrichtungs-Assistenten an.
4. Gib den Setup-Token aus dem Log ein und vergebe deinen gewünschten Admin-Benutzernamen und ein sicheres Passwort.
5. Nach dem Erstellen des Kontos wird der Setup-Token dauerhaft ungültig.

---

## 4. Container-Verwaltungsaktionen freischalten

Im Auslieferungszustand läuft Orbit aus Sicherheitsgründen im **reinen Monitoring-Modus** (`ORBIT_ENABLE_ACTIONS=false`).

Wenn du Container über Orbit starten, stoppen, neustarten oder löschen möchtest, aktiviere die Umgebungsvariablen in der `docker-compose.yml`:

```yaml
    environment:
      - ORBIT_ENABLE_ACTIONS=true       # Container starten, stoppen, neustarten
      - ORBIT_ENABLE_DESTRUCTIVE=true   # Container, Images und Volumes löschen / Bereinigen
```

Anschließend neu starten mit `docker compose up -d`.

---

## 5. NVIDIA GPU Telemetrie (Optional)

Wenn dein CasaOS-Server über eine NVIDIA GPU verfügt, installiere das [NVIDIA Container Toolkit](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/latest/install-guide.html) auf dem Host und erweitere die Konfiguration:

```yaml
    deploy:
      resources:
        reservations:
          devices:
            - driver: nvidia
              count: all
              capabilities: [gpu, utility]
```
