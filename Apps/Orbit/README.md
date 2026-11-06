# Orbit — CasaOS App Store Package

Official CasaOS App Store package for **Orbit**, an ultra-fast, self-hosted Docker infrastructure management and real-time monitoring platform.

---

## Overview

Orbit runs as a single lightweight container, connects directly to the Docker daemon (`/var/run/docker.sock`), and provides:
- **Sub-Second Telemetry**: Real-time CPU, RAM, network I/O, and disk metrics.
- **Container Lifecycle**: Start, stop, restart, pause, inspect, and stream live logs.
- **UniFi-Style IPAM & Topology**: Visual subnet allocation and interactive container network flows.
- **Hardware Monitoring**: Automatic detection and metrics for NVIDIA NVML and AMD/Intel DRM GPUs.
- **Multi-Architecture**: Native support for both `amd64` (x86_64) and `arm64` (Raspberry Pi 4/5, ZimaBlade, ZimaBoard).

---

## Installation Options

### Option 1: 1-Click Custom Install (CasaOS Web UI)

You can install Orbit directly on your CasaOS instance using the built-in Custom Install feature:

1. Open your **CasaOS Web Dashboard** (`http://<casaos-ip>`).
2. Click the **App Store** icon.
3. In the top right corner of the App Store, click **Custom Install**.
4. Click the **Import** button in the upper right corner of the modal.
5. Paste the contents of `Apps/Orbit/docker-compose.yml` into the text box (or drag and drop the file).
6. Click **Submit**. CasaOS will parse the configuration, set the container title to **Orbit**, map port **8787**, and configure persistent storage under `/DATA/AppData/orbit/data`.
7. Click **Install**.
8. Once installed, an **Orbit** launcher icon will appear on your CasaOS desktop. Click it to access Orbit at `http://<casaos-ip>:8787`.

---

### Option 2: Custom AppStore Repository Setup

To register Orbit as a browseable application in the CasaOS App Store catalog:

1. Open the **App Store** in your CasaOS dashboard.
2. Click **More Apps** or click the **Manage App Stores / Add Source** icon.
3. In the repository URL field, enter the repository archive ZIP URL:
   ```text
   https://github.com/moresonsunn/Orbit/archive/refs/heads/main.zip
   ```
4. Click **Add**. CasaOS will synchronize the repository and register all apps located under `Apps/`.
5. In the App Store sidebar, navigate to the **Developer** category.
6. Click **Orbit** and select **Install**.
7. Updates are automatically tracked: when a new version is released, CasaOS will show an update badge directly on the Orbit dashboard tile.

---

## First-Time Setup (Setup Token)

For security, Orbit generates **no default credentials**. On initial launch:

1. Open a terminal or SSH into your CasaOS server and retrieve your single-use setup token:
   ```bash
   docker logs orbit | grep "SETUP TOKEN"
   ```
2. Open `http://<casaos-ip>:8787` in your web browser.
3. Paste the setup token and create your initial administrator username and password.
4. Once completed, the setup token is permanently invalidated.

---

## Configuration & Container Actions

By default, Orbit operates in read-only monitoring mode (`ORBIT_ENABLE_ACTIONS=false`).

To allow starting, stopping, restarting, or managing containers via the web dashboard:
1. Open the app settings for Orbit in the CasaOS UI (three dots on the Orbit tile -> **Settings**).
2. Set the environment variables:
   - `ORBIT_ENABLE_ACTIONS`: `true`
   - `ORBIT_ENABLE_DESTRUCTIVE`: `true` (optional, permits container and volume deletion)
3. Click **Save**.
