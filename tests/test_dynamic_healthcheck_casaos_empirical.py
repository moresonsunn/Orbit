#!/usr/bin/env python3
"""tests/test_dynamic_healthcheck_casaos_empirical.py

Empirical Adversarial Test Harness for:
1. Dynamic Healthcheck Probing Logic
   - Probing with custom ports (8989, 9999, 12345, bare ports, host:port)
   - Closed port detection (non-zero exit code)
   - Primary endpoint (/api/system/health) 200 OK handling (exit 0)
   - Fallback endpoint (/api/v1/health) 200 OK handling (exit 0)
   - Failure / error / timeout exit codes
   - Environment variable precedence and fallback logic
2. CasaOS Port Remapping & Schema Integrity
   - Official validator execution
   - CasaOS test suite execution
   - Host port remapping simulation and UI reverse-proxy alignment
"""

import http.server
import json
import os
import re
import socket
import socketserver
import subprocess
import sys
import threading
import time
import unittest
from pathlib import Path

import yaml

REPO_ROOT = Path(__file__).resolve().parent.parent
BINARY_PATH = REPO_ROOT / "tests" / "orbit_test.exe"
CASAOS_COMPOSE_PATH = REPO_ROOT / "Apps" / "Orbit" / "docker-compose.yml"


class MockHealthHandler(http.server.BaseHTTPRequestHandler):
    """Configurable mock HTTP server for healthcheck testing."""
    system_health_status = 200
    system_health_body = b'{"status":"ok"}'
    v1_health_status = 200
    v1_health_body = b'{"status":"ok"}'
    sleep_delay = 0

    def log_message(self, format, *args):
        # Suppress logging to keep output clean
        pass

    def do_GET(self):
        try:
            if self.sleep_delay > 0:
                time.sleep(self.sleep_delay)

            if self.path == "/api/system/health":
                self.send_response(self.system_health_status)
                self.send_header("Content-Type", "application/json")
                self.end_headers()
                self.wfile.write(self.system_health_body)
            elif self.path == "/api/v1/health":
                self.send_response(self.v1_health_status)
                self.send_header("Content-Type", "application/json")
                self.end_headers()
                self.wfile.write(self.v1_health_body)
            else:
                self.send_response(404)
                self.send_header("Content-Type", "application/json")
                self.end_headers()
                self.wfile.write(b'{"error":"not found"}')
        except (ConnectionResetError, ConnectionAbortedError, BrokenPipeError):
            pass


class MockServerThread:
    def __init__(self, port, system_status=200, v1_status=200, sleep_delay=0):
        self.port = port
        handler = type(
            "ConfiguredHandler",
            (MockHealthHandler,),
            {
                "system_health_status": system_status,
                "v1_health_status": v1_status,
                "sleep_delay": sleep_delay,
            },
        )
        self.server = socketserver.TCPServer(("127.0.0.1", port), handler)
        self.thread = threading.Thread(target=self.server.serve_forever)
        self.thread.daemon = True

    def start(self):
        self.thread.start()
        # Wait until port is accepting connections
        deadline = time.time() + 3
        while time.time() < deadline:
            try:
                s = socket.create_connection(("127.0.0.1", self.port), timeout=0.2)
                s.close()
                return
            except OSError:
                time.sleep(0.05)

    def stop(self):
        self.server.shutdown()
        self.server.server_close()


def ensure_binary():
    """Ensures test binary is built."""
    if not BINARY_PATH.is_file():
        backend_dir = REPO_ROOT / "backend"
        res = subprocess.run(
            ["go", "build", "-o", str(BINARY_PATH), "./cmd/orbit"],
            cwd=str(backend_dir),
            capture_output=True,
            text=True,
        )
        if res.returncode != 0:
            raise RuntimeError(f"Failed to build orbit test binary: {res.stderr}")


def run_healthcheck(env_overrides=None):
    """Runs orbit_test.exe -healthcheck with specified environment."""
    ensure_binary()
    env = os.environ.copy()
    # Remove any ambient env vars first to guarantee clean slate
    for k in ["ORBIT_LISTEN", "ORBIT_PORT", "PORT"]:
        env.pop(k, None)

    if env_overrides:
        env.update(env_overrides)

    result = subprocess.run(
        [str(BINARY_PATH), "-healthcheck"],
        env=env,
        capture_output=True,
        text=True,
        timeout=10,
    )
    return result.returncode, result.stdout, result.stderr


class TestDynamicHealthcheckEmpirical(unittest.TestCase):
    """Empirical tests verifying healthcheck probe logic against real sockets."""

    @classmethod
    def setUpClass(cls):
        ensure_binary()

    def test_01_closed_port_exits_nonzero_default_port(self):
        """When default port 8787 is closed, healthcheck must return non-zero exit code."""
        code, stdout, stderr = run_healthcheck()
        self.assertNotEqual(code, 0, f"Expected non-zero exit code when port 8787 closed, got {code}")

    def test_02_closed_port_exits_nonzero_custom_orbit_listen(self):
        """When custom port 8989 in ORBIT_LISTEN is closed, healthcheck must return non-zero."""
        code, _, _ = run_healthcheck({"ORBIT_LISTEN": ":8989"})
        self.assertNotEqual(code, 0)

    def test_03_closed_port_exits_nonzero_custom_orbit_port(self):
        """When custom port 9999 in ORBIT_PORT is closed, healthcheck must return non-zero."""
        code, _, _ = run_healthcheck({"ORBIT_PORT": "9999"})
        self.assertNotEqual(code, 0)

    def test_04_closed_port_exits_nonzero_custom_port(self):
        """When custom port 12345 in PORT is closed, healthcheck must return non-zero."""
        code, _, _ = run_healthcheck({"PORT": "12345"})
        self.assertNotEqual(code, 0)

    def test_05_primary_endpoint_200_ok_custom_port_8989(self):
        """Mock server on port 8989 with 200 OK on /api/system/health must exit 0."""
        server = MockServerThread(8989, system_status=200)
        server.start()
        try:
            code, stdout, stderr = run_healthcheck({"ORBIT_LISTEN": ":8989"})
            self.assertEqual(code, 0, f"Healthcheck failed on port 8989: {stderr}")
        finally:
            server.stop()

    def test_06_primary_endpoint_200_ok_bare_port_8989(self):
        """Mock server on port 8989 with bare port ORBIT_LISTEN='8989' must exit 0."""
        server = MockServerThread(8989, system_status=200)
        server.start()
        try:
            code, stdout, stderr = run_healthcheck({"ORBIT_LISTEN": "8989"})
            self.assertEqual(code, 0, f"Bare port normalization failed in healthcheck: {stderr}")
        finally:
            server.stop()

    def test_07_primary_endpoint_200_ok_host_port_8989(self):
        """Mock server on port 8989 with ORBIT_LISTEN='0.0.0.0:8989' must exit 0."""
        server = MockServerThread(8989, system_status=200)
        server.start()
        try:
            code, stdout, stderr = run_healthcheck({"ORBIT_LISTEN": "0.0.0.0:8989"})
            self.assertEqual(code, 0, f"Host:port resolution failed in healthcheck: {stderr}")
        finally:
            server.stop()

    def test_08_primary_endpoint_200_ok_custom_orbit_port_9999(self):
        """Mock server on port 9999 with ORBIT_PORT='9999' must exit 0."""
        server = MockServerThread(9999, system_status=200)
        server.start()
        try:
            code, stdout, stderr = run_healthcheck({"ORBIT_PORT": "9999"})
            self.assertEqual(code, 0, f"ORBIT_PORT resolution failed in healthcheck: {stderr}")
        finally:
            server.stop()

    def test_09_primary_endpoint_200_ok_custom_port_12345(self):
        """Mock server on port 12345 with PORT='12345' must exit 0."""
        server = MockServerThread(12345, system_status=200)
        server.start()
        try:
            code, stdout, stderr = run_healthcheck({"PORT": "12345"})
            self.assertEqual(code, 0, f"PORT env resolution failed in healthcheck: {stderr}")
        finally:
            server.stop()

    def test_10_primary_endpoint_200_ok_default_8787(self):
        """Mock server on standard default port 8787 without env vars must exit 0."""
        server = MockServerThread(8787, system_status=200)
        server.start()
        try:
            code, stdout, stderr = run_healthcheck()
            self.assertEqual(code, 0, f"Default port 8787 healthcheck failed: {stderr}")
        finally:
            server.stop()

    def test_11_fallback_endpoint_200_ok_when_primary_404(self):
        """When /api/system/health returns 404, fallback /api/v1/health returns 200 -> must exit 0."""
        server = MockServerThread(8989, system_status=404, v1_status=200)
        server.start()
        try:
            code, stdout, stderr = run_healthcheck({"ORBIT_LISTEN": ":8989"})
            self.assertEqual(code, 0, f"Fallback to /api/v1/health failed when primary returned 404: {stderr}")
        finally:
            server.stop()

    def test_12_fallback_endpoint_200_ok_when_primary_500(self):
        """When /api/system/health returns 500, fallback /api/v1/health returns 200 -> must exit 0."""
        server = MockServerThread(8989, system_status=500, v1_status=200)
        server.start()
        try:
            code, stdout, stderr = run_healthcheck({"ORBIT_LISTEN": ":8989"})
            self.assertEqual(code, 0, f"Fallback to /api/v1/health failed when primary returned 500: {stderr}")
        finally:
            server.stop()

    def test_13_both_endpoints_fail_exits_nonzero(self):
        """When both /api/system/health and /api/v1/health return 500 -> must exit 1."""
        server = MockServerThread(8989, system_status=500, v1_status=500)
        server.start()
        try:
            code, stdout, stderr = run_healthcheck({"ORBIT_LISTEN": ":8989"})
            self.assertNotEqual(code, 0, "Healthcheck must exit non-zero when both endpoints fail")
        finally:
            server.stop()

    def test_14_both_endpoints_404_exits_nonzero(self):
        """When both /api/system/health and /api/v1/health return 404 -> must exit 1."""
        server = MockServerThread(8989, system_status=404, v1_status=404)
        server.start()
        try:
            code, stdout, stderr = run_healthcheck({"ORBIT_LISTEN": ":8989"})
            self.assertNotEqual(code, 0, "Healthcheck must exit non-zero when both endpoints return 404")
        finally:
            server.stop()

    def test_15_server_timeout_exits_nonzero(self):
        """When mock server response delays past client timeout (2s), healthcheck must exit non-zero."""
        server = MockServerThread(8989, system_status=200, sleep_delay=3)
        server.start()
        try:
            start = time.time()
            code, stdout, stderr = run_healthcheck({"ORBIT_LISTEN": ":8989"})
            elapsed = time.time() - start
            self.assertNotEqual(code, 0, "Healthcheck must exit non-zero on timeout")
            # Timeout is 2s for primary, then 2s for fallback = ~4s
            self.assertGreaterEqual(elapsed, 2.0, "Healthcheck should have waited for timeout")
        finally:
            server.stop()

    def test_16_precedence_orbit_port_overrides_default_listen(self):
        """When ORBIT_LISTEN is ':8787' (default) and ORBIT_PORT is '9999', server on 9999 responds."""
        server = MockServerThread(9999, system_status=200)
        server.start()
        try:
            code, stdout, stderr = run_healthcheck({"ORBIT_LISTEN": ":8787", "ORBIT_PORT": "9999"})
            self.assertEqual(code, 0, "ORBIT_PORT must override default ORBIT_LISTEN in healthcheck")
        finally:
            server.stop()

    def test_17_precedence_custom_listen_overrides_orbit_port(self):
        """When ORBIT_LISTEN is ':8989' (custom) and ORBIT_PORT is '9999', server on 8989 responds."""
        server = MockServerThread(8989, system_status=200)
        server.start()
        try:
            code, stdout, stderr = run_healthcheck({"ORBIT_LISTEN": ":8989", "ORBIT_PORT": "9999"})
            self.assertEqual(code, 0, "Explicit custom ORBIT_LISTEN must take precedence over ORBIT_PORT")
        finally:
            server.stop()

    def test_18_safe_fallback_on_invalid_port_env(self):
        """When PORT='invalid', healthcheck safely falls back to standard 8787 without crashing."""
        server = MockServerThread(8787, system_status=200)
        server.start()
        try:
            code, stdout, stderr = run_healthcheck({"PORT": "invalid-port-string"})
            self.assertEqual(code, 0, "Healthcheck must safely fall back to 8787 on invalid PORT")
        finally:
            server.stop()


class TestCasaOSPortRemappingEmpirical(unittest.TestCase):
    """Empirical tests verifying CasaOS schema compliance and host port remapping mechanics."""

    @classmethod
    def setUpClass(cls):
        cls.compose_text = CASAOS_COMPOSE_PATH.read_text(encoding="utf-8")
        cls.compose_data = yaml.safe_load(cls.compose_text)

    def test_20_casaos_official_validator(self):
        """Run tests/validate_casaos.py on Apps/Orbit/docker-compose.yml."""
        validator_script = REPO_ROOT / "tests" / "validate_casaos.py"
        res = subprocess.run(
            [sys.executable, str(validator_script), str(CASAOS_COMPOSE_PATH)],
            capture_output=True,
            text=True,
        )
        self.assertEqual(
            res.returncode,
            0,
            f"validate_casaos.py failed with exit code {res.returncode}:\n{res.stdout}\n{res.stderr}",
        )
        self.assertIn("completely valid", res.stdout)

    def test_21_casaos_metadata_port_map_quoted_string_8787(self):
        """Verify x-casaos.port_map is explicitly a quoted string '8787'."""
        xcasaos = self.compose_data.get("x-casaos", {})
        port_map = xcasaos.get("port_map")
        self.assertIsInstance(port_map, str, f"x-casaos.port_map must be string, got {type(port_map)}")
        self.assertEqual(port_map, "8787")

    def test_22_service_container_port_aligned_to_8787(self):
        """Verify service orbit ports and x-casaos container ports match 8787."""
        orbit_svc = self.compose_data.get("services", {}).get("orbit", {})
        ports = orbit_svc.get("ports", [])
        self.assertIn("8787:8787", ports, f"Expected '8787:8787' in ports list, got {ports}")

        xcasaos_svc = orbit_svc.get("x-casaos", {})
        svc_ports = xcasaos_svc.get("ports", [])
        self.assertTrue(len(svc_ports) > 0, "Missing services.orbit.x-casaos.ports")
        container_port = svc_ports[0].get("container")
        self.assertEqual(str(container_port), "8787", f"Expected container port '8787', got {container_port}")

    def test_23_service_env_orbit_listen_aligned_to_8787(self):
        """Verify services.orbit.environment.ORBIT_LISTEN is ':8787'."""
        orbit_svc = self.compose_data.get("services", {}).get("orbit", {})
        env = orbit_svc.get("environment", {})
        orbit_listen = env.get("ORBIT_LISTEN")
        self.assertEqual(orbit_listen, ":8787")

    def test_24_simulate_host_port_remapping_to_3000(self):
        """Simulate user changing host port to 3000 in CasaOS Web UI.
        
        When user remaps host port to 3000:
        1. Compose port mapping becomes '3000:8787'.
        2. CasaOS Web UI dashboard reads x-casaos.port_map ('8787'), finds the host port
           bound to container port 8787 (which is now 3000), and creates shortcut:
           http://<casaos_host>:3000/
        3. Inside the container, ORBIT_LISTEN is ':8787', so the container server listens
           on port 8787.
        4. Inside the container, healthcheck probes loopback 127.0.0.1:8787, which is unaffected
           by host port forwarding.
        """
        custom_host_port = 3000
        target_container_port = "8787"

        # Simulate CasaOS port transformation
        simulated_ports = [f"{custom_host_port}:{target_container_port}"]
        
        # Parse port mapping
        host_port, cont_port = simulated_ports[0].split(":")
        self.assertEqual(host_port, "3000")
        self.assertEqual(cont_port, target_container_port)

        # Verify CasaOS UI shortcut derivation
        # CasaOS logic: find port mapping whose container port matches x-casaos.port_map
        casaos_port_map = self.compose_data.get("x-casaos", {}).get("port_map")
        matched_host_port = None
        for p in simulated_ports:
            h, c = p.split(":")
            if c == casaos_port_map:
                matched_host_port = h
                break

        self.assertEqual(
            matched_host_port,
            "3000",
            f"CasaOS failed to route UI to remapped host port 3000 (matched: {matched_host_port})",
        )

        # Scheme and path check
        scheme = self.compose_data.get("x-casaos", {}).get("scheme", "http")
        index = self.compose_data.get("x-casaos", {}).get("index", "/")
        ui_url = f"{scheme}://192.168.1.100:{matched_host_port}{index}"
        self.assertEqual(ui_url, "http://192.168.1.100:3000/")

    def test_25_simulate_multiple_custom_host_ports(self):
        """Simulate remapping across various standard and non-standard host ports."""
        test_ports = [80, 443, 8080, 9090, 18787, 65534]
        casaos_port_map = self.compose_data.get("x-casaos", {}).get("port_map")

        for hp in test_ports:
            mapping = f"{hp}:{casaos_port_map}"
            h, c = mapping.split(":")
            self.assertEqual(c, "8787")
            self.assertEqual(int(h), hp)

    def test_26_simulate_full_container_port_remapping(self):
        """If advanced user overrides container port to 9090 via ORBIT_LISTEN,
        verify that healthcheck resolves dynamic port 9090."""
        # Start mock server on 9090
        server = MockServerThread(9090, system_status=200)
        server.start()
        try:
            code, _, _ = run_healthcheck({"ORBIT_LISTEN": ":9090"})
            self.assertEqual(code, 0, "Container dynamic healthcheck failed when container port remapped to 9090")
        finally:
            server.stop()


if __name__ == "__main__":
    try:
        unittest.main(verbosity=2)
    finally:
        if BINARY_PATH.is_file():
            try:
                BINARY_PATH.unlink()
            except OSError:
                pass
